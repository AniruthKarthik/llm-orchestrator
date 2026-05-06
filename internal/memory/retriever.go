package memory

import (
	"context"
	"fmt"
	"strings"
)

type Retriever struct {
	store MemoryStore
	topK  int
}

// NewRetriever creates a new Retriever with the specified store and result limit.
func NewRetriever(s MemoryStore, topK int) *Retriever {
	if topK <= 0 {
		topK = 5
	}
	return &Retriever{store: s, topK: topK}
}

// RetrieveContext searches the memory store and returns a formatted string of relevant snippets.
func (r *Retriever) RetrieveContext(ctx context.Context, query string) (string, error) {
	results, err := r.store.Search(ctx, query, r.topK)
	if err != nil {
		return "", fmt.Errorf("retriever: search failed: %w", err)
	}
	const minScore = 0.30

	var sb strings.Builder
	count := 0
	for _, res := range results {
		if res.Score < minScore {
			continue
		}
		if count == 0 {
			sb.WriteString("RELEVANT CONTEXT FROM MEMORY:\n")
		}
		sb.WriteString(fmt.Sprintf("[%d] (score %.3f) key=%s\n", count+1, res.Score, res.Key))
		for k, v := range res.Value {
			sb.WriteString(fmt.Sprintf("    %s: %v\n", k, v))
		}
		count++
	}

	return sb.String(), nil
}

// StoreTaskResult formats and persists a task's output into the memory store.
func (r *Retriever) StoreTaskResult(ctx context.Context, jobID, taskID, taskType string, result map[string]any) error {
	key := fmt.Sprintf("job:%s:task:%s", jobID, taskID)
	value := map[string]any{
		"job_id":    jobID,
		"task_id":   taskID,
		"task_type": taskType,
		"result":    result,
	}
	for k, v := range result {
		if _, conflict := value[k]; !conflict {
			value[k] = v
		}
	}
	if err := r.store.Store(ctx, key, value); err != nil {
		return fmt.Errorf("retriever: store task result failed: %w", err)
	}
	return nil
}

// StoreJobGoal persists the primary objective of a job into the memory store.
func (r *Retriever) StoreJobGoal(ctx context.Context, jobID, goal string) error {
	key := fmt.Sprintf("job:%s:goal", jobID)
	return r.store.Store(ctx, key, map[string]any{
		"job_id": jobID,
		"goal":   goal,
		"type":   "job_goal",
	})
}
