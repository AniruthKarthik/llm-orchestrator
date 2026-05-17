package distributed

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	nodeKeyPrefix      = "nodes:"
	heartbeatSortedSet = "node_heartbeats"
	leasePrefix        = "lease:task:"
	leaderPrefix       = "leader:"
)

type RedisCoordinator struct {
	client *redis.Client
}

func NewRedisCoordinator(client *redis.Client) *RedisCoordinator {
	return &RedisCoordinator{
		client: client,
	}
}

// --- HeartbeatManager Implementation ---

func (r *RedisCoordinator) RegisterNode(ctx context.Context, nodeID, host string) error {
	nodeData := map[string]any{
		"id":   nodeID,
		"host": host,
	}
	data, _ := json.Marshal(nodeData)

	pipe := r.client.Pipeline()
	pipe.HSet(ctx, nodeKeyPrefix+nodeID, "metadata", data)
	pipe.ZAdd(ctx, heartbeatSortedSet, redis.Z{
		Score:  float64(time.Now().Unix()),
		Member: nodeID,
	})
	_, err := pipe.Exec(ctx)
	return err
}

func (r *RedisCoordinator) SendHeartbeat(ctx context.Context, nodeID string) error {
	return r.client.ZAdd(ctx, heartbeatSortedSet, redis.Z{
		Score:  float64(time.Now().Unix()),
		Member: nodeID,
	}).Err()
}

func (r *RedisCoordinator) GetDeadNodes(ctx context.Context, timeout time.Duration) ([]string, error) {
	threshold := time.Now().Add(-timeout).Unix()
	return r.client.ZRangeByScore(ctx, heartbeatSortedSet, &redis.ZRangeBy{
		Min: "-inf",
		Max: fmt.Sprintf("%d", threshold),
	}).Result()
}

// --- LeaseManager Implementation ---

func (r *RedisCoordinator) AcquireLease(ctx context.Context, taskID, nodeID string, ttl time.Duration) (*TaskLease, error) {
	key := leasePrefix + taskID
	ok, err := r.client.SetNX(ctx, key, nodeID, ttl).Result()
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("lease already held by another node")
	}

	return &TaskLease{
		TaskID:    taskID,
		NodeID:    nodeID,
		ExpiresAt: time.Now().Add(ttl),
	}, nil
}

func (r *RedisCoordinator) RenewLease(ctx context.Context, taskID, nodeID string, ttl time.Duration) (*TaskLease, error) {
	key := leasePrefix + taskID
	// Lua script to renew only if owned by this node
	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("pexpire", KEYS[1], ARGV[2])
		else
			return 0
		end
	`
	res, err := r.client.Eval(ctx, script, []string{key}, nodeID, int64(ttl/time.Millisecond)).Int()
	if err != nil {
		return nil, err
	}
	if res == 0 {
		return nil, fmt.Errorf("lease not found or owned by another node")
	}

	return &TaskLease{
		TaskID:    taskID,
		NodeID:    nodeID,
		ExpiresAt: time.Now().Add(ttl),
	}, nil
}

func (r *RedisCoordinator) ReleaseLease(ctx context.Context, taskID, nodeID string) error {
	key := leasePrefix + taskID
	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`
	res, err := r.client.Eval(ctx, script, []string{key}, nodeID).Int()
	if err != nil {
		return err
	}
	if res == 0 {
		return fmt.Errorf("lease not found or owned by another node")
	}
	return nil
}

// --- Coordinator Implementation ---

func (r *RedisCoordinator) ClaimTask(ctx context.Context, taskID, nodeID string) error {
	// For ClaimTask, we use a default TTL, e.g., 30 seconds
	_, err := r.AcquireLease(ctx, taskID, nodeID, 30*time.Second)
	return err
}

func (r *RedisCoordinator) ReleaseTask(ctx context.Context, taskID, nodeID string) error {
	return r.ReleaseLease(ctx, taskID, nodeID)
}

// --- LeaderElectionManager Implementation ---

func (r *RedisCoordinator) TryAcquireLeadership(ctx context.Context, role, nodeID string, ttl time.Duration) (bool, error) {
	key := leaderPrefix + role
	return r.client.SetNX(ctx, key, nodeID, ttl).Result()
}

func (r *RedisCoordinator) GetLeader(ctx context.Context, role string) (string, error) {
	key := leaderPrefix + role
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	return val, err
}

func (r *RedisCoordinator) ResignLeadership(ctx context.Context, role, nodeID string) error {
	key := leaderPrefix + role
	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`
	res, err := r.client.Eval(ctx, script, []string{key}, nodeID).Int()
	if err != nil {
		return err
	}
	if res == 0 {
		return fmt.Errorf("not the leader for role: %s", role)
	}
	return nil
}
