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
)

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
			
					tDataJSON, _ := json.Marshal(t.Data().ToMap())
					
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
					
							_, err = tx.Exec(ctx, tQuery,						t.ID().String(),
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
			var iData struct {
				ID              string
				ParentID        *string
				WorkflowID      string
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
			if err := json.Unmarshal([]byte(cachedIDataJSON), &iData); err == nil {
				transitions, err := r.loadTransitions(ctx, iData.ID)
				if err != nil {
					return nil, err
				}
				return r.mapToInstance(iData.ID, iData.ParentID, iData.WorkflowID, iData.CurrentState, iData.CurrentSubState, 
					iData.Status, iData.Version, iData.CreatedAt, iData.UpdatedAt, iData.CompletedAt, 
					iData.Data, iData.Variables, iData.CurrentActor, transitions), nil
			}
			fmt.Printf("Error unmarshalling cached instance iData %s: %v\n", id.String(), err)
		}
	}

	// 2. If not in cache, get from DB
	query := `
		SELECT id, parent_id, workflow_id, current_state, current_sub_state, status, version, 
		       created_at, updated_at, completed_at, data, variables, current_actor 
		FROM workflow_instances 
		WHERE id = $1
	`

	var iData struct {
		ID              string
		ParentID        *string
		WorkflowID      string
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

	err := r.db.QueryRow(ctx, query, id.String()).Scan(
		&iData.ID,
		&iData.ParentID,
		&iData.WorkflowID,
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
		} else {
			fmt.Printf("Error marshalling instance iData %s for cache: %v\n", id.String(), marshalErr)
		}
	}

	// Load Transitions
	transitions, err := r.loadTransitions(ctx, id.String())
	if err != nil {
		return nil, err
	}

	return r.mapToInstance(iData.ID, iData.ParentID, iData.WorkflowID, iData.CurrentState, iData.CurrentSubState, 
		iData.Status, iData.Version, iData.CreatedAt, iData.UpdatedAt, iData.CompletedAt, 
		iData.Data, iData.Variables, iData.CurrentActor, transitions), nil
}

func (r *InstanceRepository) scanInstances(ctx context.Context, rows pgx.Rows) ([]*instance.Instance, error) {
	defer rows.Close()
	var instances []*instance.Instance

	for rows.Next() {
		var iData struct {
			ID              string
			ParentID        *string
			WorkflowID      string
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

		if err := rows.Scan(
			&iData.ID, &iData.ParentID, &iData.WorkflowID, &iData.CurrentState, &iData.CurrentSubState,
			&iData.Status, &iData.Version, &iData.CreatedAt, &iData.UpdatedAt,
			&iData.CompletedAt, &iData.Data, &iData.Variables, &iData.CurrentActor,
		); err != nil {
			return nil, fmt.Errorf("failed to scan instance: %w", err)
		}

		instances = append(instances, r.mapToInstance(iData.ID, iData.ParentID, iData.WorkflowID, iData.CurrentState, iData.CurrentSubState, 
			iData.Status, iData.Version, iData.CreatedAt, iData.UpdatedAt, iData.CompletedAt, 
			iData.Data, iData.Variables, iData.CurrentActor, nil))
	}

	return instances, nil
}

func (r *InstanceRepository) mapToInstance(
	id string, parentID *string, workflowID, currentState string, currentSubState *string,
	status string, version int, createdAt, updatedAt time.Time, completedAt *time.Time,
	dataBytes, varsBytes []byte, currentActor *string, transitions []*instance.Transition,
) *instance.Instance {
	instID, _ := shared.ParseID(id)
	wfID, _ := shared.ParseID(workflowID)
	
	pID := shared.NilID()
	if parentID != nil {
		pID, _ = shared.ParseID(*parentID)
	}
	
	actorID := shared.NilID()
	if currentActor != nil {
		actorID, _ = shared.ParseID(*currentActor)
	}
	
	var subState instance.SubState
	if currentSubState != nil {
		subState = instance.RestoreSubState(*currentSubState)
	}

	instStatus := instance.Status(status)
	instVersion, _ := instance.FromValue(int64(version))
	
	var dataMap map[string]interface{}
	_ = json.Unmarshal(dataBytes, &dataMap)
	data := instance.NewDataFromMap(dataMap)

	var varsMap map[string]interface{}
	_ = json.Unmarshal(varsBytes, &varsMap)
	variables := instance.NewVariablesFromMap(varsMap)

	tsCreatedAt := shared.From(createdAt)
	tsUpdatedAt := shared.From(updatedAt)
	tsCompletedAt := shared.Timestamp{}
	if completedAt != nil {
		tsCompletedAt = shared.From(*completedAt)
	}

	if transitions == nil {
		transitions = []*instance.Transition{}
	}

	return instance.RestoreInstance(
		instID, pID, wfID, "unknown", 
		currentState, subState, instStatus, instVersion,
		data, variables, transitions,
		tsCreatedAt, tsUpdatedAt, tsCompletedAt, actorID,
	)
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
		if tData.FromSubState != nil { fromSub = instance.RestoreSubState(*tData.FromSubState) }
		if tData.ToSubState != nil { toSub = instance.RestoreSubState(*tData.ToSubState) }

		var tDataMap map[string]interface{}
		_ = json.Unmarshal(tData.Data, &tDataMap)
		
		tr := instance.RestoreTransition(
			tID, tData.FromState, tData.ToState, tData.Event, actor,
			fromSub, toSub,
			instance.NewDataFromMap(tDataMap),
			shared.From(tData.CreatedAt),
		)
		
		meta := make(map[string]interface{})
		// Populate meta map if extra fields exist
		
		var reason, feedback string
		if tData.Reason != nil { reason = *tData.Reason }
		if tData.Feedback != nil { feedback = *tData.Feedback }
		
		tr.SetMetadata(instance.NewTransitionMetadata(reason, feedback, meta))

		transitions = append(transitions, tr)
	}
	
	return transitions, nil
}

