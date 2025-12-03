package workflow

import (
	"fmt"

	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/shared"
)

// Workflow-specific error codes
var (
	// ErrNotFound is returned when a workflow is not found.
	ErrNotFound = shared.NotFoundError("workflow not found")

	// ErrInvalidWorkflow is returned when a workflow is invalid.
	ErrInvalidWorkflow = shared.InvalidInputError("invalid workflow")

	// ErrStateNotFound is returned when a state is not found in the workflow.
	ErrStateNotFound = shared.NotFoundError("state not found")

	// ErrEventNotFound is returned when an event is not found in the workflow.
	ErrEventNotFound = shared.NotFoundError("event not found")

	// ErrStateAlreadyExists is returned when attempting to add a duplicate state.
	ErrStateAlreadyExists = shared.AlreadyExistsError("state already exists")

	// ErrEventAlreadyExists is returned when attempting to add a duplicate event.
	ErrEventAlreadyExists = shared.AlreadyExistsError("event already exists")

	// ErrInvalidStateID is returned when a state ID is invalid.
	ErrInvalidStateID = shared.InvalidInputError("invalid state ID")

	// ErrEmptyStateName is returned when a state name is empty.
	ErrEmptyStateName = shared.InvalidInputError("state name cannot be empty")

	// ErrInvalidTransition is returned when an invalid transition is attempted.
	ErrInvalidTransition = shared.InvalidStateError("invalid transition")

	// ErrCyclicDependency is returned when a cyclic dependency is detected.
	ErrCyclicDependency = shared.InvalidInputError("cyclic dependency detected")

	// ErrNoInitialState is returned when a workflow has no initial state.
	ErrNoInitialState = shared.InvalidInputError("workflow must have an initial state")

	// ErrInvalidInitialState is returned when the initial state is invalid.
	ErrInvalidInitialState = shared.InvalidInputError("invalid initial state")
)

// StateNotFoundError creates a new state not found error with context.
func StateNotFoundError(stateID string) *shared.DomainError {
	return shared.NotFoundError(fmt.Sprintf("state not found: %s", stateID)).
		WithContext("state_id", stateID)
}

// EventNotFoundError creates a new event not found error with context.
func EventNotFoundError(eventName string) *shared.DomainError {
	return shared.NotFoundError(fmt.Sprintf("event not found: %s", eventName)).
		WithContext("event_name", eventName)
}

// StateAlreadyExistsError creates a new state already exists error with context.
func StateAlreadyExistsError(stateID string) *shared.DomainError {
	return shared.AlreadyExistsError(fmt.Sprintf("state already exists: %s", stateID)).
		WithContext("state_id", stateID)
}

// EventAlreadyExistsError creates a new event already exists error with context.
func EventAlreadyExistsError(eventName string) *shared.DomainError {
	return shared.AlreadyExistsError(fmt.Sprintf("event already exists: %s", eventName)).
		WithContext("event_name", eventName)
}

// InvalidTransitionError creates a new invalid transition error with context.
func InvalidTransitionError(from State, event Event) *shared.DomainError {
	return shared.InvalidStateError(
		fmt.Sprintf("invalid transition from state '%s' with event '%s'", from.ID, event.Name),
	).WithContext("from_state", from.ID).
		WithContext("event", event.Name)
}

// InvalidWorkflowError creates a new invalid workflow error with context.
func InvalidWorkflowError(message string) *shared.DomainError {
	return shared.InvalidInputError(fmt.Sprintf("invalid workflow: %s", message))
}

// WorkflowNotFoundError creates a new workflow not found error with context.
func WorkflowNotFoundError(workflowID string) *shared.DomainError {
	return shared.NotFoundError(fmt.Sprintf("workflow not found: %s", workflowID)).
		WithContext("workflow_id", workflowID)
}
