package instance

import (
	"testing"

	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewInstance_Success(t *testing.T) {
	workflowID := shared.NewID()
	actor := shared.NewID()

	instance, err := NewInstance(workflowID, "Approval Workflow", "draft", actor)

	require.NoError(t, err)
	assert.NotNil(t, instance)
	assert.True(t, instance.ID().IsValid())
	assert.Equal(t, workflowID, instance.WorkflowID())
	assert.Equal(t, "Approval Workflow", instance.WorkflowName())
	assert.Equal(t, "draft", instance.CurrentState())
	assert.Equal(t, StatusRunning, instance.Status())
	assert.Equal(t, int64(1), instance.Version().Value())
	assert.True(t, instance.IsActive())
	assert.False(t, instance.IsFinal())
}

func TestNewInstance_InvalidInputs(t *testing.T) {
	validID := shared.NewID()

	tests := []struct {
		name         string
		workflowID   shared.ID
		workflowName string
		initialState string
		actor        shared.ID
		wantErr      bool
	}{
		{"invalid workflow ID", shared.NilID(), "Test", "draft", validID, true},
		{"empty workflow name", validID, "", "draft", validID, true},
		{"empty initial state", validID, "Test", "", validID, true},
		{"invalid actor ID", validID, "Test", "draft", shared.NilID(), true},
		{"valid inputs", validID, "Test", "draft", validID, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance, err := NewInstance(tt.workflowID, tt.workflowName, tt.initialState, tt.actor)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, instance)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, instance)
			}
		})
	}
}

func TestNewInstanceWithSubState_Success(t *testing.T) {
	workflowID := shared.NewID()
	actor := shared.NewID()
	subState, _ := NewSubState("pending_review", "Pending Review")

	instance, err := NewInstanceWithSubState(workflowID, "Test Workflow", "pending", subState, actor)

	require.NoError(t, err)
	assert.True(t, instance.HasSubState())
	assert.Equal(t, "pending_review", instance.CurrentSubState().ID())
}

func TestInstance_SetData(t *testing.T) {
	instance := createTestInstance(t)
	initialVersion := instance.Version()

	data := NewData().Set("key1", "value1")
	instance.SetData(data)

	assert.Equal(t, "value1", instance.Data().Get("key1"))
	assert.True(t, instance.Version().IsGreaterThan(initialVersion))
}

func TestInstance_UpdateData(t *testing.T) {
	instance := createTestInstance(t)

	instance.UpdateData("name", "John Doe")

	val, ok := instance.Data().GetString("name")
	assert.True(t, ok)
	assert.Equal(t, "John Doe", val)
}

func TestInstance_Transition_Success(t *testing.T) {
	instance := createTestInstance(t)
	actor := shared.NewID()
	metadata := NewTransitionMetadataWithReason("Approved by manager")

	err := instance.Transition("approved", "approve", actor, metadata)

	require.NoError(t, err)
	assert.Equal(t, "approved", instance.CurrentState())
	assert.Equal(t, 1, instance.TransitionCount())

	lastTransition := instance.GetLastTransition()
	assert.NotNil(t, lastTransition)
	assert.Equal(t, "draft", lastTransition.From())
	assert.Equal(t, "approved", lastTransition.To())
	assert.Equal(t, "approve", lastTransition.Event())
	assert.Equal(t, "Approved by manager", lastTransition.Metadata().Reason())
}

func TestInstance_TransitionWithSubState_Success(t *testing.T) {
	workflowID := shared.NewID()
	actor := shared.NewID()
	initialSubState, _ := NewSubState("pending_manager", "Pending Manager")

	instance, _ := NewInstanceWithSubState(workflowID, "Test Workflow", "pending", initialSubState, actor)

	toSubState, _ := NewSubState("pending_director", "Pending Director")
	metadata := EmptyTransitionMetadata()

	err := instance.TransitionWithSubState("pending", toSubState, "escalate", actor, metadata)

	require.NoError(t, err)
	assert.Equal(t, "pending", instance.CurrentState())
	assert.Equal(t, "pending_director", instance.CurrentSubState().ID())

	lastTransition := instance.GetLastTransition()
	assert.True(t, lastTransition.HasSubStates())
	assert.Equal(t, "pending_manager", lastTransition.FromSubState().ID())
	assert.Equal(t, "pending_director", lastTransition.ToSubState().ID())
}

