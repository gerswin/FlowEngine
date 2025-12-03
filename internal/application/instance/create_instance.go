package instance

import (
	"context"

	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/event"
	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/instance"
	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/shared"
	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/workflow"
)

// CreateInstanceCommand contains the data needed to create a new instance.
type CreateInstanceCommand struct {
	WorkflowID string
	ParentID   string // Optional: ID of the parent instance
	StartedBy  string // Actor ID
	Data       map[string]interface{}
	Variables  map[string]interface{}
}

// CreateInstanceResult contains the created instance data.
type CreateInstanceResult struct {
	ID           string
	ParentID     string // Optional
	WorkflowID   string
	WorkflowName string
	CurrentState string
	Status       string
	Version      string
	CreatedAt    string
	UpdatedAt    string
}

// CreateInstanceUseCase handles the creation of new workflow instances.
type CreateInstanceUseCase struct {
	workflowRepo workflow.Repository
	instanceRepo instance.Repository
	eventBus     event.Dispatcher
}

// NewCreateInstanceUseCase creates a new use case instance.
func NewCreateInstanceUseCase(
	workflowRepo workflow.Repository,
	instanceRepo instance.Repository,
	eventBus event.Dispatcher,
) *CreateInstanceUseCase {
	return &CreateInstanceUseCase{
		workflowRepo: workflowRepo,
		instanceRepo: instanceRepo,
		eventBus:     eventBus,
	}
}

// Execute creates a new workflow instance.
func (uc *CreateInstanceUseCase) Execute(ctx context.Context, cmd CreateInstanceCommand) (*CreateInstanceResult, error) {
	// Validate command
	if err := uc.validateCommand(cmd); err != nil {
		return nil, err
	}

	// Parse IDs
	workflowID, err := shared.ParseID(cmd.WorkflowID)
	if err != nil {
		return nil, instance.InvalidInstanceError("invalid workflow ID format")
	}

	actorID, err := shared.ParseID(cmd.StartedBy)
	if err != nil {
		return nil, instance.InvalidInstanceError("invalid actor ID format")
	}

	// Retrieve workflow to get initial state
	wf, err := uc.workflowRepo.FindByID(ctx, workflowID)
	if err != nil {
		return nil, err
	}

	var inst *instance.Instance

	// Check if it's a subprocess (R6.1)
	if cmd.ParentID != "" {
		parentID, err := shared.ParseID(cmd.ParentID)
		if err != nil {
			return nil, instance.InvalidInstanceError("invalid parent instance ID format")
		}

		// Validate parent exists
		exists, err := uc.instanceRepo.Exists(ctx, parentID)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, instance.InstanceNotFoundError(cmd.ParentID)
		}

		inst, err = instance.NewSubprocess(
			parentID,
			workflowID,
			wf.Name(),
			wf.InitialState().ID,
			actorID,
		)
		if err != nil {
			return nil, err
		}
	} else {
		// Create root instance
		inst, err = instance.NewInstance(
			workflowID,
			wf.Name(),
			wf.InitialState().ID,
			actorID,
		)
		if err != nil {
			return nil, err
		}
	}

	// Set data if provided
	if len(cmd.Data) > 0 {
		data := instance.NewData()
		for key, value := range cmd.Data {
			data = data.Set(key, value)
		}
		inst.SetData(data)
	}

	// Set variables if provided
	if len(cmd.Variables) > 0 {
		variables := instance.NewVariables()
		for key, value := range cmd.Variables {
			variables = variables.Set(key, value)
		}
		inst.SetVariables(variables)
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

	result := &CreateInstanceResult{
		ID:           inst.ID().String(),
		WorkflowID:   inst.WorkflowID().String(),
		WorkflowName: inst.WorkflowName(),
		CurrentState: inst.CurrentState(),
		Status:       inst.Status().String(),
		Version:      inst.Version().String(),
		CreatedAt:    inst.CreatedAt().Time().Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:    inst.UpdatedAt().Time().Format("2006-01-02T15:04:05Z07:00"),
	}

	if inst.ParentID().IsValid() {
		result.ParentID = inst.ParentID().String()
	}

	return result, nil
}

func (uc *CreateInstanceUseCase) validateCommand(cmd CreateInstanceCommand) error {
	if cmd.WorkflowID == "" {
		return instance.InvalidInstanceError("workflow ID is required")
	}

	if cmd.StartedBy == "" {
		return instance.InvalidInstanceError("actor ID is required")
	}

	return nil
}
