package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/shared"
	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/workflow"
	"github.com/LaFabric-LinkTIC/FlowEngine/internal/infrastructure/cache"
)

const (
	workflowCachePrefix = "wf:"
	workflowCacheTTL    = 5 * time.Minute

	// workflowColumns is the canonical SELECT column list for workflow queries.
	// Must match workflowRow scan order.
	workflowColumns = `id, name, description, version, created_at, updated_at, states, events`
)

// workflowRow holds the raw column values scanned from a workflows row.
type workflowRow struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Version     string    `json:"version"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	States      []byte    `json:"states"`
	Events      []byte    `json:"events"`
}

type WorkflowRepository struct {
	db    *pgxpool.Pool
	cache cache.Cache
}

// NewWorkflowRepository creates a new WorkflowRepository.
// It now accepts a cache.Cache implementation.
func NewWorkflowRepository(db *pgxpool.Pool, cache cache.Cache) *WorkflowRepository {
	return &WorkflowRepository{db: db, cache: cache}
}

func (r *WorkflowRepository) Save(ctx context.Context, w *workflow.Workflow) error {
	st := w.States()
	statesJSON, err := json.Marshal(st)
	if err != nil {
		return fmt.Errorf("failed to marshal states: %w", err)
	}
	if string(statesJSON) == "null" || string(statesJSON) == "[{},{}]" {
		return fmt.Errorf("failed to marshal states: produced empty or null JSON for %d states", len(st))
	}

	eventsJSON, err := json.Marshal(w.Events())
	if err != nil {
		return fmt.Errorf("failed to marshal events: %w", err)
	}

	query := `
		INSERT INTO workflows (id, name, description, version, created_at, updated_at, states, events)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			version = EXCLUDED.version,
			updated_at = EXCLUDED.updated_at,
			states = EXCLUDED.states,
			events = EXCLUDED.events
	`

	_, err = r.db.Exec(ctx, query,
		w.ID().String(),
		w.Name(),
		w.Description(),
		w.Version().String(),
		w.CreatedAt().Time(),
		w.UpdatedAt().Time(),
		statesJSON,
		eventsJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to save workflow: %w", err)
	}

	// Invalidate cache after write
	if r.cache != nil {
		_ = r.cache.Del(ctx, workflowCachePrefix+w.ID().String())
	}

	return nil
}

// scanWorkflow scans a single row into a workflowRow and converts it to a domain Workflow.
func (r *WorkflowRepository) scanWorkflow(row pgx.Row) (*workflow.Workflow, error) {
	var wr workflowRow
	err := row.Scan(
		&wr.ID, &wr.Name, &wr.Description, &wr.Version,
		&wr.CreatedAt, &wr.UpdatedAt, &wr.States, &wr.Events,
	)
	if err != nil {
		return nil, err
	}
	return r.mapToWorkflow(wr)
}

// scanWorkflows scans multiple rows into workflow domain objects.
func (r *WorkflowRepository) scanWorkflows(rows pgx.Rows) ([]*workflow.Workflow, error) {
	defer rows.Close()
	var workflows []*workflow.Workflow

	for rows.Next() {
		var wr workflowRow
		if err := rows.Scan(
			&wr.ID, &wr.Name, &wr.Description, &wr.Version,
			&wr.CreatedAt, &wr.UpdatedAt, &wr.States, &wr.Events,
		); err != nil {
			return nil, fmt.Errorf("failed to scan workflow: %w", err)
		}

		wf, err := r.mapToWorkflow(wr)
		if err != nil {
			return nil, err
		}
		workflows = append(workflows, wf)
	}

	return workflows, nil
}

func (r *WorkflowRepository) FindByID(ctx context.Context, id shared.ID) (*workflow.Workflow, error) {
	cacheKey := workflowCachePrefix + id.String()

	// 1. Try to get from cache (read-through)
	if r.cache != nil {
		cachedJSON, err := r.cache.Get(ctx, cacheKey)
		if err == nil && cachedJSON != "" {
			var wr workflowRow
			if err := json.Unmarshal([]byte(cachedJSON), &wr); err == nil {
				return r.mapToWorkflow(wr)
			}
			// Unmarshal failed: fall through to DB path silently.
		}
	}

	// 2. If not in cache, get from DB
	query := `SELECT ` + workflowColumns + ` FROM workflows WHERE id = $1 AND deleted_at IS NULL`

	var wr workflowRow
	err := r.db.QueryRow(ctx, query, id.String()).Scan(
		&wr.ID, &wr.Name, &wr.Description, &wr.Version,
		&wr.CreatedAt, &wr.UpdatedAt, &wr.States, &wr.Events,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, workflow.ErrNotFound
		}
		return nil, fmt.Errorf("failed to find workflow: %w", err)
	}

	// 3. Store raw DB result in cache for future requests
	if r.cache != nil {
		wrJSON, marshalErr := json.Marshal(wr)
		if marshalErr == nil {
			_ = r.cache.Set(ctx, cacheKey, string(wrJSON), workflowCacheTTL)
		}
	}

	return r.mapToWorkflow(wr)
}

func (r *WorkflowRepository) mapToWorkflow(wr workflowRow) (*workflow.Workflow, error) {
	wfID, _ := shared.ParseID(wr.ID)

	wfVersion, err := workflow.ParseVersion(wr.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to parse workflow version %q: %w", wr.Version, err)
	}

	var statesSlice []workflow.State
	if err := json.Unmarshal(wr.States, &statesSlice); err != nil {
		return nil, fmt.Errorf("failed to unmarshal states: %w", err)
	}

	statesMap := make(map[string]workflow.State)
	var initialState workflow.State
	if len(statesSlice) > 0 {
		initialState = statesSlice[0]
	}
	for _, s := range statesSlice {
		statesMap[s.ID] = s
	}

	var eventsSlice []workflow.Event
	if err := json.Unmarshal(wr.Events, &eventsSlice); err != nil {
		return nil, fmt.Errorf("failed to unmarshal events: %w", err)
	}
	eventsMap := make(map[string]workflow.Event)
	for _, e := range eventsSlice {
		eventsMap[e.Name] = e
	}

	return workflow.RestoreWorkflow(
		wfID,
		wr.Name,
		wfVersion,
		wr.Description,
		initialState,
		statesMap,
		eventsMap,
		shared.From(wr.CreatedAt),
		shared.From(wr.UpdatedAt),
	), nil
}

func (r *WorkflowRepository) FindAll(ctx context.Context) ([]*workflow.Workflow, error) {
	query := `SELECT ` + workflowColumns + ` FROM workflows WHERE deleted_at IS NULL`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query all workflows: %w", err)
	}
	return r.scanWorkflows(rows)
}

func (r *WorkflowRepository) FindByName(ctx context.Context, name string, version workflow.Version) (*workflow.Workflow, error) {
	query := `SELECT ` + workflowColumns + ` FROM workflows WHERE name = $1 AND version = $2 AND deleted_at IS NULL`

	wf, err := r.scanWorkflow(r.db.QueryRow(ctx, query, name, version.String()))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, workflow.ErrNotFound
		}
		return nil, fmt.Errorf("failed to find workflow by name: %w", err)
	}
	return wf, nil
}

func (r *WorkflowRepository) FindLatestByName(ctx context.Context, name string) (*workflow.Workflow, error) {
	query := `SELECT ` + workflowColumns + ` FROM workflows WHERE name = $1 AND deleted_at IS NULL ORDER BY created_at DESC LIMIT 1`

	wf, err := r.scanWorkflow(r.db.QueryRow(ctx, query, name))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, workflow.ErrNotFound
		}
		return nil, fmt.Errorf("failed to find latest workflow by name: %w", err)
	}
	return wf, nil
}

func (r *WorkflowRepository) FindAllByName(ctx context.Context, name string) ([]*workflow.Workflow, error) {
	query := `SELECT ` + workflowColumns + ` FROM workflows WHERE name = $1 AND deleted_at IS NULL ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, query, name)
	if err != nil {
		return nil, fmt.Errorf("failed to query workflows by name: %w", err)
	}
	return r.scanWorkflows(rows)
}

