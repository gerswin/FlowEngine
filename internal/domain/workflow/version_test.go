package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewVersion_Success(t *testing.T) {
	version, err := NewVersion(1, 2, 3)

	require.NoError(t, err)
	assert.Equal(t, 1, version.Major())
	assert.Equal(t, 2, version.Minor())
	assert.Equal(t, 3, version.Patch())
	assert.Equal(t, "1.2.3", version.String())
}

func TestNewVersion_NegativeValues(t *testing.T) {
	tests := []struct {
		name  string
		major int
		minor int
		patch int
	}{
		{"negative major", -1, 0, 0},
		{"negative minor", 0, -1, 0},
		{"negative patch", 0, 0, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewVersion(tt.major, tt.minor, tt.patch)
			assert.Error(t, err)
		})
	}
}

func TestParseVersion_Valid(t *testing.T) {
	version, err := ParseVersion("1.2.3")

	require.NoError(t, err)
	assert.Equal(t, 1, version.Major())
	assert.Equal(t, 2, version.Minor())
	assert.Equal(t, 3, version.Patch())
}

func TestParseVersion_Invalid(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"missing patch", "1.2"},
		{"extra component", "1.2.3.4"},
		{"non-numeric", "a.b.c"},
		{"partial non-numeric", "1.b.3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseVersion(tt.input)
			assert.Error(t, err)
		})
	}
}

func TestMustNewVersion_Success(t *testing.T) {
	assert.NotPanics(t, func() {
		version := MustNewVersion(1, 2, 3)
		assert.Equal(t, "1.2.3", version.String())
	})
}

func TestMustNewVersion_Panics(t *testing.T) {
	assert.Panics(t, func() {
		MustNewVersion(-1, 0, 0)
	})
}

func TestVersion_IncrementMajor(t *testing.T) {
	v := MustNewVersion(1, 2, 3)

	v2 := v.IncrementMajor()

	assert.Equal(t, 2, v2.Major())
	assert.Equal(t, 0, v2.Minor())
	assert.Equal(t, 0, v2.Patch())
	assert.Equal(t, "2.0.0", v2.String())

	// Original should be unchanged
	assert.Equal(t, "1.2.3", v.String())
}

func TestVersion_IncrementMinor(t *testing.T) {
	v := MustNewVersion(1, 2, 3)

	v2 := v.IncrementMinor()

	assert.Equal(t, 1, v2.Major())
	assert.Equal(t, 3, v2.Minor())
	assert.Equal(t, 0, v2.Patch())
	assert.Equal(t, "1.3.0", v2.String())
}

func TestVersion_IncrementPatch(t *testing.T) {
	v := MustNewVersion(1, 2, 3)

	v2 := v.IncrementPatch()

	assert.Equal(t, 1, v2.Major())
	assert.Equal(t, 2, v2.Minor())
	assert.Equal(t, 4, v2.Patch())
	assert.Equal(t, "1.2.4", v2.String())
}

func TestVersion_Equals(t *testing.T) {
	v1 := MustNewVersion(1, 2, 3)
	v2 := MustNewVersion(1, 2, 3)
	v3 := MustNewVersion(1, 2, 4)

	assert.True(t, v1.Equals(v2))
	assert.False(t, v1.Equals(v3))
}

func TestVersion_Comparison(t *testing.T) {
	v1 := MustNewVersion(1, 2, 3)
	v2 := MustNewVersion(1, 2, 4)
	v3 := MustNewVersion(1, 3, 0)
	v4 := MustNewVersion(2, 0, 0)

	t.Run("IsGreaterThan", func(t *testing.T) {
		assert.True(t, v2.IsGreaterThan(v1))
		assert.True(t, v3.IsGreaterThan(v2))
		assert.True(t, v4.IsGreaterThan(v3))
		assert.False(t, v1.IsGreaterThan(v2))
	})

	t.Run("IsLessThan", func(t *testing.T) {
		assert.True(t, v1.IsLessThan(v2))
		assert.True(t, v2.IsLessThan(v3))
		assert.True(t, v3.IsLessThan(v4))
		assert.False(t, v2.IsLessThan(v1))
	})

	t.Run("Compare", func(t *testing.T) {
		assert.Equal(t, -1, v1.Compare(v2))
		assert.Equal(t, 0, v1.Compare(v1))
		assert.Equal(t, 1, v2.Compare(v1))
	})
}

func TestVersion_IsZero(t *testing.T) {
	v0 := ZeroVersion()
	v1 := MustNewVersion(0, 0, 1)

	assert.True(t, v0.IsZero())
	assert.False(t, v1.IsZero())
}
