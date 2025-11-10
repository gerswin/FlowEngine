package workflow

import (
	"context"

	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/shared"
)

// Repository defines the interface for persisting workflows.
// This is a port in the hexagonal architecture.
type Repository interface {
	// Save persists a workflow.
	// If the workflow already exists (by ID), it will be updated.
	// Returns an error if the operation fails.
	Save(ctx context.Context, workflow *Workflow) error

	// FindByID retrieves a workflow by its ID.
	// Returns ErrNotFound if the workflow doesn't exist.
	FindByID(ctx context.Context, id shared.ID) (*Workflow, error)

	// FindByName retrieves a workflow by its name and version.
	// Returns ErrNotFound if the workflow doesn't exist.
	FindByName(ctx context.Context, name string, version Version) (*Workflow, error)

	// FindLatestByName retrieves the latest version of a workflow by name.
	// Returns ErrNotFound if no workflow with the given name exists.
	FindLatestByName(ctx context.Context, name string) (*Workflow, error)

	// FindAll retrieves all workflows.
	// Returns an empty slice if no workflows exist.
	FindAll(ctx context.Context) ([]*Workflow, error)

	// FindAllByName retrieves all versions of a workflow by name.
	// Returns an empty slice if no workflows with the given name exist.
	FindAllByName(ctx context.Context, name string) ([]*Workflow, error)

	// Delete deletes a workflow by its ID.
	// Returns ErrNotFound if the workflow doesn't exist.
	Delete(ctx context.Context, id shared.ID) error

	// Exists checks if a workflow exists by its ID.
	Exists(ctx context.Context, id shared.ID) (bool, error)

	// ExistsByName checks if a workflow exists by name and version.
	ExistsByName(ctx context.Context, name string, version Version) (bool, error)
}
