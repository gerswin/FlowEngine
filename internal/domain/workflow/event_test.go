package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEvent_Success(t *testing.T) {
	src, _ := NewState("draft", "Draft")
	dst, _ := NewState("approved", "Approved")

	evt, err := NewEvent("approve", []State{src}, dst)

	require.NoError(t, err)
	assert.Equal(t, "approve", evt.GetName())
	assert.Len(t, evt.GetSources(), 1)
	assert.Equal(t, "approved", evt.GetDestination().ID)
}

func TestNewEvent_EmptyName(t *testing.T) {
	src, _ := NewState("draft", "Draft")
	dst, _ := NewState("approved", "Approved")

	_, err := NewEvent("", []State{src}, dst)
	assert.Error(t, err)
}

func TestNewEvent_NoSources(t *testing.T) {
	dst, _ := NewState("approved", "Approved")

	_, err := NewEvent("approve", []State{}, dst)
	assert.Error(t, err)
}

func TestNewEvent_EmptyDestination(t *testing.T) {
	src, _ := NewState("draft", "Draft")

	_, err := NewEvent("approve", []State{src}, State{})
	assert.Error(t, err)
}

func TestEvent_WithGuards(t *testing.T) {
	src, _ := NewState("draft", "Draft")
	dst, _ := NewState("approved", "Approved")
	evt, _ := NewEvent("approve", []State{src}, dst)

	guards := []GuardConfig{
		{Type: "role_check", Params: map[string]interface{}{"role": "admin"}},
		{Type: "time_check"},
	}

	evtWithGuards := evt.WithGuards(guards)

	result := evtWithGuards.GetGuards()
	assert.Len(t, result, 2)
	assert.Equal(t, "role_check", result[0].Type)
	assert.Equal(t, "time_check", result[1].Type)

	// Verify original is unchanged (defensive copy)
	assert.Empty(t, evt.GetGuards())
}

func TestEvent_WithGuards_Nil(t *testing.T) {
	src, _ := NewState("draft", "Draft")
	dst, _ := NewState("approved", "Approved")
	evt, _ := NewEvent("approve", []State{src}, dst)

	evtWithGuards := evt.WithGuards(nil)

	assert.Empty(t, evtWithGuards.GetGuards())
}

func TestEvent_GetGuards_NilGuards(t *testing.T) {
	evt := Event{
		Name:    "test",
		Guards:  nil,
		Sources: []State{{ID: "a", Name: "A"}},
		Destination: State{ID: "b", Name: "B"},
	}

	assert.Empty(t, evt.GetGuards())
}

func TestEvent_WithActions(t *testing.T) {
	src, _ := NewState("draft", "Draft")
	dst, _ := NewState("approved", "Approved")
	evt, _ := NewEvent("approve", []State{src}, dst)

	actions := []ActionConfig{
		{Type: "send_email", Params: map[string]interface{}{"to": "admin@example.com"}},
		{Type: "log"},
	}

	evtWithActions := evt.WithActions(actions)

	result := evtWithActions.GetActions()
	assert.Len(t, result, 2)
	assert.Equal(t, "send_email", result[0].Type)
	assert.Equal(t, "log", result[1].Type)

	// Verify original is unchanged
	assert.Empty(t, evt.GetActions())
}

func TestEvent_WithActions_Nil(t *testing.T) {
	src, _ := NewState("draft", "Draft")
	dst, _ := NewState("approved", "Approved")
	evt, _ := NewEvent("approve", []State{src}, dst)

	evtWithActions := evt.WithActions(nil)

	assert.Empty(t, evtWithActions.GetActions())
}

func TestEvent_GetActions_NilActions(t *testing.T) {
	evt := Event{
		Name:    "test",
		Actions: nil,
		Sources: []State{{ID: "a", Name: "A"}},
		Destination: State{ID: "b", Name: "B"},
	}

	assert.Empty(t, evt.GetActions())
}

func TestEvent_WithRequiredData(t *testing.T) {
	src, _ := NewState("draft", "Draft")
	dst, _ := NewState("approved", "Approved")
	evt, _ := NewEvent("approve", []State{src}, dst)

	fields := []string{"reason", "approver_name"}
	evtWithData := evt.WithRequiredData(fields)

	result := evtWithData.GetRequiredData()
	assert.Len(t, result, 2)
	assert.Equal(t, "reason", result[0])
	assert.Equal(t, "approver_name", result[1])

	// Verify original is unchanged
	assert.Empty(t, evt.GetRequiredData())
}

