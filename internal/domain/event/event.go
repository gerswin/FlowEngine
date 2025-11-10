package event

import (
	"time"

	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/shared"
)

// DomainEvent represents an event that occurred in the domain.
// It follows the Event-Driven Architecture pattern.
type DomainEvent interface {
	// Type returns the event type identifier
	Type() string

	// AggregateID returns the ID of the aggregate that generated the event
	AggregateID() string

	// OccurredAt returns when the event occurred
	OccurredAt() time.Time

	// Payload returns the event data as a map
	Payload() map[string]interface{}
}

// BaseEvent provides common fields and methods for all domain events.
// Concrete events should embed this struct.
type BaseEvent struct {
	eventType   string
	aggregateID string
	occurredAt  time.Time
}

// Type returns the event type identifier.
func (e BaseEvent) Type() string {
	return e.eventType
}

// AggregateID returns the ID of the aggregate that generated the event.
func (e BaseEvent) AggregateID() string {
	return e.aggregateID
}

// OccurredAt returns when the event occurred.
func (e BaseEvent) OccurredAt() time.Time {
	return e.occurredAt
}

// newBaseEvent creates a base event with common fields.
func newBaseEvent(eventType string, aggregateID shared.ID) BaseEvent {
	return BaseEvent{
		eventType:   eventType,
		aggregateID: aggregateID.String(),
		occurredAt:  time.Now().UTC(),
	}
}

// InstanceCreated is emitted when a new workflow instance is created.
type InstanceCreated struct {
	BaseEvent
	WorkflowID   string
	WorkflowName string
	InitialState string
	StartedBy    string
	Data         map[string]interface{}
}

// NewInstanceCreated creates a new InstanceCreated event.
func NewInstanceCreated(instanceID, workflowID shared.ID, workflowName, initialState string, startedBy shared.ID, data map[string]interface{}) *InstanceCreated {
	return &InstanceCreated{
		BaseEvent:    newBaseEvent("instance.created", instanceID),
		WorkflowID:   workflowID.String(),
		WorkflowName: workflowName,
		InitialState: initialState,
		StartedBy:    startedBy.String(),
		Data:         data,
	}
}

// Payload returns the event data as a map.
func (e *InstanceCreated) Payload() map[string]interface{} {
	return map[string]interface{}{
		"instance_id":    e.AggregateID(),
		"workflow_id":    e.WorkflowID,
		"workflow_name":  e.WorkflowName,
		"initial_state":  e.InitialState,
		"started_by":     e.StartedBy,
		"data":           e.Data,
		"occurred_at":    e.OccurredAt(),
	}
}

// StateChanged is emitted when an instance transitions to a new state.
type StateChanged struct {
	BaseEvent
	InstanceID   string
	FromState    string
	ToState      string
	Event        string
	Actor        string
	Variables    map[string]interface{}
	TransitionID string
}

// NewStateChanged creates a new StateChanged event.
func NewStateChanged(instanceID shared.ID, fromState, toState, event string, actor shared.ID, transitionID shared.ID, variables map[string]interface{}) *StateChanged {
	return &StateChanged{
		BaseEvent:    newBaseEvent("instance.state_changed", instanceID),
		InstanceID:   instanceID.String(),
		FromState:    fromState,
		ToState:      toState,
		Event:        event,
		Actor:        actor.String(),
		TransitionID: transitionID.String(),
		Variables:    variables,
	}
}

// Payload returns the event data as a map.
func (e *StateChanged) Payload() map[string]interface{} {
	return map[string]interface{}{
		"instance_id":   e.InstanceID,
		"from_state":    e.FromState,
		"to_state":      e.ToState,
		"event":         e.Event,
		"actor":         e.Actor,
		"transition_id": e.TransitionID,
		"variables":     e.Variables,
		"occurred_at":   e.OccurredAt(),
	}
}

// SubStateChanged is emitted when an instance sub-state changes (R17).
type SubStateChanged struct {
	BaseEvent
	InstanceID     string
	CurrentState   string
	FromSubState   string
	ToSubState     string
	Actor          string
}

