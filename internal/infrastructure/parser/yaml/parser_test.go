package yaml

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParser_Parse_ValidSimpleWorkflow(t *testing.T) {
	yamlContent := `
version: "1.0"
workflow:
  name: "Simple Workflow"
  description: "A simple test workflow"
  initial_state: "draft"
  states:
    - id: draft
      name: Draft
      description: Initial state
    - id: review
      name: Review
      description: Under review
    - id: approved
      name: Approved
      description: Final approval
      final: true
  events:
    - name: submit
      from:
        - draft
      to: review
    - name: approve
      from:
        - review
      to: approved
`

	parser := NewParser()
	wf, err := parser.ParseBytes([]byte(yamlContent))

	require.NoError(t, err)
	assert.NotNil(t, wf)
	assert.Equal(t, "Simple Workflow", wf.Name())
	assert.Equal(t, "A simple test workflow", wf.Description())
	assert.Equal(t, "draft", wf.InitialState().ID)
	assert.Len(t, wf.States(), 3)
	assert.Len(t, wf.Events(), 2)
}

func TestParser_Parse_WithTimeout(t *testing.T) {
	yamlContent := `
version: "1.0"
workflow:
  name: "Workflow with Timeout"
  initial_state: "pending"
  states:
    - id: pending
      name: Pending
      timeout: 24h
      on_timeout: escalate
    - id: escalated
      name: Escalated
      final: true
  events:
    - name: escalate
      from:
        - pending
      to: escalated
`

	parser := NewParser()
	wf, err := parser.ParseBytes([]byte(yamlContent))

	require.NoError(t, err)
	assert.NotNil(t, wf)

	state, err := wf.GetState("pending")
	require.NoError(t, err)
	assert.Equal(t, "24h0m0s", state.Timeout.String())
	assert.Equal(t, "escalate", state.OnTimeout)
}

func TestParser_Parse_WithDaysTimeout(t *testing.T) {
	yamlContent := `
version: "1.0"
workflow:
  name: "Workflow with Days Timeout"
  initial_state: "pending"
  states:
    - id: pending
      name: Pending
      timeout: 5d
      on_timeout: expire
    - id: expired
      name: Expired
      final: true
  events:
    - name: expire
      from:
        - pending
      to: expired
`

	parser := NewParser()
	wf, err := parser.ParseBytes([]byte(yamlContent))

	require.NoError(t, err)
	assert.NotNil(t, wf)

	state, err := wf.GetState("pending")
	require.NoError(t, err)
	// 5 days = 120 hours
	assert.Equal(t, "120h0m0s", state.Timeout.String())
}

func TestParser_Parse_MissingVersion(t *testing.T) {
	yamlContent := `
workflow:
  name: "No Version"
  initial_state: "draft"
  states:
    - id: draft
      name: Draft
`

	parser := NewParser()
	_, err := parser.ParseBytes([]byte(yamlContent))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "version")
}

func TestParser_Parse_MissingInitialState(t *testing.T) {
	yamlContent := `
version: "1.0"
workflow:
  name: "No Initial State"
  states:
    - id: draft
      name: Draft
`

	parser := NewParser()
	_, err := parser.ParseBytes([]byte(yamlContent))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "initial_state")
}

func TestParser_Parse_InitialStateNotFound(t *testing.T) {
	yamlContent := `
version: "1.0"
workflow:
  name: "Invalid Initial State"
  initial_state: "nonexistent"
  states:
    - id: draft
      name: Draft
`

	parser := NewParser()
	_, err := parser.ParseBytes([]byte(yamlContent))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent")
}

func TestParser_Parse_DuplicateStateID(t *testing.T) {
	yamlContent := `
version: "1.0"
workflow:
  name: "Duplicate State"
  initial_state: "draft"
  states:
    - id: draft
      name: Draft
    - id: draft
      name: Another Draft
`

	parser := NewParser()
	_, err := parser.ParseBytes([]byte(yamlContent))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate state ID")
}

func TestParser_Parse_DuplicateEventName(t *testing.T) {
	yamlContent := `
version: "1.0"
workflow:
  name: "Duplicate Event"
  initial_state: "draft"
  states:
    - id: draft
      name: Draft
    - id: review
      name: Review
  events:
    - name: submit
      from: [draft]
      to: review
    - name: submit
      from: [draft]
      to: review
`

	parser := NewParser()
	_, err := parser.ParseBytes([]byte(yamlContent))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate event name")
}

func TestParser_Parse_InvalidStateIDFormat(t *testing.T) {
	yamlContent := `
version: "1.0"
workflow:
  name: "Invalid State ID"
  initial_state: "Draft-State"
  states:
    - id: Draft-State
      name: Draft
`

	parser := NewParser()
	_, err := parser.ParseBytes([]byte(yamlContent))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid state ID format")
}