func (r *InstanceRepository) FindByWorkflowID(ctx context.Context, workflowID shared.ID) ([]*instance.Instance, error) {
	query := `
		SELECT id, parent_id, workflow_id, current_state, current_sub_state, status, version, 
		       created_at, updated_at, completed_at, data, variables, current_actor 
		FROM workflow_instances 
		WHERE workflow_id = $1 AND status != 'deleted'
	`
	rows, err := r.db.Query(ctx, query, workflowID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to query instances by workflow: %w", err)
	}
	return r.scanInstances(ctx, rows)
}

func (r *InstanceRepository) FindByParentID(ctx context.Context, parentID shared.ID) ([]*instance.Instance, error) {
	query := `
		SELECT id, parent_id, workflow_id, current_state, current_sub_state, status, version, 
		       created_at, updated_at, completed_at, data, variables, current_actor 
		FROM workflow_instances 
		WHERE parent_id = $1 AND status != 'deleted'
	`
	rows, err := r.db.Query(ctx, query, parentID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to query instances by parent ID: %w", err)
	}
	return r.scanInstances(ctx, rows)
}

func (r *InstanceRepository) FindByStatus(ctx context.Context, status instance.Status) ([]*instance.Instance, error) {
	query := `
		SELECT id, parent_id, workflow_id, current_state, current_sub_state, status, version, 
		       created_at, updated_at, completed_at, data, variables, current_actor 
		FROM workflow_instances 
		WHERE status = $1
	`
	rows, err := r.db.Query(ctx, query, status.String())
	if err != nil {
		return nil, fmt.Errorf("failed to query instances by status: %w", err)
	}
	return r.scanInstances(ctx, rows)
}

func (r *InstanceRepository) FindByWorkflowIDAndStatus(ctx context.Context, workflowID shared.ID, status instance.Status) ([]*instance.Instance, error) {
	query := `
		SELECT id, parent_id, workflow_id, current_state, current_sub_state, status, version, 
		       created_at, updated_at, completed_at, data, variables, current_actor 
		FROM workflow_instances 
		WHERE workflow_id = $1 AND status = $2
	`
	rows, err := r.db.Query(ctx, query, workflowID.String(), status.String())
	if err != nil {
		return nil, fmt.Errorf("failed to query instances by workflow and status: %w", err)
	}
	return r.scanInstances(ctx, rows)
}

func (r *InstanceRepository) FindActive(ctx context.Context) ([]*instance.Instance, error) {
	query := `
		SELECT id, parent_id, workflow_id, current_state, current_sub_state, status, version, 
		       created_at, updated_at, completed_at, data, variables, current_actor 
		FROM workflow_instances 
		WHERE status IN ('running', 'paused')
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query active instances: %w", err)
	}
	return r.scanInstances(ctx, rows)
}

func (r *InstanceRepository) FindAll(ctx context.Context) ([]*instance.Instance, error) {
	query := `
		SELECT id, parent_id, workflow_id, current_state, current_sub_state, status, version, 
		       created_at, updated_at, completed_at, data, variables, current_actor 
		FROM workflow_instances
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query all instances: %w", err)
	}
	return r.scanInstances(ctx, rows)
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
