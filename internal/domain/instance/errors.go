package instance

import (
	"fmt"
	"strings"

	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/shared"
)

// Instance-specific error codes
var (
	// ErrInstanceNotFound is returned when an instance is not found.
	ErrInstanceNotFound = shared.NotFoundError("instance not found")

	// ErrInvalidInstance is returned when an instance is invalid.
	ErrInvalidInstance = shared.InvalidInputError("invalid instance")

	// ErrInvalidTransition is returned when an invalid transition is attempted.
	ErrInvalidTransition = shared.InvalidStateError("invalid transition")

	// ErrInstanceAlreadyCompleted is returned when trying to modify a completed instance.
	ErrInstanceAlreadyCompleted = shared.InvalidStateError("instance already completed")

	// ErrInstanceNotActive is returned when trying to operate on an inactive instance.
	ErrInstanceNotActive = shared.InvalidStateError("instance not active")

	// ErrVersionMismatch is returned when optimistic locking fails.
	ErrVersionMismatch = shared.ConflictError("version mismatch - instance was modified by another process")

	// ErrInvalidSubState is returned when a sub-state is invalid.
	ErrInvalidSubState = shared.InvalidInputError("invalid sub-state")

	// ErrTransitionMetadataValidationFailed is returned when transition metadata validation fails.
	ErrTransitionMetadataValidationFailed = shared.InvalidInputError("transition metadata validation failed")
)

// InstanceNotFoundError creates a new instance not found error with context.
func InstanceNotFoundError(instanceID string) *shared.DomainError {
	return shared.NotFoundError(fmt.Sprintf("instance not found: %s", instanceID)).
		WithContext("instance_id", instanceID)
}

// VersionConflictError creates a version conflict error for optimistic locking.
func VersionConflictError(instanceID shared.ID, expected, actual Version) *shared.DomainError {
	return shared.ConflictError(
		fmt.Sprintf("version conflict for instance %s: expected v%s, got v%s",
			instanceID.String(), expected.String(), actual.String()),
	).WithContext("instance_id", instanceID.String()).
		WithContext("expected_version", expected.String()).
		WithContext("actual_version", actual.String())
}

// InvalidTransitionError creates a new invalid transition error with context.
func InvalidTransitionError(from, to, event string) *shared.DomainError {
	return shared.InvalidStateError(
		fmt.Sprintf("invalid transition from '%s' to '%s' with event '%s'", from, to, event),
	).WithContext("from_state", from).
		WithContext("to_state", to).
		WithContext("event", event)
}

// VersionMismatchError creates a new version mismatch error with context.
func VersionMismatchError(expected, actual int64) *shared.DomainError {
	return shared.ConflictError(
		fmt.Sprintf("version mismatch: expected v%d, got v%d", expected, actual),
	).WithContext("expected_version", expected).
		WithContext("actual_version", actual)
}

// InvalidInstanceError creates a new invalid instance error with context.
func InvalidInstanceError(message string) *shared.DomainError {
	return shared.InvalidInputError(fmt.Sprintf("invalid instance: %s", message))
}

// InstanceNotActiveError creates a new instance not active error.
func InstanceNotActiveError(instanceID shared.ID, status Status) *shared.DomainError {
	return shared.InvalidStateError(
		fmt.Sprintf("instance %s is not active (status: %s)", instanceID.String(), status),
	).WithContext("instance_id", instanceID.String()).
		WithContext("status", status.String())
}

// TransitionMetadataValidationError creates a new transition metadata validation error.
func TransitionMetadataValidationError(reason string) *shared.DomainError {
	return shared.InvalidInputError(
		fmt.Sprintf("transition metadata validation failed: %s", reason),
	)
}

// MissingRequiredDataError creates an error for missing required data fields.
func MissingRequiredDataError(eventName string, missingFields []string) *shared.DomainError {
	return shared.InvalidInputError(
		fmt.Sprintf("missing required data for event '%s': %s", eventName, strings.Join(missingFields, ", ")),
	).WithContext("event", eventName).
		WithContext("missing_fields", missingFields)
}
