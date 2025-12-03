package instance

import (
	"context"

	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/instance"
	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/shared"
)

// InstanceDTO represents an instance for API responses.
type InstanceDTO struct {
	ID             string
	WorkflowID     string
	WorkflowName   string
	CurrentState   string
	Status         string
	Version        string
	Data           map[string]interface{}
	Variables      map[string]interface{}
	Transitions    []TransitionDTO
	CreatedAt      string
	UpdatedAt      string
	CompletedAt    string
	TransitionCount int
}

// TransitionDTO represents a transition in the history.
type TransitionDTO struct {
	ID        string
	From      string
	To        string
	Event     string
	Actor     string
	Timestamp string
	Reason    string
	Feedback  string
	Metadata  map[string]interface{}
}

// GetInstanceUseCase retrieves an instance by ID.
type GetInstanceUseCase struct {
	instanceRepo instance.Repository
}

// NewGetInstanceUseCase creates a new use case instance.
func NewGetInstanceUseCase(instanceRepo instance.Repository) *GetInstanceUseCase {
	return &GetInstanceUseCase{
		instanceRepo: instanceRepo,
	}
}

// Execute retrieves an instance by ID.
func (uc *GetInstanceUseCase) Execute(ctx context.Context, instanceID string) (*InstanceDTO, error) {
	id, err := shared.ParseID(instanceID)
	if err != nil {
		return nil, instance.InvalidInstanceError("invalid instance ID format")
	}

	inst, err := uc.instanceRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return uc.toDTO(inst), nil
}

// ExecuteByWorkflow retrieves all instances for a workflow.
func (uc *GetInstanceUseCase) ExecuteByWorkflow(ctx context.Context, workflowID string) ([]*InstanceDTO, error) {
	id, err := shared.ParseID(workflowID)
	if err != nil {
		return nil, instance.InvalidInstanceError("invalid workflow ID format")
	}

	instances, err := uc.instanceRepo.FindByWorkflowID(ctx, id)
	if err != nil {
		return nil, err
	}

	dtos := make([]*InstanceDTO, len(instances))
	for i, inst := range instances {
		dtos[i] = uc.toDTO(inst)
	}

	return dtos, nil
}

// ExecuteAll retrieves all instances.
func (uc *GetInstanceUseCase) ExecuteAll(ctx context.Context) ([]*InstanceDTO, error) {
	instances, err := uc.instanceRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	dtos := make([]*InstanceDTO, len(instances))
	for i, inst := range instances {
		dtos[i] = uc.toDTO(inst)
	}

	return dtos, nil
}

func (uc *GetInstanceUseCase) toDTO(inst *instance.Instance) *InstanceDTO {
	transitions := inst.GetTransitionHistory()
	transitionDTOs := make([]TransitionDTO, len(transitions))
	for i, trans := range transitions {
		metadata := trans.Metadata()
		transitionDTOs[i] = TransitionDTO{
			ID:        trans.ID().String(),
			From:      trans.From(),
			To:        trans.To(),
			Event:     trans.Event(),
			Actor:     trans.Actor().String(),
			Timestamp: trans.Timestamp().Time().Format("2006-01-02T15:04:05Z07:00"),
			Reason:    metadata.Reason(),
			Feedback:  metadata.Feedback(),
			Metadata:  metadata.Metadata(),
		}
	}

	completedAt := ""
	if !inst.CompletedAt().IsZero() {
		completedAt = inst.CompletedAt().Time().Format("2006-01-02T15:04:05Z07:00")
	}

	return &InstanceDTO{
		ID:              inst.ID().String(),
		WorkflowID:      inst.WorkflowID().String(),
		WorkflowName:    inst.WorkflowName(),
		CurrentState:    inst.CurrentState(),
		Status:          inst.Status().String(),
		Version:         inst.Version().String(),
		Data:            inst.Data().ToMap(),
		Variables:       inst.Variables().ToMap(),
		Transitions:     transitionDTOs,
		CreatedAt:       inst.CreatedAt().Time().Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:       inst.UpdatedAt().Time().Format("2006-01-02T15:04:05Z07:00"),
		CompletedAt:     completedAt,
		TransitionCount: inst.TransitionCount(),
	}
}
