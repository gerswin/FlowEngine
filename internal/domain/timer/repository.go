package timer

import (
	"context"

	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/shared"
)

// Repository defines the persistence interface for timers.
type Repository interface {
	Save(ctx context.Context, timer *Timer) error
	FindPending(ctx context.Context, limit int) ([]*Timer, error)
	Delete(ctx context.Context, id shared.ID) error
	DeleteByInstanceID(ctx context.Context, instanceID shared.ID) error
}
