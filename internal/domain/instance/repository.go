package instance

import (
	"context"

	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/shared"
)

// Repository defines the interface for persisting workflow instances.
// This is a port in the hexagonal architecture.
type Repository interface {
	// Save persists an instance.
	// If the instance already exists (by ID), it will be updated with optimistic locking.
	// Returns ErrVersionMismatch if the version doesn't match (optimistic lock failure).
	// Returns an error if the operation fails.
	Save(ctx context.Context, instance *Instance) error

	// FindByID retrieves an instance by its ID.
	// Returns ErrInstanceNotFound if the instance doesn't exist.
	FindByID(ctx context.Context, id shared.ID) (*Instance, error)

	// FindByWorkflowID retrieves all instances for a given workflow.
	// Returns an empty slice if no instances exist.
	FindByWorkflowID(ctx context.Context, workflowID shared.ID) ([]*Instance, error)

	// FindByStatus retrieves all instances with the given status.
	// Returns an empty slice if no instances exist.
	FindByStatus(ctx context.Context, status Status) ([]*Instance, error)

	// FindByWorkflowIDAndStatus retrieves instances for a workflow with specific status.
	// Returns an empty slice if no instances exist.
	FindByWorkflowIDAndStatus(ctx context.Context, workflowID shared.ID, status Status) ([]*Instance, error)

	// FindActive retrieves all active instances (Running or Paused).
	// Returns an empty slice if no instances exist.
	FindActive(ctx context.Context) ([]*Instance, error)

	// FindAll retrieves all instances.
	// This should be used with caution as it can return a large dataset.
	// Returns an empty slice if no instances exist.
	FindAll(ctx context.Context) ([]*Instance, error)

	// Delete deletes an instance by its ID.
	// Returns ErrInstanceNotFound if the instance doesn't exist.
	Delete(ctx context.Context, id shared.ID) error

	// Exists checks if an instance exists by its ID.
	Exists(ctx context.Context, id shared.ID) (bool, error)

	// Count returns the total number of instances.
	Count(ctx context.Context) (int64, error)

	// CountByWorkflowID returns the number of instances for a workflow.
	CountByWorkflowID(ctx context.Context, workflowID shared.ID) (int64, error)

	// CountByStatus returns the number of instances with a given status.
	CountByStatus(ctx context.Context, status Status) (int64, error)
}
