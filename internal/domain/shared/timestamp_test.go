package shared

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNow(t *testing.T) {
	before := time.Now().UTC()
	ts := Now()
	after := time.Now().UTC()

	assert.False(t, ts.IsZero())
	assert.True(t, ts.Time().Equal(before) || ts.Time().After(before))
	assert.True(t, ts.Time().Equal(after) || ts.Time().Before(after))
}

func TestFrom(t *testing.T) {
	timeValue := time.Date(2024, 1, 15, 10, 30, 0, 0, time.Local)
	ts := From(timeValue)

	// Should be converted to UTC
	assert.Equal(t, timeValue.UTC(), ts.Time())
}

func TestFromUnix(t *testing.T) {
	unixTime := int64(1705315800) // 2024-01-15 10:30:00 UTC

	ts := FromUnix(unixTime)

	assert.Equal(t, unixTime, ts.Unix())
}

func TestFromUnixMilli(t *testing.T) {
	unixMilli := int64(1705315800000) // 2024-01-15 10:30:00 UTC

	ts := FromUnixMilli(unixMilli)

	assert.Equal(t, unixMilli, ts.UnixMilli())
}

func TestParseTimestamp_ValidInput(t *testing.T) {
	validRFC3339 := "2024-01-15T10:30:00Z"

	ts, err := ParseTimestamp(validRFC3339)
	require.NoError(t, err)

	assert.False(t, ts.IsZero())
	assert.Equal(t, validRFC3339, ts.String())
}

func TestParseTimestamp_InvalidInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"invalid format", "not-a-timestamp"},
		{"wrong format", "2024-01-15 10:30:00"},
		{"partial timestamp", "2024-01-15"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, err := ParseTimestamp(tt.input)

			assert.Error(t, err)
			assert.True(t, ts.IsZero())
		})
	}
}

func TestTimestamp_Time(t *testing.T) {
	timeValue := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	ts := From(timeValue)

	assert.Equal(t, timeValue, ts.Time())
}

func TestTimestamp_Unix(t *testing.T) {
	unixTime := int64(1705315800)
	ts := FromUnix(unixTime)

	assert.Equal(t, unixTime, ts.Unix())
}

func TestTimestamp_UnixMilli(t *testing.T) {
	unixMilli := int64(1705315800000)
	ts := FromUnixMilli(unixMilli)

	assert.Equal(t, unixMilli, ts.UnixMilli())
}

func TestTimestamp_String(t *testing.T) {
	timeValue := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	ts := From(timeValue)

	expected := "2024-01-15T10:30:00Z"
	assert.Equal(t, expected, ts.String())
}

func TestTimestamp_IsZero(t *testing.T) {
	t.Run("new timestamp is not zero", func(t *testing.T) {
		ts := Now()
		assert.False(t, ts.IsZero())
	})

	t.Run("zero timestamp is zero", func(t *testing.T) {
		ts := ZeroTimestamp()
		assert.True(t, ts.IsZero())
	})

	t.Run("zero value timestamp is zero", func(t *testing.T) {
		var ts Timestamp
		assert.True(t, ts.IsZero())
	})
}

func TestTimestamp_Comparison(t *testing.T) {
	t1 := From(time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC))
	t2 := From(time.Date(2024, 1, 15, 11, 30, 0, 0, time.UTC))
	t3 := From(time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC))

	t.Run("Before", func(t *testing.T) {
		assert.True(t, t1.Before(t2))
		assert.False(t, t2.Before(t1))
		assert.False(t, t1.Before(t3))
	})

	t.Run("After", func(t *testing.T) {
		assert.True(t, t2.After(t1))
		assert.False(t, t1.After(t2))
		assert.False(t, t1.After(t3))
	})

	t.Run("Equal", func(t *testing.T) {
		assert.True(t, t1.Equal(t3))
		assert.False(t, t1.Equal(t2))
	})
}

func TestTimestamp_Add(t *testing.T) {
	ts := From(time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC))
	duration := 1 * time.Hour

	result := ts.Add(duration)

	expected := time.Date(2024, 1, 15, 11, 30, 0, 0, time.UTC)
	assert.Equal(t, expected, result.Time())
}

func TestTimestamp_Sub(t *testing.T) {
	t1 := From(time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC))
	t2 := From(time.Date(2024, 1, 15, 11, 30, 0, 0, time.UTC))

	duration := t2.Sub(t1)

	assert.Equal(t, 1*time.Hour, duration)
}

func TestTimestamp_MarshalJSON(t *testing.T) {
	ts := From(time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC))

	data, err := json.Marshal(ts)
	require.NoError(t, err)

	expected := `"2024-01-15T10:30:00Z"`
	assert.Equal(t, expected, string(data))
}

func TestTimestamp_UnmarshalJSON(t *testing.T) {
	t.Run("valid JSON", func(t *testing.T) {
		jsonData := `"2024-01-15T10:30:00Z"`

		var ts Timestamp
		err := json.Unmarshal([]byte(jsonData), &ts)
		require.NoError(t, err)

		expected := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
		assert.Equal(t, expected, ts.Time())
	})

	t.Run("invalid JSON", func(t *testing.T) {
		jsonData := `"not-a-timestamp"`

		var ts Timestamp
		err := json.Unmarshal([]byte(jsonData), &ts)
		assert.Error(t, err)
	})
}

func TestTimestamp_JSON_RoundTrip(t *testing.T) {
	original := From(time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC))

	// Marshal to JSON
	data, err := json.Marshal(original)
	require.NoError(t, err)

	// Unmarshal back
	var decoded Timestamp
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.True(t, original.Equal(decoded))
}

func TestTimestamp_Value_DatabaseDriver(t *testing.T) {
	ts := From(time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC))

	value, err := ts.Value()
	require.NoError(t, err)

	assert.IsType(t, time.Time{}, value)
	assert.Equal(t, ts.Time(), value)
}

func TestTimestamp_Scan_DatabaseDriver(t *testing.T) {
	t.Run("scan time.Time", func(t *testing.T) {
		timeValue := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

		var ts Timestamp
		err := ts.Scan(timeValue)
		require.NoError(t, err)

		assert.Equal(t, timeValue.UTC(), ts.Time())
	})

	t.Run("scan string", func(t *testing.T) {
		timeString := "2024-01-15T10:30:00Z"

		var ts Timestamp
		err := ts.Scan(timeString)
		require.NoError(t, err)

		assert.Equal(t, timeString, ts.String())
	})

	t.Run("scan nil", func(t *testing.T) {
		var ts Timestamp
		err := ts.Scan(nil)
		assert.Error(t, err)
	})

	t.Run("scan invalid type", func(t *testing.T) {
		var ts Timestamp
		err := ts.Scan(123)
		assert.Error(t, err)
	})

	t.Run("scan invalid string", func(t *testing.T) {
		var ts Timestamp
		err := ts.Scan("not-a-timestamp")
		assert.Error(t, err)
	})
}

func TestZeroTimestamp(t *testing.T) {
	ts := ZeroTimestamp()

	assert.True(t, ts.IsZero())
	assert.Equal(t, time.Time{}, ts.Time())
}

func TestTimestamp_Serialization(t *testing.T) {
	original := From(time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC))

	// Test JSON serialization
	jsonData, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded Timestamp
	err = json.Unmarshal(jsonData, &decoded)
	require.NoError(t, err)

	assert.True(t, original.Equal(decoded))
}
