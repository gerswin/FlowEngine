package instance

import (
	"testing"

	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/shared"
	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestInstanceWithData builds a test instance pre-populated with the
// supplied data entries.
func createTestInstanceWithData(data map[string]interface{}) *Instance {
	wfID := shared.NewID()
	actor := shared.NewID()
	inst, _ := NewInstance(wfID, "test", "draft", actor)
	for k, v := range data {
		inst.UpdateData(k, v)
	}
	return inst
}

// ---------------------------------------------------------------------------
// Guards
// ---------------------------------------------------------------------------

func TestGuard_FieldExists_Passes(t *testing.T) {
	engine := NewEngine()
	inst := createTestInstanceWithData(map[string]interface{}{"title": "hello"})
	ctx := GuardContext{Instance: inst, ActorID: shared.NewID()}
	err := engine.EvaluateGuards(ctx, []workflow.GuardConfig{
		{Type: "field_exists", Params: map[string]interface{}{"field": "title"}},
	})
	assert.NoError(t, err)
}

func TestGuard_FieldExists_Fails(t *testing.T) {
	engine := NewEngine()
	inst := createTestInstanceWithData(nil)
	ctx := GuardContext{Instance: inst, ActorID: shared.NewID()}
	err := engine.EvaluateGuards(ctx, []workflow.GuardConfig{
		{Type: "field_exists", Params: map[string]interface{}{"field": "missing"}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestGuard_FieldNotEmpty_Passes(t *testing.T) {
	engine := NewEngine()
	inst := createTestInstanceWithData(map[string]interface{}{"name": "Alice"})
	ctx := GuardContext{Instance: inst, ActorID: shared.NewID()}
	err := engine.EvaluateGuards(ctx, []workflow.GuardConfig{
		{Type: "field_not_empty", Params: map[string]interface{}{"field": "name"}},
	})
	assert.NoError(t, err)
}

func TestGuard_FieldNotEmpty_FailsEmpty(t *testing.T) {
	engine := NewEngine()
	inst := createTestInstanceWithData(map[string]interface{}{"name": ""})
	ctx := GuardContext{Instance: inst, ActorID: shared.NewID()}
	err := engine.EvaluateGuards(ctx, []workflow.GuardConfig{
		{Type: "field_not_empty", Params: map[string]interface{}{"field": "name"}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is empty")
}

func TestGuard_FieldNotEmpty_FailsNil(t *testing.T) {
	engine := NewEngine()
	inst := createTestInstanceWithData(nil)
	ctx := GuardContext{Instance: inst, ActorID: shared.NewID()}
	err := engine.EvaluateGuards(ctx, []workflow.GuardConfig{
		{Type: "field_not_empty", Params: map[string]interface{}{"field": "name"}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is empty")
}

func TestGuard_FieldEquals_Passes(t *testing.T) {
	engine := NewEngine()
	inst := createTestInstanceWithData(map[string]interface{}{"status": "active"})
	ctx := GuardContext{Instance: inst, ActorID: shared.NewID()}
	err := engine.EvaluateGuards(ctx, []workflow.GuardConfig{
		{Type: "field_equals", Params: map[string]interface{}{"field": "status", "value": "active"}},
	})
	assert.NoError(t, err)
}

func TestGuard_FieldEquals_Fails(t *testing.T) {
	engine := NewEngine()
	inst := createTestInstanceWithData(map[string]interface{}{"status": "inactive"})
	ctx := GuardContext{Instance: inst, ActorID: shared.NewID()}
	err := engine.EvaluateGuards(ctx, []workflow.GuardConfig{
		{Type: "field_equals", Params: map[string]interface{}{"field": "status", "value": "active"}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected")
}

func TestGuard_FieldMatches_Passes(t *testing.T) {
	engine := NewEngine()
	inst := createTestInstanceWithData(map[string]interface{}{"email": "user@example.com"})
	ctx := GuardContext{Instance: inst, ActorID: shared.NewID()}
	err := engine.EvaluateGuards(ctx, []workflow.GuardConfig{
		{Type: "field_matches", Params: map[string]interface{}{"field": "email", "pattern": `^.+@.+\..+$`}},
	})
	assert.NoError(t, err)
}

func TestGuard_FieldMatches_Fails(t *testing.T) {
	engine := NewEngine()
	inst := createTestInstanceWithData(map[string]interface{}{"email": "bad"})
	ctx := GuardContext{Instance: inst, ActorID: shared.NewID()}
	err := engine.EvaluateGuards(ctx, []workflow.GuardConfig{
		{Type: "field_matches", Params: map[string]interface{}{"field": "email", "pattern": `^.+@.+\..+$`}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not match")
}

func TestGuard_HasRole_Passes(t *testing.T) {
	engine := NewEngine()
	inst := createTestInstanceWithData(nil)
	ctx := GuardContext{Instance: inst, ActorID: shared.NewID(), Roles: []string{"admin", "editor"}}
	err := engine.EvaluateGuards(ctx, []workflow.GuardConfig{
		{Type: "has_role", Params: map[string]interface{}{"role": "admin"}},
	})
	assert.NoError(t, err)
}

func TestGuard_HasRole_Fails(t *testing.T) {
	engine := NewEngine()
	inst := createTestInstanceWithData(nil)
	ctx := GuardContext{Instance: inst, ActorID: shared.NewID(), Roles: []string{"viewer"}}
	err := engine.EvaluateGuards(ctx, []workflow.GuardConfig{
		{Type: "has_role", Params: map[string]interface{}{"role": "admin"}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not have role")
}

func TestGuard_HasAnyRole_Passes(t *testing.T) {
	engine := NewEngine()
	inst := createTestInstanceWithData(nil)
	ctx := GuardContext{Instance: inst, ActorID: shared.NewID(), Roles: []string{"viewer", "editor"}}
	err := engine.EvaluateGuards(ctx, []workflow.GuardConfig{
		{Type: "has_any_role", Params: map[string]interface{}{"roles": []interface{}{"admin", "editor"}}},
	})
	assert.NoError(t, err)
}

func TestGuard_HasAnyRole_Fails(t *testing.T) {
	engine := NewEngine()
	inst := createTestInstanceWithData(nil)
	ctx := GuardContext{Instance: inst, ActorID: shared.NewID(), Roles: []string{"viewer"}}
	err := engine.EvaluateGuards(ctx, []workflow.GuardConfig{
		{Type: "has_any_role", Params: map[string]interface{}{"roles": []interface{}{"admin", "editor"}}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not have any")
}

func TestGuard_ValidateRequiredFields_Passes(t *testing.T) {
	engine := NewEngine()
	inst := createTestInstanceWithData(map[string]interface{}{"a": 1, "b": 2})
	ctx := GuardContext{Instance: inst, ActorID: shared.NewID()}
	err := engine.EvaluateGuards(ctx, []workflow.GuardConfig{
		{Type: "validate_required_fields", Params: map[string]interface{}{"fields": []interface{}{"a", "b"}}},
	})
	assert.NoError(t, err)
}

func TestGuard_ValidateRequiredFields_ListsMissing(t *testing.T) {
	engine := NewEngine()
	inst := createTestInstanceWithData(map[string]interface{}{"a": 1})
	ctx := GuardContext{Instance: inst, ActorID: shared.NewID()}
	err := engine.EvaluateGuards(ctx, []workflow.GuardConfig{
		{Type: "validate_required_fields", Params: map[string]interface{}{"fields": []interface{}{"a", "b", "c"}}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "b")
	assert.Contains(t, err.Error(), "c")
}

func TestGuard_IsAssignedToActor_Passes(t *testing.T) {
	engine := NewEngine()
	actor := shared.NewID()
	inst := createTestInstanceWithData(map[string]interface{}{"assigned_to": actor.String()})
	ctx := GuardContext{Instance: inst, ActorID: actor}
	err := engine.EvaluateGuards(ctx, []workflow.GuardConfig{
		{Type: "is_assigned_to_actor", Params: nil},
	})
	assert.NoError(t, err)
}

func TestGuard_IsAssignedToActor_Fails(t *testing.T) {
	engine := NewEngine()
	actor := shared.NewID()
	otherActor := shared.NewID()
	inst := createTestInstanceWithData(map[string]interface{}{"assigned_to": otherActor.String()})
	ctx := GuardContext{Instance: inst, ActorID: actor}
	err := engine.EvaluateGuards(ctx, []workflow.GuardConfig{
		{Type: "is_assigned_to_actor", Params: nil},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not assigned to the current actor")
}

func TestGuard_IsNotAssigned_Passes(t *testing.T) {
	engine := NewEngine()
	inst := createTestInstanceWithData(nil)
	ctx := GuardContext{Instance: inst, ActorID: shared.NewID()}
	err := engine.EvaluateGuards(ctx, []workflow.GuardConfig{
		{Type: "is_not_assigned", Params: nil},
	})
	assert.NoError(t, err)
}

func TestGuard_IsNotAssigned_Fails(t *testing.T) {
	engine := NewEngine()
	inst := createTestInstanceWithData(map[string]interface{}{"assigned_to": "someone"})
	ctx := GuardContext{Instance: inst, ActorID: shared.NewID()}
	err := engine.EvaluateGuards(ctx, []workflow.GuardConfig{
		{Type: "is_not_assigned", Params: nil},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already assigned")
}

func TestEvaluateGuards_StopsOnFirstFailure(t *testing.T) {
	engine := NewEngine()
	inst := createTestInstanceWithData(nil)
	ctx := GuardContext{Instance: inst, ActorID: shared.NewID()}
	// First guard fails, second should not be evaluated.
	err := engine.EvaluateGuards(ctx, []workflow.GuardConfig{
		{Type: "field_exists", Params: map[string]interface{}{"field": "missing"}},
		{Type: "field_exists", Params: map[string]interface{}{"field": "also_missing"}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing")
	// Ensure only the first field is mentioned.
	assert.NotContains(t, err.Error(), "also_missing")
}

func TestEvaluateGuards_UnknownType(t *testing.T) {
	engine := NewEngine()
	inst := createTestInstanceWithData(nil)
	ctx := GuardContext{Instance: inst, ActorID: shared.NewID()}
	err := engine.EvaluateGuards(ctx, []workflow.GuardConfig{
		{Type: "nonexistent_guard", Params: nil},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown guard type")
}

// ---------------------------------------------------------------------------
// Actions
// ---------------------------------------------------------------------------

func TestAction_SetMetadata(t *testing.T) {
	engine := NewEngine()
	inst := createTestInstanceWithData(nil)
	actor := shared.NewID()
	ctx := ActionContext{Instance: inst, ActorID: actor, Params: map[string]interface{}{
		"key": "priority", "value": "high",
	}}
	err := engine.ExecuteActions(ctx, []workflow.ActionConfig{
		{Type: "set_metadata", Params: map[string]interface{}{"key": "priority", "value": "high"}},
	})
	require.NoError(t, err)
	assert.Equal(t, "high", inst.Data().Get("priority"))
}

func TestAction_SetMetadata_Now(t *testing.T) {
	engine := NewEngine()
	inst := createTestInstanceWithData(nil)
	actor := shared.NewID()
	err := engine.ExecuteActions(ActionContext{Instance: inst, ActorID: actor}, []workflow.ActionConfig{
		{Type: "set_metadata", Params: map[string]interface{}{"key": "ts", "value": "$now"}},
	})
	require.NoError(t, err)
	val, ok := inst.Data().GetString("ts")
	assert.True(t, ok)
	assert.NotEmpty(t, val)
	// Should look like an RFC3339 timestamp.
	assert.Contains(t, val, "T")
}

func TestAction_IncrementField_FromZero(t *testing.T) {
	engine := NewEngine()
	inst := createTestInstanceWithData(nil)
	actor := shared.NewID()
	err := engine.ExecuteActions(ActionContext{Instance: inst, ActorID: actor}, []workflow.ActionConfig{
		{Type: "increment_field", Params: map[string]interface{}{"field": "counter"}},
	})
	require.NoError(t, err)
	v, _ := inst.Data().GetInt("counter")
	assert.Equal(t, 1, v)
}

func TestAction_IncrementField_FromExisting(t *testing.T) {
	engine := NewEngine()
	inst := createTestInstanceWithData(map[string]interface{}{"counter": 5})
	actor := shared.NewID()
	err := engine.ExecuteActions(ActionContext{Instance: inst, ActorID: actor}, []workflow.ActionConfig{
		{Type: "increment_field", Params: map[string]interface{}{"field": "counter"}},
	})
	require.NoError(t, err)
	v, _ := inst.Data().GetInt("counter")
	assert.Equal(t, 6, v)
}

func TestAction_AssignToUser_Specified(t *testing.T) {
	engine := NewEngine()
	inst := createTestInstanceWithData(nil)
	targetUser := shared.NewID()
	actor := shared.NewID()
	err := engine.ExecuteActions(ActionContext{Instance: inst, ActorID: actor}, []workflow.ActionConfig{
		{Type: "assign_to_user", Params: map[string]interface{}{"user_id": targetUser.String()}},
	})
	require.NoError(t, err)
	assert.Equal(t, targetUser.String(), inst.Data().Get("assigned_to"))
}

func TestAction_AssignToUser_DefaultsToActor(t *testing.T) {
	engine := NewEngine()
	inst := createTestInstanceWithData(nil)
	actor := shared.NewID()
	err := engine.ExecuteActions(ActionContext{Instance: inst, ActorID: actor}, []workflow.ActionConfig{
		{Type: "assign_to_user", Params: map[string]interface{}{}},
	})
	require.NoError(t, err)
	assert.Equal(t, actor.String(), inst.Data().Get("assigned_to"))
}

func TestAction_MarkAsApproved(t *testing.T) {
	engine := NewEngine()
	inst := createTestInstanceWithData(nil)
	actor := shared.NewID()
	err := engine.ExecuteActions(ActionContext{Instance: inst, ActorID: actor}, []workflow.ActionConfig{
		{Type: "mark_as_approved", Params: nil},
	})
	require.NoError(t, err)
	approved, ok := inst.Data().GetBool("approved")
	assert.True(t, ok)
	assert.True(t, approved)
	assert.Equal(t, actor.String(), inst.Data().Get("approved_by"))
	approvedAt, ok := inst.Data().GetString("approved_at")
	assert.True(t, ok)
	assert.NotEmpty(t, approvedAt)
}

func TestAction_IncrementRejectionCount(t *testing.T) {
	engine := NewEngine()
	inst := createTestInstanceWithData(map[string]interface{}{"rejection_count": 2})
	actor := shared.NewID()
	err := engine.ExecuteActions(ActionContext{Instance: inst, ActorID: actor}, []workflow.ActionConfig{
		{Type: "increment_rejection_count", Params: nil},
	})
	require.NoError(t, err)
	v, _ := inst.Data().GetInt("rejection_count")
	assert.Equal(t, 3, v)
}

func TestExecuteActions_MultipleInOrder(t *testing.T) {
	engine := NewEngine()
	inst := createTestInstanceWithData(nil)
	actor := shared.NewID()
	err := engine.ExecuteActions(ActionContext{Instance: inst, ActorID: actor}, []workflow.ActionConfig{
		{Type: "set_metadata", Params: map[string]interface{}{"key": "step", "value": "one"}},
		{Type: "set_metadata", Params: map[string]interface{}{"key": "step", "value": "two"}},
	})
	require.NoError(t, err)
	// Last write wins.
	assert.Equal(t, "two", inst.Data().Get("step"))
}

func TestExecuteActions_UnknownTypeSkipped(t *testing.T) {
	engine := NewEngine()
	inst := createTestInstanceWithData(nil)
	actor := shared.NewID()
	err := engine.ExecuteActions(ActionContext{Instance: inst, ActorID: actor}, []workflow.ActionConfig{
		{Type: "totally_unknown_action", Params: nil},
		{Type: "set_metadata", Params: map[string]interface{}{"key": "ran", "value": true}},
	})
	require.NoError(t, err)
	assert.Equal(t, true, inst.Data().Get("ran"))
}

// ---------------------------------------------------------------------------
// Custom registration
// ---------------------------------------------------------------------------

func TestRegisterGuard_CustomGuard(t *testing.T) {
	engine := NewEngine()
	engine.RegisterGuard("always_pass", func(ctx GuardContext, params map[string]interface{}) error {
		return nil
	})
	inst := createTestInstanceWithData(nil)
	ctx := GuardContext{Instance: inst, ActorID: shared.NewID()}
	err := engine.EvaluateGuards(ctx, []workflow.GuardConfig{
		{Type: "always_pass", Params: nil},
	})
	assert.NoError(t, err)
}

func TestRegisterAction_CustomAction(t *testing.T) {
	engine := NewEngine()
	engine.RegisterAction("custom_stamp", func(ctx ActionContext) error {
		ctx.Instance.UpdateData("custom_stamp", "done")
		return nil
	})
	inst := createTestInstanceWithData(nil)
	actor := shared.NewID()
	err := engine.ExecuteActions(ActionContext{Instance: inst, ActorID: actor}, []workflow.ActionConfig{
		{Type: "custom_stamp", Params: nil},
	})
	require.NoError(t, err)
	assert.Equal(t, "done", inst.Data().Get("custom_stamp"))
}
