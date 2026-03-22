package messaging

import (
	"context"

	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/event"
)

// MultiDispatcher fans out events to multiple inner dispatchers.
// This is the composition pattern that allows combining InMemory + Webhook +
// future RabbitMQ/Kafka dispatchers without changing callers.
type MultiDispatcher struct {
	dispatchers []event.Dispatcher
}

// NewMultiDispatcher creates a MultiDispatcher that delegates to all provided dispatchers.
func NewMultiDispatcher(dispatchers ...event.Dispatcher) *MultiDispatcher {
	return &MultiDispatcher{
		dispatchers: dispatchers,
	}
}

// Dispatch sends the event to every inner dispatcher.
// It collects errors but does not stop on the first failure.
// Returns the first error encountered, if any.
func (m *MultiDispatcher) Dispatch(ctx context.Context, evt event.DomainEvent) error {
	var firstErr error
	for _, d := range m.dispatchers {
		if err := d.Dispatch(ctx, evt); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// DispatchBatch sends the batch of events to every inner dispatcher.
// It collects errors but does not stop on the first failure.
// Returns the first error encountered, if any.
func (m *MultiDispatcher) DispatchBatch(ctx context.Context, events []event.DomainEvent) error {
	var firstErr error
	for _, d := range m.dispatchers {
		if err := d.DispatchBatch(ctx, events); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
