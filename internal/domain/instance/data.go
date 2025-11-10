package instance

import (
	"encoding/json"
	"fmt"
)

// Data represents arbitrary data associated with an instance.
// It wraps a map[string]interface{} for type-safe access.
type Data struct {
	data map[string]interface{}
}

// NewData creates a new empty Data.
func NewData() Data {
	return Data{
		data: make(map[string]interface{}),
	}
}

// NewDataFromMap creates Data from an existing map.
// The map is copied to ensure immutability.
func NewDataFromMap(m map[string]interface{}) Data {
	if m == nil {
		return NewData()
	}

	data := make(map[string]interface{}, len(m))
	for k, v := range m {
		data[k] = v
	}

	return Data{data: data}
}

// Get retrieves a value by key.
// Returns nil if the key doesn't exist.
func (d Data) Get(key string) interface{} {
	return d.data[key]
}

// GetString retrieves a string value by key.
func (d Data) GetString(key string) (string, bool) {
	val, ok := d.data[key]
	if !ok {
		return "", false
	}
	str, ok := val.(string)
	return str, ok
}

// GetInt retrieves an int value by key.
func (d Data) GetInt(key string) (int, bool) {
	val, ok := d.data[key]
	if !ok {
		return 0, false
	}

	// Handle both int and float64 (JSON unmarshaling uses float64 for numbers)
	switch v := val.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	default:
		return 0, false
	}
}

// GetBool retrieves a bool value by key.
func (d Data) GetBool(key string) (bool, bool) {
	val, ok := d.data[key]
	if !ok {
		return false, false
	}
	b, ok := val.(bool)
	return b, ok
}

// Set creates a new Data with the key-value pair set.
// This maintains immutability by creating a new copy.
func (d Data) Set(key string, value interface{}) Data {
	newData := make(map[string]interface{}, len(d.data)+1)
	for k, v := range d.data {
		newData[k] = v
	}
	newData[key] = value

	return Data{data: newData}
}

// Delete creates a new Data with the key removed.
func (d Data) Delete(key string) Data {
	newData := make(map[string]interface{}, len(d.data))
	for k, v := range d.data {
		if k != key {
			newData[k] = v
		}
	}

	return Data{data: newData}
}

// Has checks if a key exists in the data.
func (d Data) Has(key string) bool {
	_, ok := d.data[key]
	return ok
}

// Keys returns all keys in the data.
func (d Data) Keys() []string {
	keys := make([]string, 0, len(d.data))
	for k := range d.data {
		keys = append(keys, k)
	}
	return keys
}

// ToMap returns a copy of the underlying map.
func (d Data) ToMap() map[string]interface{} {
	m := make(map[string]interface{}, len(d.data))
	for k, v := range d.data {
		m[k] = v
	}
	return m
}

// IsEmpty returns true if the data is empty.
func (d Data) IsEmpty() bool {
	return len(d.data) == 0
}

// Size returns the number of key-value pairs in the data.
func (d Data) Size() int {
	return len(d.data)
}

// MarshalJSON implements json.Marshaler interface.
func (d Data) MarshalJSON() ([]byte, error) {
	if d.data == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(d.data)
}

// UnmarshalJSON implements json.Unmarshaler interface.
func (d *Data) UnmarshalJSON(b []byte) error {
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	d.data = m
	return nil
}

// String returns a string representation of the data.
func (d Data) String() string {
	b, _ := json.Marshal(d.data)
	return string(b)
}

// Merge creates a new Data with values from both Data objects.
// Values from 'other' will override values from 'd' if keys overlap.
func (d Data) Merge(other Data) Data {
	newData := make(map[string]interface{}, len(d.data)+len(other.data))

	// Copy from d
	for k, v := range d.data {
		newData[k] = v
	}

	// Override/add from other
	for k, v := range other.data {
		newData[k] = v
	}

	return Data{data: newData}
}

// Validate validates the data against a schema.
// For now, this is a placeholder that can be extended with JSON schema validation.
func (d Data) Validate() error {
	// TODO: Implement schema validation when needed
	return nil
}

// DeepCopy creates a deep copy of the data.
// Note: This only works for JSON-serializable types.
func (d Data) DeepCopy() (Data, error) {
	b, err := json.Marshal(d.data)
	if err != nil {
		return Data{}, fmt.Errorf("failed to serialize data: %w", err)
	}

	var newData map[string]interface{}
	if err := json.Unmarshal(b, &newData); err != nil {
		return Data{}, fmt.Errorf("failed to deserialize data: %w", err)
	}

	return Data{data: newData}, nil
}
