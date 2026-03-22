package timer

import (
	"math"
	"time"

	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/shared"
)

// Timer status constants.
const (
	StatusPending    = "pending"
	StatusProcessing = "processing"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"

	DefaultMaxRetries    = 3
	RetryBackoffBase     = 30 * time.Second
)

// Timer represents a scheduled event execution (timeout).
type Timer struct {
	id             shared.ID
	instanceID     shared.ID
	state          string
	eventOnTimeout string
	expiresAt      shared.Timestamp
	firedAt        shared.Timestamp // Zero if not fired
	createdAt      shared.Timestamp

	// Retry fields
	retryCount int
	maxRetries int
	nextRetryAt shared.Timestamp
	lastError  string
	status     string
}

// NewTimer creates a new Timer.
func NewTimer(instanceID shared.ID, state, eventOnTimeout string, duration time.Duration) *Timer {
	now := shared.Now()
	return &Timer{
		id:             shared.NewID(),
		instanceID:     instanceID,
		state:          state,
		eventOnTimeout: eventOnTimeout,
		expiresAt:      shared.From(now.Time().Add(duration)),
		createdAt:      now,
		retryCount:     0,
		maxRetries:     DefaultMaxRetries,
		status:         StatusPending,
	}
}

// RestoreTimer reconstitutes a timer from persistence.
func RestoreTimer(
	id, instanceID shared.ID,
	state, eventOnTimeout string,
	expiresAt, firedAt, createdAt shared.Timestamp,
) *Timer {
	return &Timer{
		id:             id,
		instanceID:     instanceID,
		state:          state,
		eventOnTimeout: eventOnTimeout,
		expiresAt:      expiresAt,
		firedAt:        firedAt,
		createdAt:      createdAt,
		maxRetries:     DefaultMaxRetries,
		status:         StatusPending,
	}
}

// RestoreTimerFull reconstitutes a timer from persistence including retry fields.
func RestoreTimerFull(
	id, instanceID shared.ID,
	state, eventOnTimeout string,
	expiresAt, firedAt, createdAt shared.Timestamp,
	retryCount, maxRetries int,
	nextRetryAt shared.Timestamp,
	lastError, status string,
) *Timer {
	if maxRetries == 0 {
		maxRetries = DefaultMaxRetries
	}
	if status == "" {
		status = StatusPending
	}
	return &Timer{
		id:             id,
		instanceID:     instanceID,
		state:          state,
		eventOnTimeout: eventOnTimeout,
		expiresAt:      expiresAt,
		firedAt:        firedAt,
		createdAt:      createdAt,
		retryCount:     retryCount,
		maxRetries:     maxRetries,
		nextRetryAt:    nextRetryAt,
		lastError:      lastError,
		status:         status,
	}
}

// ID returns the timer ID.
func (t *Timer) ID() shared.ID {
	return t.id
}

// InstanceID returns the associated instance ID.
func (t *Timer) InstanceID() shared.ID {
	return t.instanceID
}

// State returns the state associated with this timer.
func (t *Timer) State() string {
	return t.state
}

// EventOnTimeout returns the event to trigger when the timer expires.
func (t *Timer) EventOnTimeout() string {
	return t.eventOnTimeout
}

// ExpiresAt returns the expiration timestamp.
func (t *Timer) ExpiresAt() shared.Timestamp {
	return t.expiresAt
}

// FiredAt returns the time the timer fired.
func (t *Timer) FiredAt() shared.Timestamp {
	return t.firedAt
}

// MarkFired marks the timer as fired.
func (t *Timer) MarkFired() {
	t.firedAt = shared.Now()
}

// IsExpired checks if the timer has expired.
func (t *Timer) IsExpired() bool {
	return time.Now().After(t.expiresAt.Time())
}

// HasFired checks if the timer has already fired.
func (t *Timer) HasFired() bool {
	return !t.firedAt.IsZero()
}

// RetryCount returns the current retry count.
func (t *Timer) RetryCount() int {
	return t.retryCount
}

// MaxRetries returns the maximum number of retries.
func (t *Timer) MaxRetries() int {
	return t.maxRetries
}

// NextRetryAt returns the next retry timestamp.
func (t *Timer) NextRetryAt() shared.Timestamp {
	return t.nextRetryAt
}

// LastError returns the last error message.
func (t *Timer) LastError() string {
	return t.lastError
}

// Status returns the timer status.
func (t *Timer) Status() string {
	return t.status
}

// CanRetry returns true if the timer has not exhausted its retries.
func (t *Timer) CanRetry() bool {
	return t.retryCount < t.maxRetries
}

// IncrementRetry increments the retry count, records the error, and calculates
// the next retry time using exponential backoff (base 30s: 30s, 60s, 120s, 240s...).
func (t *Timer) IncrementRetry(err error) {
	t.retryCount++
	t.lastError = err.Error()
	backoff := time.Duration(math.Pow(2, float64(t.retryCount-1))) * RetryBackoffBase
	t.nextRetryAt = shared.From(time.Now().UTC().Add(backoff))
	t.status = StatusPending
}

// MarkFailed sets the timer status to failed.
func (t *Timer) MarkFailed() {
	t.status = StatusFailed
}

// MarkCompleted sets the timer status to completed and records the fired time.
func (t *Timer) MarkCompleted() {
	t.status = StatusCompleted
	t.firedAt = shared.Now()
}
