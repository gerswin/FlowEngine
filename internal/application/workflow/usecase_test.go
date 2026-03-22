package workflow

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	eventMocks "github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/event/mocks"
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

// ==================== CreateWorkflowUseCase ====================

func TestCreateWorkflowUseCase_Success(t *testing.T) {
	mockWorkflowRepo := workflowMocks.NewRepository(t)
	mockEventBus := eventMocks.NewDispatcher(t)

	mockWorkflowRepo.On("Save", mock.Anything, mock.AnythingOfType("*workflow.Workflow")).Return(nil)
	mockEventBus.On("DispatchBatch", mock.Anything, mock.Anything).Return(nil)

	uc := NewCreateWorkflowUseCase(mockWorkflowRepo, mockEventBus)

	result, err := uc.Execute(context.Background(), CreateWorkflowCommand{
		Name: "MyWorkflow",
		InitialState: StateDTO{
			ID:   "draft",
			Name: "Draft",
		},
		States: []StateDTO{
			{ID: "draft", Name: "Draft"},
			{ID: "review", Name: "Review"},
			{ID: "approved", Name: "Approved", IsFinal: true},
		},
		Events: []EventDTO{
			{
				Name:        "submit",
				Sources:     []string{"draft"},
				Destination: "review",
			},
			{
				Name:        "approve",
				Sources:     []string{"review"},
				Destination: "approved",
			},
		},
		CreatedBy: shared.NewID().String(),
	})

	require.NoError(t, err)
	assert.NotEmpty(t, result.ID)
	assert.Equal(t, "MyWorkflow", result.Name)
	assert.NotEmpty(t, result.Version)
	assert.NotEmpty(t, result.CreatedAt)
}

