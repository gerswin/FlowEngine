package memory

import (
	"context"
	"testing"
	"time"

	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/instance"
	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/shared"
	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/timer"
	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Helpers ---

func createTestInstance(t *testing.T) *instance.Instance {
	t.Helper()
	workflowID := shared.NewID()
	actor := shared.NewID()
	inst, err := instance.NewInstance(workflowID, "Test Workflow", "draft", actor)
	require.NoError(t, err)
	return inst
}

func createTestInstanceWithWorkflowID(t *testing.T, workflowID shared.ID) *instance.Instance {
	t.Helper()
	actor := shared.NewID()
	inst, err := instance.NewInstance(workflowID, "Test Workflow", "draft", actor)
	require.NoError(t, err)
	return inst
}

func createTestWorkflow(t *testing.T, name string) *workflow.Workflow {
	t.Helper()
	state, err := workflow.NewState("draft", "Draft")
	require.NoError(t, err)
	actor := shared.NewID()
	wf, err := workflow.NewWorkflow(name, state, actor)
	require.NoError(t, err)
	return wf
}

func createTestTimer(t *testing.T, instanceID shared.ID, duration time.Duration) *timer.Timer {
	t.Helper()
	return timer.NewTimer(instanceID, "pending", "timeout_event", duration)
}

// --- InstanceInMemoryRepository Tests ---

func TestInstanceRepo_SaveAndFindByID(t *testing.T) {
	repo := NewInstanceInMemoryRepository()
	ctx := context.Background()
	inst := createTestInstance(t)

	err := repo.Save(ctx, inst)
	require.NoError(t, err)

	found, err := repo.FindByID(ctx, inst.ID())
	require.NoError(t, err)
	assert.Equal(t, inst.ID(), found.ID())
	assert.Equal(t, inst.WorkflowName(), found.WorkflowName())
}

func TestInstanceRepo_FindByID_NotFound(t *testing.T) {
	repo := NewInstanceInMemoryRepository()
	ctx := context.Background()

	_, err := repo.FindByID(ctx, shared.NewID())
	assert.Error(t, err)
}

func TestInstanceRepo_FindByWorkflowID_FiltersCorrectly(t *testing.T) {
	repo := NewInstanceInMemoryRepository()
	ctx := context.Background()
	workflowID := shared.NewID()

	inst1 := createTestInstanceWithWorkflowID(t, workflowID)
	inst2 := createTestInstanceWithWorkflowID(t, workflowID)
	inst3 := createTestInstance(t) // different workflow

	require.NoError(t, repo.Save(ctx, inst1))
	require.NoError(t, repo.Save(ctx, inst2))
	require.NoError(t, repo.Save(ctx, inst3))

	results, err := repo.FindByWorkflowID(ctx, workflowID)
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestInstanceRepo_FindByStatus_FiltersCorrectly(t *testing.T) {
	repo := NewInstanceInMemoryRepository()
	ctx := context.Background()

	inst1 := createTestInstance(t) // StatusRunning
	inst2 := createTestInstance(t) // StatusRunning
	inst3 := createTestInstance(t)
	require.NoError(t, inst3.Complete(shared.NewID())) // StatusCompleted

	require.NoError(t, repo.Save(ctx, inst1))
	require.NoError(t, repo.Save(ctx, inst2))
	require.NoError(t, repo.Save(ctx, inst3))

	running, err := repo.FindByStatus(ctx, instance.StatusRunning)
	require.NoError(t, err)
	assert.Len(t, running, 2)

	completed, err := repo.FindByStatus(ctx, instance.StatusCompleted)
	require.NoError(t, err)
	assert.Len(t, completed, 1)
}

func TestInstanceRepo_FindAll(t *testing.T) {
	repo := NewInstanceInMemoryRepository()
	ctx := context.Background()

	inst1 := createTestInstance(t)
	inst2 := createTestInstance(t)
	inst3 := createTestInstance(t)

	require.NoError(t, repo.Save(ctx, inst1))
	require.NoError(t, repo.Save(ctx, inst2))
	require.NoError(t, repo.Save(ctx, inst3))

	all, err := repo.FindAll(ctx)
	require.NoError(t, err)
	assert.Len(t, all, 3)
}

func TestInstanceRepo_List_WithPagination(t *testing.T) {
	repo := NewInstanceInMemoryRepository()
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		inst := createTestInstance(t)
		require.NoError(t, repo.Save(ctx, inst))
	}

	q := shared.ListQuery{Offset: 0, Limit: 2}
	results, total, err := repo.List(ctx, q, nil)
	require.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, int64(5), total)

	q2 := shared.ListQuery{Offset: 3, Limit: 10}
	results2, total2, err := repo.List(ctx, q2, nil)
	require.NoError(t, err)
	assert.Len(t, results2, 2) // only 2 remaining after offset 3
	assert.Equal(t, int64(5), total2)
}

