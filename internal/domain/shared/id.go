package shared

import (
	"database/sql/driver"
	"fmt"

	"github.com/google/uuid"
)

// ID represents a unique identifier based on UUID.
// It's an immutable value object used across all domain entities.
type ID struct {
	value uuid.UUID
}

// NewID generates a new unique ID.
func NewID() ID {
	return ID{value: uuid.New()}
}

// ParseID parses a string representation of an ID.
// Returns an error if the string is not a valid UUID.
func ParseID(s string) (ID, error) {
	parsed, err := uuid.Parse(s)
	if err != nil {
		return ID{}, fmt.Errorf("invalid ID format: %w", err)
	}
	return ID{value: parsed}, nil
}

// MustParseID parses a string representation of an ID and panics on error.
// Should only be used in tests or when the ID is known to be valid.
func MustParseID(s string) ID {
	id, err := ParseID(s)
	if err != nil {
		panic(err)
	}
	return id
}

// String returns the string representation of the ID.
func (id ID) String() string {
	return id.value.String()
}

// IsValid checks if the ID is valid (not zero value).
func (id ID) IsValid() bool {
	return id.value != uuid.Nil
}

// IsZero checks if the ID is the zero value.
func (id ID) IsZero() bool {
	return id.value == uuid.Nil
}

// Equals compares two IDs for equality.
func (id ID) Equals(other ID) bool {
	return id.value == other.value
}

// MarshalJSON implements json.Marshaler interface.
func (id ID) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, id.value.String())), nil
}

// UnmarshalJSON implements json.Unmarshaler interface.
func (id *ID) UnmarshalJSON(data []byte) error {
	// Remove quotes
	s := string(data)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}

	parsed, err := ParseID(s)
	if err != nil {
		return err
	}

	*id = parsed
	return nil
}

// Value implements driver.Valuer interface for database serialization.
func (id ID) Value() (driver.Value, error) {
	return id.value.String(), nil
}

// Scan implements sql.Scanner interface for database deserialization.
func (id *ID) Scan(value interface{}) error {
	if value == nil {
		return fmt.Errorf("cannot scan nil into ID")
	}

	switch v := value.(type) {
	case string:
		parsed, err := ParseID(v)
		if err != nil {
			return err
		}
		*id = parsed
		return nil
	case []byte:
		parsed, err := ParseID(string(v))
		if err != nil {
			return err
		}
		*id = parsed
		return nil
	default:
		return fmt.Errorf("cannot scan type %T into ID", value)
	}
}

// NilID returns a zero value ID.
func NilID() ID {
	return ID{value: uuid.Nil}
}
