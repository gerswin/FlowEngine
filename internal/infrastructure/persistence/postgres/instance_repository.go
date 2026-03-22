package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/instance"
	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/shared"
	"github.com/LaFabric-LinkTIC/FlowEngine/internal/infrastructure/cache" // Import cache package
)

const (
	instanceCachePrefix = "inst:"
	instanceCacheTTL    = 1 * time.Minute // Shorter TTL for instances due to frequent changes

	// instanceColumns is the canonical SELECT column list for instance queries.
	// It JOINs workflows to resolve the workflow name. Must match instanceRow scan order.
	instanceColumns = `wi.id, wi.parent_id, wi.workflow_id, w.name, wi.current_state, wi.current_sub_state, wi.status, wi.version, wi.created_at, wi.updated_at, wi.completed_at, wi.data, wi.variables, wi.current_actor`

	// instanceFrom is the FROM clause with the workflows JOIN.
	instanceFrom = `FROM workflow_instances wi LEFT JOIN workflows w ON wi.workflow_id = w.id`
)

// instanceRow holds the raw column values scanned from a workflow_instances row.
type instanceRow struct {
	ID              string
	ParentID        *string
	WorkflowID      string
	WorkflowName    string // Resolved via JOIN on workflows table
	CurrentState    string
	CurrentSubState *string
	Status          string
	Version         int
	CreatedAt       time.Time
	UpdatedAt       time.Time
	CompletedAt     *time.Time
	Data            []byte
	Variables       []byte
	CurrentActor    *string
}

type InstanceRepository struct {
	db    *pgxpool.Pool
	cache cache.Cache // Add cache dependency
}

func NewInstanceRepository(db *pgxpool.Pool, cache cache.Cache) *InstanceRepository {
	return &InstanceRepository{db: db, cache: cache}
}

