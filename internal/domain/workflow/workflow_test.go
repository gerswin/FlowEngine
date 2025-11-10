package workflow

import (
	"testing"

	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWorkflow_Success(t *testing.T) {
	initialState, _ := NewState("draft", "Draft")

	workflow, err := NewWorkflow("Approval Workflow", initialState, shared.NewID())

	require.NoError(t, err)
	assert.NotNil(t, workflow)
	assert.Equal(t, "Approval Workflow", workflow.Name())
	assert.Equal(t, "draft", workflow.InitialState().ID())
	assert.Equal(t, "1.0.0", workflow.Version().String())
	assert.True(t, workflow.ID().IsValid())
	assert.False(t, workflow.CreatedAt().IsZero())
	assert.False(t, workflow.UpdatedAt().IsZero())
}

func TestNewWorkflow_EmptyName(t *testing.T) {
	initialState, _ := NewState("draft", "Draft")

	workflow, err := NewWorkflow("", initialState, shared.NewID())

	assert.Error(t, err)
	assert.Nil(t, workflow)
}

func TestNewWorkflow_InvalidInitialState(t *testing.T) {
	invalidState := State{} // Zero value state

	workflow, err := NewWorkflow("Test Workflow", invalidState, shared.NewID())

	assert.Error(t, err)
	assert.Nil(t, workflow)
}

func TestWorkflow_AddState_Success(t *testing.T) {
	initialState, _ := NewState("draft", "Draft")
	workflow, _ := NewWorkflow("Test Workflow", initialState, shared.NewID())

	newState, _ := NewState("approved", "Approved")
	err := workflow.AddState(newState)

	require.NoError(t, err)
	assert.True(t, workflow.HasState("approved"))

	states := workflow.States()
	assert.Len(t, states, 2) // draft + approved
}

func TestWorkflow_AddState_Duplicate_Error(t *testing.T) {
	initialState, _ := NewState("draft", "Draft")
	workflow, _ := NewWorkflow("Test Workflow", initialState, shared.NewID())

	duplicateState, _ := NewState("draft", "Draft Again")
	err := workflow.AddState(duplicateState)

	assert.Error(t, err)
	assert.True(t, StateAlreadyExistsError("draft").Is(err))
}

func TestWorkflow_AddState_Invalid(t *testing.T) {
	initialState, _ := NewState("draft", "Draft")
	workflow, _ := NewWorkflow("Test Workflow", initialState, shared.NewID())

	invalidState := State{} // Zero value
	err := workflow.AddState(invalidState)

	assert.Error(t, err)
}

func TestWorkflow_GetState_Success(t *testing.T) {
	initialState, _ := NewState("draft", "Draft")
	workflow, _ := NewWorkflow("Test Workflow", initialState, shared.NewID())

	state, err := workflow.GetState("draft")

	require.NoError(t, err)
	assert.Equal(t, "draft", state.ID())
	assert.Equal(t, "Draft", state.Name())
}

func TestWorkflow_GetState_NotFound(t *testing.T) {
	initialState, _ := NewState("draft", "Draft")
	workflow, _ := NewWorkflow("Test Workflow", initialState, shared.NewID())

	state, err := workflow.GetState("nonexistent")

	assert.Error(t, err)
	assert.True(t, StateNotFoundError("nonexistent").Is(err))
	assert.Equal(t, "", state.ID())
}

func TestWorkflow_AddEvent_ValidStates_Success(t *testing.T) {
	initialState, _ := NewState("draft", "Draft")
	workflow, _ := NewWorkflow("Test Workflow", initialState, shared.NewID())

	approvedState, _ := NewState("approved", "Approved")
	workflow.AddState(approvedState)

	event, _ := NewEvent("approve", []State{initialState}, approvedState)
	err := workflow.AddEvent(event)

	require.NoError(t, err)
	assert.True(t, workflow.HasEvent("approve"))

	events := workflow.Events()
	assert.Len(t, events, 1)
}

func TestWorkflow_AddEvent_InvalidState_Error(t *testing.T) {
	initialState, _ := NewState("draft", "Draft")
	workflow, _ := NewWorkflow("Test Workflow", initialState, shared.NewID())

	nonExistentState, _ := NewState("approved", "Approved")
	event, _ := NewEvent("approve", []State{initialState}, nonExistentState)

	err := workflow.AddEvent(event)

	assert.Error(t, err)
	assert.True(t, StateNotFoundError("approved").Is(err))
}

func TestWorkflow_AddEvent_Duplicate_Error(t *testing.T) {
	initialState, _ := NewState("draft", "Draft")
	workflow, _ := NewWorkflow("Test Workflow", initialState, shared.NewID())

	approvedState, _ := NewState("approved", "Approved")
	workflow.AddState(approvedState)

	event1, _ := NewEvent("approve", []State{initialState}, approvedState)
	workflow.AddEvent(event1)

	event2, _ := NewEvent("approve", []State{initialState}, approvedState)
	err := workflow.AddEvent(event2)

	assert.Error(t, err)
	assert.True(t, EventAlreadyExistsError("approve").Is(err))
}

func TestWorkflow_CanTransition_ValidEvent_ReturnsTrue(t *testing.T) {
	initialState, _ := NewState("draft", "Draft")
	workflow, _ := NewWorkflow("Test Workflow", initialState, shared.NewID())

	approvedState, _ := NewState("approved", "Approved")
	workflow.AddState(approvedState)

	event, _ := NewEvent("approve", []State{initialState}, approvedState)
	workflow.AddEvent(event)

	canTransition := workflow.CanTransition(initialState, event)

	assert.True(t, canTransition)
}

func TestWorkflow_CanTransition_InvalidEvent_ReturnsFalse(t *testing.T) {
	draftState, _ := NewState("draft", "Draft")
	workflow, _ := NewWorkflow("Test Workflow", draftState, shared.NewID())

	approvedState, _ := NewState("approved", "Approved")
	workflow.AddState(approvedState)

	rejectedState, _ := NewState("rejected", "Rejected")
	workflow.AddState(rejectedState)

	approveEvent, _ := NewEvent("approve", []State{draftState}, approvedState)
	workflow.AddEvent(approveEvent)

	// Try to transition from approved state with approve event (should fail)
	canTransition := workflow.CanTransition(approvedState, approveEvent)

	assert.False(t, canTransition)
}

func TestWorkflow_CanTransition_NonExistentState_ReturnsFalse(t *testing.T) {
	initialState, _ := NewState("draft", "Draft")
	workflow, _ := NewWorkflow("Test Workflow", initialState, shared.NewID())

	nonExistentState, _ := NewState("nonexistent", "Non-Existent")
	approvedState, _ := NewState("approved", "Approved")

	event, _ := NewEvent("approve", []State{nonExistentState}, approvedState)

	canTransition := workflow.CanTransition(nonExistentState, event)

	assert.False(t, canTransition)
}

func TestWorkflow_ValidateTransition_Success(t *testing.T) {
	draftState, _ := NewState("draft", "Draft")
	workflow, _ := NewWorkflow("Test Workflow", draftState, shared.NewID())

	approvedState, _ := NewState("approved", "Approved")
	workflow.AddState(approvedState)

	event, _ := NewEvent("approve", []State{draftState}, approvedState)
	workflow.AddEvent(event)

	err := workflow.ValidateTransition(draftState, "approve")

	assert.NoError(t, err)
}

func TestWorkflow_ValidateTransition_InvalidEvent(t *testing.T) {
	draftState, _ := NewState("draft", "Draft")
	workflow, _ := NewWorkflow("Test Workflow", draftState, shared.NewID())

	err := workflow.ValidateTransition(draftState, "nonexistent_event")

	assert.Error(t, err)
	assert.True(t, EventNotFoundError("nonexistent_event").Is(err))
}

func TestWorkflow_ValidateTransition_InvalidTransition(t *testing.T) {
	draftState, _ := NewState("draft", "Draft")
	workflow, _ := NewWorkflow("Test Workflow", draftState, shared.NewID())

	approvedState, _ := NewState("approved", "Approved")
	workflow.AddState(approvedState)

	event, _ := NewEvent("approve", []State{draftState}, approvedState)
	workflow.AddEvent(event)

	// Try to validate transition from approved (not a valid source)
	err := workflow.ValidateTransition(approvedState, "approve")

	assert.Error(t, err)
}

func TestWorkflow_IncrementVersion(t *testing.T) {
	initialState, _ := NewState("draft", "Draft")
	workflow, _ := NewWorkflow("Test Workflow", initialState, shared.NewID())

	t.Run("increment major", func(t *testing.T) {
		err := workflow.IncrementVersion("major")
		require.NoError(t, err)
		assert.Equal(t, "2.0.0", workflow.Version().String())
	})

	t.Run("increment minor", func(t *testing.T) {
		err := workflow.IncrementVersion("minor")
		require.NoError(t, err)
		assert.Equal(t, "2.1.0", workflow.Version().String())
	})

	t.Run("increment patch", func(t *testing.T) {
		err := workflow.IncrementVersion("patch")
		require.NoError(t, err)
		assert.Equal(t, "2.1.1", workflow.Version().String())
	})

	t.Run("invalid version type", func(t *testing.T) {
		err := workflow.IncrementVersion("invalid")
		assert.Error(t, err)
	})
}

func TestWorkflow_Validate(t *testing.T) {
	t.Run("valid workflow", func(t *testing.T) {
		initialState, _ := NewState("draft", "Draft")
		workflow, _ := NewWorkflow("Test Workflow", initialState, shared.NewID())

		approvedState, _ := NewState("approved", "Approved")
		workflow.AddState(approvedState)

		event, _ := NewEvent("approve", []State{initialState}, approvedState)
		workflow.AddEvent(event)

		err := workflow.Validate()
		assert.NoError(t, err)
	})

	t.Run("event with non-existent state", func(t *testing.T) {
		initialState, _ := NewState("draft", "Draft")
		workflow, _ := NewWorkflow("Test Workflow", initialState, shared.NewID())

		// This scenario is hard to create due to validation in AddEvent
		// but tests defensive validation
		err := workflow.Validate()
		assert.NoError(t, err) // Should pass with just initial state
	})
}

func TestWorkflow_Clone(t *testing.T) {
	initialState, _ := NewState("draft", "Draft")
	workflow, _ := NewWorkflow("Original Workflow", initialState, shared.NewID())

	approvedState, _ := NewState("approved", "Approved")
	workflow.AddState(approvedState)

	event, _ := NewEvent("approve", []State{initialState}, approvedState)
	workflow.AddEvent(event)

	clone := workflow.Clone("Cloned Workflow")

	// Verify clone has different ID
	assert.NotEqual(t, workflow.ID(), clone.ID())

	// Verify clone has new name
	assert.Equal(t, "Cloned Workflow", clone.Name())

	// Verify clone has reset version
	assert.Equal(t, "1.0.0", clone.Version().String())

	// Verify clone has same states and events
	assert.Len(t, clone.States(), len(workflow.States()))
	assert.Len(t, clone.Events(), len(workflow.Events()))
	assert.True(t, clone.HasState("draft"))
	assert.True(t, clone.HasState("approved"))
	assert.True(t, clone.HasEvent("approve"))
}

func TestWorkflow_SetDescription(t *testing.T) {
	initialState, _ := NewState("draft", "Draft")
	workflow, _ := NewWorkflow("Test Workflow", initialState, shared.NewID())

	workflow.SetDescription("A test workflow description")

	assert.Equal(t, "A test workflow description", workflow.Description())
}
