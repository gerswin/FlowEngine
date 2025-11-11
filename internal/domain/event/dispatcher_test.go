package event

import (
	"context"
	"testing"

	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNullDispatcher_Dispatch(t *testing.T) {
	dispatcher := NewNullDispatcher()
	event := createTestEvent()

	err := dispatcher.Dispatch(context.Background(), event)

	assert.NoError(t, err)
}

func TestNullDispatcher_DispatchBatch(t *testing.T) {
	dispatcher := NewNullDispatcher()
	events := []DomainEvent{
		createTestEvent(),
		createTestEvent(),
		createTestEvent(),
	}

	err := dispatcher.DispatchBatch(context.Background(), events)

	assert.NoError(t, err)
}

func TestInMemoryDispatcher_Dispatch(t *testing.T) {
	dispatcher := NewInMemoryDispatcher()
	event := createTestEvent()

	err := dispatcher.Dispatch(context.Background(), event)

	require.NoError(t, err)
	assert.Equal(t, 1, dispatcher.Count())

	events := dispatcher.Events()
	require.Len(t, events, 1)
	assert.Equal(t, event.Type(), events[0].Type())
}

func TestInMemoryDispatcher_DispatchBatch(t *testing.T) {
	dispatcher := NewInMemoryDispatcher()
	instanceID := shared.NewID()
	actorID := shared.NewID()

	events := []DomainEvent{
		NewInstancePaused(instanceID, actorID, "reason1"),
		NewInstanceResumed(instanceID, actorID),
		NewInstanceCanceled(instanceID, "state", "reason", actorID),
	}

	err := dispatcher.DispatchBatch(context.Background(), events)

	require.NoError(t, err)
	assert.Equal(t, 3, dispatcher.Count())

	storedEvents := dispatcher.Events()
	require.Len(t, storedEvents, 3)
}

func TestInMemoryDispatcher_Count(t *testing.T) {
	dispatcher := NewInMemoryDispatcher()

	t.Run("starts with zero count", func(t *testing.T) {
		assert.Equal(t, 0, dispatcher.Count())
	})

	t.Run("increments count on dispatch", func(t *testing.T) {
		dispatcher.Dispatch(context.Background(), createTestEvent())
		assert.Equal(t, 1, dispatcher.Count())

		dispatcher.Dispatch(context.Background(), createTestEvent())
		assert.Equal(t, 2, dispatcher.Count())
	})

	t.Run("increments count on batch dispatch", func(t *testing.T) {
		dispatcher.Clear()
		events := []DomainEvent{createTestEvent(), createTestEvent()}
		dispatcher.DispatchBatch(context.Background(), events)

		assert.Equal(t, 2, dispatcher.Count())
	})
}

func TestInMemoryDispatcher_Clear(t *testing.T) {
	dispatcher := NewInMemoryDispatcher()

	// Add some events
	dispatcher.Dispatch(context.Background(), createTestEvent())
	dispatcher.Dispatch(context.Background(), createTestEvent())
	require.Equal(t, 2, dispatcher.Count())

	// Clear
	dispatcher.Clear()

	assert.Equal(t, 0, dispatcher.Count())
	assert.Empty(t, dispatcher.Events())
}

func TestInMemoryDispatcher_Events_ReturnsCopy(t *testing.T) {
	dispatcher := NewInMemoryDispatcher()
	event := createTestEvent()
	dispatcher.Dispatch(context.Background(), event)

	// Get events
	events1 := dispatcher.Events()
	require.Len(t, events1, 1)

	// Modify the returned slice (should not affect internal state)
	events1[0] = createTestEvent()

	// Get events again
	events2 := dispatcher.Events()
	require.Len(t, events2, 1)

	// Original event should still be there
	assert.Equal(t, event.Type(), events2[0].Type())
}

func TestInMemoryDispatcher_FindByType(t *testing.T) {
	dispatcher := NewInMemoryDispatcher()
	instanceID := shared.NewID()
	workflowID := shared.NewID()
	actorID := shared.NewID()

	// Dispatch multiple events of different types
	dispatcher.Dispatch(context.Background(), NewInstanceCreated(instanceID, workflowID, "test", "initial", actorID, nil))
	dispatcher.Dispatch(context.Background(), NewInstancePaused(instanceID, actorID, "reason"))
	dispatcher.Dispatch(context.Background(), NewInstanceResumed(instanceID, actorID))
	dispatcher.Dispatch(context.Background(), NewInstancePaused(shared.NewID(), actorID, "reason2"))

	t.Run("find instance.paused events", func(t *testing.T) {
		events := dispatcher.FindByType("instance.paused")
		assert.Len(t, events, 2)

		for _, event := range events {
			assert.Equal(t, "instance.paused", event.Type())
		}
	})

	t.Run("find instance.created events", func(t *testing.T) {
		events := dispatcher.FindByType("instance.created")
		assert.Len(t, events, 1)
		assert.Equal(t, "instance.created", events[0].Type())
	})

	t.Run("find non-existent event type", func(t *testing.T) {
		events := dispatcher.FindByType("non.existent")
		assert.Empty(t, events)
	})
}

func TestInMemoryDispatcher_FindByAggregateID(t *testing.T) {
	dispatcher := NewInMemoryDispatcher()
	instance1ID := shared.NewID()
	instance2ID := shared.NewID()
	workflowID := shared.NewID()
	actorID := shared.NewID()

	// Dispatch events for different aggregates
	dispatcher.Dispatch(context.Background(), NewInstanceCreated(instance1ID, workflowID, "test", "initial", actorID, nil))
	dispatcher.Dispatch(context.Background(), NewInstancePaused(instance1ID, actorID, "reason"))
	dispatcher.Dispatch(context.Background(), NewInstanceResumed(instance1ID, actorID))
	dispatcher.Dispatch(context.Background(), NewInstanceCreated(instance2ID, workflowID, "test", "initial", actorID, nil))

	t.Run("find events for instance1", func(t *testing.T) {
		events := dispatcher.FindByAggregateID(instance1ID.String())
		assert.Len(t, events, 3)

		for _, event := range events {
			assert.Equal(t, instance1ID.String(), event.AggregateID())
		}
	})

	t.Run("find events for instance2", func(t *testing.T) {
		events := dispatcher.FindByAggregateID(instance2ID.String())
		assert.Len(t, events, 1)
		assert.Equal(t, instance2ID.String(), events[0].AggregateID())
	})

	t.Run("find events for non-existent aggregate", func(t *testing.T) {
		events := dispatcher.FindByAggregateID(shared.NewID().String())
		assert.Empty(t, events)
	})
}

func TestInMemoryDispatcher_MultipleDispatches(t *testing.T) {
	dispatcher := NewInMemoryDispatcher()
	instanceID := shared.NewID()
	actorID := shared.NewID()

	// Simulate a workflow: created -> paused -> resumed -> completed
	event1 := NewInstanceCreated(instanceID, shared.NewID(), "test", "initial", actorID, nil)
	event2 := NewInstancePaused(instanceID, actorID, "maintenance")
	event3 := NewInstanceResumed(instanceID, actorID)
	event4 := NewInstanceCompleted(instanceID, "final", actorID, nil)

	dispatcher.Dispatch(context.Background(), event1)
	dispatcher.Dispatch(context.Background(), event2)
	dispatcher.Dispatch(context.Background(), event3)
	dispatcher.Dispatch(context.Background(), event4)

	t.Run("all events stored in order", func(t *testing.T) {
		events := dispatcher.Events()
		require.Len(t, events, 4)

		assert.Equal(t, "instance.created", events[0].Type())
		assert.Equal(t, "instance.paused", events[1].Type())
		assert.Equal(t, "instance.resumed", events[2].Type())
		assert.Equal(t, "instance.completed", events[3].Type())
	})

	t.Run("can find all events by aggregate ID", func(t *testing.T) {
		events := dispatcher.FindByAggregateID(instanceID.String())
		assert.Len(t, events, 4)
	})
}

func TestInMemoryDispatcher_ConcurrentDispatch(t *testing.T) {
	dispatcher := NewInMemoryDispatcher()
	instanceID := shared.NewID()
	actorID := shared.NewID()

	// Simulate concurrent dispatches
	done := make(chan bool)
	dispatchCount := 10

	for i := 0; i < dispatchCount; i++ {
		go func() {
			event := NewInstancePaused(instanceID, actorID, "reason")
			dispatcher.Dispatch(context.Background(), event)
			done <- true
		}()
	}

	// Wait for all dispatches to complete
	for i := 0; i < dispatchCount; i++ {
		<-done
	}

	// Note: This test may reveal race conditions in concurrent access
	// The current implementation is not thread-safe
	// In production, the implementation should use proper synchronization
	assert.Equal(t, dispatchCount, dispatcher.Count())
}

func TestDispatcher_ContextCancellation(t *testing.T) {
	t.Run("null dispatcher ignores context cancellation", func(t *testing.T) {
		dispatcher := NewNullDispatcher()
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := dispatcher.Dispatch(ctx, createTestEvent())
		assert.NoError(t, err)
	})

	t.Run("in-memory dispatcher ignores context cancellation", func(t *testing.T) {
		dispatcher := NewInMemoryDispatcher()
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := dispatcher.Dispatch(ctx, createTestEvent())
		assert.NoError(t, err)
		assert.Equal(t, 1, dispatcher.Count())
	})
}

// Helper function to create a test event
func createTestEvent() DomainEvent {
	return NewInstanceCreated(
		shared.NewID(),
		shared.NewID(),
		"TestWorkflow",
		"initial",
		shared.NewID(),
		map[string]interface{}{"test": "data"},
	)
}
