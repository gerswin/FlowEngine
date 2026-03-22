package actor

import (
	"testing"

	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewActor_Success(t *testing.T) {
	id := shared.NewID()
	roles := []string{RoleAdmin, RoleOperator}

	a := NewActor(id, "Alice", roles)

	require.NotNil(t, a)
	assert.Equal(t, id, a.ID())
	assert.Equal(t, "Alice", a.Name())
	assert.Equal(t, roles, a.Roles())
}

func TestNewActor_DefensiveCopyOfInputRoles(t *testing.T) {
	id := shared.NewID()
	roles := []string{RoleAdmin, RoleOperator}

	a := NewActor(id, "Alice", roles)

	// Mutating the original slice should not affect the actor.
	roles[0] = "tampered"

	assert.Equal(t, RoleAdmin, a.Roles()[0])
}

func TestRoles_ReturnsDefensiveCopy(t *testing.T) {
	a := NewActor(shared.NewID(), "Alice", []string{RoleAdmin, RoleOperator})

	returned := a.Roles()
	returned[0] = "tampered"

	assert.Equal(t, RoleAdmin, a.Roles()[0], "mutating returned slice must not affect actor internals")
}

func TestRestoreActor(t *testing.T) {
	id := shared.NewID()
	a := RestoreActor(id, "Bob", []string{RoleReviewer})

	assert.Equal(t, id, a.ID())
	assert.Equal(t, "Bob", a.Name())
	assert.Equal(t, []string{RoleReviewer}, a.Roles())
}

func TestHasRole(t *testing.T) {
	a := NewActor(shared.NewID(), "Alice", []string{RoleAdmin, RoleOperator})

	tests := []struct {
		name string
		role string
		want bool
	}{
		{"has admin", RoleAdmin, true},
		{"has operator", RoleOperator, true},
		{"does not have reviewer", RoleReviewer, false},
		{"does not have unknown role", "unknown", false},
		{"empty string role", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, a.HasRole(tt.role))
		})
	}
}

func TestHasAnyRole(t *testing.T) {
	a := NewActor(shared.NewID(), "Alice", []string{RoleAdmin, RoleOperator})

	tests := []struct {
		name  string
		roles []string
		want  bool
	}{
		{"matches first role", []string{RoleAdmin}, true},
		{"matches second role", []string{RoleOperator}, true},
		{"matches one of several", []string{RoleReviewer, RoleOperator}, true},
		{"no match at all", []string{RoleReviewer, RoleApprover}, false},
		{"empty roles slice", []string{}, false},
		{"nil roles slice", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, a.HasAnyRole(tt.roles))
		})
	}
}

