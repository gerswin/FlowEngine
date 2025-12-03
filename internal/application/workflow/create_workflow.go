package workflow

import (
	"context"

	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/event"
	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/shared"
	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/workflow"
)

// CreateWorkflowCommand contains the data needed to create a new workflow.
type CreateWorkflowCommand struct {
	Name         string
	Description  string
	InitialState StateDTO
	States       []StateDTO
	Events       []EventDTO
	CreatedBy    string // Actor ID as string
}

// StateDTO represents a state in the workflow definition.
type StateDTO struct {
	ID          string
	Name        string
	Description string
	IsFinal     bool
}

// EventDTO represents an event (transition) in the workflow definition.
type EventDTO struct {
	Name        string
	Sources     []string // State IDs
	Destination string   // State ID
}

// CreateWorkflowResult contains the created workflow data.
type CreateWorkflowResult struct {
	ID          string
	Name        string
	Description string
	Version     string
	CreatedAt   string
	UpdatedAt   string
	States      []StateDTO
	Events      []EventDTO
}

// CreateWorkflowUseCase handles the creation of new workflows.
type CreateWorkflowUseCase struct {
	workflowRepo workflow.Repository
	eventBus     event.Dispatcher
}

// NewCreateWorkflowUseCase creates a new use case instance.
func NewCreateWorkflowUseCase(
	workflowRepo workflow.Repository,
	eventBus event.Dispatcher,
) *CreateWorkflowUseCase {
	return &CreateWorkflowUseCase{
		workflowRepo: workflowRepo,
		eventBus:     eventBus,
	}
}

// Execute creates a new workflow based on the command.
func (uc *CreateWorkflowUseCase) Execute(ctx context.Context, cmd CreateWorkflowCommand) (*CreateWorkflowResult, error) {
	// Validate command
	if err := uc.validateCommand(cmd); err != nil {
		return nil, err
	}

	// Parse creator actor ID
	creatorID, err := shared.ParseID(cmd.CreatedBy)
	if err != nil {
		return nil, workflow.InvalidWorkflowError("invalid creator ID")
	}

	// Create initial state
	initialState, err := uc.createState(cmd.InitialState)
	if err != nil {
		return nil, err
	}

	// Create workflow aggregate
	wf, err := workflow.NewWorkflow(cmd.Name, initialState, creatorID)
	if err != nil {
		return nil, err
	}

	// Set description if provided
	if cmd.Description != "" {
		wf.SetDescription(cmd.Description)
	}

	// Add all states (excluding initial state as it's already added)
	for _, stateDTO := range cmd.States {
		if stateDTO.ID == cmd.InitialState.ID {
			continue // Skip initial state
		}

		state, err := uc.createState(stateDTO)
		if err != nil {
			return nil, err
		}

		if err := wf.AddState(state); err != nil {
			return nil, err
		}
	}

	// Add all events
	for _, eventDTO := range cmd.Events {
		evt, err := uc.createEvent(wf, eventDTO)
		if err != nil {
			return nil, err
		}

		if err := wf.AddEvent(evt); err != nil {
			return nil, err
		}
	}

	// Validate the complete workflow
	if err := wf.Validate(); err != nil {
		return nil, err
	}

	// Persist workflow
	if err := uc.workflowRepo.Save(ctx, wf); err != nil {
		return nil, err
	}

	// Dispatch domain events
	events := wf.DomainEvents()
	if len(events) > 0 {
		_ = uc.eventBus.DispatchBatch(ctx, events)
	}

	return &CreateWorkflowResult{
		ID:          wf.ID().String(),
		Name:        wf.Name(),
		Description: wf.Description(),
		Version:     wf.Version().String(),
		CreatedAt:   wf.CreatedAt().Time().Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   wf.UpdatedAt().Time().Format("2006-01-02T15:04:05Z07:00"),
		States:      cmd.States, // Returning the input states for now (plus initial?)
		Events:      cmd.Events,
	}, nil
}

func (uc *CreateWorkflowUseCase) validateCommand(cmd CreateWorkflowCommand) error {
	if cmd.Name == "" {
		return workflow.InvalidWorkflowError("workflow name is required")
	}

	if cmd.InitialState.ID == "" {
		return workflow.InvalidWorkflowError("initial state is required")
	}

	if cmd.CreatedBy == "" {
		return workflow.InvalidWorkflowError("creator ID is required")
	}

	return nil
}

func (uc *CreateWorkflowUseCase) createState(dto StateDTO) (workflow.State, error) {
	state, err := workflow.NewState(dto.ID, dto.Name)
	if err != nil {
		return workflow.State{}, err
	}

	if dto.Description != "" {
		state = state.WithDescription(dto.Description)
	}

	if dto.IsFinal {
		state = state.AsFinal()
	}

	return state, nil
}

func (uc *CreateWorkflowUseCase) createEvent(wf *workflow.Workflow, dto EventDTO) (workflow.Event, error) {
	// Get source states
	sources := make([]workflow.State, 0, len(dto.Sources))
	for _, sourceID := range dto.Sources {
		state, err := wf.GetState(sourceID)
		if err != nil {
			return workflow.Event{}, err
		}
		sources = append(sources, state)
	}

	// Get destination state
	destination, err := wf.GetState(dto.Destination)
	if err != nil {
		return workflow.Event{}, err
	}

	return workflow.NewEvent(dto.Name, sources, destination)
}
