package distributed

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestRedisCoordinator(t *testing.T) {
	redisURL := os.Getenv("TEST_REDIS_URL")
	if redisURL == "" {
		redisURL = "localhost:6379"
	}

	client := redis.NewClient(&redis.Options{
		Addr: redisURL,
	})
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Skipping RedisCoordinator test: redis not available at %s: %v", redisURL, err)
		return
	}
	defer client.Close()
	defer client.FlushAll(ctx)

	coord := NewRedisCoordinator(client)

	// 1. Test Node Registration and Heartbeats
	nodeID := "node-1"
	err := coord.RegisterNode(ctx, nodeID, "localhost")
	if err != nil {
		t.Errorf("RegisterNode failed: %v", err)
	}

	err = coord.SendHeartbeat(ctx, nodeID)
	if err != nil {
		t.Errorf("SendHeartbeat failed: %v", err)
	}

	deadNodes, err := coord.GetDeadNodes(ctx, 10*time.Second)
	if err != nil {
		t.Errorf("GetDeadNodes failed: %v", err)
	}
	if len(deadNodes) != 0 {
		t.Errorf("Expected 0 dead nodes, got %v", deadNodes)
	}

	// Test dead node detection (simulate time pass)
	// We'll manually add a node with an old score
	client.ZAdd(ctx, heartbeatSortedSet, redis.Z{
		Score:  float64(time.Now().Add(-20 * time.Second).Unix()),
		Member: "dead-node",
	})

	deadNodes, err = coord.GetDeadNodes(ctx, 10*time.Second)
	if err != nil {
		t.Errorf("GetDeadNodes failed: %v", err)
	}
	found := false
	for _, n := range deadNodes {
		if n == "dead-node" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected dead-node to be found")
	}

	// 2. Test Leases
	taskID := "task-123"
	lease, err := coord.AcquireLease(ctx, taskID, nodeID, 5*time.Second)
	if err != nil {
		t.Errorf("AcquireLease failed: %v", err)
	}
	if lease.NodeID != nodeID {
		t.Errorf("Expected nodeID %s, got %s", nodeID, lease.NodeID)
	}

	// Try to acquire again
	_, err = coord.AcquireLease(ctx, taskID, "node-2", 5*time.Second)
	if err == nil {
		t.Errorf("Expected error when acquiring already held lease")
	}

	// Renew lease
	_, err = coord.RenewLease(ctx, taskID, nodeID, 10*time.Second)
	if err != nil {
		t.Errorf("RenewLease failed: %v", err)
	}

	// Release lease
	err = coord.ReleaseLease(ctx, taskID, nodeID)
	if err != nil {
		t.Errorf("ReleaseLease failed: %v", err)
	}

	// Acquire again after release
	_, err = coord.AcquireLease(ctx, taskID, "node-2", 5*time.Second)
	if err != nil {
		t.Errorf("AcquireLease failed after release: %v", err)
	}

	// 3. Test Leader Election
	role := "scheduler"
	ok, err := coord.TryAcquireLeadership(ctx, role, nodeID, 5*time.Second)
	if err != nil {
		t.Errorf("TryAcquireLeadership failed: %v", err)
	}
	if !ok {
		t.Errorf("Expected to acquire leadership")
	}

	leader, err := coord.GetLeader(ctx, role)
	if err != nil {
		t.Errorf("GetLeader failed: %v", err)
	}
	if leader != nodeID {
		t.Errorf("Expected leader %s, got %s", nodeID, leader)
	}

	// Try to acquire as another node
	ok, err = coord.TryAcquireLeadership(ctx, role, "node-2", 5*time.Second)
	if err != nil {
		t.Errorf("TryAcquireLeadership failed: %v", err)
	}
	if ok {
		t.Errorf("Expected not to acquire leadership when already held")
	}

	// Resign leadership
	err = coord.ResignLeadership(ctx, role, nodeID)
	if err != nil {
		t.Errorf("ResignLeadership failed: %v", err)
	}

	leader, _ = coord.GetLeader(ctx, role)
	if leader != "" {
		t.Errorf("Expected no leader after resignation, got %s", leader)
	}
}
