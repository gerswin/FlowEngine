package event

import (
	"testing"
	"time"

	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBaseEvent_Type(t *testing.T) {
	instanceID := shared.NewID()
	base := newBaseEvent("test.event", instanceID)

	assert.Equal(t, "test.event", base.Type())
}

func TestBaseEvent_AggregateID(t *testing.T) {
	instanceID := shared.NewID()
	base := newBaseEvent("test.event", instanceID)

	assert.Equal(t, instanceID.String(), base.AggregateID())
}

func TestBaseEvent_OccurredAt(t *testing.T) {
	instanceID := shared.NewID()
	beforeCreation := time.Now().UTC()

	base := newBaseEvent("test.event", instanceID)

	afterCreation := time.Now().UTC()

	assert.False(t, base.OccurredAt().Before(beforeCreation))
	assert.False(t, base.OccurredAt().After(afterCreation))
}

func TestInstanceCreated_Event(t *testing.T) {
	instanceID := shared.NewID()
	workflowID := shared.NewID()
	actorID := shared.NewID()
	data := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
	}

	event := NewInstanceCreated(instanceID, workflowID, "TestWorkflow", "initial", actorID, data)

	t.Run("has correct type", func(t *testing.T) {
		assert.Equal(t, "instance.created", event.Type())
	})

	t.Run("has correct aggregate ID", func(t *testing.T) {
		assert.Equal(t, instanceID.String(), event.AggregateID())
	})

	t.Run("has correct fields", func(t *testing.T) {
		assert.Equal(t, workflowID.String(), event.WorkflowID)
		assert.Equal(t, "TestWorkflow", event.WorkflowName)
		assert.Equal(t, "initial", event.InitialState)
		assert.Equal(t, actorID.String(), event.StartedBy)
		assert.Equal(t, data, event.Data)
	})

	t.Run("payload contains all data", func(t *testing.T) {
		payload := event.Payload()

		assert.Equal(t, instanceID.String(), payload["instance_id"])
		assert.Equal(t, workflowID.String(), payload["workflow_id"])
		assert.Equal(t, "TestWorkflow", payload["workflow_name"])
		assert.Equal(t, "initial", payload["initial_state"])
		assert.Equal(t, actorID.String(), payload["started_by"])
		assert.Equal(t, data, payload["data"])
		assert.NotNil(t, payload["occurred_at"])
	})
}

func TestStateChanged_Event(t *testing.T) {
	instanceID := shared.NewID()
	actorID := shared.NewID()
	transitionID := shared.NewID()
	variables := map[string]interface{}{
		"var1": "value1",
		"var2": 456,
	}

	event := NewStateChanged(instanceID, "state_a", "state_b", "submit", actorID, transitionID, variables)

	t.Run("has correct type", func(t *testing.T) {
		assert.Equal(t, "instance.state_changed", event.Type())
	})

	t.Run("has correct fields", func(t *testing.T) {
		assert.Equal(t, instanceID.String(), event.InstanceID)
		assert.Equal(t, "state_a", event.FromState)
		assert.Equal(t, "state_b", event.ToState)
		assert.Equal(t, "submit", event.Event)
		assert.Equal(t, actorID.String(), event.Actor)
		assert.Equal(t, transitionID.String(), event.TransitionID)
		assert.Equal(t, variables, event.Variables)
	})

	t.Run("payload contains all data", func(t *testing.T) {
		payload := event.Payload()

		assert.Equal(t, instanceID.String(), payload["instance_id"])
		assert.Equal(t, "state_a", payload["from_state"])
		assert.Equal(t, "state_b", payload["to_state"])
		assert.Equal(t, "submit", payload["event"])
		assert.Equal(t, actorID.String(), payload["actor"])
		assert.Equal(t, transitionID.String(), payload["transition_id"])
		assert.Equal(t, variables, payload["variables"])
	})
}

