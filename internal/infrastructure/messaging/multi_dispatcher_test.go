package messaging

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMultiDispatcher_Dispatch_ToAllDispatchers(t *testing.T) {
	d1 := event.NewInMemoryDispatcher()
	d2 := event.NewInMemoryDispatcher()
	multi := NewMultiDispatcher(d1, d2)

	evt := newTestEvent("instance.created", "agg-1")

	err := multi.Dispatch(context.Background(), evt)
	require.NoError(t, err)

	assert.Equal(t, 1, d1.Count())
	assert.Equal(t, 1, d2.Count())
}

func TestMultiDispatcher_Dispatch_NoDispatchers(t *testing.T) {
	multi := NewMultiDispatcher()

	evt := newTestEvent("instance.created", "agg-1")
	err := multi.Dispatch(context.Background(), evt)

	assert.NoError(t, err)
}

func TestMultiDispatcher_Dispatch_ErrorFromOneDoesNotStopOthers(t *testing.T) {
	failing := &failingDispatcher{err: fmt.Errorf("dispatch failed")}
	d2 := event.NewInMemoryDispatcher()
	multi := NewMultiDispatcher(failing, d2)

	evt := newTestEvent("instance.created", "agg-1")
	err := multi.Dispatch(context.Background(), evt)

	// Returns first error
	assert.Error(t, err)
	assert.Equal(t, "dispatch failed", err.Error())

	// But second dispatcher still received the event
	assert.Equal(t, 1, d2.Count())
}

func TestMultiDispatcher_Dispatch_ReturnsFirstError(t *testing.T) {
	failing1 := &failingDispatcher{err: fmt.Errorf("first error")}
	failing2 := &failingDispatcher{err: fmt.Errorf("second error")}
	multi := NewMultiDispatcher(failing1, failing2)

	evt := newTestEvent("instance.created", "agg-1")
	err := multi.Dispatch(context.Background(), evt)

	assert.Error(t, err)
	assert.Equal(t, "first error", err.Error())
}

func TestMultiDispatcher_DispatchBatch_ToAllDispatchers(t *testing.T) {
	d1 := event.NewInMemoryDispatcher()
	d2 := event.NewInMemoryDispatcher()
	multi := NewMultiDispatcher(d1, d2)

	events := []event.DomainEvent{
		newTestEvent("instance.created", "agg-1"),
		newTestEvent("instance.completed", "agg-2"),
	}

	err := multi.DispatchBatch(context.Background(), events)
	require.NoError(t, err)

	assert.Equal(t, 2, d1.Count())
	assert.Equal(t, 2, d2.Count())
}

func TestMultiDispatcher_DispatchBatch_ErrorFromOneDoesNotStopOthers(t *testing.T) {
	failing := &failingDispatcher{err: fmt.Errorf("batch failed")}
	d2 := event.NewInMemoryDispatcher()
	multi := NewMultiDispatcher(failing, d2)

	events := []event.DomainEvent{
		newTestEvent("instance.created", "agg-1"),
	}

	err := multi.DispatchBatch(context.Background(), events)

	assert.Error(t, err)
	assert.Equal(t, "batch failed", err.Error())
	assert.Equal(t, 1, d2.Count())
}

func TestLogDispatcher_Dispatch(t *testing.T) {
	d := NewLogDispatcher()

	evt := &testEvent{
		eventType:   "instance.created",
		aggregateID: "agg-1",
		occurredAt:  time.Now(),
		payload:     map[string]interface{}{"key": "value"},
	}

	err := d.Dispatch(context.Background(), evt)
	assert.NoError(t, err)
}

func TestLogDispatcher_DispatchBatch(t *testing.T) {
	d := NewLogDispatcher()

	events := []event.DomainEvent{
		newTestEvent("instance.created", "agg-1"),
		newTestEvent("instance.completed", "agg-2"),
	}

	err := d.DispatchBatch(context.Background(), events)
	assert.NoError(t, err)
}
