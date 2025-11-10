package workflow

import (
	"fmt"

	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/shared"
)

// Workflow represents a workflow definition with states and events.
// It's the aggregate root for the workflow domain.
type Workflow struct {
	id           shared.ID
	name         string
	version      Version
	description  string
	initialState State
	states       map[string]State // keyed by state ID
	events       map[string]Event // keyed by event name
	createdAt    shared.Timestamp
	updatedAt    shared.Timestamp
}

// NewWorkflow creates a new Workflow with the given name and initial state.
func NewWorkflow(name string, initialState State) (*Workflow, error) {
	if name == "" {
		return nil, InvalidWorkflowError("name cannot be empty")
	}

	if err := initialState.Validate(); err != nil {
		return nil, InvalidWorkflowError(fmt.Sprintf("invalid initial state: %v", err))
	}

	now := shared.Now()
	version, _ := NewVersion(1, 0, 0)

	w := &Workflow{
		id:           shared.NewID(),
		name:         name,
		version:      version,
		initialState: initialState,
		states:       make(map[string]State),
		events:       make(map[string]Event),
		createdAt:    now,
		updatedAt:    now,
	}

	// Add initial state to states map
	w.states[initialState.ID()] = initialState

	return w, nil
}

// ID returns the workflow ID.
func (w *Workflow) ID() shared.ID {
	return w.id
}

// Name returns the workflow name.
func (w *Workflow) Name() string {
	return w.name
}

// Version returns the workflow version.
func (w *Workflow) Version() Version {
	return w.version
}

// Description returns the workflow description.
func (w *Workflow) Description() string {
	return w.description
}

// InitialState returns the initial state.
func (w *Workflow) InitialState() State {
	return w.initialState
}

// CreatedAt returns the creation timestamp.
func (w *Workflow) CreatedAt() shared.Timestamp {
	return w.createdAt
}

// UpdatedAt returns the last update timestamp.
func (w *Workflow) UpdatedAt() shared.Timestamp {
	return w.updatedAt
}

// SetDescription sets the workflow description.
func (w *Workflow) SetDescription(description string) {
	w.description = description
	w.updatedAt = shared.Now()
}

// States returns a copy of all states in the workflow.
func (w *Workflow) States() []State {
	states := make([]State, 0, len(w.states))
	for _, state := range w.states {
		states = append(states, state)
	}
	return states
}

// Events returns a copy of all events in the workflow.
func (w *Workflow) Events() []Event {
	events := make([]Event, 0, len(w.events))
	for _, event := range w.events {
		events = append(events, event)
	}
	return events
}

// AddState adds a new state to the workflow.
func (w *Workflow) AddState(state State) error {
	if err := state.Validate(); err != nil {
		return InvalidWorkflowError(fmt.Sprintf("invalid state: %v", err))
	}

	if _, exists := w.states[state.ID()]; exists {
		return StateAlreadyExistsError(state.ID())
	}

	w.states[state.ID()] = state
	w.updatedAt = shared.Now()

	return nil
}

// GetState retrieves a state by ID.
func (w *Workflow) GetState(stateID string) (State, error) {
	state, exists := w.states[stateID]
	if !exists {
		return State{}, StateNotFoundError(stateID)
	}
	return state, nil
}

// HasState checks if a state exists in the workflow.
func (w *Workflow) HasState(stateID string) bool {
	_, exists := w.states[stateID]
	return exists
}

// AddEvent adds a new event to the workflow.
func (w *Workflow) AddEvent(event Event) error {
	if err := event.Validate(); err != nil {
		return InvalidWorkflowError(fmt.Sprintf("invalid event: %v", err))
	}

	// Verify all source states exist
	for _, source := range event.Sources() {
		if !w.HasState(source.ID()) {
			return StateNotFoundError(source.ID())
		}
	}

	// Verify destination state exists
	if !w.HasState(event.Destination().ID()) {
		return StateNotFoundError(event.Destination().ID())
	}

	if _, exists := w.events[event.Name()]; exists {
		return EventAlreadyExistsError(event.Name())
	}

	w.events[event.Name()] = event
	w.updatedAt = shared.Now()

	return nil
}

