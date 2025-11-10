package workflow

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewState_Success(t *testing.T) {
	state, err := NewState("pending", "Pending")

	require.NoError(t, err)
	assert.Equal(t, "pending", state.ID())
	assert.Equal(t, "Pending", state.Name())
	assert.Empty(t, state.Description())
	assert.Equal(t, time.Duration(0), state.Timeout())
	assert.Empty(t, state.OnTimeout())
	assert.False(t, state.IsFinal())
}

func TestNewState_InvalidID(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{"empty ID", "", true},
		{"starts with uppercase", "Pending", true},
		{"starts with number", "1pending", true},
		{"contains spaces", "pending state", true},
		{"contains hyphens", "pending-state", true},
		{"valid lowercase", "pending", false},
		{"valid with underscores", "pending_approval", false},
		{"valid with numbers", "pending1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewState(tt.id, "Test State")

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewState_EmptyName(t *testing.T) {
	_, err := NewState("pending", "")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name cannot be empty")
}

func TestState_WithDescription(t *testing.T) {
	state, _ := NewState("pending", "Pending")

	stateWithDesc := state.WithDescription("Waiting for approval")

	assert.Equal(t, "Waiting for approval", stateWithDesc.Description())
	assert.Equal(t, state.ID(), stateWithDesc.ID())
	assert.Equal(t, state.Name(), stateWithDesc.Name())
}

func TestState_WithTimeout(t *testing.T) {
	state, _ := NewState("pending", "Pending")

	stateWithTimeout := state.WithTimeout(1*time.Hour, "timeout_event")

	assert.Equal(t, 1*time.Hour, stateWithTimeout.Timeout())
	assert.Equal(t, "timeout_event", stateWithTimeout.OnTimeout())
}

func TestState_AsFinal(t *testing.T) {
	state, _ := NewState("completed", "Completed")

	finalState := state.AsFinal()

	assert.True(t, finalState.IsFinal())
	assert.False(t, state.IsFinal()) // Original should be unchanged (immutability)
}

func TestState_Validate(t *testing.T) {
	t.Run("valid state", func(t *testing.T) {
		state, _ := NewState("pending", "Pending")
		err := state.Validate()
		assert.NoError(t, err)
	})

	t.Run("timeout without onTimeout event", func(t *testing.T) {
		state, _ := NewState("pending", "Pending")
		state = state.WithTimeout(1*time.Hour, "")

		err := state.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "onTimeout event must be set")
	})
}

func TestState_Equals(t *testing.T) {
	state1, _ := NewState("pending", "Pending")
	state2, _ := NewState("pending", "Different Name")
	state3, _ := NewState("approved", "Approved")

	assert.True(t, state1.Equals(state2), "states with same ID should be equal")
	assert.False(t, state1.Equals(state3), "states with different IDs should not be equal")
}

func TestState_String(t *testing.T) {
	state, _ := NewState("pending", "Pending")

	str := state.String()

	assert.Contains(t, str, "pending")
	assert.Contains(t, str, "Pending")
}
