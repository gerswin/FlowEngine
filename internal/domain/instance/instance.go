package instance

import (
	"fmt"

	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/event"
	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/shared"
)

// Instance represents a running instance of a workflow.
// It's the aggregate root for the instance domain.
type Instance struct {
	id              shared.ID
	parentID        shared.ID // Parent instance ID (for subprocesses)
	workflowID      shared.ID
	workflowName    string
	currentState    string   // State ID
	currentSubState SubState // R17: Sub-state support
	status          Status
	version         Version // For optimistic locking
	data            Data
	variables       Variables
	transitions     []*Transition
	domainEvents    []event.DomainEvent // Domain events to be published
	createdAt       shared.Timestamp
	updatedAt       shared.Timestamp
	completedAt     shared.Timestamp
	startedBy       shared.ID
}

// RestoreInstance reconstitutes an instance from persistence.
// This should ONLY be used by repositories.
func RestoreInstance(
	id shared.ID,
	parentID shared.ID,
	workflowID shared.ID,
	workflowName string,
	currentState string,
	currentSubState SubState,
	status Status,
	version Version,
	data Data,
	variables Variables,
	transitions []*Transition,
	createdAt shared.Timestamp,
	updatedAt shared.Timestamp,
	completedAt shared.Timestamp,
	startedBy shared.ID,
) *Instance {
	return &Instance{
		id:              id,
		parentID:        parentID,
		workflowID:      workflowID,
		workflowName:    workflowName,
		currentState:    currentState,
		currentSubState: currentSubState,
		status:          status,
		version:         version,
		data:            data,
		variables:       variables,
		transitions:     transitions,
		domainEvents:    []event.DomainEvent{}, // Restored instances start with no pending events
		createdAt:       createdAt,
		updatedAt:       updatedAt,
		completedAt:     completedAt,
		startedBy:       startedBy,
	}
}

// NewInstance creates a new workflow instance.
func NewInstance(workflowID shared.ID, workflowName, initialState string, startedBy shared.ID) (*Instance, error) {
	if !workflowID.IsValid() {
		return nil, InvalidInstanceError("workflow ID is invalid")
	}

	if workflowName == "" {
		return nil, InvalidInstanceError("workflow name cannot be empty")
	}

	if initialState == "" {
		return nil, InvalidInstanceError("initial state cannot be empty")
	}

	if !startedBy.IsValid() {
		return nil, InvalidInstanceError("started by actor ID is invalid")
	}

	now := shared.Now()

	instance := &Instance{
		id:           shared.NewID(),
		workflowID:   workflowID,
		workflowName: workflowName,
		currentState: initialState,
		status:       StatusRunning,
		version:      NewVersion(),
		data:         NewData(),
		variables:    NewVariables(),
		transitions:  []*Transition{},
		domainEvents: []event.DomainEvent{},
		createdAt:    now,
		updatedAt:    now,
		startedBy:    startedBy,
	}

	// Record instance created event
	instance.recordEvent(event.NewInstanceCreated(
		instance.id,
		workflowID,
		workflowName,
		initialState,
		startedBy,
		instance.data.ToMap(),
	))

	return instance, nil
}

// NewInstanceWithSubState creates a new workflow instance with an initial sub-state (R17).
func NewInstanceWithSubState(workflowID shared.ID, workflowName, initialState string, initialSubState SubState, startedBy shared.ID) (*Instance, error) {
	instance, err := NewInstance(workflowID, workflowName, initialState, startedBy)
	if err != nil {
		return nil, err
	}

	if err := initialSubState.Validate(); err != nil {
		return nil, InvalidInstanceError(fmt.Sprintf("invalid initial sub-state: %v", err))
	}

	instance.currentSubState = initialSubState
	return instance, nil
}

// NewSubprocess creates a new subprocess instance.
func NewSubprocess(parentID, workflowID shared.ID, workflowName, initialState string, startedBy shared.ID) (*Instance, error) {
	instance, err := NewInstance(workflowID, workflowName, initialState, startedBy)
	if err != nil {
		return nil, err
	}
	if !parentID.IsValid() {
		return nil, InvalidInstanceError("parent instance ID is invalid")
	}
	instance.parentID = parentID
	return instance, nil
}

