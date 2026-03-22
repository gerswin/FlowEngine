package instance

import (
	"encoding/json"
	"testing"

	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Instance lifecycle tests
// ---------------------------------------------------------------------------

func TestInstance_Complete_FailsIfNotActive(t *testing.T) {
	inst := createTestInstance(t)
	actor := shared.NewID()

	// Complete once
	require.NoError(t, inst.Complete(actor))

	// Try to complete again
	err := inst.Complete(actor)
	assert.Error(t, err)
}

func TestInstance_Cancel_FailsIfNotActive(t *testing.T) {
	inst := createTestInstance(t)
	actor := shared.NewID()

	require.NoError(t, inst.Complete(actor))

	err := inst.Cancel(actor, "too late")
	assert.Error(t, err)
}

func TestInstance_Cancel_WithEmptyReason(t *testing.T) {
	inst := createTestInstance(t)
	actor := shared.NewID()

	err := inst.Cancel(actor, "")
	require.NoError(t, err)
	assert.Equal(t, StatusCanceled, inst.Status())
	// No cancellation_reason key when reason is empty
	assert.False(t, inst.Data().Has("cancellation_reason"))
}

func TestInstance_Fail_FailsIfNotActive(t *testing.T) {
	inst := createTestInstance(t)
	actor := shared.NewID()

	require.NoError(t, inst.Complete(actor))

	err := inst.Fail(actor, "some error")
	assert.Error(t, err)
}

func TestInstance_Fail_WithEmptyReason(t *testing.T) {
	inst := createTestInstance(t)
	actor := shared.NewID()

	err := inst.Fail(actor, "")
	require.NoError(t, err)
	assert.Equal(t, StatusFailed, inst.Status())
	assert.False(t, inst.Data().Has("failure_reason"))
}

func TestInstance_Pause_FailsIfNotRunning(t *testing.T) {
	inst := createTestInstance(t)
	actor := shared.NewID()

	// Pause first
	require.NoError(t, inst.Pause(actor, "reason"))

	// Try to pause again (now paused, not running)
	err := inst.Pause(actor, "again")
	assert.Error(t, err)
}

func TestInstance_Resume_FailsIfNotPaused(t *testing.T) {
	inst := createTestInstance(t)
	actor := shared.NewID()

	// Instance is running, not paused
	err := inst.Resume(actor)
	assert.Error(t, err)
}

func TestInstance_Resume_FailsIfCompleted(t *testing.T) {
	inst := createTestInstance(t)
	actor := shared.NewID()

	require.NoError(t, inst.Complete(actor))

	err := inst.Resume(actor)
	assert.Error(t, err)
}

func TestInstance_SetVariables(t *testing.T) {
	inst := createTestInstance(t)
	initialVersion := inst.Version()

	vars := NewVariablesFromMap(map[string]interface{}{"key": "value"})
	inst.SetVariables(vars)

	assert.Equal(t, "value", inst.Variables().Get("key"))
	assert.True(t, inst.Version().IsGreaterThan(initialVersion))
}

func TestInstance_UpdateVariable(t *testing.T) {
	inst := createTestInstance(t)

	inst.UpdateVariable("counter", 42)

	val, ok := inst.Variables().GetInt("counter")
	assert.True(t, ok)
	assert.Equal(t, 42, val)
}

func TestInstance_GetLastTransition_NilWhenEmpty(t *testing.T) {
	inst := createTestInstance(t)
	assert.Nil(t, inst.GetLastTransition())
}

func TestInstance_GetLastTransition_ReturnsLast(t *testing.T) {
	inst := createTestInstance(t)
	actor := shared.NewID()

	require.NoError(t, inst.Transition("state2", "event1", actor, EmptyTransitionMetadata(), nil))
	require.NoError(t, inst.Transition("state3", "event2", actor, EmptyTransitionMetadata(), nil))

	last := inst.GetLastTransition()
	require.NotNil(t, last)
	assert.Equal(t, "state3", last.To())
	assert.Equal(t, "event2", last.Event())
}

func TestInstance_TransitionCount(t *testing.T) {
	inst := createTestInstance(t)
	actor := shared.NewID()

	assert.Equal(t, 0, inst.TransitionCount())

	require.NoError(t, inst.Transition("state2", "event1", actor, EmptyTransitionMetadata(), nil))
	assert.Equal(t, 1, inst.TransitionCount())

	require.NoError(t, inst.Transition("state3", "event2", actor, EmptyTransitionMetadata(), nil))
	assert.Equal(t, 2, inst.TransitionCount())
}

func TestInstance_Transition_EmptyToState(t *testing.T) {
	inst := createTestInstance(t)
	actor := shared.NewID()

	err := inst.Transition("", "event", actor, EmptyTransitionMetadata(), nil)
	assert.Error(t, err)
}

func TestInstance_Transition_EmptyEvent(t *testing.T) {
	inst := createTestInstance(t)
	actor := shared.NewID()

	err := inst.Transition("state2", "", actor, EmptyTransitionMetadata(), nil)
	assert.Error(t, err)
}

func TestInstance_Transition_InvalidActor(t *testing.T) {
	inst := createTestInstance(t)

	err := inst.Transition("state2", "event", shared.NilID(), EmptyTransitionMetadata(), nil)
	assert.Error(t, err)
}

func TestInstance_Transition_MissingRequiredData(t *testing.T) {
	inst := createTestInstance(t)
	actor := shared.NewID()

	err := inst.Transition("state2", "approve", actor, EmptyTransitionMetadata(), []string{"approval_doc", "reason"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "approval_doc")
	assert.Contains(t, err.Error(), "reason")
}

func TestInstance_Transition_RequiredDataPresent(t *testing.T) {
	inst := createTestInstance(t)
	actor := shared.NewID()

	inst.UpdateData("approval_doc", "doc.pdf")
	inst.UpdateData("reason", "looks good")

	err := inst.Transition("state2", "approve", actor, EmptyTransitionMetadata(), []string{"approval_doc", "reason"})
	assert.NoError(t, err)
}

func TestInstance_Validate_AllCases(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(inst *Instance)
		wantErr bool
	}{
		{
			name:    "valid instance",
			mutate:  func(inst *Instance) {},
			wantErr: false,
		},
		{
			name:    "invalid ID",
			mutate:  func(inst *Instance) { inst.id = shared.NilID() },
			wantErr: true,
		},
		{
			name:    "invalid workflow ID",
			mutate:  func(inst *Instance) { inst.workflowID = shared.NilID() },
			wantErr: true,
		},
		{
			name:    "empty workflow name",
			mutate:  func(inst *Instance) { inst.workflowName = "" },
			wantErr: true,
		},
		{
			name:    "empty current state",
			mutate:  func(inst *Instance) { inst.currentState = "" },
			wantErr: true,
		},
		{
			name:    "invalid status",
			mutate:  func(inst *Instance) { inst.status = Status("INVALID") },
			wantErr: true,
		},
		{
			name:    "zero version",
			mutate:  func(inst *Instance) { inst.version = ZeroVersion() },
			wantErr: true,
		},
		{
			name:    "invalid startedBy",
			mutate:  func(inst *Instance) { inst.startedBy = shared.NilID() },
			wantErr: true,
		},
		{
			name: "invalid sub-state",
			mutate: func(inst *Instance) {
				inst.currentSubState = RestoreSubState("INVALID-ID")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inst := createTestInstance(t)
			tt.mutate(inst)
			err := inst.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestInstance_String_WithoutParent(t *testing.T) {
	inst := createTestInstance(t)
	s := inst.String()

	assert.Contains(t, s, "Instance{")
	assert.Contains(t, s, "workflow=Test Workflow")
	assert.Contains(t, s, "state=draft")
	assert.Contains(t, s, "status=RUNNING")
	assert.NotContains(t, s, "parent=")
}

func TestInstance_String_WithParent(t *testing.T) {
	parentID := shared.NewID()
	workflowID := shared.NewID()
	actor := shared.NewID()

	inst, err := NewSubprocess(parentID, workflowID, "Sub Workflow", "init", actor)
	require.NoError(t, err)

	s := inst.String()
	assert.Contains(t, s, "parent=")
}

func TestInstance_String_WithSubState(t *testing.T) {
	workflowID := shared.NewID()
	actor := shared.NewID()
	subState, _ := NewSubState("pending_review", "Pending Review")

	inst, err := NewInstanceWithSubState(workflowID, "Test", "pending", subState, actor)
	require.NoError(t, err)

	s := inst.String()
	assert.Contains(t, s, "pending.pending_review")
}

func TestNewSubprocess_Success(t *testing.T) {
	parentID := shared.NewID()
	workflowID := shared.NewID()
	actor := shared.NewID()

	inst, err := NewSubprocess(parentID, workflowID, "Sub", "init", actor)
	require.NoError(t, err)
	assert.Equal(t, parentID, inst.ParentID())
	assert.True(t, inst.ParentID().IsValid())
}

func TestNewSubprocess_InvalidParentID(t *testing.T) {
	workflowID := shared.NewID()
	actor := shared.NewID()

	inst, err := NewSubprocess(shared.NilID(), workflowID, "Sub", "init", actor)
	assert.Error(t, err)
	assert.Nil(t, inst)
}

func TestNewInstanceWithSubState_InvalidSubState(t *testing.T) {
	workflowID := shared.NewID()
	actor := shared.NewID()

	// RestoreSubState bypasses validation, so the sub-state will have an invalid ID format
	badSubState := RestoreSubState("INVALID-ID")
	inst, err := NewInstanceWithSubState(workflowID, "Test", "pending", badSubState, actor)
	assert.Error(t, err)
	assert.Nil(t, inst)
}

func TestInstance_DomainEvents_ReturnsAndClears(t *testing.T) {
	inst := createTestInstance(t)

	// NewInstance records a creation event
	events := inst.DomainEvents()
	assert.NotEmpty(t, events)

	// Calling again should return empty
	events2 := inst.DomainEvents()
	assert.Empty(t, events2)
}

func TestInstance_Transitions_ReturnsCopy(t *testing.T) {
	inst := createTestInstance(t)
	actor := shared.NewID()
	require.NoError(t, inst.Transition("state2", "event1", actor, EmptyTransitionMetadata(), nil))

	transitions := inst.Transitions()
	assert.Equal(t, 1, len(transitions))

	// Modifying the returned slice should not affect the instance
	transitions[0] = nil
	assert.NotNil(t, inst.Transitions()[0])
}

func TestInstance_HasSubState(t *testing.T) {
	inst := createTestInstance(t)
	assert.False(t, inst.HasSubState())

	workflowID := shared.NewID()
	actor := shared.NewID()
	subState, _ := NewSubState("review", "Review")
	inst2, _ := NewInstanceWithSubState(workflowID, "Test", "pending", subState, actor)
	assert.True(t, inst2.HasSubState())
}

// ---------------------------------------------------------------------------
// Data value object tests
// ---------------------------------------------------------------------------

func TestNewData_CreatesEmpty(t *testing.T) {
	d := NewData()
	assert.True(t, d.IsEmpty())
	assert.Equal(t, 0, d.Size())
}

func TestNewDataFromMap_CopiesMap(t *testing.T) {
	original := map[string]interface{}{"key": "value"}
	d := NewDataFromMap(original)

	// Mutating original should not affect Data
	original["key"] = "changed"
	assert.Equal(t, "value", d.Get("key"))
}

func TestNewDataFromMap_NilMap(t *testing.T) {
	d := NewDataFromMap(nil)
	assert.True(t, d.IsEmpty())
}

func TestData_GetString(t *testing.T) {
	d := NewData().Set("name", "Alice").Set("count", 42)

	val, ok := d.GetString("name")
	assert.True(t, ok)
	assert.Equal(t, "Alice", val)

	// Wrong type
	_, ok = d.GetString("count")
	assert.False(t, ok)

	// Missing key
	_, ok = d.GetString("missing")
	assert.False(t, ok)
}

func TestData_GetInt(t *testing.T) {
	d := NewData().Set("count", 42).Set("float_num", float64(7)).Set("int64_num", int64(99)).Set("name", "Alice")

	val, ok := d.GetInt("count")
	assert.True(t, ok)
	assert.Equal(t, 42, val)

	val, ok = d.GetInt("float_num")
	assert.True(t, ok)
	assert.Equal(t, 7, val)

	val, ok = d.GetInt("int64_num")
	assert.True(t, ok)
	assert.Equal(t, 99, val)

	// Wrong type
	_, ok = d.GetInt("name")
	assert.False(t, ok)

	// Missing key
	_, ok = d.GetInt("missing")
	assert.False(t, ok)
}

func TestData_GetBool(t *testing.T) {
	d := NewData().Set("flag", true).Set("name", "Alice")

	val, ok := d.GetBool("flag")
	assert.True(t, ok)
	assert.True(t, val)

	// Wrong type
	_, ok = d.GetBool("name")
	assert.False(t, ok)

	// Missing key
	_, ok = d.GetBool("missing")
	assert.False(t, ok)
}

func TestData_Set_Immutability(t *testing.T) {
	d1 := NewData().Set("a", 1)
	d2 := d1.Set("b", 2)

	assert.True(t, d1.Has("a"))
	assert.False(t, d1.Has("b")) // d1 is unchanged
	assert.True(t, d2.Has("a"))
	assert.True(t, d2.Has("b"))
}

func TestData_Delete(t *testing.T) {
	d := NewData().Set("a", 1).Set("b", 2)
	d2 := d.Delete("a")

	assert.False(t, d2.Has("a"))
	assert.True(t, d2.Has("b"))
	// Original unchanged
	assert.True(t, d.Has("a"))
}

func TestData_Has(t *testing.T) {
	d := NewData().Set("key", "value")

	assert.True(t, d.Has("key"))
	assert.False(t, d.Has("missing"))
}

func TestData_Keys(t *testing.T) {
	d := NewData().Set("b", 2).Set("a", 1)
	keys := d.Keys()

	assert.Equal(t, 2, len(keys))
	assert.Contains(t, keys, "a")
	assert.Contains(t, keys, "b")
}

func TestData_IsEmpty_Size(t *testing.T) {
	d := NewData()
	assert.True(t, d.IsEmpty())
	assert.Equal(t, 0, d.Size())

	d = d.Set("key", "val")
	assert.False(t, d.IsEmpty())
	assert.Equal(t, 1, d.Size())
}

func TestData_Merge(t *testing.T) {
	d1 := NewData().Set("a", 1).Set("b", 2)
	d2 := NewData().Set("b", 99).Set("c", 3)

	merged := d1.Merge(d2)

	assert.Equal(t, 1, merged.Get("a"))
	assert.Equal(t, 99, merged.Get("b")) // d2 overrides
	assert.Equal(t, 3, merged.Get("c"))
}

func TestData_DeepCopy(t *testing.T) {
	d := NewData().Set("key", "value").Set("nested", map[string]interface{}{"inner": "data"})

	copied, err := d.DeepCopy()
	require.NoError(t, err)

	assert.Equal(t, "value", copied.Get("key"))
	// Ensure it's a different underlying map
	d2 := d.Set("key", "changed")
	assert.Equal(t, "value", copied.Get("key"))
	_ = d2
}

func TestData_MarshalJSON(t *testing.T) {
	d := NewData().Set("name", "Alice")
	b, err := json.Marshal(d)
	require.NoError(t, err)
	assert.Contains(t, string(b), `"name":"Alice"`)
}

func TestData_MarshalJSON_NilData(t *testing.T) {
	d := Data{} // nil internal map
	b, err := d.MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, "{}", string(b))
}

func TestData_UnmarshalJSON(t *testing.T) {
	var d Data
	err := json.Unmarshal([]byte(`{"name":"Bob","age":30}`), &d)
	require.NoError(t, err)

	val, ok := d.GetString("name")
	assert.True(t, ok)
	assert.Equal(t, "Bob", val)
}

func TestData_UnmarshalJSON_Invalid(t *testing.T) {
	var d Data
	err := json.Unmarshal([]byte(`not json`), &d)
	assert.Error(t, err)
}

func TestData_String(t *testing.T) {
	d := NewData().Set("key", "val")
	s := d.String()
	assert.Contains(t, s, "key")
	assert.Contains(t, s, "val")
}

func TestData_ToMap_ReturnsCopy(t *testing.T) {
	d := NewData().Set("a", 1)
	m := d.ToMap()
	m["a"] = 999

	// Original should be unaffected
	assert.Equal(t, 1, d.Get("a"))
}

// ---------------------------------------------------------------------------
// TransitionMetadata tests
// ---------------------------------------------------------------------------

func TestNewTransitionMetadata_DefensiveCopy(t *testing.T) {
	original := map[string]interface{}{"key": "value"}
	tm := NewTransitionMetadata("reason", "feedback", original)

	// Mutate original
	original["key"] = "changed"
	meta := tm.Metadata()
	assert.Equal(t, "value", meta["key"])
}

func TestTransitionMetadata_WithReason_Immutability(t *testing.T) {
	tm := EmptyTransitionMetadata()
	tm2 := tm.WithReason("new reason")

	assert.Equal(t, "", tm.Reason())
	assert.Equal(t, "new reason", tm2.Reason())
}

func TestTransitionMetadata_WithFeedback_Immutability(t *testing.T) {
	tm := EmptyTransitionMetadata()
	tm2 := tm.WithFeedback("great work")

	assert.Equal(t, "", tm.Feedback())
	assert.Equal(t, "great work", tm2.Feedback())
}

func TestTransitionMetadata_WithMetadata_Immutability(t *testing.T) {
	tm := EmptyTransitionMetadata()
	tm2 := tm.WithMetadata("priority", "high")

	_, ok := tm.GetMetadata("priority")
	assert.False(t, ok)

	val, ok := tm2.GetMetadata("priority")
	assert.True(t, ok)
	assert.Equal(t, "high", val)
}

func TestTransitionMetadata_IsEmpty(t *testing.T) {
	assert.True(t, EmptyTransitionMetadata().IsEmpty())
	assert.False(t, NewTransitionMetadataWithReason("reason").IsEmpty())
	assert.False(t, EmptyTransitionMetadata().WithFeedback("fb").IsEmpty())
	assert.False(t, EmptyTransitionMetadata().WithMetadata("k", "v").IsEmpty())
}

func TestTransitionMetadata_String(t *testing.T) {
	tm := NewTransitionMetadata("reason", "feedback", map[string]interface{}{"a": 1})
	s := tm.String()
	assert.Contains(t, s, "reason")
	assert.Contains(t, s, "feedback")
	assert.Contains(t, s, "metadata_count=1")
}

func TestTransitionMetadata_Validate_NilSchema(t *testing.T) {
	tm := EmptyTransitionMetadata()
	assert.NoError(t, tm.Validate(nil))
}

func TestTransitionMetadata_Validate_RequiredFieldMissing(t *testing.T) {
	schema := NewMetadataSchema().WithRequired("approval_doc")
	tm := EmptyTransitionMetadata()

	err := tm.Validate(schema)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "approval_doc")
}

func TestTransitionMetadata_Validate_RequiredFieldPresent(t *testing.T) {
	schema := NewMetadataSchema().WithRequired("approval_doc")
	tm := EmptyTransitionMetadata().WithMetadata("approval_doc", "doc.pdf")

	err := tm.Validate(schema)
	assert.NoError(t, err)
}

func TestMetadataSchema_TypeValidation(t *testing.T) {
	tests := []struct {
		name     string
		typeName string
		value    interface{}
		wantErr  bool
	}{
		{"string valid", "string", "hello", false},
		{"string invalid", "string", 42, true},
		{"int valid (int)", "int", 42, false},
		{"int valid (int64)", "int", int64(42), false},
		{"int valid (float64)", "int", float64(42), false},
		{"int invalid", "int", "not-a-number", true},
		{"bool valid", "bool", true, false},
		{"bool invalid", "bool", "true", true},
		{"object valid", "object", map[string]interface{}{"a": 1}, false},
		{"object invalid", "object", "not-an-object", true},
		{"array valid", "array", []interface{}{1, 2}, false},
		{"array invalid", "array", "not-an-array", true},
		{"unknown type", "unknown_type", "val", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := NewMetadataSchema().WithType("field", tt.typeName)
			tm := EmptyTransitionMetadata().WithMetadata("field", tt.value)

			err := tm.Validate(schema)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMetadataSchema_CustomValidator(t *testing.T) {
	schema := NewMetadataSchema().WithValidator("priority", func(value interface{}) error {
		s, ok := value.(string)
		if !ok || (s != "low" && s != "medium" && s != "high") {
			return assert.AnError
		}
		return nil
	})

	tm := EmptyTransitionMetadata().WithMetadata("priority", "high")
	assert.NoError(t, tm.Validate(schema))

	tm2 := EmptyTransitionMetadata().WithMetadata("priority", "critical")
	assert.Error(t, tm2.Validate(schema))
}

// ---------------------------------------------------------------------------
// Version tests
// ---------------------------------------------------------------------------

func TestNewVersion(t *testing.T) {
	v := NewVersion()
	assert.Equal(t, int64(1), v.Value())
	assert.False(t, v.IsZero())
}

func TestFromValue_Valid(t *testing.T) {
	v, err := FromValue(5)
	require.NoError(t, err)
	assert.Equal(t, int64(5), v.Value())
}

func TestFromValue_Invalid(t *testing.T) {
	_, err := FromValue(0)
	assert.Error(t, err)

	_, err = FromValue(-1)
	assert.Error(t, err)
}

func TestVersion_Increment(t *testing.T) {
	v := NewVersion()
	v2 := v.Increment()

	assert.Equal(t, int64(1), v.Value())  // immutable
	assert.Equal(t, int64(2), v2.Value())
}

func TestVersion_IsZero(t *testing.T) {
	assert.True(t, ZeroVersion().IsZero())
	assert.False(t, NewVersion().IsZero())
}

func TestVersion_String(t *testing.T) {
	v := NewVersion()
	assert.Equal(t, "v1", v.String())
}

func TestVersion_Equals(t *testing.T) {
	v1 := NewVersion()
	v2 := NewVersion()

	assert.True(t, v1.Equals(v2))
	assert.False(t, v1.Equals(v2.Increment()))
}

func TestVersion_IsGreaterThan_IsLessThan(t *testing.T) {
	v1 := NewVersion()
	v2 := v1.Increment()

	assert.True(t, v2.IsGreaterThan(v1))
	assert.False(t, v1.IsGreaterThan(v2))
	assert.True(t, v1.IsLessThan(v2))
	assert.False(t, v2.IsLessThan(v1))
}

// ---------------------------------------------------------------------------
// SubState tests
// ---------------------------------------------------------------------------

func TestNewSubState_Valid(t *testing.T) {
	ss, err := NewSubState("pending_review", "Pending Review")
	require.NoError(t, err)
	assert.Equal(t, "pending_review", ss.ID())
	assert.Equal(t, "Pending Review", ss.Name())
}

func TestNewSubState_EmptyID(t *testing.T) {
	_, err := NewSubState("", "Name")
	assert.Error(t, err)
}

func TestNewSubState_InvalidIDFormat(t *testing.T) {
	_, err := NewSubState("Invalid-ID", "Name")
	assert.Error(t, err)

	_, err = NewSubState("123start", "Name")
	assert.Error(t, err)
}

func TestNewSubState_EmptyName(t *testing.T) {
	_, err := NewSubState("valid_id", "")
	assert.Error(t, err)
}

func TestZeroSubState_IsZero(t *testing.T) {
	ss := ZeroSubState()
	assert.True(t, ss.IsZero())
	assert.Equal(t, "", ss.ID())
}

func TestSubState_Validate(t *testing.T) {
	ss, _ := NewSubState("valid_id", "Valid")
	assert.NoError(t, ss.Validate())

	// Zero sub-state validation fails
	zero := ZeroSubState()
	assert.Error(t, zero.Validate())

	// Restored with invalid format
	bad := RestoreSubState("BAD-FORMAT")
	assert.Error(t, bad.Validate())
}

func TestSubState_Equals(t *testing.T) {
	ss1, _ := NewSubState("review", "Review")
	ss2, _ := NewSubState("review", "Different Name")
	ss3, _ := NewSubState("approved", "Approved")

	assert.True(t, ss1.Equals(ss2))  // same ID
	assert.False(t, ss1.Equals(ss3)) // different ID
}

func TestSubState_WithDescription(t *testing.T) {
	ss, _ := NewSubState("review", "Review")
	ss2 := ss.WithDescription("Detailed description")

	assert.Equal(t, "", ss.Description())
	assert.Equal(t, "Detailed description", ss2.Description())
}

func TestSubState_String(t *testing.T) {
	ss, _ := NewSubState("review", "Review")
	s := ss.String()
	assert.Contains(t, s, "review")
	assert.Contains(t, s, "Review")
}

// ---------------------------------------------------------------------------
// Status tests
// ---------------------------------------------------------------------------

func TestStatus_IsActive(t *testing.T) {
	assert.True(t, StatusRunning.IsActive())
	assert.True(t, StatusPaused.IsActive())
	assert.False(t, StatusCompleted.IsActive())
	assert.False(t, StatusCanceled.IsActive())
	assert.False(t, StatusFailed.IsActive())
}

func TestStatus_IsFinal(t *testing.T) {
	assert.False(t, StatusRunning.IsFinal())
	assert.False(t, StatusPaused.IsFinal())
	assert.True(t, StatusCompleted.IsFinal())
	assert.True(t, StatusCanceled.IsFinal())
	assert.True(t, StatusFailed.IsFinal())
}

func TestStatus_IsValid(t *testing.T) {
	assert.True(t, StatusRunning.IsValid())
	assert.True(t, StatusPaused.IsValid())
	assert.True(t, StatusCompleted.IsValid())
	assert.True(t, StatusCanceled.IsValid())
	assert.True(t, StatusFailed.IsValid())
	assert.False(t, Status("INVALID").IsValid())
	assert.False(t, Status("").IsValid())
}

func TestStatus_String(t *testing.T) {
	assert.Equal(t, "RUNNING", StatusRunning.String())
	assert.Equal(t, "COMPLETED", StatusCompleted.String())
}

func TestParseStatus_Valid(t *testing.T) {
	s, err := ParseStatus("RUNNING")
	require.NoError(t, err)
	assert.Equal(t, StatusRunning, s)
}

func TestParseStatus_Invalid(t *testing.T) {
	_, err := ParseStatus("INVALID")
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// Variables tests
// ---------------------------------------------------------------------------

func TestNewVariables_Empty(t *testing.T) {
	v := NewVariables()
	assert.True(t, v.IsEmpty())
	assert.Equal(t, 0, v.Size())
}

func TestNewVariablesFromMap(t *testing.T) {
	m := map[string]interface{}{"key": "value"}
	v := NewVariablesFromMap(m)

	assert.Equal(t, "value", v.Get("key"))
	// Mutating original shouldn't affect variables
	m["key"] = "changed"
	assert.Equal(t, "value", v.Get("key"))
}

func TestVariables_SetDeleteHas(t *testing.T) {
	v := NewVariables().Set("a", 1).Set("b", 2)

	assert.True(t, v.Has("a"))
	assert.True(t, v.Has("b"))

	v2 := v.Delete("a")
	assert.False(t, v2.Has("a"))
	assert.True(t, v2.Has("b"))
	// Original unchanged
	assert.True(t, v.Has("a"))
}

func TestVariables_GetBool(t *testing.T) {
	v := NewVariables().Set("flag", true)
	val, ok := v.GetBool("flag")
	assert.True(t, ok)
	assert.True(t, val)
}

func TestVariables_Merge(t *testing.T) {
	v1 := NewVariables().Set("a", 1).Set("b", 2)
	v2 := NewVariables().Set("b", 99).Set("c", 3)

	merged := v1.Merge(v2)
	assert.Equal(t, 1, merged.Get("a"))
	assert.Equal(t, 99, merged.Get("b"))
	assert.Equal(t, 3, merged.Get("c"))
}

func TestVariables_DeepCopy(t *testing.T) {
	v := NewVariables().Set("key", "value")
	copied, err := v.DeepCopy()
	require.NoError(t, err)
	assert.Equal(t, "value", copied.Get("key"))
}

func TestVariables_MarshalUnmarshalJSON(t *testing.T) {
	v := NewVariables().Set("name", "test")
	b, err := json.Marshal(v)
	require.NoError(t, err)

	var v2 Variables
	err = json.Unmarshal(b, &v2)
	require.NoError(t, err)

	val, ok := v2.GetString("name")
	assert.True(t, ok)
	assert.Equal(t, "test", val)
}

func TestVariables_String(t *testing.T) {
	v := NewVariables().Set("key", "val")
	s := v.String()
	assert.Contains(t, s, "key")
}

func TestVariables_Keys(t *testing.T) {
	v := NewVariables().Set("x", 1).Set("y", 2)
	keys := v.Keys()
	assert.Equal(t, 2, len(keys))
	assert.Contains(t, keys, "x")
	assert.Contains(t, keys, "y")
}

func TestVariables_ToMap(t *testing.T) {
	v := NewVariables().Set("a", 1)
	m := v.ToMap()
	m["a"] = 999
	// Original unchanged
	assert.Equal(t, 1, v.Get("a"))
}

// ---------------------------------------------------------------------------
// Transition entity tests
// ---------------------------------------------------------------------------

func TestNewTransition(t *testing.T) {
	actor := shared.NewID()
	tr := NewTransition("draft", "review", "submit", actor)

	assert.True(t, tr.ID().IsValid())
	assert.Equal(t, "draft", tr.From())
	assert.Equal(t, "review", tr.To())
	assert.Equal(t, "submit", tr.Event())
	assert.Equal(t, actor, tr.Actor())
	assert.False(t, tr.HasSubStates())
}

func TestNewTransitionWithSubStates(t *testing.T) {
	actor := shared.NewID()
	fromSS, _ := NewSubState("pending_review", "Pending Review")
	toSS, _ := NewSubState("pending_approval", "Pending Approval")

	tr := NewTransitionWithSubStates("pending", "pending", "escalate", actor, fromSS, toSS)

	assert.True(t, tr.HasSubStates())
	assert.Equal(t, "pending_review", tr.FromSubState().ID())
	assert.Equal(t, "pending_approval", tr.ToSubState().ID())
}

func TestTransition_String_WithoutSubStates(t *testing.T) {
	actor := shared.NewID()
	tr := NewTransition("draft", "review", "submit", actor)
	s := tr.String()
	assert.Contains(t, s, "draft -> review (submit)")
}

func TestTransition_String_WithSubStates(t *testing.T) {
	actor := shared.NewID()
	fromSS, _ := NewSubState("sub_a", "Sub A")
	toSS, _ := NewSubState("sub_b", "Sub B")
	tr := NewTransitionWithSubStates("state1", "state2", "event", actor, fromSS, toSS)
	s := tr.String()
	assert.Contains(t, s, "state1.sub_a -> state2.sub_b (event)")
}

func TestTransition_SetMetadata(t *testing.T) {
	actor := shared.NewID()
	tr := NewTransition("a", "b", "ev", actor)

	tm := NewTransitionMetadataWithReason("test reason")
	tr.SetMetadata(tm)

	assert.Equal(t, "test reason", tr.Metadata().Reason())
}

func TestTransition_SetData(t *testing.T) {
	actor := shared.NewID()
	tr := NewTransition("a", "b", "ev", actor)

	d := NewData().Set("snapshot", "value")
	tr.SetData(d)

	assert.Equal(t, "value", tr.Data().Get("snapshot"))
}

func TestTransition_Duration(t *testing.T) {
	actor := shared.NewID()
	tr := NewTransition("a", "b", "ev", actor)
	// Duration should be >= 0 (just created)
	assert.True(t, tr.Duration() >= 0)
}
