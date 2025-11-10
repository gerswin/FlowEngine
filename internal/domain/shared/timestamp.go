package shared

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// Timestamp represents a point in time.
// It's an immutable value object that wraps time.Time with domain-specific behavior.
type Timestamp struct {
	value time.Time
}

// Now creates a Timestamp with the current time.
func Now() Timestamp {
	return Timestamp{value: time.Now().UTC()}
}

// From creates a Timestamp from a time.Time.
func From(t time.Time) Timestamp {
	return Timestamp{value: t.UTC()}
}

// FromUnix creates a Timestamp from Unix timestamp in seconds.
func FromUnix(sec int64) Timestamp {
	return Timestamp{value: time.Unix(sec, 0).UTC()}
}

// FromUnixMilli creates a Timestamp from Unix timestamp in milliseconds.
func FromUnixMilli(ms int64) Timestamp {
	return Timestamp{value: time.UnixMilli(ms).UTC()}
}

// ParseTimestamp parses a RFC3339 formatted string into a Timestamp.
func ParseTimestamp(s string) (Timestamp, error) {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return Timestamp{}, fmt.Errorf("invalid timestamp format: %w", err)
	}
	return Timestamp{value: t.UTC()}, nil
}

// Time returns the underlying time.Time value.
func (ts Timestamp) Time() time.Time {
	return ts.value
}

// Unix returns the Unix timestamp in seconds.
func (ts Timestamp) Unix() int64 {
	return ts.value.Unix()
}

// UnixMilli returns the Unix timestamp in milliseconds.
func (ts Timestamp) UnixMilli() int64 {
	return ts.value.UnixMilli()
}

// String returns the RFC3339 formatted string representation.
func (ts Timestamp) String() string {
	return ts.value.Format(time.RFC3339)
}

// IsZero checks if the timestamp is the zero value.
func (ts Timestamp) IsZero() bool {
	return ts.value.IsZero()
}

// Before reports whether the timestamp ts is before other.
func (ts Timestamp) Before(other Timestamp) bool {
	return ts.value.Before(other.value)
}

// After reports whether the timestamp ts is after other.
func (ts Timestamp) After(other Timestamp) bool {
	return ts.value.After(other.value)
}

// Equal reports whether ts and other represent the same time instant.
func (ts Timestamp) Equal(other Timestamp) bool {
	return ts.value.Equal(other.value)
}

// Add returns the timestamp ts+d.
func (ts Timestamp) Add(d time.Duration) Timestamp {
	return Timestamp{value: ts.value.Add(d)}
}

// Sub returns the duration ts-other.
func (ts Timestamp) Sub(other Timestamp) time.Duration {
	return ts.value.Sub(other.value)
}

// MarshalJSON implements json.Marshaler interface.
func (ts Timestamp) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, ts.value.Format(time.RFC3339))), nil
}

// UnmarshalJSON implements json.Unmarshaler interface.
func (ts *Timestamp) UnmarshalJSON(data []byte) error {
	// Remove quotes
	s := string(data)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}

	parsed, err := ParseTimestamp(s)
	if err != nil {
		return err
	}

	*ts = parsed
	return nil
}

// Value implements driver.Valuer interface for database serialization.
func (ts Timestamp) Value() (driver.Value, error) {
	return ts.value, nil
}

// Scan implements sql.Scanner interface for database deserialization.
func (ts *Timestamp) Scan(value interface{}) error {
	if value == nil {
		return fmt.Errorf("cannot scan nil into Timestamp")
	}

	switch v := value.(type) {
	case time.Time:
		*ts = From(v)
		return nil
	case string:
		parsed, err := ParseTimestamp(v)
		if err != nil {
			return err
		}
		*ts = parsed
		return nil
	default:
		return fmt.Errorf("cannot scan type %T into Timestamp", value)
	}
}

// ZeroTimestamp returns a zero value Timestamp.
func ZeroTimestamp() Timestamp {
	return Timestamp{value: time.Time{}}
}
