package yaml

import (
	"fmt"
	"strings"
)

// ParseError represents an error that occurred during YAML parsing.
type ParseError struct {
	Line    int
	Column  int
	Field   string
	Message string
}

func (e ParseError) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("line %d, column %d: %s: %s", e.Line, e.Column, e.Field, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationError represents a semantic validation error.
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationErrors represents a collection of validation errors.
type ValidationErrors struct {
	Errors []ValidationError
}

func (e ValidationErrors) Error() string {
	if len(e.Errors) == 0 {
		return "no validation errors"
	}

	var sb strings.Builder
	sb.WriteString("validation failed with ")
	sb.WriteString(fmt.Sprintf("%d error(s):\n", len(e.Errors)))

	for i, err := range e.Errors {
		sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, err.Error()))
	}

	return sb.String()
}

// HasErrors returns true if there are validation errors.
func (e *ValidationErrors) HasErrors() bool {
	return len(e.Errors) > 0
}

// Add adds a validation error.
func (e *ValidationErrors) Add(field, message string) {
	e.Errors = append(e.Errors, ValidationError{
		Field:   field,
		Message: message,
	})
}

// InvalidYAMLError represents a YAML syntax error.
type InvalidYAMLError struct {
	Cause error
}

func (e InvalidYAMLError) Error() string {
	return fmt.Sprintf("invalid YAML syntax: %v", e.Cause)
}

func (e InvalidYAMLError) Unwrap() error {
	return e.Cause
}

// DuplicateStateError represents a duplicate state ID error.
type DuplicateStateError struct {
	StateID string
}

func (e DuplicateStateError) Error() string {
	return fmt.Sprintf("duplicate state ID: %s", e.StateID)
}

// DuplicateEventError represents a duplicate event name error.
type DuplicateEventError struct {
	EventName string
}

func (e DuplicateEventError) Error() string {
	return fmt.Sprintf("duplicate event name: %s", e.EventName)
}

// InvalidDurationError represents an invalid duration format error.
type InvalidDurationError struct {
	Value string
	Field string
}

func (e InvalidDurationError) Error() string {
	return fmt.Sprintf("invalid duration format for %s: %s (expected format: 1h, 30m, 24h, etc.)", e.Field, e.Value)
}

// StateNotFoundError represents a reference to a non-existent state.
type StateNotFoundError struct {
	StateID    string
	ReferencedBy string
}

func (e StateNotFoundError) Error() string {
	return fmt.Sprintf("state '%s' referenced by %s does not exist", e.StateID, e.ReferencedBy)
}

// EventNotFoundError represents a reference to a non-existent event.
type EventNotFoundError struct {
	EventName    string
	ReferencedBy string
}

func (e EventNotFoundError) Error() string {
	return fmt.Sprintf("event '%s' referenced by %s does not exist", e.EventName, e.ReferencedBy)
}

// MissingInitialStateError represents a missing initial state error.
type MissingInitialStateError struct{}

func (e MissingInitialStateError) Error() string {
	return "initial_state is required but not specified"
}

// InitialStateNotFoundError represents when initial state doesn't exist in states.
type InitialStateNotFoundError struct {
	StateID string
}

func (e InitialStateNotFoundError) Error() string {
	return fmt.Sprintf("initial_state '%s' is not defined in states list", e.StateID)
}
