package instance

// Variables represents workflow variables associated with an instance.
// It's essentially the same as Data but used for a different semantic purpose.
type Variables struct {
	data Data
}

// NewVariables creates a new empty Variables.
func NewVariables() Variables {
	return Variables{
		data: NewData(),
	}
}

// NewVariablesFromMap creates Variables from an existing map.
func NewVariablesFromMap(m map[string]interface{}) Variables {
	return Variables{
		data: NewDataFromMap(m),
	}
}

// Get retrieves a variable value by key.
func (v Variables) Get(key string) interface{} {
	return v.data.Get(key)
}

// GetString retrieves a string variable by key.
func (v Variables) GetString(key string) (string, bool) {
	return v.data.GetString(key)
}

// GetInt retrieves an int variable by key.
func (v Variables) GetInt(key string) (int, bool) {
	return v.data.GetInt(key)
}

// GetBool retrieves a bool variable by key.
func (v Variables) GetBool(key string) (bool, bool) {
	return v.data.GetBool(key)
}

// Set creates a new Variables with the key-value pair set.
func (v Variables) Set(key string, value interface{}) Variables {
	return Variables{
		data: v.data.Set(key, value),
	}
}

// Delete creates a new Variables with the key removed.
func (v Variables) Delete(key string) Variables {
	return Variables{
		data: v.data.Delete(key),
	}
}

// Has checks if a variable exists.
func (v Variables) Has(key string) bool {
	return v.data.Has(key)
}

// Keys returns all variable keys.
func (v Variables) Keys() []string {
	return v.data.Keys()
}

// ToMap returns a copy of the underlying map.
func (v Variables) ToMap() map[string]interface{} {
	return v.data.ToMap()
}

// IsEmpty returns true if there are no variables.
func (v Variables) IsEmpty() bool {
	return v.data.IsEmpty()
}

// Size returns the number of variables.
func (v Variables) Size() int {
	return v.data.Size()
}

// MarshalJSON implements json.Marshaler interface.
func (v Variables) MarshalJSON() ([]byte, error) {
	return v.data.MarshalJSON()
}

// UnmarshalJSON implements json.Unmarshaler interface.
func (v *Variables) UnmarshalJSON(b []byte) error {
	return v.data.UnmarshalJSON(b)
}

// String returns a string representation of the variables.
func (v Variables) String() string {
	return v.data.String()
}

// Merge creates a new Variables with values from both Variables objects.
func (v Variables) Merge(other Variables) Variables {
	return Variables{
		data: v.data.Merge(other.data),
	}
}

// DeepCopy creates a deep copy of the variables.
func (v Variables) DeepCopy() (Variables, error) {
	copiedData, err := v.data.DeepCopy()
	if err != nil {
		return Variables{}, err
	}
	return Variables{data: copiedData}, nil
}