func (r *InstanceRepository) Save(ctx context.Context, i *instance.Instance) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Serialize JSONB fields
	dataJSON, err := json.Marshal(i.Data().ToMap())
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}
	variablesJSON, err := json.Marshal(i.Variables().ToMap())
	if err != nil {
		return fmt.Errorf("failed to marshal variables: %w", err)
	}

	// Determine if Insert or Update based on Version
	currentVersion := int(i.Version().Value())

	if currentVersion == 1 {
		// Insert
		query := `
			INSERT INTO workflow_instances (
				id, parent_id, workflow_id, current_state, current_sub_state, status, version,
				created_at, updated_at, completed_at, data, variables,
				current_actor
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		`
		var subStateID *string
		if i.HasSubState() {
			s := i.CurrentSubState().ID()
			subStateID = &s
		}

		var completedAt *time.Time
		if !i.CompletedAt().IsZero() {
			t := i.CompletedAt().Time()
			completedAt = &t
		}

		var currentActor *string
		if i.StartedBy().IsValid() {
			s := i.StartedBy().String()
			currentActor = &s
		}

		var parentID *string
		if i.ParentID().IsValid() {
			s := i.ParentID().String()
			parentID = &s
		}

		_, err = tx.Exec(ctx, query,
			i.ID().String(),
			parentID,
			i.WorkflowID().String(),
			i.CurrentState(),
			subStateID,
			i.Status().String(),
			currentVersion,
			i.CreatedAt().Time(),
			i.UpdatedAt().Time(),
			completedAt,
			dataJSON,
			variablesJSON,
			currentActor,
		)
		if err != nil {
			return fmt.Errorf("failed to insert instance: %w", err)
		}

	} else {
		// Update
		expectedVersion := currentVersion - 1
		query := `
			UPDATE workflow_instances SET
				current_state = $1,
				current_sub_state = $2,
				status = $3,
				version = $4,
				updated_at = $5,
				completed_at = $6,
				data = $7,
				variables = $8
			WHERE id = $9 AND version = $10
		`

		var subStateID *string
		if i.HasSubState() {
			s := i.CurrentSubState().ID()
			subStateID = &s
		}

		var completedAt *time.Time
		if !i.CompletedAt().IsZero() {
			t := i.CompletedAt().Time()
			completedAt = &t
		}

		cmdTag, err := tx.Exec(ctx, query,
			i.CurrentState(),
			subStateID,
			i.Status().String(),
			currentVersion,
			i.UpdatedAt().Time(),
			completedAt,
			dataJSON,
			variablesJSON,
			i.ID().String(),
			expectedVersion,
		)
		if err != nil {
			return fmt.Errorf("failed to update instance: %w", err)
		}

		if cmdTag.RowsAffected() == 0 {
			var exists bool
			checkQuery := `SELECT EXISTS(SELECT 1 FROM workflow_instances WHERE id = $1)`
			_ = tx.QueryRow(ctx, checkQuery, i.ID().String()).Scan(&exists)

			if !exists {
				return instance.ErrInstanceNotFound
			}
			return instance.ErrVersionMismatch
		}
	}

	// Save Transitions
	transitions := i.Transitions()
	for _, t := range transitions {
		tQuery := `
			INSERT INTO workflow_transitions (
				id, instance_id, event, from_state, to_state,
				from_sub_state, to_sub_state,
				actor, actor_role, data, created_at, reason, feedback
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
			ON CONFLICT (id) DO NOTHING
		`

		tDataJSON, err := json.Marshal(t.Data().ToMap())
		if err != nil {
			return fmt.Errorf("failed to marshal transition data for %s: %w", t.ID(), err)
		}

		var fromSub, toSub *string
		if !t.FromSubState().IsZero() {
			s := t.FromSubState().ID()
			fromSub = &s
		}
		if !t.ToSubState().IsZero() {
			s := t.ToSubState().ID()
			toSub = &s
		}

		meta := t.Metadata()
		reason := meta.Reason()
		feedback := meta.Feedback()

		_, err = tx.Exec(ctx, tQuery,
			t.ID().String(),
			i.ID().String(),
			t.Event(),
			t.From(),
			t.To(),
			fromSub,
			toSub,
			t.Actor().String(),
			"",
			tDataJSON,
			t.Timestamp().Time(),
			reason,
			feedback,
		)
		if err != nil {
			return fmt.Errorf("failed to save transition %s: %w", t.ID(), err)
		}
	}

	if r.cache != nil {
		_ = r.cache.Del(ctx, instanceCachePrefix+i.ID().String()) // Invalidate instance cache
	}

	return tx.Commit(ctx)
}

func (r *InstanceRepository) FindByID(ctx context.Context, id shared.ID) (*instance.Instance, error) {
	cacheKey := instanceCachePrefix + id.String()

	// 1. Try to get from cache (read-through)
	if r.cache != nil {
		cachedIDataJSON, err := r.cache.Get(ctx, cacheKey)
		if err == nil && cachedIDataJSON != "" {
			var iData instanceRow
			if err := json.Unmarshal([]byte(cachedIDataJSON), &iData); err == nil {
				transitions, err := r.loadTransitions(ctx, iData.ID)
				if err != nil {
					return nil, err
				}
				return r.mapToInstance(iData, transitions)
			}
			// Unmarshal failed: fall through to DB path silently.
		}
	}

	// 2. If not in cache, get from DB
	query := `
		SELECT ` + instanceColumns + `
		` + instanceFrom + `
		WHERE wi.id = $1
	`

	var iData instanceRow

	err := r.db.QueryRow(ctx, query, id.String()).Scan(
		&iData.ID,
		&iData.ParentID,
		&iData.WorkflowID,
		&iData.WorkflowName,
		&iData.CurrentState,
		&iData.CurrentSubState,
		&iData.Status,
		&iData.Version,
		&iData.CreatedAt,
		&iData.UpdatedAt,
		&iData.CompletedAt,
		&iData.Data,
		&iData.Variables,
		&iData.CurrentActor,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, instance.ErrInstanceNotFound
		}
		return nil, fmt.Errorf("failed to find instance: %w", err)
	}

	// 3. Store raw DB result in cache for future requests
	if r.cache != nil {
		iDataJSON, marshalErr := json.Marshal(iData)
		if marshalErr == nil {
			_ = r.cache.Set(ctx, cacheKey, string(iDataJSON), instanceCacheTTL)
		}
		// Marshalling failure is non-fatal; the instance is still returned.
	}

	// Load Transitions
	transitions, err := r.loadTransitions(ctx, id.String())
	if err != nil {
		return nil, err
	}

	return r.mapToInstance(iData, transitions)
}