func TestInstanceRepo_List_WithWorkflowIDFilter(t *testing.T) {
	repo := NewInstanceInMemoryRepository()
	ctx := context.Background()
	workflowID := shared.NewID()

	inst1 := createTestInstanceWithWorkflowID(t, workflowID)
	inst2 := createTestInstanceWithWorkflowID(t, workflowID)
	inst3 := createTestInstance(t) // different workflow

	require.NoError(t, repo.Save(ctx, inst1))
	require.NoError(t, repo.Save(ctx, inst2))
	require.NoError(t, repo.Save(ctx, inst3))

	q := shared.ListQuery{Offset: 0, Limit: 10}
	results, total, err := repo.List(ctx, q, &workflowID)
	require.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, int64(2), total)
}

func TestInstanceRepo_Delete(t *testing.T) {
	repo := NewInstanceInMemoryRepository()
	ctx := context.Background()
	inst := createTestInstance(t)

	require.NoError(t, repo.Save(ctx, inst))

	err := repo.Delete(ctx, inst.ID())
	require.NoError(t, err)

	_, err = repo.FindByID(ctx, inst.ID())
	assert.Error(t, err)
}

func TestInstanceRepo_Delete_NotFound(t *testing.T) {
	repo := NewInstanceInMemoryRepository()
	ctx := context.Background()

	err := repo.Delete(ctx, shared.NewID())
	assert.Error(t, err)
}

func TestInstanceRepo_Exists(t *testing.T) {
	repo := NewInstanceInMemoryRepository()
	ctx := context.Background()
	inst := createTestInstance(t)

	exists, err := repo.Exists(ctx, inst.ID())
	require.NoError(t, err)
	assert.False(t, exists)

	require.NoError(t, repo.Save(ctx, inst))

	exists, err = repo.Exists(ctx, inst.ID())
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestInstanceRepo_Count(t *testing.T) {
	repo := NewInstanceInMemoryRepository()
	ctx := context.Background()

	count, err := repo.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)

	require.NoError(t, repo.Save(ctx, createTestInstance(t)))
	require.NoError(t, repo.Save(ctx, createTestInstance(t)))

	count, err = repo.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

// --- WorkflowInMemoryRepository Tests ---

func TestWorkflowRepo_SaveAndFindByID(t *testing.T) {
	repo := NewWorkflowInMemoryRepository()
	ctx := context.Background()
	wf := createTestWorkflow(t, "approval")

	err := repo.Save(ctx, wf)
	require.NoError(t, err)

	found, err := repo.FindByID(ctx, wf.ID())
	require.NoError(t, err)
	assert.Equal(t, wf.ID(), found.ID())
	assert.Equal(t, "approval", found.Name())
}

func TestWorkflowRepo_FindByID_NotFound(t *testing.T) {
	repo := NewWorkflowInMemoryRepository()
	ctx := context.Background()

	_, err := repo.FindByID(ctx, shared.NewID())
	assert.Error(t, err)
}

func TestWorkflowRepo_FindAll(t *testing.T) {
	repo := NewWorkflowInMemoryRepository()
	ctx := context.Background()

	require.NoError(t, repo.Save(ctx, createTestWorkflow(t, "wf1")))
	require.NoError(t, repo.Save(ctx, createTestWorkflow(t, "wf2")))
	require.NoError(t, repo.Save(ctx, createTestWorkflow(t, "wf3")))

	all, err := repo.FindAll(ctx)
	require.NoError(t, err)
	assert.Len(t, all, 3)
}

func TestWorkflowRepo_FindByName_WithVersion(t *testing.T) {
	repo := NewWorkflowInMemoryRepository()
	ctx := context.Background()
	wf := createTestWorkflow(t, "approval")

	require.NoError(t, repo.Save(ctx, wf))

	found, err := repo.FindByName(ctx, "approval", wf.Version())
	require.NoError(t, err)
	assert.Equal(t, wf.ID(), found.ID())

	// Not found with wrong version
	v2, _ := workflow.NewVersion(2, 0, 0)
	_, err = repo.FindByName(ctx, "approval", v2)
	assert.Error(t, err)
}

func TestWorkflowRepo_FindLatestByName(t *testing.T) {
	repo := NewWorkflowInMemoryRepository()
	ctx := context.Background()

	wf1 := createTestWorkflow(t, "approval")
	require.NoError(t, repo.Save(ctx, wf1))

	wf2 := createTestWorkflow(t, "approval")
	require.NoError(t, wf2.IncrementVersion("minor")) // 1.1.0
	require.NoError(t, repo.Save(ctx, wf2))

	wf3 := createTestWorkflow(t, "approval")
	require.NoError(t, wf3.IncrementVersion("major")) // 2.0.0
	require.NoError(t, repo.Save(ctx, wf3))

	latest, err := repo.FindLatestByName(ctx, "approval")
	require.NoError(t, err)
	assert.Equal(t, wf3.ID(), latest.ID())

	// Not found for nonexistent name
	_, err = repo.FindLatestByName(ctx, "nonexistent")
	assert.Error(t, err)
}

func TestWorkflowRepo_List_WithPagination(t *testing.T) {
	repo := NewWorkflowInMemoryRepository()
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		require.NoError(t, repo.Save(ctx, createTestWorkflow(t, "wf")))
	}

	q := shared.ListQuery{Offset: 0, Limit: 3}
	results, total, err := repo.List(ctx, q)
	require.NoError(t, err)
	assert.Len(t, results, 3)
	assert.Equal(t, int64(5), total)

	q2 := shared.ListQuery{Offset: 4, Limit: 10}
	results2, _, err := repo.List(ctx, q2)
	require.NoError(t, err)
	assert.Len(t, results2, 1)
}