func (r *WorkflowRepository) List(ctx context.Context, q shared.ListQuery) ([]*workflow.Workflow, int64, error) {
	query := `
		SELECT ` + workflowColumns + `,
		       COUNT(*) OVER() AS total_count
		FROM workflows
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(ctx, query, q.Limit, q.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list workflows: %w", err)
	}
	defer rows.Close()

	var workflows []*workflow.Workflow
	var totalCount int64

	for rows.Next() {
		var wr workflowRow
		var rowTotal int64

		if err := rows.Scan(
			&wr.ID, &wr.Name, &wr.Description, &wr.Version,
			&wr.CreatedAt, &wr.UpdatedAt, &wr.States, &wr.Events,
			&rowTotal,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan workflow: %w", err)
		}

		totalCount = rowTotal
		wf, err := r.mapToWorkflow(wr)
		if err != nil {
			return nil, 0, err
		}
		workflows = append(workflows, wf)
	}

	if len(workflows) == 0 {
		var count int64
		_ = r.db.QueryRow(ctx, "SELECT COUNT(*) FROM workflows WHERE deleted_at IS NULL").Scan(&count)
		totalCount = count
	}

	return workflows, totalCount, nil
}

func (r *WorkflowRepository) Delete(ctx context.Context, id shared.ID) error {
	query := `UPDATE workflows SET deleted_at = $1 WHERE id = $2`

	cmdTag, err := r.db.Exec(ctx, query, time.Now(), id.String())
	if err != nil {
		return fmt.Errorf("failed to delete workflow: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return workflow.ErrNotFound
	}

	if r.cache != nil {
		_ = r.cache.Del(ctx, workflowCachePrefix+id.String())
	}

	return nil
}

func (r *WorkflowRepository) Exists(ctx context.Context, id shared.ID) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM workflows WHERE id = $1 AND deleted_at IS NULL)`
	err := r.db.QueryRow(ctx, query, id.String()).Scan(&exists)
	return exists, err
}

func (r *WorkflowRepository) ExistsByName(ctx context.Context, name string, version workflow.Version) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM workflows WHERE name = $1 AND version = $2 AND deleted_at IS NULL)`
	err := r.db.QueryRow(ctx, query, name, version.String()).Scan(&exists)
	return exists, err
}