func TestSubStateChanged_Event(t *testing.T) {
	instanceID := shared.NewID()
	actorID := shared.NewID()

	event := NewSubStateChanged(instanceID, "pending", "sub_a", "sub_b", actorID)

	t.Run("has correct type", func(t *testing.T) {
		assert.Equal(t, "instance.substate_changed", event.Type())
	})

	t.Run("has correct fields", func(t *testing.T) {
		assert.Equal(t, instanceID.String(), event.InstanceID)
		assert.Equal(t, "pending", event.CurrentState)
		assert.Equal(t, "sub_a", event.FromSubState)
		assert.Equal(t, "sub_b", event.ToSubState)
		assert.Equal(t, actorID.String(), event.Actor)
	})

	t.Run("payload contains all data", func(t *testing.T) {
		payload := event.Payload()

		assert.Equal(t, instanceID.String(), payload["instance_id"])
		assert.Equal(t, "pending", payload["current_state"])
		assert.Equal(t, "sub_a", payload["from_substate"])
		assert.Equal(t, "sub_b", payload["to_substate"])
		assert.Equal(t, actorID.String(), payload["actor"])
	})
}

func TestInstancePaused_Event(t *testing.T) {
	instanceID := shared.NewID()
	actorID := shared.NewID()

	event := NewInstancePaused(instanceID, actorID, "maintenance window")

	t.Run("has correct type", func(t *testing.T) {
		assert.Equal(t, "instance.paused", event.Type())
	})

	t.Run("has correct fields", func(t *testing.T) {
		assert.Equal(t, instanceID.String(), event.InstanceID)
		assert.Equal(t, "maintenance window", event.Reason)
		assert.Equal(t, actorID.String(), event.PausedBy)
	})

	t.Run("payload contains all data", func(t *testing.T) {
		payload := event.Payload()

		assert.Equal(t, instanceID.String(), payload["instance_id"])
		assert.Equal(t, "maintenance window", payload["reason"])
		assert.Equal(t, actorID.String(), payload["paused_by"])
	})
}

func TestInstanceResumed_Event(t *testing.T) {
	instanceID := shared.NewID()
	actorID := shared.NewID()

	event := NewInstanceResumed(instanceID, actorID)

	t.Run("has correct type", func(t *testing.T) {
		assert.Equal(t, "instance.resumed", event.Type())
	})

	t.Run("has correct fields", func(t *testing.T) {
		assert.Equal(t, instanceID.String(), event.InstanceID)
		assert.Equal(t, actorID.String(), event.ResumedBy)
	})

	t.Run("payload contains all data", func(t *testing.T) {
		payload := event.Payload()

		assert.Equal(t, instanceID.String(), payload["instance_id"])
		assert.Equal(t, actorID.String(), payload["resumed_by"])
	})
}

func TestInstanceCompleted_Event(t *testing.T) {
	instanceID := shared.NewID()
	actorID := shared.NewID()
	result := map[string]interface{}{
		"status": "approved",
		"score":  95,
	}

	event := NewInstanceCompleted(instanceID, "final_state", actorID, result)

	t.Run("has correct type", func(t *testing.T) {
		assert.Equal(t, "instance.completed", event.Type())
	})

	t.Run("has correct fields", func(t *testing.T) {
		assert.Equal(t, instanceID.String(), event.InstanceID)
		assert.Equal(t, "final_state", event.FinalState)
		assert.Equal(t, actorID.String(), event.CompletedBy)
		assert.Equal(t, result, event.Result)
	})

	t.Run("payload contains all data", func(t *testing.T) {
		payload := event.Payload()

		assert.Equal(t, instanceID.String(), payload["instance_id"])
		assert.Equal(t, "final_state", payload["final_state"])
		assert.Equal(t, actorID.String(), payload["completed_by"])
		assert.Equal(t, result, payload["result"])
	})
}