func TestInstance_Transition_InvalidWhenNotActive(t *testing.T) {
	instance := createTestInstance(t)
	actor := shared.NewID()

	// Complete the instance
	instance.Complete(actor)

	// Try to transition
	err := instance.Transition("rejected", "reject", actor, EmptyTransitionMetadata())

	assert.Error(t, err)
	assert.True(t, InstanceNotActiveError(instance.ID(), instance.Status()).Is(err))
}

func TestInstance_Complete_Success(t *testing.T) {
	instance := createTestInstance(t)
	actor := shared.NewID()

	err := instance.Complete(actor)

	require.NoError(t, err)
	assert.Equal(t, StatusCompleted, instance.Status())
	assert.True(t, instance.IsFinal())
	assert.False(t, instance.IsActive())
	assert.False(t, instance.CompletedAt().IsZero())
}

func TestInstance_Cancel_Success(t *testing.T) {
	instance := createTestInstance(t)
	actor := shared.NewID()

	err := instance.Cancel(actor, "Canceled by user")

	require.NoError(t, err)
	assert.Equal(t, StatusCanceled, instance.Status())
	assert.True(t, instance.IsFinal())

	reason, ok := instance.Data().GetString("cancellation_reason")
	assert.True(t, ok)
	assert.Equal(t, "Canceled by user", reason)
}

func TestInstance_Fail_Success(t *testing.T) {
	instance := createTestInstance(t)
	actor := shared.NewID()

	err := instance.Fail(actor, "System error occurred")

	require.NoError(t, err)
	assert.Equal(t, StatusFailed, instance.Status())
	assert.True(t, instance.IsFinal())

	reason, ok := instance.Data().GetString("failure_reason")
	assert.True(t, ok)
	assert.Equal(t, "System error occurred", reason)
}

func TestInstance_PauseAndResume(t *testing.T) {
	instance := createTestInstance(t)
	actor := shared.NewID()

	// Pause
	err := instance.Pause(actor)
	require.NoError(t, err)
	assert.Equal(t, StatusPaused, instance.Status())
	assert.True(t, instance.IsActive())

	// Resume
	err = instance.Resume(actor)
	require.NoError(t, err)
	assert.Equal(t, StatusRunning, instance.Status())
	assert.True(t, instance.IsActive())
}

func TestInstance_Pause_InvalidWhenNotRunning(t *testing.T) {
	instance := createTestInstance(t)
	actor := shared.NewID()

	instance.Complete(actor)

	err := instance.Pause(actor)
	assert.Error(t, err)
}

func TestInstance_Resume_InvalidWhenNotPaused(t *testing.T) {
	instance := createTestInstance(t)
	actor := shared.NewID()

	err := instance.Resume(actor)
	assert.Error(t, err)
}

func TestInstance_Validate(t *testing.T) {
	t.Run("valid instance", func(t *testing.T) {
		instance := createTestInstance(t)
		err := instance.Validate()
		assert.NoError(t, err)
	})

	t.Run("invalid instance with zero version", func(t *testing.T) {
		instance := createTestInstance(t)
		instance.version = ZeroVersion()

		err := instance.Validate()
		assert.Error(t, err)
	})
}

func TestInstance_Variables(t *testing.T) {
	instance := createTestInstance(t)

	instance.UpdateVariable("approved_by", "manager@example.com")
	instance.UpdateVariable("approval_count", 3)

	email, ok := instance.Variables().GetString("approved_by")
	assert.True(t, ok)
	assert.Equal(t, "manager@example.com", email)

	count, ok := instance.Variables().GetInt("approval_count")
	assert.True(t, ok)
	assert.Equal(t, 3, count)
}

// Helper function to create a test instance
func createTestInstance(t *testing.T) *Instance {
	workflowID := shared.NewID()
	actor := shared.NewID()

	instance, err := NewInstance(workflowID, "Test Workflow", "draft", actor)
	require.NoError(t, err)

	return instance
}