func (r *InstanceRepository) scanInstances(ctx context.Context, rows pgx.Rows) ([]*instance.Instance, error) {
	defer rows.Close()
	var instances []*instance.Instance

	for rows.Next() {
		var iData instanceRow

		if err := rows.Scan(
			&iData.ID, &iData.ParentID, &iData.WorkflowID, &iData.WorkflowName,
			&iData.CurrentState, &iData.CurrentSubState,
			&iData.Status, &iData.Version, &iData.CreatedAt, &iData.UpdatedAt,
			&iData.CompletedAt, &iData.Data, &iData.Variables, &iData.CurrentActor,
		); err != nil {
			return nil, fmt.Errorf("failed to scan instance: %w", err)
		}

		inst, err := r.mapToInstance(iData, nil)
		if err != nil {
			return nil, err
		}
		instances = append(instances, inst)
	}

	return instances, nil
}

func (r *InstanceRepository) mapToInstance(row instanceRow, transitions []*instance.Transition) (*instance.Instance, error) {
	instID, _ := shared.ParseID(row.ID)
	wfID, _ := shared.ParseID(row.WorkflowID)

	pID := shared.NilID()
	if row.ParentID != nil {
		pID, _ = shared.ParseID(*row.ParentID)
	}

	actorID := shared.NilID()
	if row.CurrentActor != nil {
		actorID, _ = shared.ParseID(*row.CurrentActor)
	}

	var subState instance.SubState
	if row.CurrentSubState != nil {
		subState = instance.RestoreSubState(*row.CurrentSubState)
	}

	instStatus := instance.Status(row.Status)
	instVersion, _ := instance.FromValue(int64(row.Version))

	var dataMap map[string]interface{}
	if err := json.Unmarshal(row.Data, &dataMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal instance data for %s: %w", row.ID, err)
	}
	data := instance.NewDataFromMap(dataMap)

	var varsMap map[string]interface{}
	if err := json.Unmarshal(row.Variables, &varsMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal instance variables for %s: %w", row.ID, err)
	}
	variables := instance.NewVariablesFromMap(varsMap)

	tsCreatedAt := shared.From(row.CreatedAt)
	tsUpdatedAt := shared.From(row.UpdatedAt)
	tsCompletedAt := shared.Timestamp{}
	if row.CompletedAt != nil {
		tsCompletedAt = shared.From(*row.CompletedAt)
	}

	if transitions == nil {
		transitions = []*instance.Transition{}
	}

	return instance.RestoreInstance(
		instID, pID, wfID, row.WorkflowName,
		row.CurrentState, subState, instStatus, instVersion,
		data, variables, transitions,
		tsCreatedAt, tsUpdatedAt, tsCompletedAt, actorID,
	), nil
}

