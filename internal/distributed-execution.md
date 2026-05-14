# Distributed Execution

## Node Management
- **Cluster Nodes:** Individual workers register themselves in the cluster.
- **Heartbeats:** Nodes send periodic heartbeats to prove liveness.
- **Node Health:** Dead nodes are detected via missed heartbeats, triggering recovery.

## Task Coordination
- **Leasing:** Nodes must acquire a `TaskLease` before executing a task.
- **Claiming:** The `Coordinator` ensures exclusive task ownership to prevent duplicate execution.
- **Lease Expiration:** Expired leases allow tasks to be reclaimed by healthy nodes.
- **Recovery:** Dead-node tasks are released and rescheduled.
