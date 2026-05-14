package distributed

import "context"

// Coordinator defines the top-level interface for distributed execution coordination.
type Coordinator interface {
	LeaseManager
	HeartbeatManager

	// ClaimTask attempts to claim a task for a specific node, typically wrapping a lease acquisition.
	ClaimTask(ctx context.Context, taskID, nodeID string) error
	// ReleaseTask releases a previously claimed task.
	ReleaseTask(ctx context.Context, taskID, nodeID string) error
}