func (r *InstanceRepository) loadTransitions(ctx context.Context, instanceID string) ([]*instance.Transition, error) {
	query := `
		SELECT id, event, from_state, to_state, from_sub_state, to_sub_state,
		       actor, data, created_at, reason, feedback
		FROM workflow_transitions
		WHERE instance_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.db.Query(ctx, query, instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query transitions: %w", err)
	}
	defer rows.Close()

	var transitions []*instance.Transition
	for rows.Next() {
		var tData struct {
			ID           string
			Event        string
			FromState    string
			ToState      string
			FromSubState *string
			ToSubState   *string
			Actor        string
			Data         []byte
			CreatedAt    time.Time
			Reason       *string
			Feedback     *string
		}

		if err := rows.Scan(
			&tData.ID, &tData.Event, &tData.FromState, &tData.ToState,
			&tData.FromSubState, &tData.ToSubState, &tData.Actor, &tData.Data,
			&tData.CreatedAt, &tData.Reason, &tData.Feedback,
		); err != nil {
			return nil, err
		}

		tID, _ := shared.ParseID(tData.ID)
		actor, _ := shared.ParseID(tData.Actor)

		var fromSub, toSub instance.SubState
		if tData.FromSubState != nil {
			fromSub = instance.RestoreSubState(*tData.FromSubState)
		}
		if tData.ToSubState != nil {
			toSub = instance.RestoreSubState(*tData.ToSubState)
		}

		var tDataMap map[string]interface{}
		if err := json.Unmarshal(tData.Data, &tDataMap); err != nil {
			return nil, fmt.Errorf("failed to unmarshal transition data for %s: %w", tData.ID, err)
		}

		tr := instance.RestoreTransition(
			tID, tData.FromState, tData.ToState, tData.Event, actor,
			fromSub, toSub,
			instance.NewDataFromMap(tDataMap),
			shared.From(tData.CreatedAt),
		)

		meta := make(map[string]interface{})
		// Populate meta map if extra fields exist

		var reason, feedback string
		if tData.Reason != nil {
			reason = *tData.Reason
		}
		if tData.Feedback != nil {
			feedback = *tData.Feedback
		}

		tr.SetMetadata(instance.NewTransitionMetadata(reason, feedback, meta))

		transitions = append(transitions, tr)
	}

	return transitions, nil
}

func (r *InstanceRepository) FindByWorkflowID(ctx context.Context, workflowID shared.ID) ([]*instance.Instance, error) {
	query := `
		SELECT ` + instanceColumns + `
		` + instanceFrom + `
		WHERE wi.workflow_id = $1 AND wi.status != 'deleted'
	`
	rows, err := r.db.Query(ctx, query, workflowID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to query instances by workflow: %w", err)
	}
	return r.scanInstances(ctx, rows)
}

func (r *InstanceRepository) FindByParentID(ctx context.Context, parentID shared.ID) ([]*instance.Instance, error) {
	query := `
		SELECT ` + instanceColumns + `
		` + instanceFrom + `
		WHERE wi.parent_id = $1 AND wi.status != 'deleted'
	`
	rows, err := r.db.Query(ctx, query, parentID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to query instances by parent ID: %w", err)
	}
	return r.scanInstances(ctx, rows)
}

func (r *InstanceRepository) FindByStatus(ctx context.Context, status instance.Status) ([]*instance.Instance, error) {
	query := `
		SELECT ` + instanceColumns + `
		` + instanceFrom + `
		WHERE wi.status = $1
	`
	rows, err := r.db.Query(ctx, query, status.String())
	if err != nil {
		return nil, fmt.Errorf("failed to query instances by status: %w", err)
	}
	return r.scanInstances(ctx, rows)
}

func (r *InstanceRepository) FindByWorkflowIDAndStatus(ctx context.Context, workflowID shared.ID, status instance.Status) ([]*instance.Instance, error) {
	query := `
		SELECT ` + instanceColumns + `
		` + instanceFrom + `
		WHERE wi.workflow_id = $1 AND wi.status = $2
	`
	rows, err := r.db.Query(ctx, query, workflowID.String(), status.String())
	if err != nil {
		return nil, fmt.Errorf("failed to query instances by workflow and status: %w", err)
	}
	return r.scanInstances(ctx, rows)
}

func (r *InstanceRepository) FindActive(ctx context.Context) ([]*instance.Instance, error) {
	query := `
		SELECT ` + instanceColumns + `
		` + instanceFrom + `
		WHERE wi.status IN ('running', 'paused')
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query active instances: %w", err)
	}
	return r.scanInstances(ctx, rows)
}

