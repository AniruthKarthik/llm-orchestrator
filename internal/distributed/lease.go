package distributed

import (
	"context"
	"time"
)

// TaskLease represents an exclusive lock on a task for a specific node.
type TaskLease struct {
	TaskID    string
	NodeID    string
	ExpiresAt time.Time
}

// LeaseManager defines the interface for managing distributed task leases.
type LeaseManager interface {
	// AcquireLease attempts to get an exclusive lease for a task.
	AcquireLease(ctx context.Context, taskID, nodeID string, ttl time.Duration) (*TaskLease, error)
	// RenewLease extends the expiration of an existing lease.
	RenewLease(ctx context.Context, taskID, nodeID string, ttl time.Duration) (*TaskLease, error)
	// ReleaseLease explicitly releases a task lease.
	ReleaseLease(ctx context.Context, taskID, nodeID string) error
}
