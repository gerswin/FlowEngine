package messaging

import (
	"context"

	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/event"
	"github.com/LaFabric-LinkTIC/FlowEngine/pkg/logger"
)

// LogDispatcher writes domain events as structured JSON log entries.
// It serves as an audit log, a lightweight alternative to a message broker
// during development, and a foundation for log-based event streaming
// (e.g., CloudWatch -> Lambda).
type LogDispatcher struct{}

// NewLogDispatcher creates a new LogDispatcher.
func NewLogDispatcher() *LogDispatcher {
	return &LogDispatcher{}
}

// Dispatch logs the event as a structured JSON entry.
func (d *LogDispatcher) Dispatch(ctx context.Context, evt event.DomainEvent) error {
	logger.Info("domain_event",
		"event_type", evt.Type(),
		"aggregate_id", evt.AggregateID(),
		"payload", evt.Payload(),
	)
	return nil
}

// DispatchBatch logs each event in the batch as a structured JSON entry.
func (d *LogDispatcher) DispatchBatch(ctx context.Context, events []event.DomainEvent) error {
	for _, evt := range events {
		d.Dispatch(ctx, evt)
	}
	return nil
}
