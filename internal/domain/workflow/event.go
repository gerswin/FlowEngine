package workflow

import (
	"fmt"
)

// Event represents a transition event in a workflow.
// It defines which states can transition to a destination state via this event.
type Event struct {
	name        string
	sources     []State
	destination State
	validators  []string // Optional validator function names
}

// NewEvent creates a new Event.
func NewEvent(name string, sources []State, destination State) (Event, error) {
	if name == "" {
		return Event{}, fmt.Errorf("event name cannot be empty")
	}

	if len(sources) == 0 {
		return Event{}, fmt.Errorf("event must have at least one source state")
	}

	if destination.id == "" {
		return Event{}, fmt.Errorf("destination state cannot be empty")
	}

	// Create defensive copy of sources
	sourcesCopy := make([]State, len(sources))
	copy(sourcesCopy, sources)

	return Event{
		name:        name,
		sources:     sourcesCopy,
		destination: destination,
		validators:  []string{},
	}, nil
}

// Name returns the event name.
func (e Event) Name() string {
	return e.name
}

// Sources returns a copy of the source states.
func (e Event) Sources() []State {
	sourcesCopy := make([]State, len(e.sources))
	copy(sourcesCopy, e.sources)
	return sourcesCopy
}

// Destination returns the destination state.
func (e Event) Destination() State {
	return e.destination
}

// Validators returns a copy of the validator function names.
func (e Event) Validators() []string {
	if e.validators == nil {
		return []string{}
	}
	validatorsCopy := make([]string, len(e.validators))
	copy(validatorsCopy, e.validators)
	return validatorsCopy
}

// WithValidators returns a new Event with the given validators.
func (e Event) WithValidators(validators []string) Event {
	if validators == nil {
		e.validators = []string{}
	} else {
		e.validators = make([]string, len(validators))
		copy(e.validators, validators)
	}
	return e
}

// CanTransitionFrom checks if this event can transition from the given state.
func (e Event) CanTransitionFrom(state State) bool {
	for _, source := range e.sources {
		if source.Equals(state) {
			return true
		}
	}
	return false
}

// Validate validates the event.
func (e Event) Validate() error {
	if e.name == "" {
		return fmt.Errorf("event name cannot be empty")
	}

	if len(e.sources) == 0 {
		return fmt.Errorf("event must have at least one source state")
	}

	for i, source := range e.sources {
		if err := source.Validate(); err != nil {
			return fmt.Errorf("invalid source state at index %d: %w", i, err)
		}
	}

	if err := e.destination.Validate(); err != nil {
		return fmt.Errorf("invalid destination state: %w", err)
	}

	return nil
}

// String returns a string representation of the event.
func (e Event) String() string {
	sourceIDs := make([]string, len(e.sources))
	for i, source := range e.sources {
		sourceIDs[i] = source.ID()
	}
	return fmt.Sprintf("Event{name=%s, sources=%v, destination=%s}", e.name, sourceIDs, e.destination.ID())
}
