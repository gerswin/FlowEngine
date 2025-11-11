package event

import "context"

// Dispatcher defines the contract for dispatching domain events.
// This is a port interface that will be implemented by the infrastructure layer.
//
// Implementations should handle:
// - Async event publishing (non-blocking)
// - Retry logic for failed dispatches
// - Event ordering guarantees (if required)
// - Integration with message brokers (RabbitMQ, Kafka, etc.)
type Dispatcher interface {
	// Dispatch publishes a single domain event.
	//
	// The implementation should be non-blocking and return quickly.
	// Errors should be logged but not block the caller.
	//
	// Context can be used for timeout control and cancellation.
	//
	// Returns an error if the event cannot be queued for dispatch.
	Dispatch(ctx context.Context, event DomainEvent) error

	// DispatchBatch publishes multiple domain events atomically.
	//
	// All events should be dispatched together or none at all.
	// This is useful for maintaining consistency when multiple events
	// are generated from a single aggregate operation.
	//
	// Context can be used for timeout control and cancellation.
	//
	// Returns an error if any event cannot be queued for dispatch.
	DispatchBatch(ctx context.Context, events []DomainEvent) error
}

// NullDispatcher is a no-op implementation of Dispatcher.
// Useful for testing or when event dispatching is disabled.
type NullDispatcher struct{}

// NewNullDispatcher creates a new NullDispatcher.
func NewNullDispatcher() *NullDispatcher {
	return &NullDispatcher{}
}

// Dispatch does nothing and returns nil.
func (d *NullDispatcher) Dispatch(ctx context.Context, event DomainEvent) error {
	return nil
}

// DispatchBatch does nothing and returns nil.
func (d *NullDispatcher) DispatchBatch(ctx context.Context, events []DomainEvent) error {
	return nil
}

// InMemoryDispatcher is a simple in-memory implementation that collects events.
// Useful for testing to verify which events were dispatched.
type InMemoryDispatcher struct {
	events []DomainEvent
}

// NewInMemoryDispatcher creates a new InMemoryDispatcher.
func NewInMemoryDispatcher() *InMemoryDispatcher {
	return &InMemoryDispatcher{
		events: make([]DomainEvent, 0),
	}
}

// Dispatch stores the event in memory.
func (d *InMemoryDispatcher) Dispatch(ctx context.Context, event DomainEvent) error {
	d.events = append(d.events, event)
	return nil
}

// DispatchBatch stores all events in memory.
func (d *InMemoryDispatcher) DispatchBatch(ctx context.Context, events []DomainEvent) error {
	d.events = append(d.events, events...)
	return nil
}

// Events returns all dispatched events.
func (d *InMemoryDispatcher) Events() []DomainEvent {
	// Return a copy to prevent external modification
	result := make([]DomainEvent, len(d.events))
	copy(result, d.events)
	return result
}

// Clear removes all stored events.
func (d *InMemoryDispatcher) Clear() {
	d.events = make([]DomainEvent, 0)
}

// Count returns the number of dispatched events.
func (d *InMemoryDispatcher) Count() int {
	return len(d.events)
}

// FindByType returns all events of the given type.
func (d *InMemoryDispatcher) FindByType(eventType string) []DomainEvent {
	result := make([]DomainEvent, 0)
	for _, event := range d.events {
		if event.Type() == eventType {
			result = append(result, event)
		}
	}
	return result
}

// FindByAggregateID returns all events for the given aggregate ID.
func (d *InMemoryDispatcher) FindByAggregateID(aggregateID string) []DomainEvent {
	result := make([]DomainEvent, 0)
	for _, event := range d.events {
		if event.AggregateID() == aggregateID {
			result = append(result, event)
		}
	}
	return result
}