func TestHasPermission_PerRole(t *testing.T) {
	tests := []struct {
		name string
		role string
		perm Permission
		want bool
	}{
		// Admin has all permissions.
		{"admin can create workflow", RoleAdmin, PermCreateWorkflow, true},
		{"admin can read workflow", RoleAdmin, PermReadWorkflow, true},
		{"admin can delete workflow", RoleAdmin, PermDeleteWorkflow, true},
		{"admin can create instance", RoleAdmin, PermCreateInstance, true},
		{"admin can read instance", RoleAdmin, PermReadInstance, true},
		{"admin can transition", RoleAdmin, PermTransition, true},
		{"admin can clone instance", RoleAdmin, PermCloneInstance, true},
		{"admin can delete instance", RoleAdmin, PermDeleteInstance, true},

		// Operator permissions.
		{"operator can read workflow", RoleOperator, PermReadWorkflow, true},
		{"operator can create instance", RoleOperator, PermCreateInstance, true},
		{"operator can clone instance", RoleOperator, PermCloneInstance, true},
		{"operator cannot create workflow", RoleOperator, PermCreateWorkflow, false},
		{"operator cannot delete workflow", RoleOperator, PermDeleteWorkflow, false},
		{"operator cannot delete instance", RoleOperator, PermDeleteInstance, false},

		// Reviewer permissions.
		{"reviewer can read workflow", RoleReviewer, PermReadWorkflow, true},
		{"reviewer can read instance", RoleReviewer, PermReadInstance, true},
		{"reviewer can transition", RoleReviewer, PermTransition, true},
		{"reviewer cannot create workflow", RoleReviewer, PermCreateWorkflow, false},
		{"reviewer cannot clone instance", RoleReviewer, PermCloneInstance, false},

		// Approver permissions.
		{"approver can transition", RoleApprover, PermTransition, true},
		{"approver cannot create instance", RoleApprover, PermCreateInstance, false},

		// Supervisor permissions.
		{"supervisor can clone instance", RoleSupervisor, PermCloneInstance, true},
		{"supervisor cannot create workflow", RoleSupervisor, PermCreateWorkflow, false},
		{"supervisor cannot delete instance", RoleSupervisor, PermDeleteInstance, false},

		// Auditor only has read permissions.
		{"auditor can read workflow", RoleAuditor, PermReadWorkflow, true},
		{"auditor can read instance", RoleAuditor, PermReadInstance, true},
		{"auditor cannot create workflow", RoleAuditor, PermCreateWorkflow, false},
		{"auditor cannot delete workflow", RoleAuditor, PermDeleteWorkflow, false},
		{"auditor cannot create instance", RoleAuditor, PermCreateInstance, false},
		{"auditor cannot transition", RoleAuditor, PermTransition, false},
		{"auditor cannot clone instance", RoleAuditor, PermCloneInstance, false},
		{"auditor cannot delete instance", RoleAuditor, PermDeleteInstance, false},

		// System permissions.
		{"system can create workflow", RoleSystem, PermCreateWorkflow, true},
		{"system can read workflow", RoleSystem, PermReadWorkflow, true},
		{"system cannot delete workflow", RoleSystem, PermDeleteWorkflow, false},
		{"system cannot delete instance", RoleSystem, PermDeleteInstance, false},

		// Unknown role has no permissions.
		{"unknown cannot create workflow", "unknown", PermCreateWorkflow, false},
		{"unknown cannot read workflow", "unknown", PermReadWorkflow, false},
		{"unknown cannot delete workflow", "unknown", PermDeleteWorkflow, false},
		{"unknown cannot create instance", "unknown", PermCreateInstance, false},
		{"unknown cannot read instance", "unknown", PermReadInstance, false},
		{"unknown cannot transition", "unknown", PermTransition, false},
		{"unknown cannot clone instance", "unknown", PermCloneInstance, false},
		{"unknown cannot delete instance", "unknown", PermDeleteInstance, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, HasPermission(tt.role, tt.perm))
		})
	}
}

func TestHasPermission_AdminHasAllPermissions(t *testing.T) {
	allPerms := []Permission{
		PermCreateWorkflow, PermReadWorkflow, PermDeleteWorkflow,
		PermCreateInstance, PermReadInstance, PermTransition, PermCloneInstance, PermDeleteInstance,
	}

	for _, perm := range allPerms {
		assert.True(t, HasPermission(RoleAdmin, perm), "admin should have permission %s", perm)
	}
}

func TestHasPermission_AuditorOnlyReadPermissions(t *testing.T) {
	readPerms := []Permission{PermReadWorkflow, PermReadInstance}
	writePerms := []Permission{
		PermCreateWorkflow, PermDeleteWorkflow,
		PermCreateInstance, PermTransition, PermCloneInstance, PermDeleteInstance,
	}

	for _, perm := range readPerms {
		assert.True(t, HasPermission(RoleAuditor, perm), "auditor should have read permission %s", perm)
	}
	for _, perm := range writePerms {
		assert.False(t, HasPermission(RoleAuditor, perm), "auditor should not have write permission %s", perm)
	}
}

func TestHasPermission_UnknownRoleHasNoPermissions(t *testing.T) {
	allPerms := []Permission{
		PermCreateWorkflow, PermReadWorkflow, PermDeleteWorkflow,
		PermCreateInstance, PermReadInstance, PermTransition, PermCloneInstance, PermDeleteInstance,
	}

	for _, perm := range allPerms {
		assert.False(t, HasPermission("nonexistent", perm), "unknown role should not have permission %s", perm)
	}
}

func TestHasAnyPermission(t *testing.T) {
	tests := []struct {
		name  string
		roles []string
		perm  Permission
		want  bool
	}{
		{"single role with permission", []string{RoleAdmin}, PermCreateWorkflow, true},
		{"single role without permission", []string{RoleAuditor}, PermCreateWorkflow, false},
		{"multiple roles one has permission", []string{RoleAuditor, RoleOperator}, PermCreateInstance, true},
		{"multiple roles none has permission", []string{RoleAuditor, RoleReviewer}, PermCreateWorkflow, false},
		{"empty roles", []string{}, PermReadWorkflow, false},
		{"nil roles", nil, PermReadWorkflow, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, HasAnyPermission(tt.roles, tt.perm))
		})
	}
}
