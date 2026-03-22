package timer

import (
	"errors"
	"testing"
	"time"

	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTimer_Defaults(t *testing.T) {
	instanceID := shared.NewID()
	tmr := NewTimer(instanceID, "awaiting_approval", "timeout", 5*time.Minute)

	require.NotNil(t, tmr)
	assert.True(t, tmr.ID().IsValid())
	assert.Equal(t, instanceID, tmr.InstanceID())
	assert.Equal(t, "awaiting_approval", tmr.State())
	assert.Equal(t, "timeout", tmr.EventOnTimeout())
	assert.Equal(t, StatusPending, tmr.Status())
	assert.Equal(t, 0, tmr.RetryCount())
	assert.Equal(t, DefaultMaxRetries, tmr.MaxRetries())
	assert.False(t, tmr.HasFired())
	assert.True(t, tmr.ExpiresAt().After(shared.Now()))
}

func TestTimer_Getters(t *testing.T) {
	instanceID := shared.NewID()
	tmr := NewTimer(instanceID, "state1", "event1", 10*time.Second)

	assert.Equal(t, instanceID, tmr.InstanceID())
	assert.Equal(t, "state1", tmr.State())
	assert.Equal(t, "event1", tmr.EventOnTimeout())
	assert.True(t, tmr.FiredAt().IsZero())
	assert.True(t, tmr.NextRetryAt().IsZero())
	assert.Empty(t, tmr.LastError())
}

func TestTimer_HasFired_MarkFired(t *testing.T) {
	tmr := NewTimer(shared.NewID(), "s", "e", time.Minute)

	assert.False(t, tmr.HasFired())

	tmr.MarkFired()

	assert.True(t, tmr.HasFired())
	assert.False(t, tmr.FiredAt().IsZero())
}

func TestTimer_IncrementRetry(t *testing.T) {
	tmr := NewTimer(shared.NewID(), "s", "e", time.Minute)
	testErr := errors.New("connection refused")

	// First retry: backoff = 2^0 * 30s = 30s
	tmr.IncrementRetry(testErr)
	assert.Equal(t, 1, tmr.RetryCount())
	assert.Equal(t, "connection refused", tmr.LastError())
	assert.Equal(t, StatusPending, tmr.Status())
	assert.False(t, tmr.NextRetryAt().IsZero())

	// Second retry: backoff = 2^1 * 30s = 60s
	beforeRetry := time.Now().UTC()
	tmr.IncrementRetry(errors.New("timeout"))
	assert.Equal(t, 2, tmr.RetryCount())
	assert.Equal(t, "timeout", tmr.LastError())
	expectedMinRetry := shared.From(beforeRetry.Add(60 * time.Second))
	// nextRetryAt should be approximately 60s from now (allow 2s tolerance)
	diff := tmr.NextRetryAt().Time().Sub(expectedMinRetry.Time())
	assert.Less(t, diff.Abs(), 2*time.Second)

	// Third retry: backoff = 2^2 * 30s = 120s
	tmr.IncrementRetry(errors.New("err3"))
	assert.Equal(t, 3, tmr.RetryCount())
}

func TestTimer_CanRetry(t *testing.T) {
	tmr := NewTimer(shared.NewID(), "s", "e", time.Minute)

	assert.True(t, tmr.CanRetry()) // 0 < 3

	tmr.IncrementRetry(errors.New("e1"))
	assert.True(t, tmr.CanRetry()) // 1 < 3

	tmr.IncrementRetry(errors.New("e2"))
	assert.True(t, tmr.CanRetry()) // 2 < 3

	tmr.IncrementRetry(errors.New("e3"))
	assert.False(t, tmr.CanRetry()) // 3 == 3
}

func TestTimer_MarkFailed(t *testing.T) {
	tmr := NewTimer(shared.NewID(), "s", "e", time.Minute)

	tmr.MarkFailed()

	assert.Equal(t, StatusFailed, tmr.Status())
}

func TestTimer_MarkCompleted(t *testing.T) {
	tmr := NewTimer(shared.NewID(), "s", "e", time.Minute)

	tmr.MarkCompleted()

	assert.Equal(t, StatusCompleted, tmr.Status())
	assert.True(t, tmr.HasFired())
}

func TestTimer_IsExpired(t *testing.T) {
	// Timer that expires in the past
	tmr := NewTimer(shared.NewID(), "s", "e", -1*time.Second)
	assert.True(t, tmr.IsExpired())

	// Timer that expires in the future
	tmr2 := NewTimer(shared.NewID(), "s", "e", 10*time.Minute)
	assert.False(t, tmr2.IsExpired())
}

func TestRestoreTimer(t *testing.T) {
	id := shared.NewID()
	instID := shared.NewID()
	now := shared.Now()
	expires := shared.From(now.Time().Add(5 * time.Minute))

	tmr := RestoreTimer(id, instID, "waiting", "expire", expires, shared.ZeroTimestamp(), now)

	assert.Equal(t, id, tmr.ID())
	assert.Equal(t, instID, tmr.InstanceID())
	assert.Equal(t, "waiting", tmr.State())
	assert.Equal(t, "expire", tmr.EventOnTimeout())
	assert.Equal(t, expires, tmr.ExpiresAt())
	assert.True(t, tmr.FiredAt().IsZero())
	assert.Equal(t, DefaultMaxRetries, tmr.MaxRetries())
	assert.Equal(t, StatusPending, tmr.Status())
	assert.Equal(t, 0, tmr.RetryCount())
}

func TestRestoreTimerFull(t *testing.T) {
	id := shared.NewID()
	instID := shared.NewID()
	now := shared.Now()
	expires := shared.From(now.Time().Add(5 * time.Minute))
	nextRetry := shared.From(now.Time().Add(30 * time.Second))

	tmr := RestoreTimerFull(
		id, instID, "waiting", "expire",
		expires, shared.ZeroTimestamp(), now,
		2, 5, nextRetry,
		"last err", StatusProcessing,
	)

	assert.Equal(t, id, tmr.ID())
	assert.Equal(t, instID, tmr.InstanceID())
	assert.Equal(t, 2, tmr.RetryCount())
	assert.Equal(t, 5, tmr.MaxRetries())
	assert.Equal(t, "last err", tmr.LastError())
	assert.Equal(t, StatusProcessing, tmr.Status())
	assert.Equal(t, nextRetry, tmr.NextRetryAt())
}

func TestRestoreTimerFull_DefaultMaxRetries(t *testing.T) {
	tmr := RestoreTimerFull(
		shared.NewID(), shared.NewID(), "s", "e",
		shared.Now(), shared.ZeroTimestamp(), shared.Now(),
		0, 0, shared.ZeroTimestamp(), // maxRetries=0 should default
		"", "", // empty status should default
	)

	assert.Equal(t, DefaultMaxRetries, tmr.MaxRetries())
	assert.Equal(t, StatusPending, tmr.Status())
}
