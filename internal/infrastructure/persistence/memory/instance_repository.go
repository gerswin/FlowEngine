package memory

import (
	"context"
	"sort"
	"sync"

	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/instance"
	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/shared"
)

// InstanceInMemoryRepository implements instance.Repository using in-memory storage.
// This is useful for testing and development without database dependencies.
type InstanceInMemoryRepository struct {
	mu        sync.RWMutex
	instances map[string]*instance.Instance // keyed by instance ID
}

// NewInstanceInMemoryRepository creates a new in-memory instance repository.
func NewInstanceInMemoryRepository() *InstanceInMemoryRepository {
	return &InstanceInMemoryRepository{
		instances: make(map[string]*instance.Instance),
	}
}

// Save persists an instance to memory.
// This implements optimistic locking by checking the version.
func (r *InstanceInMemoryRepository) Save(ctx context.Context, inst *instance.Instance) error {
	if inst == nil {
		return instance.InvalidInstanceError("instance cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check for optimistic locking conflict
	existing, exists := r.instances[inst.ID().String()]
	if exists && !existing.Version().Equals(inst.Version()) {
		return instance.VersionConflictError(inst.ID(), existing.Version(), inst.Version())
	}

	r.instances[inst.ID().String()] = inst
	return nil
}

// FindByID retrieves an instance by its ID.
func (r *InstanceInMemoryRepository) FindByID(ctx context.Context, id shared.ID) (*instance.Instance, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	inst, exists := r.instances[id.String()]
	if !exists {
		return nil, instance.InstanceNotFoundError(id.String())
	}

	return inst, nil
}

// FindByWorkflowID retrieves all instances for a given workflow.
func (r *InstanceInMemoryRepository) FindByWorkflowID(ctx context.Context, workflowID shared.ID) ([]*instance.Instance, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	instances := make([]*instance.Instance, 0)
	for _, inst := range r.instances {
		if inst.WorkflowID().Equals(workflowID) {
			instances = append(instances, inst)
		}
	}

	return instances, nil
}

// FindAll retrieves all instances.
func (r *InstanceInMemoryRepository) FindAll(ctx context.Context) ([]*instance.Instance, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	instances := make([]*instance.Instance, 0, len(r.instances))
	for _, inst := range r.instances {
		instances = append(instances, inst)
	}

	return instances, nil
}

// FindByParentID retrieves all child instances (subprocesses) for a given parent instance.
func (r *InstanceInMemoryRepository) FindByParentID(ctx context.Context, parentID shared.ID) ([]*instance.Instance, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	instances := make([]*instance.Instance, 0)
	for _, inst := range r.instances {
		if inst.ParentID().Equals(parentID) {
			instances = append(instances, inst)
		}
	}

	return instances, nil
}

// FindByStatus retrieves all instances with a specific status.
func (r *InstanceInMemoryRepository) FindByStatus(ctx context.Context, status instance.Status) ([]*instance.Instance, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	instances := make([]*instance.Instance, 0)
	for _, inst := range r.instances {
		if inst.Status() == status {
			instances = append(instances, inst)
		}
	}

	return instances, nil
}

// Delete removes an instance from memory.
func (r *InstanceInMemoryRepository) Delete(ctx context.Context, id shared.ID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.instances[id.String()]; !exists {
		return instance.InstanceNotFoundError(id.String())
	}

	delete(r.instances, id.String())
	return nil
}

// Clear removes all instances (useful for testing).
func (r *InstanceInMemoryRepository) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.instances = make(map[string]*instance.Instance)
}

// FindByWorkflowIDAndStatus retrieves instances for a workflow with specific status.
func (r *InstanceInMemoryRepository) FindByWorkflowIDAndStatus(ctx context.Context, workflowID shared.ID, status instance.Status) ([]*instance.Instance, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	instances := make([]*instance.Instance, 0)
	for _, inst := range r.instances {
		if inst.WorkflowID().Equals(workflowID) && inst.Status() == status {
			instances = append(instances, inst)
		}
	}

	return instances, nil
}

// FindActive retrieves all active instances (Running or Paused).
func (r *InstanceInMemoryRepository) FindActive(ctx context.Context) ([]*instance.Instance, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	instances := make([]*instance.Instance, 0)
	for _, inst := range r.instances {
		if inst.IsActive() {
			instances = append(instances, inst)
		}
	}

	return instances, nil
}

// Exists checks if an instance exists by ID.
func (r *InstanceInMemoryRepository) Exists(ctx context.Context, id shared.ID) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.instances[id.String()]
	return exists, nil
}

// Count returns the total number of instances (implements Repository interface).
func (r *InstanceInMemoryRepository) Count(ctx context.Context) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return int64(len(r.instances)), nil
}

// CountByWorkflowID returns the number of instances for a workflow.
func (r *InstanceInMemoryRepository) CountByWorkflowID(ctx context.Context, workflowID shared.ID) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := int64(0)
	for _, inst := range r.instances {
		if inst.WorkflowID().Equals(workflowID) {
			count++
		}
	}

	return count, nil
}

// List retrieves a paginated list of instances, optionally filtered by workflow ID.
func (r *InstanceInMemoryRepository) List(ctx context.Context, q shared.ListQuery, workflowID *shared.ID) ([]*instance.Instance, int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Collect matching instances
	var filtered []*instance.Instance
	for _, inst := range r.instances {
		if workflowID != nil && !inst.WorkflowID().Equals(*workflowID) {
			continue
		}
		filtered = append(filtered, inst)
	}

	// Sort by created_at descending for consistency with postgres
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].CreatedAt().Time().After(filtered[j].CreatedAt().Time())
	})

	total := int64(len(filtered))

	// Apply offset/limit
	start := q.Offset
	if start > len(filtered) {
		start = len(filtered)
	}
	end := start + q.Limit
	if end > len(filtered) {
		end = len(filtered)
	}

	return filtered[start:end], total, nil
}

// CountByStatus returns the number of instances with a given status.
func (r *InstanceInMemoryRepository) CountByStatus(ctx context.Context, status instance.Status) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := int64(0)
	for _, inst := range r.instances {
		if inst.Status() == status {
			count++
		}
	}

	return count, nil
}
