package actor

import "github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/shared"

// Actor represents a user or system that can perform actions in workflows.
type Actor struct {
	id    shared.ID
	name  string
	roles []string
}

// NewActor creates a new Actor with a defensive copy of roles.
func NewActor(id shared.ID, name string, roles []string) *Actor {
	rolesCopy := make([]string, len(roles))
	copy(rolesCopy, roles)
	return &Actor{id: id, name: name, roles: rolesCopy}
}

// RestoreActor reconstructs an Actor from persistence.
func RestoreActor(id shared.ID, name string, roles []string) *Actor {
	return NewActor(id, name, roles)
}

// ID returns the actor's unique identifier.
func (a *Actor) ID() shared.ID { return a.id }

// Name returns the actor's name.
func (a *Actor) Name() string { return a.name }

// Roles returns a defensive copy of the actor's roles.
func (a *Actor) Roles() []string {
	r := make([]string, len(a.roles))
	copy(r, a.roles)
	return r
}

// HasRole checks if the actor has a specific role.
func (a *Actor) HasRole(role string) bool {
	for _, r := range a.roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasAnyRole checks if the actor has at least one of the specified roles.
func (a *Actor) HasAnyRole(roles []string) bool {
	for _, required := range roles {
		if a.HasRole(required) {
			return true
		}
	}
	return false
}