func TestParser_Parse_InvalidTimeoutFormat(t *testing.T) {
	yamlContent := `
version: "1.0"
workflow:
  name: "Invalid Timeout"
  initial_state: "pending"
  states:
    - id: pending
      name: Pending
      timeout: invalid
      on_timeout: escalate
    - id: escalated
      name: Escalated
      final: true
  events:
    - name: escalate
      from: [pending]
      to: escalated
`

	parser := NewParser()
	_, err := parser.ParseBytes([]byte(yamlContent))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid duration format")
}

func TestParser_Parse_TimeoutWithoutOnTimeout(t *testing.T) {
	yamlContent := `
version: "1.0"
workflow:
  name: "Timeout without event"
  initial_state: "pending"
  states:
    - id: pending
      name: Pending
      timeout: 24h
`

	parser := NewParser()
	_, err := parser.ParseBytes([]byte(yamlContent))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "on_timeout event is required")
}

func TestParser_Parse_EventReferencesNonexistentState(t *testing.T) {
	yamlContent := `
version: "1.0"
workflow:
  name: "Bad Event Reference"
  initial_state: "draft"
  states:
    - id: draft
      name: Draft
  events:
    - name: submit
      from: [draft]
      to: nonexistent
`

	parser := NewParser()
	_, err := parser.ParseBytes([]byte(yamlContent))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent")
}

func TestParser_Parse_OnTimeoutReferencesNonexistentEvent(t *testing.T) {
	yamlContent := `
version: "1.0"
workflow:
  name: "Bad OnTimeout Reference"
  initial_state: "pending"
  states:
    - id: pending
      name: Pending
      timeout: 24h
      on_timeout: nonexistent_event
    - id: done
      name: Done
      final: true
  events:
    - name: complete
      from: [pending]
      to: done
`

	parser := NewParser()
	_, err := parser.ParseBytes([]byte(yamlContent))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent_event")
}

func TestParser_Parse_WithGuards(t *testing.T) {
	yamlContent := `
version: "1.0"
workflow:
  name: "Workflow with Guards"
  initial_state: "draft"
  states:
    - id: draft
      name: Draft
    - id: review
      name: Review
      final: true
  events:
    - name: submit
      from: [draft]
      to: review
      guards:
        - type: has_role
          params:
            role: editor
        - type: validate_required_fields
          params:
            required_fields:
              - title
              - content
`

	parser := NewParser()
	wf, err := parser.ParseBytes([]byte(yamlContent))

	require.NoError(t, err)
	assert.NotNil(t, wf)

	event, err := wf.GetEvent("submit")
	require.NoError(t, err)
	assert.Len(t, event.GetValidators(), 2)
	assert.Contains(t, event.GetValidators(), "has_role")
	assert.Contains(t, event.GetValidators(), "validate_required_fields")
}

func TestParser_Parse_UnknownGuardType(t *testing.T) {
	yamlContent := `
version: "1.0"
workflow:
  name: "Unknown Guard"
  initial_state: "draft"
  states:
    - id: draft
      name: Draft
    - id: review
      name: Review
      final: true
  events:
    - name: submit
      from: [draft]
      to: review
      guards:
        - type: unknown_guard_type
`

	parser := NewParser()
	_, err := parser.ParseBytes([]byte(yamlContent))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown guard type")
}

func TestParser_Parse_UnknownActionType(t *testing.T) {
	yamlContent := `
version: "1.0"
workflow:
  name: "Unknown Action"
  initial_state: "draft"
  states:
    - id: draft
      name: Draft
    - id: review
      name: Review
      final: true
  events:
    - name: submit
      from: [draft]
      to: review
      actions:
        - type: unknown_action_type
`

	parser := NewParser()
	_, err := parser.ParseBytes([]byte(yamlContent))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown action type")
}

func TestParser_Parse_CustomRegisteredAction(t *testing.T) {
	yamlContent := `
version: "1.0"
workflow:
  name: "Custom Action"
  initial_state: "draft"
  states:
    - id: draft
      name: Draft
    - id: review
      name: Review
      final: true
  events:
    - name: submit
      from: [draft]
      to: review
      actions:
        - type: my_custom_action
`

	parser := NewParser()
	parser.RegisterAction("my_custom_action")

	wf, err := parser.ParseBytes([]byte(yamlContent))

	require.NoError(t, err)
	assert.NotNil(t, wf)
}

