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
		INSERT INTO workflow_timers (id, instance_id, state, event_on_timeout, expires_at, fired_at, created_at,
			retry_count, max_retries, next_retry_at, last_error, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (id) DO UPDATE SET
			fired_at = EXCLUDED.fired_at,
			retry_count = EXCLUDED.retry_count,
			max_retries = EXCLUDED.max_retries,
			next_retry_at = EXCLUDED.next_retry_at,
			last_error = EXCLUDED.last_error,
			status = EXCLUDED.status
	`
	var firedAt *time.Time
	if !t.FiredAt().IsZero() {
		val := t.FiredAt().Time()
		firedAt = &val
	}

	var nextRetryAt *time.Time
	if !t.NextRetryAt().IsZero() {
		val := t.NextRetryAt().Time()
		nextRetryAt = &val
	}

	_, err := r.db.Exec(ctx, query,
		t.ID().String(),
		t.InstanceID().String(),
		t.State(),
		t.EventOnTimeout(),
		t.ExpiresAt().Time(),
		firedAt,
		time.Now(), // CreatedAt logic usually in domain, simplified here
		t.RetryCount(),
		t.MaxRetries(),
		nextRetryAt,
		t.LastError(),
		t.Status(),
	)
	return err
}

func (r *TimerRepository) FindPending(ctx context.Context, limit int) ([]*timer.Timer, error) {
	query := `
		SELECT id, instance_id, state, event_on_timeout, expires_at, created_at,
			COALESCE(retry_count, 0), COALESCE(max_retries, 3), next_retry_at, COALESCE(last_error, ''), COALESCE(status, 'pending')
		FROM workflow_timers
		WHERE (status IS NULL OR status = 'pending')
			AND expires_at <= $1
			AND (next_retry_at IS NULL OR next_retry_at <= $1)
			AND fired_at IS NULL
		LIMIT $2
	`
	now := time.Now()
	rows, err := r.db.Query(ctx, query, now, limit)
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
			RetryCount     int
			MaxRetries     int
			NextRetryAt    *time.Time
			LastError      string
			Status         string
		}
		if err := rows.Scan(
			&tData.ID, &tData.InstanceID, &tData.State, &tData.EventOnTimeout,
			&tData.ExpiresAt, &tData.CreatedAt,
			&tData.RetryCount, &tData.MaxRetries, &tData.NextRetryAt, &tData.LastError, &tData.Status,
		); err != nil {
			return nil, err
		}

		tID, _ := shared.ParseID(tData.ID)
		iID, _ := shared.ParseID(tData.InstanceID)

		var nextRetryAt shared.Timestamp
		if tData.NextRetryAt != nil {
			nextRetryAt = shared.From(*tData.NextRetryAt)
		}

		tm := timer.RestoreTimerFull(
			tID, iID, tData.State, tData.EventOnTimeout,
			shared.From(tData.ExpiresAt), shared.ZeroTimestamp(), shared.From(tData.CreatedAt),
			tData.RetryCount, tData.MaxRetries, nextRetryAt, tData.LastError, tData.Status,
		)
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
