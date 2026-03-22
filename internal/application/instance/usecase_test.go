package instance

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	eventMocks "github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/event/mocks"
	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/instance"
	instanceMocks "github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/instance/mocks"
	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/shared"
	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/workflow"
	workflowMocks "github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/workflow/mocks"
)

// ---------- helpers ----------

func newTestWorkflow(t *testing.T) *workflow.Workflow {
	t.Helper()
	initialState, err := workflow.NewState("draft", "Draft")
	require.NoError(t, err)
	creatorID := shared.NewID()
	wf, err := workflow.NewWorkflow("TestWorkflow", initialState, creatorID)
	require.NoError(t, err)
	return wf
}

func newTestWorkflowWithTransition(t *testing.T) *workflow.Workflow {
	t.Helper()
	wf := newTestWorkflow(t)

	reviewState, err := workflow.NewState("review", "Review")
	require.NoError(t, err)
	err = wf.AddState(reviewState)
	require.NoError(t, err)

	approvedState, err := workflow.NewState("approved", "Approved")
	require.NoError(t, err)
	approvedState = approvedState.AsFinal()
	err = wf.AddState(approvedState)
	require.NoError(t, err)

	draftState, _ := wf.GetState("draft")
	evt, err := workflow.NewEvent("submit", []workflow.State{draftState}, reviewState)
	require.NoError(t, err)
	err = wf.AddEvent(evt)
	require.NoError(t, err)

	reviewStateFromWf, _ := wf.GetState("review")
	approveEvt, err := workflow.NewEvent("approve", []workflow.State{reviewStateFromWf}, approvedState)
	require.NoError(t, err)
	err = wf.AddEvent(approveEvt)
	require.NoError(t, err)

	return wf
}

func newTestInstance(t *testing.T, wf *workflow.Workflow) *instance.Instance {
	t.Helper()
	actorID := shared.NewID()
	inst, err := instance.NewInstance(
		wf.ID(),
		wf.Name(),
		wf.InitialState().ID,
		actorID,
	)
	require.NoError(t, err)
	return inst
}

// ==================== CreateInstanceUseCase ====================

func TestCreateInstanceUseCase_Success(t *testing.T) {
	mockInstanceRepo := instanceMocks.NewRepository(t)
	mockWorkflowRepo := workflowMocks.NewRepository(t)
	mockEventBus := eventMocks.NewDispatcher(t)

	wf := newTestWorkflow(t)
	actorID := shared.NewID()

	mockWorkflowRepo.On("FindByID", mock.Anything, mock.AnythingOfType("shared.ID")).Return(wf, nil)
	mockInstanceRepo.On("Save", mock.Anything, mock.AnythingOfType("*instance.Instance")).Return(nil)
	mockEventBus.On("DispatchBatch", mock.Anything, mock.Anything).Return(nil)

	uc := NewCreateInstanceUseCase(mockWorkflowRepo, mockInstanceRepo, mockEventBus)

	result, err := uc.Execute(context.Background(), CreateInstanceCommand{
		WorkflowID: wf.ID().String(),
		StartedBy:  actorID.String(),
	})

	require.NoError(t, err)
	assert.NotEmpty(t, result.ID)
	assert.Equal(t, wf.ID().String(), result.WorkflowID)
	assert.Equal(t, "TestWorkflow", result.WorkflowName)
	assert.Equal(t, "draft", result.CurrentState)
	assert.Equal(t, "RUNNING", result.Status)
	assert.Empty(t, result.ParentID)
}