func TestWorkflowRepo_Delete(t *testing.T) {
	repo := NewWorkflowInMemoryRepository()
	ctx := context.Background()
	wf := createTestWorkflow(t, "approval")

	require.NoError(t, repo.Save(ctx, wf))

	err := repo.Delete(ctx, wf.ID())
	require.NoError(t, err)

	_, err = repo.FindByID(ctx, wf.ID())
	assert.Error(t, err)
}

func TestWorkflowRepo_Delete_NotFound(t *testing.T) {
	repo := NewWorkflowInMemoryRepository()
	ctx := context.Background()

	err := repo.Delete(ctx, shared.NewID())
	assert.Error(t, err)
}

func TestWorkflowRepo_Exists(t *testing.T) {
	repo := NewWorkflowInMemoryRepository()
	ctx := context.Background()
	wf := createTestWorkflow(t, "approval")

	exists, err := repo.Exists(ctx, wf.ID())
	require.NoError(t, err)
	assert.False(t, exists)

	require.NoError(t, repo.Save(ctx, wf))

	exists, err = repo.Exists(ctx, wf.ID())
	require.NoError(t, err)
	assert.True(t, exists)
}

// --- TimerInMemoryRepository Tests ---

func TestTimerRepo_SaveAndFindPending(t *testing.T) {
	repo := NewTimerInMemoryRepository()
	ctx := context.Background()
	instanceID := shared.NewID()

	// Create a timer that expires immediately (negative duration)
	tmr := createTestTimer(t, instanceID, -1*time.Second)

	require.NoError(t, repo.Save(ctx, tmr))

	pending, err := repo.FindPending(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, pending, 1)
	assert.Equal(t, tmr.ID(), pending[0].ID())
}

func TestTimerRepo_FindPending_OnlyReturnsDueTimers(t *testing.T) {
	repo := NewTimerInMemoryRepository()
	ctx := context.Background()
	instanceID := shared.NewID()

	// Expired timer (due)
	expired := createTestTimer(t, instanceID, -1*time.Second)
	require.NoError(t, repo.Save(ctx, expired))

	// Future timer (not due)
	future := createTestTimer(t, instanceID, 1*time.Hour)
	require.NoError(t, repo.Save(ctx, future))

	pending, err := repo.FindPending(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, pending, 1)
	assert.Equal(t, expired.ID(), pending[0].ID())
}

func TestTimerRepo_FindPending_RespectsLimit(t *testing.T) {
	repo := NewTimerInMemoryRepository()
	ctx := context.Background()
	instanceID := shared.NewID()

	for i := 0; i < 5; i++ {
		tmr := createTestTimer(t, instanceID, -1*time.Second)
		require.NoError(t, repo.Save(ctx, tmr))
	}

	pending, err := repo.FindPending(ctx, 2)
	require.NoError(t, err)
	assert.Len(t, pending, 2)
}

