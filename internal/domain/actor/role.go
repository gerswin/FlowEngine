package actor

// Predefined roles for the system.
const (
	RoleAdmin      = "admin"
	RoleOperator   = "operator"
	RoleReviewer   = "reviewer"
	RoleApprover   = "approver"
	RoleSupervisor = "supervisor"
	RoleAuditor    = "auditor"
	RoleSystem     = "system"
)

// Permission represents an action that can be performed.
type Permission string

const (
	PermCreateWorkflow Permission = "workflow:create"
	PermReadWorkflow   Permission = "workflow:read"
	PermDeleteWorkflow Permission = "workflow:delete"
	PermCreateInstance Permission = "instance:create"
	PermReadInstance   Permission = "instance:read"
	PermTransition     Permission = "instance:transition"
	PermCloneInstance  Permission = "instance:clone"
	PermDeleteInstance Permission = "instance:delete"
)

// RolePermissions maps roles to their allowed permissions.
var RolePermissions = map[string][]Permission{
	RoleAdmin: {
		PermCreateWorkflow, PermReadWorkflow, PermDeleteWorkflow,
		PermCreateInstance, PermReadInstance, PermTransition, PermCloneInstance, PermDeleteInstance,
	},
	RoleOperator: {
		PermReadWorkflow,
		PermCreateInstance, PermReadInstance, PermTransition, PermCloneInstance,
	},
	RoleReviewer: {
		PermReadWorkflow, PermReadInstance, PermTransition,
	},
	RoleApprover: {
		PermReadWorkflow, PermReadInstance, PermTransition,
	},
	RoleSupervisor: {
		PermReadWorkflow,
		PermReadInstance, PermTransition, PermCloneInstance,
	},
	RoleAuditor: {
		PermReadWorkflow, PermReadInstance,
	},
	RoleSystem: {
		PermCreateWorkflow, PermReadWorkflow,
		PermCreateInstance, PermReadInstance, PermTransition, PermCloneInstance,
	},
}

// HasPermission checks if a role has a specific permission.
func HasPermission(role string, perm Permission) bool {
	perms, ok := RolePermissions[role]
	if !ok {
		return false
	}
	for _, p := range perms {
		if p == perm {
			return true
		}
	}
	return false
}

// HasAnyPermission checks if any of the roles has the permission.
func HasAnyPermission(roles []string, perm Permission) bool {
	for _, role := range roles {
		if HasPermission(role, perm) {
			return true
		}
	}
	return false
}
