package workflow

import (
	"fmt"
)

// Event represents a transition event in a workflow.
// It's an entity that records the history of state changes.
type Event struct {
	Name         string         `json:"name"`
	Sources      []State        `json:"sources"`
	Destination  State          `json:"destination"`
	Validators   []string       `json:"validators"`
	RequiredData []string       `json:"required_data,omitempty"`
	Guards       []GuardConfig  `json:"guards,omitempty"`
	Actions      []ActionConfig `json:"actions,omitempty"`
}

// NewEvent creates a new Event.
func NewEvent(name string, sources []State, destination State) (Event, error) {
	if name == "" {
		return Event{}, fmt.Errorf("event name cannot be empty")
	}

	if len(sources) == 0 {
		return Event{}, fmt.Errorf("event must have at least one source state")
	}

	if destination.ID == "" {
		return Event{}, fmt.Errorf("destination state cannot be empty")
	}

	// Create defensive copy of sources
	sourcesCopy := make([]State, len(sources))
	copy(sourcesCopy, sources)

	return Event{
		Name:        name,
		Sources:     sourcesCopy,
		Destination: destination,
		Validators:  []string{},
	}, nil
}

// GetName returns the event name.
func (e Event) GetName() string {
	return e.Name
}

// GetSources returns a copy of the source states.
func (e Event) GetSources() []State {
	sourcesCopy := make([]State, len(e.Sources))
	copy(sourcesCopy, e.Sources)
	return sourcesCopy
}

// GetDestination returns the destination state.
func (e Event) GetDestination() State {
	return e.Destination
}

// GetValidators returns a copy of the validator function names.
func (e Event) GetValidators() []string {
	if e.Validators == nil {
		return []string{}
	}
	validatorsCopy := make([]string, len(e.Validators))
	copy(validatorsCopy, e.Validators)
	return validatorsCopy
}

// WithValidators returns a new Event with the given validators.
func (e Event) WithValidators(validators []string) Event {
	if validators == nil {
		e.Validators = []string{}
	} else {
		e.Validators = make([]string, len(validators))
		copy(e.Validators, validators)
	}
	return e
}

// WithRequiredData returns a new Event with the given required data fields.
func (e Event) WithRequiredData(fields []string) Event {
	if fields == nil {
		e.RequiredData = []string{}
	} else {
		e.RequiredData = make([]string, len(fields))
		copy(e.RequiredData, fields)
	}
	return e
}

// GetRequiredData returns a copy of the required data field names.
func (e Event) GetRequiredData() []string {
	if e.RequiredData == nil {
		return []string{}
	}
	fieldsCopy := make([]string, len(e.RequiredData))
	copy(fieldsCopy, e.RequiredData)
	return fieldsCopy
}

// WithGuards returns a new Event with the given guards.
func (e Event) WithGuards(guards []GuardConfig) Event {
	if guards == nil {
		e.Guards = []GuardConfig{}
	} else {
		e.Guards = make([]GuardConfig, len(guards))
		copy(e.Guards, guards)
	}
	return e
}

// WithActions returns a new Event with the given actions.
func (e Event) WithActions(actions []ActionConfig) Event {
	if actions == nil {
		e.Actions = []ActionConfig{}
	} else {
		e.Actions = make([]ActionConfig, len(actions))
		copy(e.Actions, actions)
	}
	return e
}

// GetGuards returns a copy of the guard configurations.
func (e Event) GetGuards() []GuardConfig {
	if e.Guards == nil {
		return []GuardConfig{}
	}
	guardsCopy := make([]GuardConfig, len(e.Guards))
	copy(guardsCopy, e.Guards)
	return guardsCopy
}

// GetActions returns a copy of the action configurations.
func (e Event) GetActions() []ActionConfig {
	if e.Actions == nil {
		return []ActionConfig{}
	}
	actionsCopy := make([]ActionConfig, len(e.Actions))
	copy(actionsCopy, e.Actions)
	return actionsCopy
}

// CanTransitionFrom checks if this event can transition from the given state.
func (e Event) CanTransitionFrom(state State) bool {
	for _, source := range e.Sources {
		if source.Equals(state) {
			return true
		}
	}
	return false
}

// Validate validates the event.
func (e Event) Validate() error {
	if e.Name == "" {
		return fmt.Errorf("event name cannot be empty")
	}

	if len(e.Sources) == 0 {
		return fmt.Errorf("event must have at least one source state")
	}

	for i, source := range e.Sources {
		if err := source.Validate(); err != nil {
			return fmt.Errorf("invalid source state at index %d: %w", i, err)
		}
	}

	if err := e.Destination.Validate(); err != nil {
		return fmt.Errorf("invalid destination state: %w", err)
	}

	return nil
}

// String returns a string representation of the event.
func (e Event) String() string {
	sourceIDs := make([]string, len(e.Sources))
	for i, source := range e.Sources {
		sourceIDs[i] = source.GetID()
	}
	return fmt.Sprintf("Event{name=%s, sources=%v, destination=%s}", e.Name, sourceIDs, e.Destination.GetID())
}