func TestCreateWorkflowUseCase_MissingName(t *testing.T) {
	mockWorkflowRepo := workflowMocks.NewRepository(t)
	mockEventBus := eventMocks.NewDispatcher(t)

	uc := NewCreateWorkflowUseCase(mockWorkflowRepo, mockEventBus)

	_, err := uc.Execute(context.Background(), CreateWorkflowCommand{
		Name: "",
		InitialState: StateDTO{
			ID:   "draft",
			Name: "Draft",
		},
		CreatedBy: shared.NewID().String(),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "workflow name is required")
}

func TestCreateWorkflowUseCase_MissingCreatorID(t *testing.T) {
	mockWorkflowRepo := workflowMocks.NewRepository(t)
	mockEventBus := eventMocks.NewDispatcher(t)

	uc := NewCreateWorkflowUseCase(mockWorkflowRepo, mockEventBus)

	_, err := uc.Execute(context.Background(), CreateWorkflowCommand{
		Name: "MyWorkflow",
		InitialState: StateDTO{
			ID:   "draft",
			Name: "Draft",
		},
		CreatedBy: "",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "creator ID is required")
}

func TestCreateWorkflowUseCase_MissingInitialState(t *testing.T) {
	mockWorkflowRepo := workflowMocks.NewRepository(t)
	mockEventBus := eventMocks.NewDispatcher(t)

	uc := NewCreateWorkflowUseCase(mockWorkflowRepo, mockEventBus)

	_, err := uc.Execute(context.Background(), CreateWorkflowCommand{
		Name:         "MyWorkflow",
		InitialState: StateDTO{ID: "", Name: "Draft"},
		CreatedBy:    shared.NewID().String(),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "initial state is required")
}

func TestCreateWorkflowUseCase_InvalidStateRefInEvent(t *testing.T) {
	mockWorkflowRepo := workflowMocks.NewRepository(t)
	mockEventBus := eventMocks.NewDispatcher(t)

	uc := NewCreateWorkflowUseCase(mockWorkflowRepo, mockEventBus)

	_, err := uc.Execute(context.Background(), CreateWorkflowCommand{
		Name: "MyWorkflow",
		InitialState: StateDTO{
			ID:   "draft",
			Name: "Draft",
		},
		States: []StateDTO{
			{ID: "draft", Name: "Draft"},
		},
		Events: []EventDTO{
			{
				Name:        "submit",
				Sources:     []string{"draft"},
				Destination: "nonexistent_state",
			},
		},
		CreatedBy: shared.NewID().String(),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestCreateWorkflowUseCase_WithDescription(t *testing.T) {
	mockWorkflowRepo := workflowMocks.NewRepository(t)
	mockEventBus := eventMocks.NewDispatcher(t)

	mockWorkflowRepo.On("Save", mock.Anything, mock.AnythingOfType("*workflow.Workflow")).Return(nil)
	mockEventBus.On("DispatchBatch", mock.Anything, mock.Anything).Return(nil)

	uc := NewCreateWorkflowUseCase(mockWorkflowRepo, mockEventBus)

	result, err := uc.Execute(context.Background(), CreateWorkflowCommand{
		Name:        "MyWorkflow",
		Description: "A test workflow",
		InitialState: StateDTO{
			ID:   "draft",
			Name: "Draft",
		},
		States: []StateDTO{
			{ID: "draft", Name: "Draft"},
		},
		Events:    []EventDTO{},
		CreatedBy: shared.NewID().String(),
	})

	require.NoError(t, err)
	assert.Equal(t, "A test workflow", result.Description)
}

func TestCreateWorkflowUseCase_SaveError(t *testing.T) {
	mockWorkflowRepo := workflowMocks.NewRepository(t)
	mockEventBus := eventMocks.NewDispatcher(t)

	mockWorkflowRepo.On("Save", mock.Anything, mock.AnythingOfType("*workflow.Workflow")).
		Return(errors.New("database error"))

	uc := NewCreateWorkflowUseCase(mockWorkflowRepo, mockEventBus)

	_, err := uc.Execute(context.Background(), CreateWorkflowCommand{
		Name: "MyWorkflow",
		InitialState: StateDTO{
			ID:   "draft",
			Name: "Draft",
		},
		States:    []StateDTO{{ID: "draft", Name: "Draft"}},
		Events:    []EventDTO{},
		CreatedBy: shared.NewID().String(),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "database error")
}

// ==================== GetWorkflowUseCase ====================

func TestGetWorkflowUseCase_Execute_Success(t *testing.T) {
	mockWorkflowRepo := workflowMocks.NewRepository(t)

	wf := newTestWorkflow(t)
	mockWorkflowRepo.On("FindByID", mock.Anything, mock.AnythingOfType("shared.ID")).Return(wf, nil)

	uc := NewGetWorkflowUseCase(mockWorkflowRepo)

	dto, err := uc.Execute(context.Background(), wf.ID().String())

	require.NoError(t, err)
	assert.Equal(t, wf.ID().String(), dto.ID)
	assert.Equal(t, wf.Name(), dto.Name)
	assert.Equal(t, "draft", dto.InitialState)
	assert.NotEmpty(t, dto.States)
	assert.NotEmpty(t, dto.Events)
}

func TestGetWorkflowUseCase_Execute_InvalidID(t *testing.T) {
	mockWorkflowRepo := workflowMocks.NewRepository(t)

	uc := NewGetWorkflowUseCase(mockWorkflowRepo)

	_, err := uc.Execute(context.Background(), "bad-id")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid workflow ID format")
}

func TestGetWorkflowUseCase_Execute_NotFound(t *testing.T) {
	mockWorkflowRepo := workflowMocks.NewRepository(t)

	wfID := shared.NewID()
	mockWorkflowRepo.On("FindByID", mock.Anything, mock.AnythingOfType("shared.ID")).
		Return(nil, errors.New("workflow not found"))

	uc := NewGetWorkflowUseCase(mockWorkflowRepo)

	_, err := uc.Execute(context.Background(), wfID.String())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetWorkflowUseCase_ExecuteAll_Success(t *testing.T) {
	mockWorkflowRepo := workflowMocks.NewRepository(t)

	wf1 := newTestWorkflow(t)
	wf2 := newTestWorkflow(t)

	mockWorkflowRepo.On("FindAll", mock.Anything).
		Return([]*workflow.Workflow{wf1, wf2}, nil)

	uc := NewGetWorkflowUseCase(mockWorkflowRepo)

	dtos, err := uc.ExecuteAll(context.Background())

	require.NoError(t, err)
	assert.Len(t, dtos, 2)
}

func TestGetWorkflowUseCase_ExecuteAll_Error(t *testing.T) {
	mockWorkflowRepo := workflowMocks.NewRepository(t)

	mockWorkflowRepo.On("FindAll", mock.Anything).
		Return(nil, errors.New("database error"))

	uc := NewGetWorkflowUseCase(mockWorkflowRepo)

	_, err := uc.ExecuteAll(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "database error")
}

func TestGetWorkflowUseCase_ExecuteList_Success(t *testing.T) {
	mockWorkflowRepo := workflowMocks.NewRepository(t)

	wf := newTestWorkflow(t)

	mockWorkflowRepo.On("List", mock.Anything, mock.AnythingOfType("shared.ListQuery")).
		Return([]*workflow.Workflow{wf}, int64(1), nil)

	uc := NewGetWorkflowUseCase(mockWorkflowRepo)

	result, err := uc.ExecuteList(context.Background(), 1, 10)

	require.NoError(t, err)
	assert.Len(t, result.Items, 1)
	assert.Equal(t, int64(1), result.Total)
	assert.Equal(t, wf.ID().String(), result.Items[0].ID)
}

func TestGetWorkflowUseCase_ExecuteList_Error(t *testing.T) {
	mockWorkflowRepo := workflowMocks.NewRepository(t)

	mockWorkflowRepo.On("List", mock.Anything, mock.AnythingOfType("shared.ListQuery")).
		Return(nil, int64(0), errors.New("database error"))

	uc := NewGetWorkflowUseCase(mockWorkflowRepo)

	_, err := uc.ExecuteList(context.Background(), 1, 10)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "database error")
}

func TestGetWorkflowUseCase_ExecuteAll_Empty(t *testing.T) {
	mockWorkflowRepo := workflowMocks.NewRepository(t)

	mockWorkflowRepo.On("FindAll", mock.Anything).
		Return([]*workflow.Workflow{}, nil)

	uc := NewGetWorkflowUseCase(mockWorkflowRepo)

	dtos, err := uc.ExecuteAll(context.Background())

	require.NoError(t, err)
	assert.Len(t, dtos, 0)
}

// ==================== CreateWorkflowFromYAMLUseCase ====================

func TestCreateWorkflowFromYAMLUseCase_Execute_Success(t *testing.T) {
	mockWorkflowRepo := workflowMocks.NewRepository(t)
	mockEventBus := eventMocks.NewDispatcher(t)

	mockWorkflowRepo.On("Save", mock.Anything, mock.AnythingOfType("*workflow.Workflow")).Return(nil)
	mockEventBus.On("DispatchBatch", mock.Anything, mock.Anything).Return(nil)

	uc := NewCreateWorkflowFromYAMLUseCase(mockWorkflowRepo, mockEventBus)

	yamlContent := []byte(`
version: "1.0"
workflow:
  name: Approval
  description: A simple approval flow
  initial_state: draft
  states:
    - id: draft
      name: Draft
    - id: approved
      name: Approved
      is_final: true
  events:
    - name: approve
      from: [draft]
      to: approved
`)

	result, err := uc.Execute(context.Background(), CreateWorkflowFromYAMLCommand{
		YAMLContent: yamlContent,
		CreatedBy:   shared.NewID().String(),
	})

	require.NoError(t, err)
	assert.NotEmpty(t, result.ID)
	assert.Equal(t, "Approval", result.Name)
	assert.Equal(t, "A simple approval flow", result.Description)
	assert.Equal(t, "draft", result.InitialState)
	assert.Equal(t, 2, result.StatesCount)
	assert.Equal(t, 1, result.EventsCount)
}

func TestCreateWorkflowFromYAMLUseCase_Execute_EmptyYAML(t *testing.T) {
	mockWorkflowRepo := workflowMocks.NewRepository(t)
	mockEventBus := eventMocks.NewDispatcher(t)

	uc := NewCreateWorkflowFromYAMLUseCase(mockWorkflowRepo, mockEventBus)

	_, err := uc.Execute(context.Background(), CreateWorkflowFromYAMLCommand{
		YAMLContent: []byte{},
		CreatedBy:   shared.NewID().String(),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "YAML content is required")
}

func TestCreateWorkflowFromYAMLUseCase_Execute_EmptyCreator(t *testing.T) {
	mockWorkflowRepo := workflowMocks.NewRepository(t)
	mockEventBus := eventMocks.NewDispatcher(t)

	uc := NewCreateWorkflowFromYAMLUseCase(mockWorkflowRepo, mockEventBus)

	_, err := uc.Execute(context.Background(), CreateWorkflowFromYAMLCommand{
		YAMLContent: []byte("name: test"),
		CreatedBy:   "",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "creator ID is required")
}

func TestCreateWorkflowFromYAMLUseCase_Execute_InvalidYAML(t *testing.T) {
	mockWorkflowRepo := workflowMocks.NewRepository(t)
	mockEventBus := eventMocks.NewDispatcher(t)

	uc := NewCreateWorkflowFromYAMLUseCase(mockWorkflowRepo, mockEventBus)

	_, err := uc.Execute(context.Background(), CreateWorkflowFromYAMLCommand{
		YAMLContent: []byte("this is not valid yaml: [unclosed"),
		CreatedBy:   shared.NewID().String(),
	})

	require.Error(t, err)
}

func TestCreateWorkflowFromYAMLUseCase_Execute_SaveError(t *testing.T) {
	mockWorkflowRepo := workflowMocks.NewRepository(t)
	mockEventBus := eventMocks.NewDispatcher(t)

	mockWorkflowRepo.On("Save", mock.Anything, mock.AnythingOfType("*workflow.Workflow")).
		Return(errors.New("save failed"))

	uc := NewCreateWorkflowFromYAMLUseCase(mockWorkflowRepo, mockEventBus)

	yamlContent := []byte(`
version: "1.0"
workflow:
  name: Approval
  initial_state: draft
  states:
    - id: draft
      name: Draft
    - id: approved
      name: Approved
      is_final: true
  events:
    - name: approve
      from: [draft]
      to: approved
`)

	_, err := uc.Execute(context.Background(), CreateWorkflowFromYAMLCommand{
		YAMLContent: yamlContent,
		CreatedBy:   shared.NewID().String(),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "save failed")
}

func TestCreateWorkflowFromYAMLUseCase_ExecuteFromReader_Success(t *testing.T) {
	mockWorkflowRepo := workflowMocks.NewRepository(t)
	mockEventBus := eventMocks.NewDispatcher(t)

	mockWorkflowRepo.On("Save", mock.Anything, mock.AnythingOfType("*workflow.Workflow")).Return(nil)
	mockEventBus.On("DispatchBatch", mock.Anything, mock.Anything).Return(nil)

	uc := NewCreateWorkflowFromYAMLUseCase(mockWorkflowRepo, mockEventBus)

	yamlContent := `
version: "1.0"
workflow:
  name: ReaderWorkflow
  initial_state: draft
  states:
    - id: draft
      name: Draft
    - id: done
      name: Done
      is_final: true
  events:
    - name: finish
      from: [draft]
      to: done
`
	reader := strings.NewReader(yamlContent)

	result, err := uc.ExecuteFromReader(context.Background(), reader, shared.NewID().String())

	require.NoError(t, err)
	assert.Equal(t, "ReaderWorkflow", result.Name)
}

func TestCreateWorkflowFromYAMLUseCase_ExecuteWithDetails_Success(t *testing.T) {
	mockWorkflowRepo := workflowMocks.NewRepository(t)
	mockEventBus := eventMocks.NewDispatcher(t)

	mockWorkflowRepo.On("Save", mock.Anything, mock.AnythingOfType("*workflow.Workflow")).Return(nil)
	mockEventBus.On("DispatchBatch", mock.Anything, mock.Anything).Return(nil)

	uc := NewCreateWorkflowFromYAMLUseCase(mockWorkflowRepo, mockEventBus)

	yamlContent := []byte(`
version: "1.0"
workflow:
  name: DetailWorkflow
  initial_state: draft
  states:
    - id: draft
      name: Draft
    - id: approved
      name: Approved
      is_final: true
  events:
    - name: approve
      from: [draft]
      to: approved
`)

	result, err := uc.ExecuteWithDetails(context.Background(), CreateWorkflowFromYAMLCommand{
		YAMLContent: yamlContent,
		CreatedBy:   shared.NewID().String(),
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "DetailWorkflow", result.Name)
}

func TestCreateWorkflowFromYAMLUseCase_ExecuteWithDetails_EmptyYAML(t *testing.T) {
	mockWorkflowRepo := workflowMocks.NewRepository(t)
	mockEventBus := eventMocks.NewDispatcher(t)

	uc := NewCreateWorkflowFromYAMLUseCase(mockWorkflowRepo, mockEventBus)

	_, err := uc.ExecuteWithDetails(context.Background(), CreateWorkflowFromYAMLCommand{
		YAMLContent: []byte{},
		CreatedBy:   shared.NewID().String(),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "YAML content is required")
}

func TestCreateWorkflowFromYAMLUseCase_ExecuteWithDetails_EmptyCreator(t *testing.T) {
	mockWorkflowRepo := workflowMocks.NewRepository(t)
	mockEventBus := eventMocks.NewDispatcher(t)

	uc := NewCreateWorkflowFromYAMLUseCase(mockWorkflowRepo, mockEventBus)

	_, err := uc.ExecuteWithDetails(context.Background(), CreateWorkflowFromYAMLCommand{
		YAMLContent: []byte("name: test"),
		CreatedBy:   "",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "creator ID is required")
}

func TestCreateWorkflowFromYAMLUseCase_RegisterActionAndGuard(t *testing.T) {
	mockWorkflowRepo := workflowMocks.NewRepository(t)
	mockEventBus := eventMocks.NewDispatcher(t)

	uc := NewCreateWorkflowFromYAMLUseCase(mockWorkflowRepo, mockEventBus)

	// These should not panic
	assert.NotPanics(t, func() {
		uc.RegisterAction("custom_action")
		uc.RegisterGuard("custom_guard")
	})
}

func TestCreateWorkflowUseCase_InvalidCreatorID(t *testing.T) {
	mockWorkflowRepo := workflowMocks.NewRepository(t)
	mockEventBus := eventMocks.NewDispatcher(t)

	uc := NewCreateWorkflowUseCase(mockWorkflowRepo, mockEventBus)

	_, err := uc.Execute(context.Background(), CreateWorkflowCommand{
		Name: "MyWorkflow",
		InitialState: StateDTO{
			ID:   "draft",
			Name: "Draft",
		},
		States:    []StateDTO{{ID: "draft", Name: "Draft"}},
		Events:    []EventDTO{},
		CreatedBy: "invalid-not-a-uuid",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid creator ID")
}

func TestCreateWorkflowUseCase_WithGuardsAndActions(t *testing.T) {
	mockWorkflowRepo := workflowMocks.NewRepository(t)
	mockEventBus := eventMocks.NewDispatcher(t)

	mockWorkflowRepo.On("Save", mock.Anything, mock.AnythingOfType("*workflow.Workflow")).Return(nil)
	mockEventBus.On("DispatchBatch", mock.Anything, mock.Anything).Return(nil)

	uc := NewCreateWorkflowUseCase(mockWorkflowRepo, mockEventBus)

	result, err := uc.Execute(context.Background(), CreateWorkflowCommand{
		Name: "GuardedWorkflow",
		InitialState: StateDTO{
			ID:   "draft",
			Name: "Draft",
		},
		States: []StateDTO{
			{ID: "draft", Name: "Draft"},
			{ID: "approved", Name: "Approved", IsFinal: true, Description: "Final state"},
		},
		Events: []EventDTO{
			{
				Name:         "approve",
				Sources:      []string{"draft"},
				Destination:  "approved",
				RequiredData: []string{"reason"},
				Guards: []GuardDTO{
					{Type: "role_check", Params: map[string]interface{}{"role": "admin"}},
				},
				Actions: []ActionDTO{
					{Type: "send_email", Params: map[string]interface{}{"to": "admin@example.com"}},
				},
			},
		},
		CreatedBy: shared.NewID().String(),
	})

	require.NoError(t, err)
	assert.Equal(t, "GuardedWorkflow", result.Name)
}