func TestTimerRepo_Delete(t *testing.T) {
	repo := NewTimerInMemoryRepository()
	ctx := context.Background()
	instanceID := shared.NewID()

	tmr := createTestTimer(t, instanceID, -1*time.Second)
	require.NoError(t, repo.Save(ctx, tmr))

	err := repo.Delete(ctx, tmr.ID())
	require.NoError(t, err)

	pending, err := repo.FindPending(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, pending, 0)
}

func TestTimerRepo_DeleteByInstanceID(t *testing.T) {
	repo := NewTimerInMemoryRepository()
	ctx := context.Background()

	instanceID1 := shared.NewID()
	instanceID2 := shared.NewID()

	// Two timers for instanceID1
	tmr1 := createTestTimer(t, instanceID1, -1*time.Second)
	tmr2 := createTestTimer(t, instanceID1, -1*time.Second)
	// One timer for instanceID2
	tmr3 := createTestTimer(t, instanceID2, -1*time.Second)

	require.NoError(t, repo.Save(ctx, tmr1))
	require.NoError(t, repo.Save(ctx, tmr2))
	require.NoError(t, repo.Save(ctx, tmr3))

	err := repo.DeleteByInstanceID(ctx, instanceID1)
	require.NoError(t, err)

	// Only instanceID2 timer should remain
	pending, err := repo.FindPending(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, pending, 1)
	assert.Equal(t, tmr3.ID(), pending[0].ID())
}

func TestInstanceRepo_SaveNil(t *testing.T) {
	repo := NewInstanceInMemoryRepository()
	ctx := context.Background()

	err := repo.Save(ctx, nil)
	assert.Error(t, err)
}