// NewSubStateChanged creates a new SubStateChanged event.
func NewSubStateChanged(instanceID shared.ID, currentState, fromSubState, toSubState string, actor shared.ID) *SubStateChanged {
	return &SubStateChanged{
		BaseEvent:      newBaseEvent("instance.substate_changed", instanceID),
		InstanceID:     instanceID.String(),
		CurrentState:   currentState,
		FromSubState:   fromSubState,
		ToSubState:     toSubState,
		Actor:          actor.String(),
	}
}

// Payload returns the event data as a map.
func (e *SubStateChanged) Payload() map[string]interface{} {
	return map[string]interface{}{
		"instance_id":    e.InstanceID,
		"current_state":  e.CurrentState,
		"from_substate":  e.FromSubState,
		"to_substate":    e.ToSubState,
		"actor":          e.Actor,
		"occurred_at":    e.OccurredAt(),
	}
}

// InstancePaused is emitted when an instance is paused.
type InstancePaused struct {
	BaseEvent
	InstanceID string
	Reason     string
	PausedBy   string
}

// NewInstancePaused creates a new InstancePaused event.
func NewInstancePaused(instanceID, pausedBy shared.ID, reason string) *InstancePaused {
	return &InstancePaused{
		BaseEvent:  newBaseEvent("instance.paused", instanceID),
		InstanceID: instanceID.String(),
		Reason:     reason,
		PausedBy:   pausedBy.String(),
	}
}

// Payload returns the event data as a map.
func (e *InstancePaused) Payload() map[string]interface{} {
	return map[string]interface{}{
		"instance_id": e.InstanceID,
		"reason":      e.Reason,
		"paused_by":   e.PausedBy,
		"occurred_at": e.OccurredAt(),
	}
}

// InstanceResumed is emitted when a paused instance is resumed.
type InstanceResumed struct {
	BaseEvent
	InstanceID string
	ResumedBy  string
}

// NewInstanceResumed creates a new InstanceResumed event.
func NewInstanceResumed(instanceID, resumedBy shared.ID) *InstanceResumed {
	return &InstanceResumed{
		BaseEvent:  newBaseEvent("instance.resumed", instanceID),
		InstanceID: instanceID.String(),
		ResumedBy:  resumedBy.String(),
	}
}

// Payload returns the event data as a map.
func (e *InstanceResumed) Payload() map[string]interface{} {
	return map[string]interface{}{
		"instance_id": e.InstanceID,
		"resumed_by":  e.ResumedBy,
		"occurred_at": e.OccurredAt(),
	}
}

// InstanceCompleted is emitted when an instance reaches a final state.
type InstanceCompleted struct {
	BaseEvent
	InstanceID   string
	FinalState   string
	CompletedBy  string
	Result       map[string]interface{}
}

// NewInstanceCompleted creates a new InstanceCompleted event.
func NewInstanceCompleted(instanceID shared.ID, finalState string, completedBy shared.ID, result map[string]interface{}) *InstanceCompleted {
	return &InstanceCompleted{
		BaseEvent:    newBaseEvent("instance.completed", instanceID),
		InstanceID:   instanceID.String(),
		FinalState:   finalState,
		CompletedBy:  completedBy.String(),
		Result:       result,
	}
}

// Payload returns the event data as a map.
func (e *InstanceCompleted) Payload() map[string]interface{} {
	return map[string]interface{}{
		"instance_id":  e.InstanceID,
		"final_state":  e.FinalState,
		"completed_by": e.CompletedBy,
		"result":       e.Result,
		"occurred_at":  e.OccurredAt(),
	}
}

// InstanceCanceled is emitted when an instance is explicitly canceled.
type InstanceCanceled struct {
	BaseEvent
	InstanceID   string
	CurrentState string
	Reason       string
	CanceledBy   string
}

// NewInstanceCanceled creates a new InstanceCanceled event.
func NewInstanceCanceled(instanceID shared.ID, currentState string, reason string, canceledBy shared.ID) *InstanceCanceled {
	return &InstanceCanceled{
		BaseEvent:    newBaseEvent("instance.canceled", instanceID),
		InstanceID:   instanceID.String(),
		CurrentState: currentState,
		Reason:       reason,
		CanceledBy:   canceledBy.String(),
	}
}

// Payload returns the event data as a map.
func (e *InstanceCanceled) Payload() map[string]interface{} {
	return map[string]interface{}{
		"instance_id":   e.InstanceID,
		"current_state": e.CurrentState,
		"reason":        e.Reason,
		"canceled_by":   e.CanceledBy,
		"occurred_at":   e.OccurredAt(),
	}
}

