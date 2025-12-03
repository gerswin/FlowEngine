package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/shared"
	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/timer"
)

type TimerRepository struct {
	db *pgxpool.Pool
}

func NewTimerRepository(db *pgxpool.Pool) *TimerRepository {
	return &TimerRepository{db: db}
}

func (r *TimerRepository) Save(ctx context.Context, t *timer.Timer) error {
	query := `
		INSERT INTO workflow_timers (id, instance_id, state, event_on_timeout, expires_at, fired_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO UPDATE SET
			fired_at = EXCLUDED.fired_at
	`
	var firedAt *time.Time
	if !t.FiredAt().IsZero() {
		val := t.FiredAt().Time()
		firedAt = &val
	}

	_, err := r.db.Exec(ctx, query,
		t.ID().String(),
		t.InstanceID().String(),
		t.State(),
		t.EventOnTimeout(),
		t.ExpiresAt().Time(),
		firedAt,
		time.Now(), // CreatedAt logic usually in domain, simplified here
	)
	return err
}

func (r *TimerRepository) FindPending(ctx context.Context, limit int) ([]*timer.Timer, error) {
	query := `
		SELECT id, instance_id, state, event_on_timeout, expires_at, created_at 
		FROM workflow_timers 
		WHERE fired_at IS NULL AND expires_at <= $1
		LIMIT $2
	`
	rows, err := r.db.Query(ctx, query, time.Now(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var timers []*timer.Timer
	for rows.Next() {
		var tData struct {
			ID             string
			InstanceID     string
			State          string
			EventOnTimeout string
			ExpiresAt      time.Time
			CreatedAt      time.Time
		}
		if err := rows.Scan(&tData.ID, &tData.InstanceID, &tData.State, &tData.EventOnTimeout, &tData.ExpiresAt, &tData.CreatedAt); err != nil {
			return nil, err
		}

		tID, _ := shared.ParseID(tData.ID)
		iID, _ := shared.ParseID(tData.InstanceID)

		// Reconstruct (simplified, ideally use RestoreTimer)
		// I need to add RestoreTimer to domain if fields are private.
		// For now, assuming I can add it or Timer struct needs update.
		// Checking Timer struct... fields are private.
		// I need RestoreTimer.
		
		// WORKAROUND: I will add RestoreTimer to domain/timer/timer.go in next step.
		// For now, assuming it exists.
		tm := timer.RestoreTimer(tID, iID, tData.State, tData.EventOnTimeout, shared.From(tData.ExpiresAt), shared.ZeroTimestamp(), shared.From(tData.CreatedAt))
		timers = append(timers, tm)
	}
	return timers, nil
}

func (r *TimerRepository) Delete(ctx context.Context, id shared.ID) error {
	query := `DELETE FROM workflow_timers WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id.String())
	return err
}

func (r *TimerRepository) DeleteByInstanceID(ctx context.Context, instanceID shared.ID) error {
	query := `DELETE FROM workflow_timers WHERE instance_id = $1`
	_, err := r.db.Exec(ctx, query, instanceID.String())
	return err
}