func TestInstanceCanceled_Event(t *testing.T) {
	instanceID := shared.NewID()
	actorID := shared.NewID()

	event := NewInstanceCanceled(instanceID, "pending", "user requested", actorID)

	t.Run("has correct type", func(t *testing.T) {
		assert.Equal(t, "instance.canceled", event.Type())
	})

	t.Run("has correct fields", func(t *testing.T) {
		assert.Equal(t, instanceID.String(), event.InstanceID)
		assert.Equal(t, "pending", event.CurrentState)
		assert.Equal(t, "user requested", event.Reason)
		assert.Equal(t, actorID.String(), event.CanceledBy)
	})

	t.Run("payload contains all data", func(t *testing.T) {
		payload := event.Payload()

		assert.Equal(t, instanceID.String(), payload["instance_id"])
		assert.Equal(t, "pending", payload["current_state"])
		assert.Equal(t, "user requested", payload["reason"])
		assert.Equal(t, actorID.String(), payload["canceled_by"])
	})
}

func TestInstanceFailed_Event(t *testing.T) {
	instanceID := shared.NewID()
	actorID := shared.NewID()

	event := NewInstanceFailed(instanceID, "processing", "database connection lost", actorID)

	t.Run("has correct type", func(t *testing.T) {
		assert.Equal(t, "instance.failed", event.Type())
	})

	t.Run("has correct fields", func(t *testing.T) {
		assert.Equal(t, instanceID.String(), event.InstanceID)
		assert.Equal(t, "processing", event.CurrentState)
		assert.Equal(t, "database connection lost", event.Error)
		assert.Equal(t, actorID.String(), event.FailedBy)
	})

	t.Run("payload contains all data", func(t *testing.T) {
		payload := event.Payload()

		assert.Equal(t, instanceID.String(), payload["instance_id"])
		assert.Equal(t, "processing", payload["current_state"])
		assert.Equal(t, "database connection lost", payload["error"])
		assert.Equal(t, actorID.String(), payload["failed_by"])
	})
}

func TestDocumentReclassified_Event(t *testing.T) {
	instanceID := shared.NewID()
	actorID := shared.NewID()

	event := NewDocumentReclassified(instanceID, "type_a", "type_b", "incorrect classification", actorID)

	t.Run("has correct type", func(t *testing.T) {
		assert.Equal(t, "instance.document_reclassified", event.Type())
	})

	t.Run("has correct fields", func(t *testing.T) {
		assert.Equal(t, instanceID.String(), event.InstanceID)
		assert.Equal(t, "type_a", event.OldType)
		assert.Equal(t, "type_b", event.NewType)
		assert.Equal(t, "incorrect classification", event.Reason)
		assert.Equal(t, actorID.String(), event.ReclassifiedBy)
	})

	t.Run("payload contains all data", func(t *testing.T) {
		payload := event.Payload()

		assert.Equal(t, instanceID.String(), payload["instance_id"])
		assert.Equal(t, "type_a", payload["old_type"])
		assert.Equal(t, "type_b", payload["new_type"])
		assert.Equal(t, "incorrect classification", payload["reason"])
		assert.Equal(t, actorID.String(), payload["reclassified_by"])
	})
}

func TestWorkflowCreated_Event(t *testing.T) {
	workflowID := shared.NewID()
	actorID := shared.NewID()

	event := NewWorkflowCreated(workflowID, "TestWorkflow", "initial", actorID)

	t.Run("has correct type", func(t *testing.T) {
		assert.Equal(t, "workflow.created", event.Type())
	})

	t.Run("has correct fields", func(t *testing.T) {
		assert.Equal(t, workflowID.String(), event.WorkflowID)
		assert.Equal(t, "TestWorkflow", event.Name)
		assert.Equal(t, "initial", event.InitialState)
		assert.Equal(t, actorID.String(), event.CreatedBy)
	})

	t.Run("payload contains all data", func(t *testing.T) {
		payload := event.Payload()

		assert.Equal(t, workflowID.String(), payload["workflow_id"])
		assert.Equal(t, "TestWorkflow", payload["name"])
		assert.Equal(t, "initial", payload["initial_state"])
		assert.Equal(t, actorID.String(), payload["created_by"])
	})
}