// InstanceFailed is emitted when an instance fails due to an error.
type InstanceFailed struct {
	BaseEvent
	InstanceID   string
	CurrentState string
	Error        string
	FailedBy     string
}

// NewInstanceFailed creates a new InstanceFailed event.
func NewInstanceFailed(instanceID shared.ID, currentState, errorMsg string, failedBy shared.ID) *InstanceFailed {
	return &InstanceFailed{
		BaseEvent:    newBaseEvent("instance.failed", instanceID),
		InstanceID:   instanceID.String(),
		CurrentState: currentState,
		Error:        errorMsg,
		FailedBy:     failedBy.String(),
	}
}

// Payload returns the event data as a map.
func (e *InstanceFailed) Payload() map[string]interface{} {
	return map[string]interface{}{
		"instance_id":   e.InstanceID,
		"current_state": e.CurrentState,
		"error":         e.Error,
		"failed_by":     e.FailedBy,
		"occurred_at":   e.OccurredAt(),
	}
}

// DocumentReclassified is emitted when an instance document type is reclassified (R19).
type DocumentReclassified struct {
	BaseEvent
	InstanceID   string
	OldType      string
	NewType      string
	Reason       string
	ReclassifiedBy string
}

// NewDocumentReclassified creates a new DocumentReclassified event.
func NewDocumentReclassified(instanceID shared.ID, oldType, newType, reason string, reclassifiedBy shared.ID) *DocumentReclassified {
	return &DocumentReclassified{
		BaseEvent:      newBaseEvent("instance.document_reclassified", instanceID),
		InstanceID:     instanceID.String(),
		OldType:        oldType,
		NewType:        newType,
		Reason:         reason,
		ReclassifiedBy: reclassifiedBy.String(),
	}
}

// Payload returns the event data as a map.
func (e *DocumentReclassified) Payload() map[string]interface{} {
	return map[string]interface{}{
		"instance_id":      e.InstanceID,
		"old_type":         e.OldType,
		"new_type":         e.NewType,
		"reason":           e.Reason,
		"reclassified_by":  e.ReclassifiedBy,
		"occurred_at":      e.OccurredAt(),
	}
}

// WorkflowCreated is emitted when a new workflow is created.
type WorkflowCreated struct {
	BaseEvent
	WorkflowID   string
	Name         string
	InitialState string
	CreatedBy    string
}

// NewWorkflowCreated creates a new WorkflowCreated event.
func NewWorkflowCreated(workflowID shared.ID, name, initialState string, createdBy shared.ID) *WorkflowCreated {
	return &WorkflowCreated{
		BaseEvent:    newBaseEvent("workflow.created", workflowID),
		WorkflowID:   workflowID.String(),
		Name:         name,
		InitialState: initialState,
		CreatedBy:    createdBy.String(),
	}
}

// Payload returns the event data as a map.
func (e *WorkflowCreated) Payload() map[string]interface{} {
	return map[string]interface{}{
		"workflow_id":   e.WorkflowID,
		"name":          e.Name,
		"initial_state": e.InitialState,
		"created_by":    e.CreatedBy,
		"occurred_at":   e.OccurredAt(),
	}
}

// WorkflowUpdated is emitted when a workflow definition is updated.
type WorkflowUpdated struct {
	BaseEvent
	WorkflowID string
	Version    string
	UpdatedBy  string
	Changes    map[string]interface{}
}

// NewWorkflowUpdated creates a new WorkflowUpdated event.
func NewWorkflowUpdated(workflowID shared.ID, version string, updatedBy shared.ID, changes map[string]interface{}) *WorkflowUpdated {
	return &WorkflowUpdated{
		BaseEvent:  newBaseEvent("workflow.updated", workflowID),
		WorkflowID: workflowID.String(),
		Version:    version,
		UpdatedBy:  updatedBy.String(),
		Changes:    changes,
	}
}

// Payload returns the event data as a map.
func (e *WorkflowUpdated) Payload() map[string]interface{} {
	return map[string]interface{}{
		"workflow_id": e.WorkflowID,
		"version":     e.Version,
		"updated_by":  e.UpdatedBy,
		"changes":     e.Changes,
		"occurred_at": e.OccurredAt(),
	}
}