func TestParser_Parse_WithSubstates(t *testing.T) {
	yamlContent := `
version: "1.0"
workflow:
  name: "Workflow with Substates"
  initial_state: "in_progress"
  states:
    - id: in_progress
      name: In Progress
      substates:
        - id: working
          name: Working
          default: true
        - id: escalated
          name: Escalated
    - id: done
      name: Done
      final: true
  events:
    - name: complete
      from: [in_progress]
      to: done
`

	parser := NewParser()
	wf, err := parser.ParseBytes([]byte(yamlContent))

	require.NoError(t, err)
	assert.NotNil(t, wf)
}

func TestParser_Parse_InvalidYAMLSyntax(t *testing.T) {
	yamlContent := `
version: "1.0"
workflow:
  name: "Invalid YAML
  initial_state: draft
`

	parser := NewParser()
	_, err := parser.ParseBytes([]byte(yamlContent))

	require.Error(t, err)
	var yamlErr InvalidYAMLError
	assert.ErrorAs(t, err, &yamlErr)
}

func TestParser_Parse_EmptyStates(t *testing.T) {
	yamlContent := `
version: "1.0"
workflow:
  name: "No States"
  initial_state: "draft"
  states: []
`

	parser := NewParser()
	_, err := parser.ParseBytes([]byte(yamlContent))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one state is required")
}

func TestParser_Parse_FromReader(t *testing.T) {
	yamlContent := `
version: "1.0"
workflow:
  name: "Reader Test"
  initial_state: "draft"
  states:
    - id: draft
      name: Draft
    - id: done
      name: Done
      final: true
  events:
    - name: complete
      from: [draft]
      to: done
`

	parser := NewParser()
	reader := strings.NewReader(yamlContent)
	wf, err := parser.Parse(reader)

	require.NoError(t, err)
	assert.NotNil(t, wf)
	assert.Equal(t, "Reader Test", wf.Name())
}

func TestParser_ParseWithDetails(t *testing.T) {
	yamlContent := `
version: "1.0"
workflow:
  id: "this-will-be-ignored"
  name: "Details Test"
  initial_state: "draft"
  states:
    - id: draft
      name: Draft
    - id: done
      name: Done
      final: true
  events:
    - name: complete
      from: [draft]
      to: done
`

	parser := NewParser()
	reader := strings.NewReader(yamlContent)
	result, err := parser.ParseWithDetails(reader)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.Workflow)
	assert.NotNil(t, result.Schema)
	// Should have warning about ID being ignored
	assert.Len(t, result.Warnings, 1)
	assert.Contains(t, result.Warnings[0], "workflow.id is ignored")
}

func TestParser_Parse_WithWebhooks(t *testing.T) {
	yamlContent := `
version: "1.0"
workflow:
  name: "Webhooks Test"
  initial_state: "draft"
  states:
    - id: draft
      name: Draft
    - id: done
      name: Done
      final: true
  events:
    - name: complete
      from: [draft]
      to: done
  webhooks:
    - url: "https://example.com/webhook"
      events:
        - state.changed
      secret: "secret123"
      active: true
      retry_config:
        max_retries: 3
        backoff_multiplier: 2
        initial_interval: 1s
`

	parser := NewParser()
	wf, err := parser.ParseBytes([]byte(yamlContent))

	require.NoError(t, err)
	assert.NotNil(t, wf)
}

func TestParser_Parse_WebhookInvalidRetryInterval(t *testing.T) {
	yamlContent := `
version: "1.0"
workflow:
  name: "Bad Webhook"
  initial_state: "draft"
  states:
    - id: draft
      name: Draft
  webhooks:
    - url: "https://example.com/webhook"
      events:
        - state.changed
      retry_config:
        initial_interval: invalid
      active: true
`

	parser := NewParser()
	_, err := parser.ParseBytes([]byte(yamlContent))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid duration format")
}

func TestValidationErrors_HasErrors(t *testing.T) {
	ve := &ValidationErrors{}
	assert.False(t, ve.HasErrors())

	ve.Add("field1", "error1")
	assert.True(t, ve.HasErrors())
}

func TestValidationErrors_Error(t *testing.T) {
	ve := &ValidationErrors{}
	ve.Add("field1", "error message 1")
	ve.Add("field2", "error message 2")

	errStr := ve.Error()
	assert.Contains(t, errStr, "2 error(s)")
	assert.Contains(t, errStr, "field1")
	assert.Contains(t, errStr, "field2")
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		hasError bool
	}{
		{"24h", "24h0m0s", false},
		{"30m", "30m0s", false},
		{"5d", "120h0m0s", false},
		{"1h30m", "1h30m0s", false},
		{"invalid", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			d, err := parseDuration(tt.input)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, d.String())
			}
		})
	}
}