func (r *InstanceRepository) FindAll(ctx context.Context) ([]*instance.Instance, error) {
	query := `
		SELECT ` + instanceColumns + `
		` + instanceFrom + `
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query all instances: %w", err)
	}
	return r.scanInstances(ctx, rows)
}

func (r *InstanceRepository) List(ctx context.Context, q shared.ListQuery, workflowID *shared.ID) ([]*instance.Instance, int64, error) {
	var args []interface{}
	argIdx := 1

	where := ""
	if workflowID != nil {
		where = fmt.Sprintf("WHERE wi.workflow_id = $%d", argIdx)
		args = append(args, workflowID.String())
		argIdx++
	}

	query := fmt.Sprintf(`
		SELECT %s, COUNT(*) OVER() AS total_count
		%s
		%s
		ORDER BY wi.created_at DESC
		LIMIT $%d OFFSET $%d
	`, instanceColumns, instanceFrom, where, argIdx, argIdx+1)
	args = append(args, q.Limit, q.Offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list instances: %w", err)
	}
	defer rows.Close()

	var instances []*instance.Instance
	var totalCount int64

	for rows.Next() {
		var iData instanceRow
		var rowTotal int64

		if err := rows.Scan(
			&iData.ID, &iData.ParentID, &iData.WorkflowID, &iData.WorkflowName,
			&iData.CurrentState, &iData.CurrentSubState,
			&iData.Status, &iData.Version, &iData.CreatedAt, &iData.UpdatedAt,
			&iData.CompletedAt, &iData.Data, &iData.Variables, &iData.CurrentActor,
			&rowTotal,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan instance: %w", err)
		}

		totalCount = rowTotal
		inst, err := r.mapToInstance(iData, nil)
		if err != nil {
			return nil, 0, err
		}
		instances = append(instances, inst)
	}

	// If no rows returned, get the count separately for correct total
	if len(instances) == 0 {
		countQuery := "SELECT COUNT(*) FROM workflow_instances wi"
		if workflowID != nil {
			countQuery += " WHERE wi.workflow_id = $1"
			_ = r.db.QueryRow(ctx, countQuery, workflowID.String()).Scan(&totalCount)
		} else {
			_ = r.db.QueryRow(ctx, countQuery).Scan(&totalCount)
		}
	}

	return instances, totalCount, nil
}

func (r *InstanceRepository) Delete(ctx context.Context, id shared.ID) error {
	query := `DELETE FROM workflow_instances WHERE id = $1`
	cmdTag, err := r.db.Exec(ctx, query, id.String())
	if err != nil {
		return fmt.Errorf("failed to delete instance: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return instance.ErrInstanceNotFound
	}

	if r.cache != nil {
		_ = r.cache.Del(ctx, instanceCachePrefix+id.String())
	}

	return nil
}

func (r *InstanceRepository) Exists(ctx context.Context, id shared.ID) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM workflow_instances WHERE id = $1)`
	err := r.db.QueryRow(ctx, query, id.String()).Scan(&exists)
	return exists, err
}

func (r *InstanceRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM workflow_instances`
	err := r.db.QueryRow(ctx, query).Scan(&count)
	return count, err
}

func (r *InstanceRepository) CountByWorkflowID(ctx context.Context, workflowID shared.ID) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM workflow_instances WHERE workflow_id = $1`
	err := r.db.QueryRow(ctx, query, workflowID.String()).Scan(&count)
	return count, err
}

func (r *InstanceRepository) CountByStatus(ctx context.Context, status instance.Status) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM workflow_instances WHERE status = $1`
	err := r.db.QueryRow(ctx, query, status.String()).Scan(&count)
	return count, err
}