// GetEvent retrieves an event by name.
func (w *Workflow) GetEvent(eventName string) (Event, error) {
	event, exists := w.events[eventName]
	if !exists {
		return Event{}, EventNotFoundError(eventName)
	}
	return event, nil
}

// FindEvent is an alias for GetEvent (for backwards compatibility).
func (w *Workflow) FindEvent(eventName string) (Event, error) {
	return w.GetEvent(eventName)
}

// HasEvent checks if an event exists in the workflow.
func (w *Workflow) HasEvent(eventName string) bool {
	_, exists := w.events[eventName]
	return exists
}

// CanTransition checks if a transition from the given state using the given event is valid.
func (w *Workflow) CanTransition(from State, event Event) bool {
	// Check if the state exists in the workflow
	if !w.HasState(from.ID()) {
		return false
	}

	// Check if the event exists in the workflow
	if !w.HasEvent(event.Name()) {
		return false
	}

	// Check if the event can transition from the given state
	return event.CanTransitionFrom(from)
}

// ValidateTransition validates a transition from the given state using the given event.
func (w *Workflow) ValidateTransition(from State, eventName string) error {
	event, err := w.GetEvent(eventName)
	if err != nil {
		return err
	}

	if !w.CanTransition(from, event) {
		return InvalidTransitionError(from, event)
	}

	return nil
}

// IncrementVersion increments the workflow version.
func (w *Workflow) IncrementVersion(versionType string) error {
	switch versionType {
	case "major":
		w.version = w.version.IncrementMajor()
	case "minor":
		w.version = w.version.IncrementMinor()
	case "patch":
		w.version = w.version.IncrementPatch()
	default:
		return InvalidWorkflowError(fmt.Sprintf("invalid version type: %s", versionType))
	}

	w.updatedAt = shared.Now()
	return nil
}

// Validate validates the entire workflow.
func (w *Workflow) Validate() error {
	if w.name == "" {
		return InvalidWorkflowError("name cannot be empty")
	}

	if err := w.initialState.Validate(); err != nil {
		return InvalidWorkflowError(fmt.Sprintf("invalid initial state: %v", err))
	}

	if !w.HasState(w.initialState.ID()) {
		return InvalidWorkflowError("initial state not found in workflow states")
	}

	// Validate all states
	for _, state := range w.states {
		if err := state.Validate(); err != nil {
			return InvalidWorkflowError(fmt.Sprintf("invalid state %s: %v", state.ID(), err))
		}
	}

	// Validate all events
	for _, event := range w.events {
		if err := event.Validate(); err != nil {
			return InvalidWorkflowError(fmt.Sprintf("invalid event %s: %v", event.Name(), err))
		}

		// Verify source and destination states exist
		for _, source := range event.Sources() {
			if !w.HasState(source.ID()) {
				return InvalidWorkflowError(fmt.Sprintf("event %s references non-existent source state: %s", event.Name(), source.ID()))
			}
		}

		if !w.HasState(event.Destination().ID()) {
			return InvalidWorkflowError(fmt.Sprintf("event %s references non-existent destination state: %s", event.Name(), event.Destination().ID()))
		}
	}

	return nil
}

// Clone creates a deep copy of the workflow with a new ID and version.
func (w *Workflow) Clone(newName string) *Workflow {
	now := shared.Now()
	version, _ := NewVersion(1, 0, 0)

	clone := &Workflow{
		id:           shared.NewID(),
		name:         newName,
		version:      version,
		description:  w.description,
		initialState: w.initialState,
		states:       make(map[string]State),
		events:       make(map[string]Event),
		createdAt:    now,
		updatedAt:    now,
	}

	// Copy states
	for id, state := range w.states {
		clone.states[id] = state
	}

	// Copy events
	for name, event := range w.events {
		clone.events[name] = event
	}

	return clone
}

// String returns a string representation of the workflow.
func (w *Workflow) String() string {
	return fmt.Sprintf("Workflow{id=%s, name=%s, version=%s, states=%d, events=%d}",
		w.id.String(), w.name, w.version.String(), len(w.states), len(w.events))
}