// ID returns the instance ID.
func (i *Instance) ID() shared.ID {
	return i.id
}

// ParentID returns the parent instance ID (if any).
func (i *Instance) ParentID() shared.ID {
	return i.parentID
}

// WorkflowID returns the workflow ID.
func (i *Instance) WorkflowID() shared.ID {
	return i.workflowID
}

// WorkflowName returns the workflow name.
func (i *Instance) WorkflowName() string {
	return i.workflowName
}

// CurrentState returns the current state ID.
func (i *Instance) CurrentState() string {
	return i.currentState
}

// CurrentSubState returns the current sub-state (R17).
func (i *Instance) CurrentSubState() SubState {
	return i.currentSubState
}

// Status returns the instance status.
func (i *Instance) Status() Status {
	return i.status
}

// Version returns the instance version (for optimistic locking).
func (i *Instance) Version() Version {
	return i.version
}

// Data returns a copy of the instance data.
func (i *Instance) Data() Data {
	return i.data
}

// Variables returns a copy of the instance variables.
func (i *Instance) Variables() Variables {
	return i.variables
}

// Transitions returns a copy of the transition history.
func (i *Instance) Transitions() []*Transition {
	transitionsCopy := make([]*Transition, len(i.transitions))
	copy(transitionsCopy, i.transitions)
	return transitionsCopy
}

// CreatedAt returns the creation timestamp.
func (i *Instance) CreatedAt() shared.Timestamp {
	return i.createdAt
}

// UpdatedAt returns the last update timestamp.
func (i *Instance) UpdatedAt() shared.Timestamp {
	return i.updatedAt
}

// CompletedAt returns the completion timestamp.
func (i *Instance) CompletedAt() shared.Timestamp {
	return i.completedAt
}

// StartedBy returns the ID of the actor who started the instance.
func (i *Instance) StartedBy() shared.ID {
	return i.startedBy
}

// DomainEvents returns all domain events and clears the internal event list.
// This follows the pattern where the aggregate generates events, and the
// application layer is responsible for dispatching them after persistence.
func (i *Instance) DomainEvents() []event.DomainEvent {
	events := make([]event.DomainEvent, len(i.domainEvents))
	copy(events, i.domainEvents)
	i.domainEvents = []event.DomainEvent{} // Clear events after returning
	return events
}

// recordEvent adds a domain event to the instance's event list.
func (i *Instance) recordEvent(evt event.DomainEvent) {
	i.domainEvents = append(i.domainEvents, evt)
}

// IsActive returns true if the instance is in an active state.
func (i *Instance) IsActive() bool {
	return i.status.IsActive()
}

// IsFinal returns true if the instance is in a final state.
func (i *Instance) IsFinal() bool {
	return i.status.IsFinal()
}

// HasSubState returns true if the instance has a sub-state set (R17).
func (i *Instance) HasSubState() bool {
	return !i.currentSubState.IsZero()
}

// SetData sets the instance data.
func (i *Instance) SetData(data Data) {
	i.data = data
	i.updatedAt = shared.Now()
	i.version = i.version.Increment()
}

// UpdateData updates specific keys in the instance data.
func (i *Instance) UpdateData(key string, value interface{}) {
	i.data = i.data.Set(key, value)
	i.updatedAt = shared.Now()
	i.version = i.version.Increment()
}

// SetVariables sets the instance variables.
func (i *Instance) SetVariables(variables Variables) {
	i.variables = variables
	i.updatedAt = shared.Now()
	i.version = i.version.Increment()
}

// UpdateVariable updates a specific variable.
func (i *Instance) UpdateVariable(key string, value interface{}) {
	i.variables = i.variables.Set(key, value)
	i.updatedAt = shared.Now()
	i.version = i.version.Increment()
}

