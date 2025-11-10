package workflow

import (
	"fmt"
	"regexp"
	"time"
)

var (
	// stateIDPattern defines the valid pattern for state IDs.
	// Must start with lowercase letter, followed by lowercase letters, digits, or underscores.
	stateIDPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
)

// State represents a state in a workflow.
// It's an immutable value object.
type State struct {
	id          string
	name        string
	description string
	timeout     time.Duration
	onTimeout   string // Event to trigger on timeout
	isFinal     bool
}

// NewState creates a new State with the given ID and name.
func NewState(id, name string) (State, error) {
	if id == "" {
		return State{}, fmt.Errorf("state ID cannot be empty")
	}

	if !stateIDPattern.MatchString(id) {
		return State{}, fmt.Errorf("invalid state ID format: %s (must match pattern: ^[a-z][a-z0-9_]*$)", id)
	}

	if name == "" {
		return State{}, fmt.Errorf("state name cannot be empty")
	}

	return State{
		id:   id,
		name: name,
	}, nil
}

// ID returns the state ID.
func (s State) ID() string {
	return s.id
}

// Name returns the state name.
func (s State) Name() string {
	return s.name
}

// Description returns the state description.
func (s State) Description() string {
	return s.description
}

// Timeout returns the state timeout duration.
func (s State) Timeout() time.Duration {
	return s.timeout
}

// OnTimeout returns the event to trigger on timeout.
func (s State) OnTimeout() string {
	return s.onTimeout
}

// IsFinal returns true if this is a final state.
func (s State) IsFinal() bool {
	return s.isFinal
}

// WithDescription returns a new State with the given description.
func (s State) WithDescription(description string) State {
	s.description = description
	return s
}

// WithTimeout returns a new State with the given timeout and timeout event.
func (s State) WithTimeout(timeout time.Duration, onTimeoutEvent string) State {
	s.timeout = timeout
	s.onTimeout = onTimeoutEvent
	return s
}

// AsFinal returns a new State marked as final.
func (s State) AsFinal() State {
	s.isFinal = true
	return s
}

// Validate validates the state.
func (s State) Validate() error {
	if s.id == "" {
		return fmt.Errorf("state ID cannot be empty")
	}

	if !stateIDPattern.MatchString(s.id) {
		return fmt.Errorf("invalid state ID format: %s", s.id)
	}

	if s.name == "" {
		return fmt.Errorf("state name cannot be empty")
	}

	// If timeout is set, onTimeout event must be set
	if s.timeout > 0 && s.onTimeout == "" {
		return fmt.Errorf("onTimeout event must be set when timeout is configured")
	}

	return nil
}

// Equals compares two states for equality based on ID.
func (s State) Equals(other State) bool {
	return s.id == other.id
}

// String returns a string representation of the state.
func (s State) String() string {
	return fmt.Sprintf("State{id=%s, name=%s, isFinal=%v}", s.id, s.name, s.isFinal)
}
