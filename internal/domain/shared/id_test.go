package shared

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewID_GeneratesValidUUID(t *testing.T) {
	id := NewID()

	assert.True(t, id.IsValid())
	assert.False(t, id.IsZero())
	assert.NotEmpty(t, id.String())
}

func TestNewID_GeneratesUniqueIDs(t *testing.T) {
	id1 := NewID()
	id2 := NewID()

	assert.NotEqual(t, id1.String(), id2.String())
	assert.False(t, id1.Equals(id2))
}

func TestParseID_ValidInput(t *testing.T) {
	validUUID := "123e4567-e89b-12d3-a456-426614174000"

	id, err := ParseID(validUUID)

	require.NoError(t, err)
	assert.True(t, id.IsValid())
	assert.Equal(t, validUUID, id.String())
}

func TestParseID_InvalidInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"invalid format", "not-a-uuid"},
		{"partial uuid", "123e4567"},
		{"malformed uuid", "123e4567-e89b-12d3-a456"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := ParseID(tt.input)

			assert.Error(t, err)
			assert.False(t, id.IsValid())
		})
	}
}

func TestMustParseID_ValidInput(t *testing.T) {
	validUUID := "123e4567-e89b-12d3-a456-426614174000"

	assert.NotPanics(t, func() {
		id := MustParseID(validUUID)
		assert.True(t, id.IsValid())
	})
}

func TestMustParseID_InvalidInput_Panics(t *testing.T) {
	invalidUUID := "not-a-uuid"

	assert.Panics(t, func() {
		MustParseID(invalidUUID)
	})
}

func TestID_String(t *testing.T) {
	validUUID := "123e4567-e89b-12d3-a456-426614174000"
	id, err := ParseID(validUUID)
	require.NoError(t, err)

	assert.Equal(t, validUUID, id.String())
}

func TestID_IsValid(t *testing.T) {
	t.Run("valid ID", func(t *testing.T) {
		id := NewID()
		assert.True(t, id.IsValid())
	})

	t.Run("nil ID", func(t *testing.T) {
		id := NilID()
		assert.False(t, id.IsValid())
	})

	t.Run("zero value ID", func(t *testing.T) {
		var id ID
		assert.False(t, id.IsValid())
	})
}

func TestID_IsZero(t *testing.T) {
	t.Run("new ID is not zero", func(t *testing.T) {
		id := NewID()
		assert.False(t, id.IsZero())
	})

	t.Run("nil ID is zero", func(t *testing.T) {
		id := NilID()
		assert.True(t, id.IsZero())
	})

	t.Run("zero value ID is zero", func(t *testing.T) {
		var id ID
		assert.True(t, id.IsZero())
	})
}

func TestID_Equals(t *testing.T) {
	t.Run("same IDs are equal", func(t *testing.T) {
		id1 := NewID()
		id2 := id1

		assert.True(t, id1.Equals(id2))
	})

	t.Run("different IDs are not equal", func(t *testing.T) {
		id1 := NewID()
		id2 := NewID()

		assert.False(t, id1.Equals(id2))
	})

	t.Run("parsed same UUID are equal", func(t *testing.T) {
		validUUID := "123e4567-e89b-12d3-a456-426614174000"
		id1, _ := ParseID(validUUID)
		id2, _ := ParseID(validUUID)

		assert.True(t, id1.Equals(id2))
	})
}

func TestID_MarshalJSON(t *testing.T) {
	validUUID := "123e4567-e89b-12d3-a456-426614174000"
	id, err := ParseID(validUUID)
	require.NoError(t, err)

	data, err := json.Marshal(id)
	require.NoError(t, err)

	expected := `"` + validUUID + `"`
	assert.Equal(t, expected, string(data))
}

func TestID_UnmarshalJSON(t *testing.T) {
	t.Run("valid JSON", func(t *testing.T) {
		validUUID := "123e4567-e89b-12d3-a456-426614174000"
		jsonData := `"` + validUUID + `"`

		var id ID
		err := json.Unmarshal([]byte(jsonData), &id)
		require.NoError(t, err)

		assert.Equal(t, validUUID, id.String())
	})

	t.Run("invalid JSON", func(t *testing.T) {
		jsonData := `"not-a-uuid"`

		var id ID
		err := json.Unmarshal([]byte(jsonData), &id)
		assert.Error(t, err)
	})
}

func TestID_JSON_RoundTrip(t *testing.T) {
	original := NewID()

	// Marshal to JSON
	data, err := json.Marshal(original)
	require.NoError(t, err)

	// Unmarshal back
	var decoded ID
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.True(t, original.Equals(decoded))
}

func TestID_Value_DatabaseDriver(t *testing.T) {
	id := NewID()

	value, err := id.Value()
	require.NoError(t, err)

	assert.IsType(t, "", value)
	assert.Equal(t, id.String(), value)
}

func TestID_Scan_DatabaseDriver(t *testing.T) {
	t.Run("scan string", func(t *testing.T) {
		validUUID := "123e4567-e89b-12d3-a456-426614174000"

		var id ID
		err := id.Scan(validUUID)
		require.NoError(t, err)

		assert.Equal(t, validUUID, id.String())
	})

	t.Run("scan byte slice", func(t *testing.T) {
		validUUID := "123e4567-e89b-12d3-a456-426614174000"

		var id ID
		err := id.Scan([]byte(validUUID))
		require.NoError(t, err)

		assert.Equal(t, validUUID, id.String())
	})

	t.Run("scan nil", func(t *testing.T) {
		var id ID
		err := id.Scan(nil)
		assert.Error(t, err)
	})

	t.Run("scan invalid type", func(t *testing.T) {
		var id ID
		err := id.Scan(123)
		assert.Error(t, err)
	})

	t.Run("scan invalid UUID", func(t *testing.T) {
		var id ID
		err := id.Scan("not-a-uuid")
		assert.Error(t, err)
	})
}

func TestNilID(t *testing.T) {
	id := NilID()

	assert.True(t, id.IsZero())
	assert.False(t, id.IsValid())
	assert.Equal(t, uuid.Nil.String(), id.String())
}
