package instance

import (
	"fmt"
	"regexp"
)

var (
	// subStateIDPattern defines the valid pattern for sub-state IDs.
	// Same as workflow state IDs: lowercase letters, digits, and underscores.
	subStateIDPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
)

// SubState represents a sub-state within a workflow state (R17).
// Sub-states allow fine-grained tracking within a single workflow state.
// Example: "pending" state might have sub-states: "pending_review", "pending_manager", etc.
type SubState struct {
	id          string
	name        string
	description string
}

// RestoreSubState creates a SubState from ID without validation.
// Should only be used by persistence layer.
func RestoreSubState(id string) SubState {
	return SubState{
		id:   id,
		name: id, // Default name to ID if not persisted
	}
}

// NewSubState creates a new SubState with the given ID and name.
func NewSubState(id, name string) (SubState, error) {
	if id == "" {
		return SubState{}, fmt.Errorf("sub-state ID cannot be empty")
	}

	if !subStateIDPattern.MatchString(id) {
		return SubState{}, fmt.Errorf("invalid sub-state ID format: %s (must match pattern: ^[a-z][a-z0-9_]*$)", id)
	}

	if name == "" {
		return SubState{}, fmt.Errorf("sub-state name cannot be empty")
	}

	return SubState{
		id:   id,
		name: name,
	}, nil
}

// ID returns the sub-state ID.
func (s SubState) ID() string {
	return s.id
}

// Name returns the sub-state name.
func (s SubState) Name() string {
	return s.name
}

// Description returns the sub-state description.
func (s SubState) Description() string {
	return s.description
}

// WithDescription returns a new SubState with the given description.
func (s SubState) WithDescription(description string) SubState {
	s.description = description
	return s
}

// Validate validates the sub-state.
func (s SubState) Validate() error {
	if s.id == "" {
		return fmt.Errorf("sub-state ID cannot be empty")
	}

	if !subStateIDPattern.MatchString(s.id) {
		return fmt.Errorf("invalid sub-state ID format: %s", s.id)
	}

	if s.name == "" {
		return fmt.Errorf("sub-state name cannot be empty")
	}

	return nil
}

// Equals compares two sub-states for equality based on ID.
func (s SubState) Equals(other SubState) bool {
	return s.id == other.id
}

// String returns a string representation of the sub-state.
func (s SubState) String() string {
	return fmt.Sprintf("SubState{id=%s, name=%s}", s.id, s.name)
}

// IsZero checks if the sub-state is the zero value.
func (s SubState) IsZero() bool {
	return s.id == ""
}

// ZeroSubState returns a zero value sub-state.
func ZeroSubState() SubState {
	return SubState{}
}