// Transition performs a state transition with the given event.
func (i *Instance) Transition(toState, eventName string, actor shared.ID, metadata TransitionMetadata, requiredData []string) error {
	return i.TransitionWithSubState(toState, ZeroSubState(), eventName, actor, metadata, requiredData)
}

// TransitionWithSubState performs a state transition with sub-state support (R17, R23).
func (i *Instance) TransitionWithSubState(toState string, toSubState SubState, eventName string, actor shared.ID, metadata TransitionMetadata, requiredData []string) error {
	// Validate instance can transition
	if !i.IsActive() {
		return InstanceNotActiveError(i.id, i.status)
	}

	if toState == "" {
		return InvalidTransitionError(i.currentState, toState, eventName)
	}

	if eventName == "" {
		return InvalidInstanceError("event name cannot be empty")
	}

	if !actor.IsValid() {
		return InvalidInstanceError("actor ID is invalid")
	}

	// Validate sub-state if provided
	if !toSubState.IsZero() {
		if err := toSubState.Validate(); err != nil {
			return InvalidInstanceError(fmt.Sprintf("invalid sub-state: %v", err))
		}
	}

	// Validate required data fields
	if len(requiredData) > 0 {
		var missingFields []string
		for _, field := range requiredData {
			if !i.data.Has(field) {
				missingFields = append(missingFields, field)
			}
		}
		if len(missingFields) > 0 {
			return MissingRequiredDataError(eventName, missingFields)
		}
	}

	// Create transition record
	var transition *Transition
	if i.HasSubState() || !toSubState.IsZero() {
		transition = NewTransitionWithSubStates(
			i.currentState,
			toState,
			eventName,
			actor,
			i.currentSubState,
			toSubState,
		)
	} else {
		transition = NewTransition(i.currentState, toState, eventName, actor)
	}

	transition.SetMetadata(metadata)
	transition.SetData(i.data) // Snapshot current data

	// Store previous state and sub-state for event
	previousState := i.currentState
	previousSubState := i.currentSubState

	// Apply transition
	i.currentState = toState
	i.currentSubState = toSubState
	i.transitions = append(i.transitions, transition)
	i.updatedAt = shared.Now()
	i.version = i.version.Increment()

	// Record state changed event
	i.recordEvent(event.NewStateChanged(
		i.id,
		previousState,
		toState,
		eventName,
		actor,
		transition.ID(),
		i.variables.ToMap(),
	))

	// If sub-state changed, record sub-state changed event (R17)
	if !previousSubState.Equals(toSubState) && (!previousSubState.IsZero() || !toSubState.IsZero()) {
		fromSubStateID := ""
		if !previousSubState.IsZero() {
			fromSubStateID = previousSubState.ID()
		}
		toSubStateID := ""
		if !toSubState.IsZero() {
			toSubStateID = toSubState.ID()
		}

		i.recordEvent(event.NewSubStateChanged(
			i.id,
			i.currentState,
			fromSubStateID,
			toSubStateID,
			actor,
		))
	}

	return nil
}

// Complete marks the instance as completed.
func (i *Instance) Complete(actor shared.ID) error {
	if !i.IsActive() {
		return InstanceNotActiveError(i.id, i.status)
	}

	i.status = StatusCompleted
	i.completedAt = shared.Now()
	i.updatedAt = shared.Now()
	i.version = i.version.Increment()

	// Record instance completed event
	i.recordEvent(event.NewInstanceCompleted(
		i.id,
		i.currentState,
		actor,
		i.data.ToMap(),
	))

	return nil
}

// Cancel marks the instance as canceled.
func (i *Instance) Cancel(actor shared.ID, reason string) error {
	if !i.IsActive() {
		return InstanceNotActiveError(i.id, i.status)
	}

	i.status = StatusCanceled
	i.completedAt = shared.Now()
	i.updatedAt = shared.Now()
	i.version = i.version.Increment()

	// Optionally store cancellation reason in data
	if reason != "" {
		i.data = i.data.Set("cancellation_reason", reason)
	}

	// Record instance canceled event
	i.recordEvent(event.NewInstanceCanceled(
		i.id,
		i.currentState,
		reason,
		actor,
	))

	return nil
}

