package timer

import (
	"time"

	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/shared"
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
