package memory

import (
	"context"
	"sync"
	"time"

	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/shared"
	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/timer"
)

// TimerInMemoryRepository implements timer.Repository using in-memory storage.
// This is useful for testing and development without database dependencies.
type TimerInMemoryRepository struct {
	mu     sync.RWMutex
	timers map[string]*timer.Timer // keyed by timer ID
}

// NewTimerInMemoryRepository creates a new in-memory timer repository.
func NewTimerInMemoryRepository() *TimerInMemoryRepository {
	return &TimerInMemoryRepository{
		timers: make(map[string]*timer.Timer),
	}
}

// Save persists a timer to memory.
func (r *TimerInMemoryRepository) Save(ctx context.Context, t *timer.Timer) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.timers[t.ID().String()] = t
	return nil
}

// FindPending returns timers that are pending, expired, and ready for retry, up to the given limit.
func (r *TimerInMemoryRepository) FindPending(ctx context.Context, limit int) ([]*timer.Timer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	now := time.Now()
	var pending []*timer.Timer
	for _, t := range r.timers {
		if t.HasFired() {
			continue
		}
		// Only fetch pending timers (status empty means legacy pending)
		status := t.Status()
		if status != "" && status != timer.StatusPending {
			continue
		}
		if now.Before(t.ExpiresAt().Time()) {
			continue
		}
		// Respect nextRetryAt for backoff
		if !t.NextRetryAt().IsZero() && now.Before(t.NextRetryAt().Time()) {
			continue
		}
		pending = append(pending, t)
		if len(pending) >= limit {
			break
		}
	}

	return pending, nil
}

// Delete removes a timer by ID.
func (r *TimerInMemoryRepository) Delete(ctx context.Context, id shared.ID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.timers, id.String())
	return nil
}

// DeleteByInstanceID removes all timers associated with a given instance ID.
func (r *TimerInMemoryRepository) DeleteByInstanceID(ctx context.Context, instanceID shared.ID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for key, t := range r.timers {
		if t.InstanceID().Equals(instanceID) {
			delete(r.timers, key)
		}
	}

	return nil
}

// Clear removes all timers (useful for testing).
func (r *TimerInMemoryRepository) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.timers = make(map[string]*timer.Timer)
}