// Fail marks the instance as failed.
func (i *Instance) Fail(actor shared.ID, reason string) error {
	if !i.IsActive() {
		return InstanceNotActiveError(i.id, i.status)
	}

	i.status = StatusFailed
	i.completedAt = shared.Now()
	i.updatedAt = shared.Now()
	i.version = i.version.Increment()

	// Store failure reason in data
	if reason != "" {
		i.data = i.data.Set("failure_reason", reason)
	}

	// Record instance failed event
	i.recordEvent(event.NewInstanceFailed(
		i.id,
		i.currentState,
		reason,
		actor,
	))

	return nil
}

// Pause pauses the instance execution.
func (i *Instance) Pause(actor shared.ID, reason string) error {
	if i.status != StatusRunning {
		return InvalidInstanceError("can only pause running instances")
	}

	i.status = StatusPaused
	i.updatedAt = shared.Now()
	i.version = i.version.Increment()

	// Record instance paused event
	i.recordEvent(event.NewInstancePaused(
		i.id,
		actor,
		reason,
	))

	return nil
}

// Resume resumes a paused instance.
func (i *Instance) Resume(actor shared.ID) error {
	if i.status != StatusPaused {
		return InvalidInstanceError("can only resume paused instances")
	}

	i.status = StatusRunning
	i.updatedAt = shared.Now()
	i.version = i.version.Increment()

	// Record instance resumed event
	i.recordEvent(event.NewInstanceResumed(
		i.id,
		actor,
	))

	return nil
}

// GetLastTransition returns the most recent transition, or nil if none exist.
func (i *Instance) GetLastTransition() *Transition {
	if len(i.transitions) == 0 {
		return nil
	}
	return i.transitions[len(i.transitions)-1]
}

// GetTransitionHistory returns the full transition history.
func (i *Instance) GetTransitionHistory() []*Transition {
	return i.Transitions()
}

// TransitionCount returns the number of transitions that have occurred.
func (i *Instance) TransitionCount() int {
	return len(i.transitions)
}

// Validate validates the instance.
func (i *Instance) Validate() error {
	if !i.id.IsValid() {
		return InvalidInstanceError("instance ID is invalid")
	}

	if !i.workflowID.IsValid() {
		return InvalidInstanceError("workflow ID is invalid")
	}

	if i.workflowName == "" {
		return InvalidInstanceError("workflow name cannot be empty")
	}

	if i.currentState == "" {
		return InvalidInstanceError("current state cannot be empty")
	}

	if !i.status.IsValid() {
		return InvalidInstanceError(fmt.Sprintf("invalid status: %s", i.status))
	}

	if i.version.IsZero() {
		return InvalidInstanceError("version must be >= 1")
	}

	if !i.startedBy.IsValid() {
		return InvalidInstanceError("started by actor ID is invalid")
	}

	// Validate sub-state if present
	if i.HasSubState() {
		if err := i.currentSubState.Validate(); err != nil {
			return InvalidInstanceError(fmt.Sprintf("invalid current sub-state: %v", err))
		}
	}

	return nil
}

// String returns a string representation of the instance.
func (i *Instance) String() string {
	parentInfo := ""
	if i.parentID.IsValid() {
		parentInfo = fmt.Sprintf(", parent=%s", i.parentID.String())
	}
	
	if i.HasSubState() {
		return fmt.Sprintf("Instance{id=%s%s, workflow=%s, state=%s.%s, status=%s, version=%s}",
			i.id.String(), parentInfo, i.workflowName, i.currentState, i.currentSubState.ID(), i.status, i.version.String())
	}
	return fmt.Sprintf("Instance{id=%s%s, workflow=%s, state=%s, status=%s, version=%s}",
		i.id.String(), parentInfo, i.workflowName, i.currentState, i.status, i.version.String())
}
