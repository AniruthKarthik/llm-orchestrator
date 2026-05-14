package distributed

import (
	"context"
	"time"
)

// HeartbeatManager defines the interface for tracking node health and liveness.
type HeartbeatManager interface {
	// RegisterNode adds a new node to the cluster tracking system.
	RegisterNode(ctx context.Context, nodeID, host string) error
	// SendHeartbeat updates the last seen timestamp for a node.
	SendHeartbeat(ctx context.Context, nodeID string) error
	// GetDeadNodes returns a list of node IDs that have not sent a heartbeat within the timeout period.
	GetDeadNodes(ctx context.Context, timeout time.Duration) ([]string, error)
}
