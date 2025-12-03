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
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Timeout     time.Duration `json:"timeout"`
	OnTimeout   string        `json:"on_timeout"`
	IsFinal     bool          `json:"is_final"`
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
		ID:   id,
		Name: name,
	}, nil
}

// ID returns the state ID.
func (s State) GetID() string {
	return s.ID
}

// Name returns the state name.
func (s State) GetName() string {
	return s.Name
}

// Description returns the state description.
func (s State) GetDescription() string {
	return s.Description
}

// Timeout returns the state timeout duration.
func (s State) GetTimeout() time.Duration {
	return s.Timeout
}

// OnTimeout returns the event to trigger on timeout.
func (s State) GetOnTimeout() string {
	return s.OnTimeout
}

// IsFinal returns true if this is a final state.
func (s State) GetIsFinal() bool {
	return s.IsFinal
}

// WithDescription returns a new State with the given description.
func (s State) WithDescription(description string) State {
	s.Description = description
	return s
}

// WithTimeout returns a new State with the given timeout and timeout event.
func (s State) WithTimeout(timeout time.Duration, onTimeoutEvent string) State {
	s.Timeout = timeout
	s.OnTimeout = onTimeoutEvent
	return s
}

// AsFinal returns a new State marked as final.
func (s State) AsFinal() State {
	s.IsFinal = true
	return s
}

// Validate validates the state.
func (s State) Validate() error {
	if s.ID == "" {
		return fmt.Errorf("state ID cannot be empty")
	}

	if !stateIDPattern.MatchString(s.ID) {
		return fmt.Errorf("invalid state ID format: %s", s.ID)
	}

	if s.Name == "" {
		return fmt.Errorf("state name cannot be empty")
	}

	// If timeout is set, onTimeout event must be set
	if s.Timeout > 0 && s.OnTimeout == "" {
		return fmt.Errorf("onTimeout event must be set when timeout is configured")
	}

	return nil
}

// Equals compares two states for equality based on ID.
func (s State) Equals(other State) bool {
	return s.ID == other.ID
}

// String returns a string representation of the state.
func (s State) String() string {
	return fmt.Sprintf("State{id=%s, name=%s, isFinal=%v}", s.ID, s.Name, s.IsFinal)
}
