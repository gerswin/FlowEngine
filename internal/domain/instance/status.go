package instance

import "fmt"

// Status represents the execution status of a workflow instance.
type Status string

const (
	// StatusRunning indicates the instance is currently running.
	StatusRunning Status = "RUNNING"

	// StatusPaused indicates the instance is paused.
	StatusPaused Status = "PAUSED"

	// StatusCompleted indicates the instance has completed successfully.
	StatusCompleted Status = "COMPLETED"

	// StatusCanceled indicates the instance was canceled.
	StatusCanceled Status = "CANCELED"

	// StatusFailed indicates the instance failed.
	StatusFailed Status = "FAILED"
)

// IsActive returns true if the instance is in an active state (can be interacted with).
func (s Status) IsActive() bool {
	return s == StatusRunning || s == StatusPaused
}

// IsFinal returns true if the instance is in a final state (no more transitions possible).
func (s Status) IsFinal() bool {
	return s == StatusCompleted || s == StatusCanceled || s == StatusFailed
}

// String returns the string representation of the status.
func (s Status) String() string {
	return string(s)
}

// IsValid checks if the status is a valid value.
func (s Status) IsValid() bool {
	switch s {
	case StatusRunning, StatusPaused, StatusCompleted, StatusCanceled, StatusFailed:
		return true
	default:
		return false
	}
}

// ParseStatus parses a string into a Status.
func ParseStatus(s string) (Status, error) {
	status := Status(s)
	if !status.IsValid() {
		return "", fmt.Errorf("invalid status: %s", s)
	}
	return status, nil
}