func TestInstanceRepo_FindByParentID(t *testing.T) {
	repo := NewInstanceInMemoryRepository()
	ctx := context.Background()

	parentID := shared.NewID()
	child1, err := instance.NewSubprocess(parentID, shared.NewID(), "Test", "draft", shared.NewID())
	require.NoError(t, err)

	child2, err := instance.NewSubprocess(parentID, shared.NewID(), "Test", "draft", shared.NewID())
	require.NoError(t, err)

	other := createTestInstance(t) // No parent set

	require.NoError(t, repo.Save(ctx, child1))
	require.NoError(t, repo.Save(ctx, child2))
	require.NoError(t, repo.Save(ctx, other))

	results, err := repo.FindByParentID(ctx, parentID)
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestInstanceRepo_FindActive(t *testing.T) {
	repo := NewInstanceInMemoryRepository()
	ctx := context.Background()

	active := createTestInstance(t) // Running by default
	completed := createTestInstance(t)
	require.NoError(t, completed.Complete(shared.NewID()))

	require.NoError(t, repo.Save(ctx, active))
	require.NoError(t, repo.Save(ctx, completed))

	results, err := repo.FindActive(ctx)
	require.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestInstanceRepo_CountByWorkflowID(t *testing.T) {
	repo := NewInstanceInMemoryRepository()
	ctx := context.Background()
	workflowID := shared.NewID()

	inst1 := createTestInstanceWithWorkflowID(t, workflowID)
	inst2 := createTestInstanceWithWorkflowID(t, workflowID)
	inst3 := createTestInstance(t) // different workflow

	require.NoError(t, repo.Save(ctx, inst1))
	require.NoError(t, repo.Save(ctx, inst2))
	require.NoError(t, repo.Save(ctx, inst3))

	count, err := repo.CountByWorkflowID(ctx, workflowID)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestInstanceRepo_CountByStatus(t *testing.T) {
	repo := NewInstanceInMemoryRepository()
	ctx := context.Background()

	inst1 := createTestInstance(t) // Running
	inst2 := createTestInstance(t) // Running
	inst3 := createTestInstance(t)
	require.NoError(t, inst3.Complete(shared.NewID()))

	require.NoError(t, repo.Save(ctx, inst1))
	require.NoError(t, repo.Save(ctx, inst2))
	require.NoError(t, repo.Save(ctx, inst3))

	runningCount, err := repo.CountByStatus(ctx, instance.StatusRunning)
	require.NoError(t, err)
	assert.Equal(t, int64(2), runningCount)

	completedCount, err := repo.CountByStatus(ctx, instance.StatusCompleted)
	require.NoError(t, err)
	assert.Equal(t, int64(1), completedCount)
}

func TestInstanceRepo_FindByWorkflowIDAndStatus(t *testing.T) {
	repo := NewInstanceInMemoryRepository()
	ctx := context.Background()
	workflowID := shared.NewID()

	running := createTestInstanceWithWorkflowID(t, workflowID)
	completed := createTestInstanceWithWorkflowID(t, workflowID)
	require.NoError(t, completed.Complete(shared.NewID()))

	require.NoError(t, repo.Save(ctx, running))
	require.NoError(t, repo.Save(ctx, completed))

	results, err := repo.FindByWorkflowIDAndStatus(ctx, workflowID, instance.StatusRunning)
	require.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestInstanceRepo_Clear(t *testing.T) {
	repo := NewInstanceInMemoryRepository()
	ctx := context.Background()

	require.NoError(t, repo.Save(ctx, createTestInstance(t)))
	require.NoError(t, repo.Save(ctx, createTestInstance(t)))

	repo.Clear()

	all, err := repo.FindAll(ctx)
	require.NoError(t, err)
	assert.Len(t, all, 0)
}

func TestInstanceRepo_List_OffsetBeyondTotal(t *testing.T) {
	repo := NewInstanceInMemoryRepository()
	ctx := context.Background()

	require.NoError(t, repo.Save(ctx, createTestInstance(t)))

	q := shared.ListQuery{Offset: 100, Limit: 10}
	results, total, err := repo.List(ctx, q, nil)
	require.NoError(t, err)
	assert.Len(t, results, 0)
	assert.Equal(t, int64(1), total)
}

func TestWorkflowRepo_SaveNil(t *testing.T) {
	repo := NewWorkflowInMemoryRepository()
	ctx := context.Background()

	err := repo.Save(ctx, nil)
	assert.Error(t, err)
}

func TestWorkflowRepo_FindAllByName(t *testing.T) {
	repo := NewWorkflowInMemoryRepository()
	ctx := context.Background()

	wf1 := createTestWorkflow(t, "approval")
	wf2 := createTestWorkflow(t, "approval")
	wf3 := createTestWorkflow(t, "other")

	require.NoError(t, repo.Save(ctx, wf1))
	require.NoError(t, repo.Save(ctx, wf2))
	require.NoError(t, repo.Save(ctx, wf3))

	results, err := repo.FindAllByName(ctx, "approval")
	require.NoError(t, err)
	assert.Len(t, results, 2)

	// Non-existent name returns empty
	empty, err := repo.FindAllByName(ctx, "nonexistent")
	require.NoError(t, err)
	assert.Len(t, empty, 0)
}

func TestWorkflowRepo_ExistsByName(t *testing.T) {
	repo := NewWorkflowInMemoryRepository()
	ctx := context.Background()

	wf := createTestWorkflow(t, "approval")
	require.NoError(t, repo.Save(ctx, wf))

	exists, err := repo.ExistsByName(ctx, "approval", wf.Version())
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = repo.ExistsByName(ctx, "nonexistent", wf.Version())
	require.NoError(t, err)
	assert.False(t, exists)

	// Wrong version
	v2, _ := workflow.NewVersion(2, 0, 0)
	exists, err = repo.ExistsByName(ctx, "approval", v2)
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestWorkflowRepo_Count(t *testing.T) {
	repo := NewWorkflowInMemoryRepository()

	assert.Equal(t, 0, repo.Count())

	ctx := context.Background()
	require.NoError(t, repo.Save(ctx, createTestWorkflow(t, "wf1")))
	require.NoError(t, repo.Save(ctx, createTestWorkflow(t, "wf2")))

	assert.Equal(t, 2, repo.Count())
}

func TestWorkflowRepo_Clear(t *testing.T) {
	repo := NewWorkflowInMemoryRepository()
	ctx := context.Background()

	require.NoError(t, repo.Save(ctx, createTestWorkflow(t, "wf1")))
	require.NoError(t, repo.Save(ctx, createTestWorkflow(t, "wf2")))

	repo.Clear()
	assert.Equal(t, 0, repo.Count())
}

func TestWorkflowRepo_List_OffsetBeyondTotal(t *testing.T) {
	repo := NewWorkflowInMemoryRepository()
	ctx := context.Background()

	require.NoError(t, repo.Save(ctx, createTestWorkflow(t, "wf")))

	q := shared.ListQuery{Offset: 100, Limit: 10}
	results, total, err := repo.List(ctx, q)
	require.NoError(t, err)
	assert.Len(t, results, 0)
	assert.Equal(t, int64(1), total)
}

func TestTimerRepo_FindPending_SkipsFiredTimers(t *testing.T) {
	repo := NewTimerInMemoryRepository()
	ctx := context.Background()
	instanceID := shared.NewID()

	tmr := createTestTimer(t, instanceID, -1*time.Second)
	tmr.MarkFired()
	require.NoError(t, repo.Save(ctx, tmr))

	pending, err := repo.FindPending(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, pending, 0)
}
