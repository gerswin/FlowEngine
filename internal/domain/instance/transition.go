package instance

import (
	"time"

	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/shared"
)

// Transition represents a state transition in a workflow instance.
// It's an entity that records the history of state changes.
type Transition struct {
	id           shared.ID
	from         string // State ID
	to           string // State ID
	fromSubState SubState
	toSubState   SubState
	event        string
	actor        shared.ID // Actor who triggered the transition
	timestamp    shared.Timestamp
	metadata     TransitionMetadata
	data         Data // Data snapshot at transition time
}

// NewTransition creates a new Transition.
func NewTransition(from, to, event string, actor shared.ID) *Transition {
	return &Transition{
		id:        shared.NewID(),
		from:      from,
		to:        to,
		event:     event,
		actor:     actor,
		timestamp: shared.Now(),
		metadata:  EmptyTransitionMetadata(),
		data:      NewData(),
	}
}

// NewTransitionWithSubStates creates a new Transition with sub-states (R17).
func NewTransitionWithSubStates(from, to, event string, actor shared.ID, fromSubState, toSubState SubState) *Transition {
	return &Transition{
		id:           shared.NewID(),
		from:         from,
		to:           to,
		fromSubState: fromSubState,
		toSubState:   toSubState,
		event:        event,
		actor:        actor,
		timestamp:    shared.Now(),
		metadata:     EmptyTransitionMetadata(),
		data:         NewData(),
	}
}

// ID returns the transition ID.
func (t *Transition) ID() shared.ID {
	return t.id
}

// From returns the source state ID.
func (t *Transition) From() string {
	return t.from
}

// To returns the destination state ID.
func (t *Transition) To() string {
	return t.to
}

// FromSubState returns the source sub-state (R17).
func (t *Transition) FromSubState() SubState {
	return t.fromSubState
}

// ToSubState returns the destination sub-state (R17).
func (t *Transition) ToSubState() SubState {
	return t.toSubState
}

// Event returns the event name that triggered the transition.
func (t *Transition) Event() string {
	return t.event
}

// Actor returns the ID of the actor who triggered the transition.
func (t *Transition) Actor() shared.ID {
	return t.actor
}

// Timestamp returns the transition timestamp.
func (t *Transition) Timestamp() shared.Timestamp {
	return t.timestamp
}

// Metadata returns the transition metadata (R23).
func (t *Transition) Metadata() TransitionMetadata {
	return t.metadata
}

// Data returns a copy of the data snapshot at transition time.
func (t *Transition) Data() Data {
	return t.data
}

// SetMetadata sets the transition metadata.
func (t *Transition) SetMetadata(metadata TransitionMetadata) {
	t.metadata = metadata
}

// SetData sets the data snapshot for the transition.
func (t *Transition) SetData(data Data) {
	t.data = data
}

// Duration calculates the time elapsed since the transition.
func (t *Transition) Duration() time.Duration {
	return shared.Now().Sub(t.timestamp)
}

// DurationSince calculates the time between this transition and another timestamp.
func (t *Transition) DurationSince(other shared.Timestamp) time.Duration {
	return t.timestamp.Sub(other)
}

// HasSubStates returns true if this transition includes sub-state information (R17).
func (t *Transition) HasSubStates() bool {
	return !t.fromSubState.IsZero() || !t.toSubState.IsZero()
}

// String returns a string representation of the transition.
func (t *Transition) String() string {
	if t.HasSubStates() {
		return t.id.String() + ": " + t.from + "." + t.fromSubState.ID() + " -> " + t.to + "." + t.toSubState.ID() + " (" + t.event + ")"
	}
	return t.id.String() + ": " + t.from + " -> " + t.to + " (" + t.event + ")"
}
