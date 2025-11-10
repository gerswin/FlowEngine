package instance

import "fmt"

// Version represents the version of an instance for optimistic locking.
// It's an immutable value object that increments with each update.
type Version struct {
	value int64
}

// NewVersion creates a new Version starting at 1.
func NewVersion() Version {
	return Version{value: 1}
}

// FromValue creates a Version from a specific value.
func FromValue(value int64) (Version, error) {
	if value < 1 {
		return Version{}, fmt.Errorf("version value must be >= 1, got: %d", value)
	}
	return Version{value: value}, nil
}

// Increment returns a new Version with the value incremented by 1.
func (v Version) Increment() Version {
	return Version{value: v.value + 1}
}

// Value returns the underlying version value.
func (v Version) Value() int64 {
	return v.value
}

// Equals compares two versions for equality.
func (v Version) Equals(other Version) bool {
	return v.value == other.value
}

// IsGreaterThan checks if this version is greater than the other.
func (v Version) IsGreaterThan(other Version) bool {
	return v.value > other.value
}

// IsLessThan checks if this version is less than the other.
func (v Version) IsLessThan(other Version) bool {
	return v.value < other.value
}

// String returns the string representation of the version.
func (v Version) String() string {
	return fmt.Sprintf("v%d", v.value)
}

// IsZero checks if the version is the zero value.
func (v Version) IsZero() bool {
	return v.value == 0
}

// ZeroVersion returns a zero version (invalid for actual use).
func ZeroVersion() Version {
	return Version{value: 0}
}