func TestEvent_WithRequiredData_Nil(t *testing.T) {
	src, _ := NewState("draft", "Draft")
	dst, _ := NewState("approved", "Approved")
	evt, _ := NewEvent("approve", []State{src}, dst)

	evtWithData := evt.WithRequiredData(nil)

	assert.Empty(t, evtWithData.GetRequiredData())
}

func TestEvent_GetRequiredData_NilData(t *testing.T) {
	evt := Event{
		Name:         "test",
		RequiredData: nil,
		Sources:      []State{{ID: "a", Name: "A"}},
		Destination:  State{ID: "b", Name: "B"},
	}

	assert.Empty(t, evt.GetRequiredData())
}

func TestEvent_WithValidators(t *testing.T) {
	src, _ := NewState("draft", "Draft")
	dst, _ := NewState("approved", "Approved")
	evt, _ := NewEvent("approve", []State{src}, dst)

	validators := []string{"validate_budget", "validate_approver"}
	evtWithValidators := evt.WithValidators(validators)

	result := evtWithValidators.GetValidators()
	assert.Len(t, result, 2)
	assert.Equal(t, "validate_budget", result[0])
	assert.Equal(t, "validate_approver", result[1])
}

func TestEvent_WithValidators_Nil(t *testing.T) {
	src, _ := NewState("draft", "Draft")
	dst, _ := NewState("approved", "Approved")
	evt, _ := NewEvent("approve", []State{src}, dst)

	evtWithValidators := evt.WithValidators(nil)

	assert.Empty(t, evtWithValidators.GetValidators())
}

func TestEvent_GetValidators_NilValidators(t *testing.T) {
	evt := Event{
		Name:       "test",
		Validators: nil,
		Sources:    []State{{ID: "a", Name: "A"}},
		Destination: State{ID: "b", Name: "B"},
	}

	assert.Empty(t, evt.GetValidators())
}

func TestEvent_CanTransitionFrom(t *testing.T) {
	src1, _ := NewState("draft", "Draft")
	src2, _ := NewState("review", "Review")
	dst, _ := NewState("approved", "Approved")
	evt, _ := NewEvent("approve", []State{src1, src2}, dst)

	assert.True(t, evt.CanTransitionFrom(src1))
	assert.True(t, evt.CanTransitionFrom(src2))

	other, _ := NewState("rejected", "Rejected")
	assert.False(t, evt.CanTransitionFrom(other))
}

func TestEvent_Validate(t *testing.T) {
	t.Run("valid event", func(t *testing.T) {
		src, _ := NewState("draft", "Draft")
		dst, _ := NewState("approved", "Approved")
		evt, _ := NewEvent("approve", []State{src}, dst)

		assert.NoError(t, evt.Validate())
	})

	t.Run("empty name", func(t *testing.T) {
		evt := Event{Name: "", Sources: []State{{ID: "a", Name: "A"}}, Destination: State{ID: "b", Name: "B"}}
		assert.Error(t, evt.Validate())
	})

	t.Run("no sources", func(t *testing.T) {
		evt := Event{Name: "test", Sources: []State{}, Destination: State{ID: "b", Name: "B"}}
		assert.Error(t, evt.Validate())
	})
}

func TestEvent_String(t *testing.T) {
	src, _ := NewState("draft", "Draft")
	dst, _ := NewState("approved", "Approved")
	evt, _ := NewEvent("approve", []State{src}, dst)

	s := evt.String()
	assert.Contains(t, s, "approve")
	assert.Contains(t, s, "draft")
	assert.Contains(t, s, "approved")
}

func TestEvent_GetSources_DefensiveCopy(t *testing.T) {
	src, _ := NewState("draft", "Draft")
	dst, _ := NewState("approved", "Approved")
	evt, _ := NewEvent("approve", []State{src}, dst)

	sources := evt.GetSources()
	sources[0] = State{ID: "modified", Name: "Modified"}

	// Original should be unchanged
	assert.Equal(t, "draft", evt.GetSources()[0].ID)
}
