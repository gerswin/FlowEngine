package workflow

import (
	"context"
	"io"
	"time"

	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/event"
	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/workflow"
	yamlparser "github.com/LaFabric-LinkTIC/FlowEngine/internal/infrastructure/parser/yaml"
)

// CreateWorkflowFromYAMLCommand contains the data needed to create a workflow from YAML.
type CreateWorkflowFromYAMLCommand struct {
	YAMLContent []byte
	CreatedBy   string
}

// CreateWorkflowFromYAMLResult contains the created workflow data and parsing info.
type CreateWorkflowFromYAMLResult struct {
	ID           string
	Name         string
	Description  string
	Version      string
	InitialState string
	StatesCount  int
	EventsCount  int
	CreatedAt    string
	Warnings     []string
}

// CreateWorkflowFromYAMLUseCase handles creating workflows from YAML definitions.
type CreateWorkflowFromYAMLUseCase struct {
	workflowRepo workflow.Repository
	eventBus     event.Dispatcher
	parser       *yamlparser.Parser
}

// NewCreateWorkflowFromYAMLUseCase creates a new use case instance.
func NewCreateWorkflowFromYAMLUseCase(
	workflowRepo workflow.Repository,
	eventBus event.Dispatcher,
) *CreateWorkflowFromYAMLUseCase {
	return &CreateWorkflowFromYAMLUseCase{
		workflowRepo: workflowRepo,
		eventBus:     eventBus,
		parser:       yamlparser.NewParser(),
	}
}

// Execute creates a new workflow from YAML content.
func (uc *CreateWorkflowFromYAMLUseCase) Execute(ctx context.Context, cmd CreateWorkflowFromYAMLCommand) (*CreateWorkflowFromYAMLResult, error) {
	// Validate command
	if len(cmd.YAMLContent) == 0 {
		return nil, workflow.InvalidWorkflowError("YAML content is required")
	}

	if cmd.CreatedBy == "" {
		return nil, workflow.InvalidWorkflowError("creator ID is required")
	}

	// Parse YAML content
	wf, err := uc.parser.ParseBytes(cmd.YAMLContent)
	if err != nil {
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

	return &CreateWorkflowFromYAMLResult{
		ID:           wf.ID().String(),
		Name:         wf.Name(),
		Description:  wf.Description(),
		Version:      wf.Version().String(),
		InitialState: wf.InitialState().ID,
		StatesCount:  len(wf.States()),
		EventsCount:  len(wf.Events()),
		CreatedAt:    wf.CreatedAt().Time().Format(time.RFC3339),
		Warnings:     []string{},
	}, nil
}

// ExecuteFromReader creates a workflow from a YAML reader.
func (uc *CreateWorkflowFromYAMLUseCase) ExecuteFromReader(ctx context.Context, reader io.Reader, createdBy string) (*CreateWorkflowFromYAMLResult, error) {
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, workflow.InvalidWorkflowError("failed to read YAML content: " + err.Error())
	}

	return uc.Execute(ctx, CreateWorkflowFromYAMLCommand{
		YAMLContent: content,
		CreatedBy:   createdBy,
	})
}

// ExecuteWithDetails creates a workflow and returns detailed parsing information.
func (uc *CreateWorkflowFromYAMLUseCase) ExecuteWithDetails(ctx context.Context, cmd CreateWorkflowFromYAMLCommand) (*CreateWorkflowFromYAMLResult, error) {
	// Validate command
	if len(cmd.YAMLContent) == 0 {
		return nil, workflow.InvalidWorkflowError("YAML content is required")
	}

	if cmd.CreatedBy == "" {
		return nil, workflow.InvalidWorkflowError("creator ID is required")
	}

	// Parse YAML with details
	result, err := uc.parser.ParseWithDetails(io.NopCloser(
		&bytesReader{data: cmd.YAMLContent, pos: 0},
	))
	if err != nil {
		return nil, err
	}

	wf := result.Workflow

	// Persist workflow
	if err := uc.workflowRepo.Save(ctx, wf); err != nil {
		return nil, err
	}

	// Dispatch domain events
	events := wf.DomainEvents()
	if len(events) > 0 {
		_ = uc.eventBus.DispatchBatch(ctx, events)
	}

	return &CreateWorkflowFromYAMLResult{
		ID:           wf.ID().String(),
		Name:         wf.Name(),
		Description:  wf.Description(),
		Version:      wf.Version().String(),
		InitialState: wf.InitialState().ID,
		StatesCount:  len(wf.States()),
		EventsCount:  len(wf.Events()),
		CreatedAt:    wf.CreatedAt().Time().Format(time.RFC3339),
		Warnings:     result.Warnings,
	}, nil
}

// bytesReader is a simple io.Reader for bytes.
type bytesReader struct {
	data []byte
	pos  int
}

func (r *bytesReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

// RegisterAction registers a custom action type with the parser.
func (uc *CreateWorkflowFromYAMLUseCase) RegisterAction(actionType string) {
	uc.parser.RegisterAction(actionType)
}

// RegisterGuard registers a custom guard type with the parser.
func (uc *CreateWorkflowFromYAMLUseCase) RegisterGuard(guardType string) {
	uc.parser.RegisterGuard(guardType)
}
