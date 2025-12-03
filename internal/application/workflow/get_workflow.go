package workflow

import (
	"context"

	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/shared"
	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/workflow"
)

// WorkflowDTO represents a workflow for API responses.
type WorkflowDTO struct {
	ID           string
	Name         string
	Description  string
	Version      string
	InitialState string
	States       []StateDTO
	Events       []EventDTO
	CreatedAt    string
	UpdatedAt    string
}

// GetWorkflowUseCase retrieves a workflow by ID.
type GetWorkflowUseCase struct {
	workflowRepo workflow.Repository
}

// NewGetWorkflowUseCase creates a new use case instance.
func NewGetWorkflowUseCase(workflowRepo workflow.Repository) *GetWorkflowUseCase {
	return &GetWorkflowUseCase{
		workflowRepo: workflowRepo,
	}
}

// Execute retrieves a workflow by ID.
func (uc *GetWorkflowUseCase) Execute(ctx context.Context, workflowID string) (*WorkflowDTO, error) {
	id, err := shared.ParseID(workflowID)
	if err != nil {
		return nil, workflow.InvalidWorkflowError("invalid workflow ID format")
	}

	wf, err := uc.workflowRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return uc.toDTO(wf), nil
}

// ExecuteAll retrieves all workflows.
func (uc *GetWorkflowUseCase) ExecuteAll(ctx context.Context) ([]*WorkflowDTO, error) {
	workflows, err := uc.workflowRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	dtos := make([]*WorkflowDTO, len(workflows))
	for i, wf := range workflows {
		dtos[i] = uc.toDTO(wf)
	}

	return dtos, nil
}

func (uc *GetWorkflowUseCase) toDTO(wf *workflow.Workflow) *WorkflowDTO {
	states := wf.States()
	stateDTOs := make([]StateDTO, len(states))
	for i, state := range states {
		stateDTOs[i] = StateDTO{
			ID:          state.ID,
			Name:        state.Name,
			Description: state.Description,
			IsFinal:     state.IsFinal,
		}
	}

	events := wf.Events()
	eventDTOs := make([]EventDTO, len(events))
	for i, evt := range events {
		sources := evt.Sources
		sourceIDs := make([]string, len(sources))
		for j, src := range sources {
			sourceIDs[j] = src.ID
		}

		eventDTOs[i] = EventDTO{
			Name:        evt.Name,
			Sources:     sourceIDs,
			Destination: evt.Destination.ID,
		}
	}

	return &WorkflowDTO{
		ID:           wf.ID().String(),
		Name:         wf.Name(),
		Description:  wf.Description(),
		Version:      wf.Version().String(),
		InitialState: wf.InitialState().ID,
		States:       stateDTOs,
		Events:       eventDTOs,
		CreatedAt:    wf.CreatedAt().Time().Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:    wf.UpdatedAt().Time().Format("2006-01-02T15:04:05Z07:00"),
	}
}
