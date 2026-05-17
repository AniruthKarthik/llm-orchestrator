package distributed

import (
	"context"
	"time"
)

// Coordinator defines the top-level interface for distributed execution coordination.
type Coordinator interface {
	LeaseManager
	HeartbeatManager
	LeaderElectionManager

	// ClaimTask attempts to claim a task for a specific node, typically wrapping a lease acquisition.
	ClaimTask(ctx context.Context, taskID, nodeID string) error
	// ReleaseTask releases a previously claimed task.
	ReleaseTask(ctx context.Context, taskID, nodeID string) error
}

// LeaderElectionManager defines the interface for distributed leader election.
type LeaderElectionManager interface {
	// TryAcquireLeadership attempts to become the leader for a specific role.
	TryAcquireLeadership(ctx context.Context, role, nodeID string, ttl time.Duration) (bool, error)
	// GetLeader returns the current leader ID for a specific role.
	GetLeader(ctx context.Context, role string) (string, error)
	// ResignLeadership releases the leadership for a specific role.
	ResignLeadership(ctx context.Context, role, nodeID string) error
}
