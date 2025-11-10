package instance

import (
	"fmt"
)

// TransitionMetadata represents metadata associated with a state transition (R23).
// It includes reason, feedback, and additional structured metadata.
type TransitionMetadata struct {
	reason   string
	feedback string
	metadata map[string]interface{}
}

// NewTransitionMetadata creates a new TransitionMetadata.
func NewTransitionMetadata(reason, feedback string, metadata map[string]interface{}) TransitionMetadata {
	// Create a defensive copy of metadata
	metadataCopy := make(map[string]interface{})
	if metadata != nil {
		for k, v := range metadata {
			metadataCopy[k] = v
		}
	}

	return TransitionMetadata{
		reason:   reason,
		feedback: feedback,
		metadata: metadataCopy,
	}
}

// NewTransitionMetadataWithReason creates metadata with just a reason.
func NewTransitionMetadataWithReason(reason string) TransitionMetadata {
	return NewTransitionMetadata(reason, "", nil)
}

// EmptyTransitionMetadata creates empty transition metadata.
func EmptyTransitionMetadata() TransitionMetadata {
	return NewTransitionMetadata("", "", nil)
}

// Reason returns the transition reason.
func (tm TransitionMetadata) Reason() string {
	return tm.reason
}

// Feedback returns the transition feedback.
func (tm TransitionMetadata) Feedback() string {
	return tm.feedback
}

// Metadata returns a copy of the metadata map.
func (tm TransitionMetadata) Metadata() map[string]interface{} {
	metadataCopy := make(map[string]interface{})
	for k, v := range tm.metadata {
		metadataCopy[k] = v
	}
	return metadataCopy
}

// GetMetadata retrieves a metadata value by key.
func (tm TransitionMetadata) GetMetadata(key string) (interface{}, bool) {
	val, ok := tm.metadata[key]
	return val, ok
}

// WithReason returns a new TransitionMetadata with the reason set.
func (tm TransitionMetadata) WithReason(reason string) TransitionMetadata {
	return TransitionMetadata{
		reason:   reason,
		feedback: tm.feedback,
		metadata: tm.Metadata(),
	}
}

// WithFeedback returns a new TransitionMetadata with the feedback set.
func (tm TransitionMetadata) WithFeedback(feedback string) TransitionMetadata {
	return TransitionMetadata{
		reason:   tm.reason,
		feedback: feedback,
		metadata: tm.Metadata(),
	}
}

// WithMetadata returns a new TransitionMetadata with the metadata set.
func (tm TransitionMetadata) WithMetadata(key string, value interface{}) TransitionMetadata {
	newMetadata := tm.Metadata()
	newMetadata[key] = value

	return TransitionMetadata{
		reason:   tm.reason,
		feedback: tm.feedback,
		metadata: newMetadata,
	}
}

// IsEmpty returns true if all fields are empty.
func (tm TransitionMetadata) IsEmpty() bool {
	return tm.reason == "" && tm.feedback == "" && len(tm.metadata) == 0
}

// Validate validates the transition metadata against a schema.
func (tm TransitionMetadata) Validate(schema *MetadataSchema) error {
	if schema == nil {
		return nil // No schema to validate against
	}

	return schema.Validate(tm)
}

// String returns a string representation of the metadata.
func (tm TransitionMetadata) String() string {
	return fmt.Sprintf("TransitionMetadata{reason=%s, feedback=%s, metadata_count=%d}",
		tm.reason, tm.feedback, len(tm.metadata))
}

// MetadataSchema defines validation rules for transition metadata (R23).
type MetadataSchema struct {
	required   []string                     // Required metadata keys
	optional   []string                     // Optional metadata keys
	types      map[string]string            // Expected types for keys (e.g., "string", "int", "bool")
	validators map[string]MetadataValidator // Custom validators for specific keys
}

// MetadataValidator is a function that validates a metadata value.
type MetadataValidator func(value interface{}) error

// NewMetadataSchema creates a new MetadataSchema.
func NewMetadataSchema() *MetadataSchema {
	return &MetadataSchema{
		required:   []string{},
		optional:   []string{},
		types:      make(map[string]string),
		validators: make(map[string]MetadataValidator),
	}
}

// WithRequired adds required fields to the schema.
func (ms *MetadataSchema) WithRequired(keys ...string) *MetadataSchema {
	ms.required = append(ms.required, keys...)
	return ms
}

// WithOptional adds optional fields to the schema.
func (ms *MetadataSchema) WithOptional(keys ...string) *MetadataSchema {
	ms.optional = append(ms.optional, keys...)
	return ms
}

// WithType specifies the expected type for a key.
func (ms *MetadataSchema) WithType(key, typeName string) *MetadataSchema {
	ms.types[key] = typeName
	return ms
}

// WithValidator adds a custom validator for a key.
func (ms *MetadataSchema) WithValidator(key string, validator MetadataValidator) *MetadataSchema {
	ms.validators[key] = validator
	return ms
}

// Validate validates transition metadata against this schema.
func (ms *MetadataSchema) Validate(tm TransitionMetadata) error {
	// Check required fields
	for _, key := range ms.required {
		if _, ok := tm.metadata[key]; !ok {
			return fmt.Errorf("required metadata field missing: %s", key)
		}
	}

	// Validate types
	for key, expectedType := range ms.types {
		value, ok := tm.metadata[key]
		if !ok {
			continue // Skip if not present (checked by required validation)
		}

		if err := ms.validateType(key, value, expectedType); err != nil {
			return err
		}
	}

	// Run custom validators
	for key, validator := range ms.validators {
		value, ok := tm.metadata[key]
		if !ok {
			continue // Skip if not present
		}

		if err := validator(value); err != nil {
			return fmt.Errorf("validation failed for metadata field %s: %w", key, err)
		}
	}

	return nil
}

// validateType checks if a value matches the expected type.
func (ms *MetadataSchema) validateType(key string, value interface{}, expectedType string) error {
	switch expectedType {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("metadata field %s must be a string, got %T", key, value)
		}
	case "int":
		switch value.(type) {
		case int, int64, float64: // JSON unmarshaling uses float64 for numbers
			// OK
		default:
			return fmt.Errorf("metadata field %s must be an int, got %T", key, value)
		}
	case "bool":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("metadata field %s must be a bool, got %T", key, value)
		}
	case "object":
		if _, ok := value.(map[string]interface{}); !ok {
			return fmt.Errorf("metadata field %s must be an object, got %T", key, value)
		}
	case "array":
		if _, ok := value.([]interface{}); !ok {
			return fmt.Errorf("metadata field %s must be an array, got %T", key, value)
		}
	default:
		return fmt.Errorf("unknown type constraint: %s", expectedType)
	}

	return nil
}