func TestCreateInstanceUseCase_InvalidWorkflowIDFormat(t *testing.T) {
	mockInstanceRepo := instanceMocks.NewRepository(t)
	mockWorkflowRepo := workflowMocks.NewRepository(t)
	mockEventBus := eventMocks.NewDispatcher(t)

	uc := NewCreateInstanceUseCase(mockWorkflowRepo, mockInstanceRepo, mockEventBus)

	_, err := uc.Execute(context.Background(), CreateInstanceCommand{
		WorkflowID: "not-a-uuid",
		StartedBy:  shared.NewID().String(),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid workflow ID format")
}

func TestCreateInstanceUseCase_MissingActorID(t *testing.T) {
	mockInstanceRepo := instanceMocks.NewRepository(t)
	mockWorkflowRepo := workflowMocks.NewRepository(t)
	mockEventBus := eventMocks.NewDispatcher(t)

	uc := NewCreateInstanceUseCase(mockWorkflowRepo, mockInstanceRepo, mockEventBus)

	_, err := uc.Execute(context.Background(), CreateInstanceCommand{
		WorkflowID: shared.NewID().String(),
		StartedBy:  "",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "actor ID is required")
}

func TestCreateInstanceUseCase_MissingWorkflowID(t *testing.T) {
	mockInstanceRepo := instanceMocks.NewRepository(t)
	mockWorkflowRepo := workflowMocks.NewRepository(t)
	mockEventBus := eventMocks.NewDispatcher(t)

	uc := NewCreateInstanceUseCase(mockWorkflowRepo, mockInstanceRepo, mockEventBus)

	_, err := uc.Execute(context.Background(), CreateInstanceCommand{
		WorkflowID: "",
		StartedBy:  shared.NewID().String(),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "workflow ID is required")
}

func TestCreateInstanceUseCase_WorkflowNotFound(t *testing.T) {
	mockInstanceRepo := instanceMocks.NewRepository(t)
	mockWorkflowRepo := workflowMocks.NewRepository(t)
	mockEventBus := eventMocks.NewDispatcher(t)

	wfID := shared.NewID()
	mockWorkflowRepo.On("FindByID", mock.Anything, mock.AnythingOfType("shared.ID")).
		Return(nil, errors.New("workflow not found"))

	uc := NewCreateInstanceUseCase(mockWorkflowRepo, mockInstanceRepo, mockEventBus)

	_, err := uc.Execute(context.Background(), CreateInstanceCommand{
		WorkflowID: wfID.String(),
		StartedBy:  shared.NewID().String(),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "workflow not found")
}

func TestCreateInstanceUseCase_WithDataAndVariables(t *testing.T) {
	mockInstanceRepo := instanceMocks.NewRepository(t)
	mockWorkflowRepo := workflowMocks.NewRepository(t)
	mockEventBus := eventMocks.NewDispatcher(t)

	wf := newTestWorkflow(t)
	actorID := shared.NewID()

	mockWorkflowRepo.On("FindByID", mock.Anything, mock.AnythingOfType("shared.ID")).Return(wf, nil)
	mockInstanceRepo.On("Save", mock.Anything, mock.AnythingOfType("*instance.Instance")).Return(nil)
	mockEventBus.On("DispatchBatch", mock.Anything, mock.Anything).Return(nil)

	uc := NewCreateInstanceUseCase(mockWorkflowRepo, mockInstanceRepo, mockEventBus)

	result, err := uc.Execute(context.Background(), CreateInstanceCommand{
		WorkflowID: wf.ID().String(),
		StartedBy:  actorID.String(),
		Data:       map[string]interface{}{"key1": "value1"},
		Variables:  map[string]interface{}{"var1": 42},
	})

	require.NoError(t, err)
	assert.NotEmpty(t, result.ID)
	assert.Equal(t, "draft", result.CurrentState)
}

// ==================== GetInstanceUseCase ====================

func TestGetInstanceUseCase_Execute_Success(t *testing.T) {
	mockInstanceRepo := instanceMocks.NewRepository(t)

	wf := newTestWorkflow(t)
	inst := newTestInstance(t, wf)

	mockInstanceRepo.On("FindByID", mock.Anything, mock.AnythingOfType("shared.ID")).Return(inst, nil)

	uc := NewGetInstanceUseCase(mockInstanceRepo)

	dto, err := uc.Execute(context.Background(), inst.ID().String())

	require.NoError(t, err)
	assert.Equal(t, inst.ID().String(), dto.ID)
	assert.Equal(t, inst.WorkflowName(), dto.WorkflowName)
	assert.Equal(t, inst.CurrentState(), dto.CurrentState)
	assert.Equal(t, "RUNNING", dto.Status)
}

func TestGetInstanceUseCase_Execute_NotFound(t *testing.T) {
	mockInstanceRepo := instanceMocks.NewRepository(t)

	instID := shared.NewID()
	mockInstanceRepo.On("FindByID", mock.Anything, mock.AnythingOfType("shared.ID")).
		Return(nil, errors.New("instance not found"))

	uc := NewGetInstanceUseCase(mockInstanceRepo)

	_, err := uc.Execute(context.Background(), instID.String())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetInstanceUseCase_Execute_InvalidID(t *testing.T) {
	mockInstanceRepo := instanceMocks.NewRepository(t)
	uc := NewGetInstanceUseCase(mockInstanceRepo)

	_, err := uc.Execute(context.Background(), "bad-id")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid instance ID format")
}

func TestGetInstanceUseCase_ExecuteAll_Success(t *testing.T) {
	mockInstanceRepo := instanceMocks.NewRepository(t)

	wf := newTestWorkflow(t)
	inst1 := newTestInstance(t, wf)
	inst2 := newTestInstance(t, wf)

	mockInstanceRepo.On("FindAll", mock.Anything).
		Return([]*instance.Instance{inst1, inst2}, nil)

	uc := NewGetInstanceUseCase(mockInstanceRepo)

	dtos, err := uc.ExecuteAll(context.Background())

	require.NoError(t, err)
	assert.Len(t, dtos, 2)
}

func TestGetInstanceUseCase_ExecuteByWorkflow_Success(t *testing.T) {
	mockInstanceRepo := instanceMocks.NewRepository(t)

	wf := newTestWorkflow(t)
	inst := newTestInstance(t, wf)

	mockInstanceRepo.On("FindByWorkflowID", mock.Anything, mock.AnythingOfType("shared.ID")).
		Return([]*instance.Instance{inst}, nil)

	uc := NewGetInstanceUseCase(mockInstanceRepo)

	dtos, err := uc.ExecuteByWorkflow(context.Background(), wf.ID().String())

	require.NoError(t, err)
	assert.Len(t, dtos, 1)
	assert.Equal(t, inst.ID().String(), dtos[0].ID)
}

func TestGetInstanceUseCase_ExecuteList_Success(t *testing.T) {
	mockInstanceRepo := instanceMocks.NewRepository(t)

	wf := newTestWorkflow(t)
	inst := newTestInstance(t, wf)

	mockInstanceRepo.On("List", mock.Anything, mock.AnythingOfType("shared.ListQuery"), mock.Anything).
		Return([]*instance.Instance{inst}, int64(1), nil)

	uc := NewGetInstanceUseCase(mockInstanceRepo)

	result, err := uc.ExecuteList(context.Background(), 1, 10, "")

	require.NoError(t, err)
	assert.Len(t, result.Items, 1)
	assert.Equal(t, int64(1), result.Total)
}

func TestGetInstanceUseCase_ExecuteList_WithWorkflowFilter(t *testing.T) {
	mockInstanceRepo := instanceMocks.NewRepository(t)

	wf := newTestWorkflow(t)
	inst := newTestInstance(t, wf)

	mockInstanceRepo.On("List", mock.Anything, mock.AnythingOfType("shared.ListQuery"), mock.AnythingOfType("*shared.ID")).
		Return([]*instance.Instance{inst}, int64(1), nil)

	uc := NewGetInstanceUseCase(mockInstanceRepo)

	result, err := uc.ExecuteList(context.Background(), 1, 10, wf.ID().String())

	require.NoError(t, err)
	assert.Len(t, result.Items, 1)
}

func TestGetInstanceUseCase_ExecuteHistory_Success(t *testing.T) {
	mockInstanceRepo := instanceMocks.NewRepository(t)

	wf := newTestWorkflowWithTransition(t)
	inst := newTestInstance(t, wf)

	// Perform a transition to have history
	metadata := instance.NewTransitionMetadata("test reason", "test feedback", nil)
	err := inst.Transition("review", "submit", shared.NewID(), metadata, nil)
	require.NoError(t, err)

	mockInstanceRepo.On("FindByID", mock.Anything, mock.AnythingOfType("shared.ID")).Return(inst, nil)

	uc := NewGetInstanceUseCase(mockInstanceRepo)

	transitions, err := uc.ExecuteHistory(context.Background(), inst.ID().String())

	require.NoError(t, err)
	assert.Len(t, transitions, 1)
	assert.Equal(t, "draft", transitions[0].From)
	assert.Equal(t, "review", transitions[0].To)
	assert.Equal(t, "submit", transitions[0].Event)
	assert.Equal(t, "test reason", transitions[0].Reason)
	assert.Equal(t, "test feedback", transitions[0].Feedback)
}

// ==================== TransitionInstanceUseCase ====================

func TestTransitionInstanceUseCase_Success(t *testing.T) {
	mockInstanceRepo := instanceMocks.NewRepository(t)
	mockWorkflowRepo := workflowMocks.NewRepository(t)
	mockEventBus := eventMocks.NewDispatcher(t)

	wf := newTestWorkflowWithTransition(t)
	inst := newTestInstance(t, wf)
	actorID := shared.NewID()

	mockInstanceRepo.On("FindByID", mock.Anything, mock.AnythingOfType("shared.ID")).Return(inst, nil)
	mockWorkflowRepo.On("FindByID", mock.Anything, mock.AnythingOfType("shared.ID")).Return(wf, nil)
	mockInstanceRepo.On("Save", mock.Anything, mock.AnythingOfType("*instance.Instance")).Return(nil)
	mockEventBus.On("DispatchBatch", mock.Anything, mock.Anything).Return(nil)

	engine := instance.NewEngine()
	uc := NewTransitionInstanceUseCase(mockWorkflowRepo, mockInstanceRepo, mockEventBus, engine)

	result, err := uc.Execute(context.Background(), TransitionInstanceCommand{
		InstanceID: inst.ID().String(),
		Event:      "submit",
		ActorID:    actorID.String(),
	})

	require.NoError(t, err)
	assert.Equal(t, "draft", result.PreviousState)
	assert.Equal(t, "review", result.CurrentState)
	assert.Equal(t, "RUNNING", result.Status)
	assert.NotEmpty(t, result.TransitionID)
}

func TestTransitionInstanceUseCase_InvalidInstanceID(t *testing.T) {
	mockInstanceRepo := instanceMocks.NewRepository(t)
	mockWorkflowRepo := workflowMocks.NewRepository(t)
	mockEventBus := eventMocks.NewDispatcher(t)

	uc := NewTransitionInstanceUseCase(mockWorkflowRepo, mockInstanceRepo, mockEventBus, nil)

	_, err := uc.Execute(context.Background(), TransitionInstanceCommand{
		InstanceID: "",
		Event:      "submit",
		ActorID:    shared.NewID().String(),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "instance ID is required")
}

func TestTransitionInstanceUseCase_MissingEvent(t *testing.T) {
	mockInstanceRepo := instanceMocks.NewRepository(t)
	mockWorkflowRepo := workflowMocks.NewRepository(t)
	mockEventBus := eventMocks.NewDispatcher(t)

	uc := NewTransitionInstanceUseCase(mockWorkflowRepo, mockInstanceRepo, mockEventBus, nil)

	_, err := uc.Execute(context.Background(), TransitionInstanceCommand{
		InstanceID: shared.NewID().String(),
		Event:      "",
		ActorID:    shared.NewID().String(),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "event name is required")
}

func TestTransitionInstanceUseCase_MissingActorID(t *testing.T) {
	mockInstanceRepo := instanceMocks.NewRepository(t)
	mockWorkflowRepo := workflowMocks.NewRepository(t)
	mockEventBus := eventMocks.NewDispatcher(t)

	uc := NewTransitionInstanceUseCase(mockWorkflowRepo, mockInstanceRepo, mockEventBus, nil)

	_, err := uc.Execute(context.Background(), TransitionInstanceCommand{
		InstanceID: shared.NewID().String(),
		Event:      "submit",
		ActorID:    "",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "actor ID is required")
}

func TestTransitionInstanceUseCase_InstanceNotFound(t *testing.T) {
	mockInstanceRepo := instanceMocks.NewRepository(t)
	mockWorkflowRepo := workflowMocks.NewRepository(t)
	mockEventBus := eventMocks.NewDispatcher(t)

	mockInstanceRepo.On("FindByID", mock.Anything, mock.AnythingOfType("shared.ID")).
		Return(nil, errors.New("instance not found"))

	uc := NewTransitionInstanceUseCase(mockWorkflowRepo, mockInstanceRepo, mockEventBus, nil)

	_, err := uc.Execute(context.Background(), TransitionInstanceCommand{
		InstanceID: shared.NewID().String(),
		Event:      "submit",
		ActorID:    shared.NewID().String(),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestTransitionInstanceUseCase_InvalidEvent(t *testing.T) {
	mockInstanceRepo := instanceMocks.NewRepository(t)
	mockWorkflowRepo := workflowMocks.NewRepository(t)
	mockEventBus := eventMocks.NewDispatcher(t)

	wf := newTestWorkflowWithTransition(t)
	inst := newTestInstance(t, wf)

	mockInstanceRepo.On("FindByID", mock.Anything, mock.AnythingOfType("shared.ID")).Return(inst, nil)
	mockWorkflowRepo.On("FindByID", mock.Anything, mock.AnythingOfType("shared.ID")).Return(wf, nil)

	uc := NewTransitionInstanceUseCase(mockWorkflowRepo, mockInstanceRepo, mockEventBus, nil)

	_, err := uc.Execute(context.Background(), TransitionInstanceCommand{
		InstanceID: inst.ID().String(),
		Event:      "nonexistent_event",
		ActorID:    shared.NewID().String(),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestTransitionInstanceUseCase_InvalidTransitionFromState(t *testing.T) {
	mockInstanceRepo := instanceMocks.NewRepository(t)
	mockWorkflowRepo := workflowMocks.NewRepository(t)
	mockEventBus := eventMocks.NewDispatcher(t)

	wf := newTestWorkflowWithTransition(t)
	inst := newTestInstance(t, wf)

	mockInstanceRepo.On("FindByID", mock.Anything, mock.AnythingOfType("shared.ID")).Return(inst, nil)
	mockWorkflowRepo.On("FindByID", mock.Anything, mock.AnythingOfType("shared.ID")).Return(wf, nil)

	uc := NewTransitionInstanceUseCase(mockWorkflowRepo, mockInstanceRepo, mockEventBus, nil)

	// "approve" can only be triggered from "review" state, not "draft"
	_, err := uc.Execute(context.Background(), TransitionInstanceCommand{
		InstanceID: inst.ID().String(),
		Event:      "approve",
		ActorID:    shared.NewID().String(),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid transition")
}

func TestTransitionInstanceUseCase_GuardFailure(t *testing.T) {
	mockInstanceRepo := instanceMocks.NewRepository(t)
	mockWorkflowRepo := workflowMocks.NewRepository(t)
	mockEventBus := eventMocks.NewDispatcher(t)

	// Build a workflow with a guard on the submit event
	initialState, _ := workflow.NewState("draft", "Draft")
	creatorID := shared.NewID()
	wf, err := workflow.NewWorkflow("GuardedWorkflow", initialState, creatorID)
	require.NoError(t, err)

	reviewState, _ := workflow.NewState("review", "Review")
	err = wf.AddState(reviewState)
	require.NoError(t, err)

	draftState, _ := wf.GetState("draft")
	evt, _ := workflow.NewEvent("submit", []workflow.State{draftState}, reviewState)
	// Add a guard that requires a field to exist
	evt = evt.WithGuards([]workflow.GuardConfig{
		{Type: "field_exists", Params: map[string]interface{}{"field": "required_field"}},
	})
	err = wf.AddEvent(evt)
	require.NoError(t, err)

	inst := newTestInstance(t, wf)
	actorID := shared.NewID()

	mockInstanceRepo.On("FindByID", mock.Anything, mock.AnythingOfType("shared.ID")).Return(inst, nil)
	mockWorkflowRepo.On("FindByID", mock.Anything, mock.AnythingOfType("shared.ID")).Return(wf, nil)

	engine := instance.NewEngine()
	uc := NewTransitionInstanceUseCase(mockWorkflowRepo, mockInstanceRepo, mockEventBus, engine)

	// Instance has no "required_field" data so the guard should fail
	_, err = uc.Execute(context.Background(), TransitionInstanceCommand{
		InstanceID: inst.ID().String(),
		Event:      "submit",
		ActorID:    actorID.String(),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "required_field")
}

func TestTransitionInstanceUseCase_WithData(t *testing.T) {
	mockInstanceRepo := instanceMocks.NewRepository(t)
	mockWorkflowRepo := workflowMocks.NewRepository(t)
	mockEventBus := eventMocks.NewDispatcher(t)

	wf := newTestWorkflowWithTransition(t)
	inst := newTestInstance(t, wf)
	actorID := shared.NewID()

	mockInstanceRepo.On("FindByID", mock.Anything, mock.AnythingOfType("shared.ID")).Return(inst, nil)
	mockWorkflowRepo.On("FindByID", mock.Anything, mock.AnythingOfType("shared.ID")).Return(wf, nil)
	mockInstanceRepo.On("Save", mock.Anything, mock.AnythingOfType("*instance.Instance")).Return(nil)
	mockEventBus.On("DispatchBatch", mock.Anything, mock.Anything).Return(nil)

	engine := instance.NewEngine()
	uc := NewTransitionInstanceUseCase(mockWorkflowRepo, mockInstanceRepo, mockEventBus, engine)

	result, err := uc.Execute(context.Background(), TransitionInstanceCommand{
		InstanceID: inst.ID().String(),
		Event:      "submit",
		ActorID:    actorID.String(),
		Data:       map[string]interface{}{"note": "urgent"},
		Reason:     "needs review",
		Feedback:   "please review",
	})

	require.NoError(t, err)
	assert.Equal(t, "review", result.CurrentState)
}

// ==================== CloneInstanceUseCase ====================

func TestCloneInstanceUseCase_Success(t *testing.T) {
	mockInstanceRepo := instanceMocks.NewRepository(t)
	mockWorkflowRepo := workflowMocks.NewRepository(t)
	mockEventBus := eventMocks.NewDispatcher(t)

	wf := newTestWorkflow(t)
	parent := newTestInstance(t, wf)

	mockInstanceRepo.On("FindByID", mock.Anything, mock.AnythingOfType("shared.ID")).Return(parent, nil)
	mockInstanceRepo.On("Save", mock.Anything, mock.AnythingOfType("*instance.Instance")).Return(nil)

	uc := NewCloneInstanceUseCase(mockInstanceRepo, mockWorkflowRepo, mockEventBus)

	result, err := uc.Execute(context.Background(), CloneInstanceCommand{
		ParentInstanceID: parent.ID().String(),
		Assignees: []CloneAssignee{
			{UserID: shared.NewID().String(), OfficeID: "office1"},
			{UserID: shared.NewID().String(), OfficeID: "office2"},
		},
		ConsolidatorID:  shared.NewID().String(),
		Reason:          "need opinions",
		TimeoutDuration: "24h",
	})

	require.NoError(t, err)
	assert.NotEmpty(t, result.CloneGroupID)
	assert.Equal(t, parent.ID().String(), result.ParentID)
	assert.Len(t, result.ClonedInstances, 2)
	assert.False(t, result.ExpiresAt.IsZero())
}

func TestCloneInstanceUseCase_ParentNotFound(t *testing.T) {
	mockInstanceRepo := instanceMocks.NewRepository(t)
	mockWorkflowRepo := workflowMocks.NewRepository(t)
	mockEventBus := eventMocks.NewDispatcher(t)

	parentID := shared.NewID()
	mockInstanceRepo.On("FindByID", mock.Anything, mock.AnythingOfType("shared.ID")).
		Return(nil, errors.New("instance not found"))

	uc := NewCloneInstanceUseCase(mockInstanceRepo, mockWorkflowRepo, mockEventBus)

	_, err := uc.Execute(context.Background(), CloneInstanceCommand{
		ParentInstanceID: parentID.String(),
		Assignees:        []CloneAssignee{{UserID: shared.NewID().String()}},
		TimeoutDuration:  "1h",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestCloneInstanceUseCase_ParentNotRunning(t *testing.T) {
	mockInstanceRepo := instanceMocks.NewRepository(t)
	mockWorkflowRepo := workflowMocks.NewRepository(t)
	mockEventBus := eventMocks.NewDispatcher(t)

	wf := newTestWorkflow(t)
	parent := newTestInstance(t, wf)

	// Complete the parent to make it non-running
	err := parent.Complete(shared.NewID())
	require.NoError(t, err)

	mockInstanceRepo.On("FindByID", mock.Anything, mock.AnythingOfType("shared.ID")).Return(parent, nil)

	uc := NewCloneInstanceUseCase(mockInstanceRepo, mockWorkflowRepo, mockEventBus)

	_, err = uc.Execute(context.Background(), CloneInstanceCommand{
		ParentInstanceID: parent.ID().String(),
		Assignees:        []CloneAssignee{{UserID: shared.NewID().String()}},
		TimeoutDuration:  "1h",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not active")
}

func TestCloneInstanceUseCase_InvalidParentID(t *testing.T) {
	mockInstanceRepo := instanceMocks.NewRepository(t)
	mockWorkflowRepo := workflowMocks.NewRepository(t)
	mockEventBus := eventMocks.NewDispatcher(t)

	uc := NewCloneInstanceUseCase(mockInstanceRepo, mockWorkflowRepo, mockEventBus)

	_, err := uc.Execute(context.Background(), CloneInstanceCommand{
		ParentInstanceID: "not-a-uuid",
		Assignees:        []CloneAssignee{{UserID: shared.NewID().String()}},
		TimeoutDuration:  "1h",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid parent ID")
}

func TestCloneInstanceUseCase_InvalidTimeoutDuration(t *testing.T) {
	mockInstanceRepo := instanceMocks.NewRepository(t)
	mockWorkflowRepo := workflowMocks.NewRepository(t)
	mockEventBus := eventMocks.NewDispatcher(t)

	wf := newTestWorkflow(t)
	parent := newTestInstance(t, wf)

	mockInstanceRepo.On("FindByID", mock.Anything, mock.AnythingOfType("shared.ID")).Return(parent, nil)

	uc := NewCloneInstanceUseCase(mockInstanceRepo, mockWorkflowRepo, mockEventBus)

	_, err := uc.Execute(context.Background(), CloneInstanceCommand{
		ParentInstanceID: parent.ID().String(),
		Assignees:        []CloneAssignee{{UserID: shared.NewID().String()}},
		TimeoutDuration:  "invalid",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid timeout duration")
}
