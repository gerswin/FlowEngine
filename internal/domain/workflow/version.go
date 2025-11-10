package workflow

import (
	"fmt"
)

// Version represents a semantic version for a workflow.
// It's an immutable value object.
type Version struct {
	major int
	minor int
	patch int
}

// NewVersion creates a new Version.
func NewVersion(major, minor, patch int) (Version, error) {
	if major < 0 {
		return Version{}, fmt.Errorf("major version cannot be negative")
	}
	if minor < 0 {
		return Version{}, fmt.Errorf("minor version cannot be negative")
	}
	if patch < 0 {
		return Version{}, fmt.Errorf("patch version cannot be negative")
	}

	return Version{
		major: major,
		minor: minor,
		patch: patch,
	}, nil
}

// MustNewVersion creates a new Version and panics on error.
// Should only be used in tests or when values are known to be valid.
func MustNewVersion(major, minor, patch int) Version {
	v, err := NewVersion(major, minor, patch)
	if err != nil {
		panic(err)
	}
	return v
}

// ParseVersion parses a semantic version string (e.g., "1.2.3").
func ParseVersion(s string) (Version, error) {
	var major, minor, patch int
	var extra string

	// Try to scan with an extra string to detect invalid format (e.g., "1.2.3.4")
	n, err := fmt.Sscanf(s, "%d.%d.%d%s", &major, &minor, &patch, &extra)

	// We expect exactly 3 items scanned, no more, no less
	if err != nil && n != 3 {
		return Version{}, fmt.Errorf("invalid version format: %s (expected: major.minor.patch)", s)
	}

	// If extra string was scanned, format is invalid
	if n == 4 {
		return Version{}, fmt.Errorf("invalid version format: %s (expected: major.minor.patch)", s)
	}

	return NewVersion(major, minor, patch)
}

// Major returns the major version number.
func (v Version) Major() int {
	return v.major
}

// Minor returns the minor version number.
func (v Version) Minor() int {
	return v.minor
}

// Patch returns the patch version number.
func (v Version) Patch() int {
	return v.patch
}

// String returns the string representation in semantic versioning format.
func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.major, v.minor, v.patch)
}

// IncrementMajor returns a new Version with the major version incremented.
// Minor and patch are reset to 0.
func (v Version) IncrementMajor() Version {
	return Version{
		major: v.major + 1,
		minor: 0,
		patch: 0,
	}
}

// IncrementMinor returns a new Version with the minor version incremented.
// Patch is reset to 0.
func (v Version) IncrementMinor() Version {
	return Version{
		major: v.major,
		minor: v.minor + 1,
		patch: 0,
	}
}

// IncrementPatch returns a new Version with the patch version incremented.
func (v Version) IncrementPatch() Version {
	return Version{
		major: v.major,
		minor: v.minor,
		patch: v.patch + 1,
	}
}

// Equals compares two versions for equality.
func (v Version) Equals(other Version) bool {
	return v.major == other.major &&
		v.minor == other.minor &&
		v.patch == other.patch
}

// IsGreaterThan checks if this version is greater than the other.
func (v Version) IsGreaterThan(other Version) bool {
	if v.major != other.major {
		return v.major > other.major
	}
	if v.minor != other.minor {
		return v.minor > other.minor
	}
	return v.patch > other.patch
}

// IsLessThan checks if this version is less than the other.
func (v Version) IsLessThan(other Version) bool {
	if v.major != other.major {
		return v.major < other.major
	}
	if v.minor != other.minor {
		return v.minor < other.minor
	}
	return v.patch < other.patch
}

// Compare compares two versions.
// Returns -1 if v < other, 0 if v == other, 1 if v > other.
func (v Version) Compare(other Version) int {
	if v.Equals(other) {
		return 0
	}
	if v.IsLessThan(other) {
		return -1
	}
	return 1
}

// IsZero checks if the version is the zero value (0.0.0).
func (v Version) IsZero() bool {
	return v.major == 0 && v.minor == 0 && v.patch == 0
}

// ZeroVersion returns a zero version (0.0.0).
func ZeroVersion() Version {
	return Version{major: 0, minor: 0, patch: 0}
}
