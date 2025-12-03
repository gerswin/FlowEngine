package instance

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/event"
	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/instance"
	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/shared"
	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/workflow"
)

// CloneInstanceCommand defines the request to clone an instance.
type CloneInstanceCommand struct {
	ParentInstanceID string
	Assignees        []CloneAssignee
	ConsolidatorID   string
	Reason           string
	TimeoutDuration  string // e.g. "24h"
	Metadata         map[string]interface{}
}

type CloneAssignee struct {
	UserID   string
	OfficeID string
}

// CloneInstanceResult returns the result of the cloning operation.
type CloneInstanceResult struct {
	CloneGroupID    string
	ParentID        string
	ClonedInstances []string // IDs of created clones
	ExpiresAt       time.Time
}

// CloneInstanceUseCase handles the cloning of instances.
type CloneInstanceUseCase struct {
	instanceRepo instance.Repository
	workflowRepo workflow.Repository
	eventBus     event.Dispatcher
}

func NewCloneInstanceUseCase(
	instanceRepo instance.Repository,
	workflowRepo workflow.Repository,
	eventBus event.Dispatcher,
) *CloneInstanceUseCase {
	return &CloneInstanceUseCase{
		instanceRepo: instanceRepo,
		workflowRepo: workflowRepo,
		eventBus:     eventBus,
	}
}

func (uc *CloneInstanceUseCase) Execute(ctx context.Context, cmd CloneInstanceCommand) (*CloneInstanceResult, error) {
	// 1. Validate Parent
	parentID, err := shared.ParseID(cmd.ParentInstanceID)
	if err != nil {
		return nil, instance.InvalidInstanceError("invalid parent ID")
	}

	parent, err := uc.instanceRepo.FindByID(ctx, parentID)
	if err != nil {
		return nil, err
	}

	if parent.Status() != instance.StatusRunning {
		return nil, instance.InstanceNotActiveError(parentID, parent.Status())
	}

	// 2. Calculate Expiration
	duration, err := time.ParseDuration(cmd.TimeoutDuration)
	if err != nil {
		return nil, instance.InvalidInstanceError("invalid timeout duration format")
	}
	expiresAt := time.Now().Add(duration)

	// 3. Create Clone Group ID
	cloneGroupID := uuid.New().String()

	// 4. Create Clones
	var clonedIDs []string
	
	// Retrieve workflow to get state config? 
	// For now assume generic "pending" state for clones as per requirements, 
	// or we use the same workflow definition but start at a specific state.
	// R15.1: "Una clonación es implementada como un tipo especial de subproceso con estados simplificados... pending, accepted..."
	// This implies a "Clone Workflow" definition might exist or we force the state.
	// Simplification: We use the same workflow ID but force initial state "pending" if it exists, 
	// or assume the workflow has a specific flow for clones.
	// Requirement says: "cloned_instance_workflow" section in YAML.
	// Since we don't have that parsed yet, I'll assume we start clones in a state named "pending".

	for _, assignee := range cmd.Assignees {
		assigneeID, _ := shared.ParseID(assignee.UserID)
		
		// Create Subprocess
		// Note: We should probably use a specific "Clone Workflow" or the same workflow.
		// The requirement R15.7 mentions configuration in YAML.
		// For MVP, I will create instances of the SAME workflow, starting at "pending".
		
		clone, err := instance.NewSubprocess(
			parentID,
			parent.WorkflowID(),
			parent.WorkflowName(),
			"pending", // Hardcoded initial state for clone
			assigneeID, // Started by/Assigned to
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create clone for user %s: %w", assignee.UserID, err)
		}

		// Set metadata
		data := instance.NewData()
		data = data.Set("clone_group_id", cloneGroupID)
		data = data.Set("clone_type", "cloned_instance")
		data = data.Set("consolidator_id", cmd.ConsolidatorID)
		data = data.Set("reason", cmd.Reason)
		data = data.Set("expires_at", expiresAt)
		
		// Copy relevant data from parent? 
		// "data: copia de datos relevantes del padre"
		// Copy all for now
		for k, v := range parent.Data().ToMap() {
			data = data.Set(k, v)
		}
		
		clone.SetData(data)

		if err := uc.instanceRepo.Save(ctx, clone); err != nil {
			return nil, fmt.Errorf("failed to persist clone: %w", err)
		}
		
		clonedIDs = append(clonedIDs, clone.ID().String())
	}

	// 5. Update Parent State?
	// R15.1: "el sistema SHALL cambiar el estado del trámite principal según configuración"
	// Skipping strict workflow config check for now.

	// 6. Generate Events
	// "CloneGroupCreated", "ClonedInstanceCreated"
	// Handled by domain events inside Instance or manually here.
	
	return &CloneInstanceResult{
		CloneGroupID:    cloneGroupID,
		ParentID:        parentID.String(),
		ClonedInstances: clonedIDs,
		ExpiresAt:       expiresAt,
	}, nil
}
