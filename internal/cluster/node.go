package cluster

import "time"

// NodeStatus represents the current state of a cluster node.
type NodeStatus string

const (
	// NodeStatusActive indicates the node is actively heartbeating.
	NodeStatusActive NodeStatus = "ACTIVE"
	// NodeStatusDead indicates the node has missed heartbeats and is considered dead.
	NodeStatusDead NodeStatus = "DEAD"
)

// Node represents a worker node in the distributed orchestrator cluster.
type Node struct {
	ID            string
	Host          string
	Status        NodeStatus
	LastHeartbeat time.Time
	Tags          map[string]string
}