func TestWorkflowUpdated_Event(t *testing.T) {
	workflowID := shared.NewID()
	actorID := shared.NewID()
	changes := map[string]interface{}{
		"added_states":   []string{"new_state"},
		"modified_event": "submit",
	}

	event := NewWorkflowUpdated(workflowID, "1.1.0", actorID, changes)

	t.Run("has correct type", func(t *testing.T) {
		assert.Equal(t, "workflow.updated", event.Type())
	})

	t.Run("has correct fields", func(t *testing.T) {
		assert.Equal(t, workflowID.String(), event.WorkflowID)
		assert.Equal(t, "1.1.0", event.Version)
		assert.Equal(t, actorID.String(), event.UpdatedBy)
		assert.Equal(t, changes, event.Changes)
	})

	t.Run("payload contains all data", func(t *testing.T) {
		payload := event.Payload()

		assert.Equal(t, workflowID.String(), payload["workflow_id"])
		assert.Equal(t, "1.1.0", payload["version"])
		assert.Equal(t, actorID.String(), payload["updated_by"])
		assert.Equal(t, changes, payload["changes"])
	})
}

func TestDomainEvent_Interface(t *testing.T) {
	// Verify that all concrete events implement DomainEvent interface
	t.Run("all events implement DomainEvent", func(t *testing.T) {
		instanceID := shared.NewID()
		workflowID := shared.NewID()
		actorID := shared.NewID()

		events := []DomainEvent{
			NewInstanceCreated(instanceID, workflowID, "test", "initial", actorID, nil),
			NewStateChanged(instanceID, "a", "b", "event", actorID, shared.NewID(), nil),
			NewSubStateChanged(instanceID, "state", "sub_a", "sub_b", actorID),
			NewInstancePaused(instanceID, actorID, "reason"),
			NewInstanceResumed(instanceID, actorID),
			NewInstanceCompleted(instanceID, "final", actorID, nil),
			NewInstanceCanceled(instanceID, "state", "reason", actorID),
			NewInstanceFailed(instanceID, "state", "error", actorID),
			NewDocumentReclassified(instanceID, "old", "new", "reason", actorID),
			NewWorkflowCreated(workflowID, "test", "initial", actorID),
			NewWorkflowUpdated(workflowID, "1.0.0", actorID, nil),
		}

		for _, event := range events {
			assert.NotEmpty(t, event.Type())
			assert.NotEmpty(t, event.AggregateID())
			assert.False(t, event.OccurredAt().IsZero())
			assert.NotNil(t, event.Payload())
		}
	})
}

func TestDomainEvent_Timestamps(t *testing.T) {
	t.Run("events have recent timestamps", func(t *testing.T) {
		before := time.Now().UTC()
		time.Sleep(1 * time.Millisecond)

		instanceID := shared.NewID()
		workflowID := shared.NewID()
		actorID := shared.NewID()
		event := NewInstanceCreated(instanceID, workflowID, "test", "initial", actorID, nil)

		time.Sleep(1 * time.Millisecond)
		after := time.Now().UTC()

		assert.True(t, event.OccurredAt().After(before))
		assert.True(t, event.OccurredAt().Before(after))
	})
}

func TestDomainEvent_PayloadStructure(t *testing.T) {
	t.Run("payloads include occurred_at", func(t *testing.T) {
		instanceID := shared.NewID()
		workflowID := shared.NewID()
		actorID := shared.NewID()

		events := []DomainEvent{
			NewInstanceCreated(instanceID, workflowID, "test", "initial", actorID, nil),
			NewStateChanged(instanceID, "a", "b", "event", actorID, shared.NewID(), nil),
			NewInstanceCompleted(instanceID, "final", actorID, nil),
		}

		for _, event := range events {
			payload := event.Payload()
			occurredAt, exists := payload["occurred_at"]

			require.True(t, exists, "payload should contain occurred_at for event type: %s", event.Type())
			assert.IsType(t, time.Time{}, occurredAt, "occurred_at should be time.Time for event type: %s", event.Type())
		}
	})
}
