package instance

import (
	"context"

	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/event"
	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/instance"
	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/shared"
	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/workflow"
)

// TransitionInstanceCommand contains the data needed to perform a state transition.
type TransitionInstanceCommand struct {
	InstanceID string
	Event      string
	ActorID    string
	Roles      []string
	Reason     string
	Feedback   string
	Metadata   map[string]interface{}
	Data       map[string]interface{} // Optional data updates
}

// TransitionInstanceResult contains the transition result.
type TransitionInstanceResult struct {
	ID            string // Instance ID
	WorkflowID    string
	PreviousState string
	CurrentState  string
	Status        string
	Version       string
	TransitionID  string
	UpdatedAt     string
}

// TransitionInstanceUseCase handles state transitions for workflow instances.
type TransitionInstanceUseCase struct {
	workflowRepo workflow.Repository
	instanceRepo instance.Repository
	eventBus     event.Dispatcher
	engine       *instance.Engine
}

// NewTransitionInstanceUseCase creates a new use case instance.
func NewTransitionInstanceUseCase(
	workflowRepo workflow.Repository,
	instanceRepo instance.Repository,
	eventBus event.Dispatcher,
	engine *instance.Engine,
) *TransitionInstanceUseCase {
	return &TransitionInstanceUseCase{
		workflowRepo: workflowRepo,
		instanceRepo: instanceRepo,
		eventBus:     eventBus,
		engine:       engine,
	}
}

// Execute performs a state transition on an instance.
func (uc *TransitionInstanceUseCase) Execute(ctx context.Context, cmd TransitionInstanceCommand) (*TransitionInstanceResult, error) {
	// Validate command
	if err := uc.validateCommand(cmd); err != nil {
		return nil, err
	}

	// Parse instance ID
	instanceID, err := shared.ParseID(cmd.InstanceID)
	if err != nil {
		return nil, instance.InvalidInstanceError("invalid instance ID format")
	}

	// Parse actor ID
	actorID, err := shared.ParseID(cmd.ActorID)
	if err != nil {
		return nil, instance.InvalidInstanceError("invalid actor ID format")
	}

	// Retrieve instance
	inst, err := uc.instanceRepo.FindByID(ctx, instanceID)
	if err != nil {
		return nil, err
	}

	// Retrieve workflow to validate transition
	wf, err := uc.workflowRepo.FindByID(ctx, inst.WorkflowID())
	if err != nil {
		return nil, err
	}

	// Get current state
	currentState, err := wf.GetState(inst.CurrentState())
	if err != nil {
		return nil, err
	}

	// Get event and validate transition
	evt, err := wf.GetEvent(cmd.Event)
	if err != nil {
		return nil, err
	}

	if !wf.CanTransition(currentState, evt) {
		return nil, instance.InvalidTransitionError(
			currentState.ID,
			evt.Destination.ID,
			cmd.Event,
		)
	}

	// Update data if provided (before guards, so transition data can satisfy guard conditions)
	if len(cmd.Data) > 0 {
		for key, value := range cmd.Data {
			inst.UpdateData(key, value)
		}
	}

	// Evaluate guards before transition
	if uc.engine != nil && len(evt.Guards) > 0 {
		guardCtx := instance.GuardContext{
			Instance: inst,
			Event:    cmd.Event,
			ActorID:  actorID,
			Roles:    cmd.Roles,
		}
		if err := uc.engine.EvaluateGuards(guardCtx, evt.Guards); err != nil {
			return nil, err
		}
	}

	// Store previous state for result
	previousState := inst.CurrentState()

	// Create transition metadata
	metadata := instance.NewTransitionMetadata(cmd.Reason, cmd.Feedback, cmd.Metadata)

	// Perform transition
	if err := inst.Transition(evt.Destination.ID, cmd.Event, actorID, metadata, evt.RequiredData); err != nil {
		return nil, err
	}

	// Execute actions after transition
	if uc.engine != nil && len(evt.Actions) > 0 {
		actionCtx := instance.ActionContext{
			Instance: inst,
			Event:    cmd.Event,
			ActorID:  actorID,
		}
		if err := uc.engine.ExecuteActions(actionCtx, evt.Actions); err != nil {
			return nil, err
		}
	}

	// Persist instance
	if err := uc.instanceRepo.Save(ctx, inst); err != nil {
		return nil, err
	}

	// Dispatch domain events
	events := inst.DomainEvents()
	if len(events) > 0 {
		_ = uc.eventBus.DispatchBatch(ctx, events)
	}

	// Get last transition for ID
	lastTransition := inst.GetLastTransition()

	return &TransitionInstanceResult{
		ID:            inst.ID().String(),
		WorkflowID:    inst.WorkflowID().String(),
		PreviousState: previousState,
		CurrentState:  inst.CurrentState(),
		Status:        inst.Status().String(),
		Version:       inst.Version().String(),
		TransitionID:  lastTransition.ID().String(),
		UpdatedAt:     inst.UpdatedAt().Time().Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

func (uc *TransitionInstanceUseCase) validateCommand(cmd TransitionInstanceCommand) error {
	if cmd.InstanceID == "" {
		return instance.InvalidInstanceError("instance ID is required")
	}

	if cmd.Event == "" {
		return instance.InvalidInstanceError("event name is required")
	}

	if cmd.ActorID == "" {
		return instance.InvalidInstanceError("actor ID is required")
	}

	return nil
}
