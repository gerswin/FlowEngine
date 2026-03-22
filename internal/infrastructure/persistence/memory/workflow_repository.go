package memory

import (
	"context"
	"sort"
	"sync"

	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/shared"
	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/workflow"
)

// WorkflowInMemoryRepository implements workflow.Repository using in-memory storage.
// This is useful for testing and development without database dependencies.
type WorkflowInMemoryRepository struct {
	mu        sync.RWMutex
	workflows map[string]*workflow.Workflow // keyed by workflow ID
}

// NewWorkflowInMemoryRepository creates a new in-memory workflow repository.
func NewWorkflowInMemoryRepository() *WorkflowInMemoryRepository {
	return &WorkflowInMemoryRepository{
		workflows: make(map[string]*workflow.Workflow),
	}
}

// Save persists a workflow to memory.
func (r *WorkflowInMemoryRepository) Save(ctx context.Context, wf *workflow.Workflow) error {
	if wf == nil {
		return workflow.InvalidWorkflowError("workflow cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.workflows[wf.ID().String()] = wf
	return nil
}

// FindByID retrieves a workflow by its ID.
func (r *WorkflowInMemoryRepository) FindByID(ctx context.Context, id shared.ID) (*workflow.Workflow, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	wf, exists := r.workflows[id.String()]
	if !exists {
		return nil, workflow.WorkflowNotFoundError(id.String())
	}

	return wf, nil
}

// FindAll retrieves all workflows.
func (r *WorkflowInMemoryRepository) FindAll(ctx context.Context) ([]*workflow.Workflow, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	workflows := make([]*workflow.Workflow, 0, len(r.workflows))
	for _, wf := range r.workflows {
		workflows = append(workflows, wf)
	}

	return workflows, nil
}

// Delete removes a workflow from memory.
func (r *WorkflowInMemoryRepository) Delete(ctx context.Context, id shared.ID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.workflows[id.String()]; !exists {
		return workflow.WorkflowNotFoundError(id.String())
	}

	delete(r.workflows, id.String())
	return nil
}

// Clear removes all workflows (useful for testing).
func (r *WorkflowInMemoryRepository) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.workflows = make(map[string]*workflow.Workflow)
}

// Count returns the number of workflows in memory.
func (r *WorkflowInMemoryRepository) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.workflows)
}

// FindByName retrieves a workflow by name and version.
func (r *WorkflowInMemoryRepository) FindByName(ctx context.Context, name string, version workflow.Version) (*workflow.Workflow, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, wf := range r.workflows {
		if wf.Name() == name && wf.Version().Equals(version) {
			return wf, nil
		}
	}

	return nil, workflow.WorkflowNotFoundError(name + " v" + version.String())
}

// FindLatestByName retrieves the latest version of a workflow by name.
func (r *WorkflowInMemoryRepository) FindLatestByName(ctx context.Context, name string) (*workflow.Workflow, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var latest *workflow.Workflow
	for _, wf := range r.workflows {
		if wf.Name() == name {
			if latest == nil || wf.Version().IsGreaterThan(latest.Version()) {
				latest = wf
			}
		}
	}

	if latest == nil {
		return nil, workflow.WorkflowNotFoundError(name)
	}

	return latest, nil
}

// FindAllByName retrieves all versions of a workflow by name.
func (r *WorkflowInMemoryRepository) FindAllByName(ctx context.Context, name string) ([]*workflow.Workflow, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	workflows := make([]*workflow.Workflow, 0)
	for _, wf := range r.workflows {
		if wf.Name() == name {
			workflows = append(workflows, wf)
		}
	}

	return workflows, nil
}

// List retrieves a paginated list of workflows.
func (r *WorkflowInMemoryRepository) List(ctx context.Context, q shared.ListQuery) ([]*workflow.Workflow, int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	all := make([]*workflow.Workflow, 0, len(r.workflows))
	for _, wf := range r.workflows {
		all = append(all, wf)
	}

	sort.Slice(all, func(i, j int) bool {
		return all[i].CreatedAt().Time().After(all[j].CreatedAt().Time())
	})

	total := int64(len(all))

	start := q.Offset
	if start > len(all) {
		start = len(all)
	}
	end := start + q.Limit
	if end > len(all) {
		end = len(all)
	}

	return all[start:end], total, nil
}

// Exists checks if a workflow exists by ID.
func (r *WorkflowInMemoryRepository) Exists(ctx context.Context, id shared.ID) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.workflows[id.String()]
	return exists, nil
}

// ExistsByName checks if a workflow exists by name and version.
func (r *WorkflowInMemoryRepository) ExistsByName(ctx context.Context, name string, version workflow.Version) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, wf := range r.workflows {
		if wf.Name() == name && wf.Version().Equals(version) {
			return true, nil
		}
	}

	return false, nil
}
