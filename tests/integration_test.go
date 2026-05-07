package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/AniruthKarthik/llm-orchestrator/internal/api"
	"github.com/AniruthKarthik/llm-orchestrator/internal/executor"
	"github.com/AniruthKarthik/llm-orchestrator/internal/llm"
	"github.com/AniruthKarthik/llm-orchestrator/internal/memory"
	"github.com/AniruthKarthik/llm-orchestrator/internal/observability"
	"github.com/AniruthKarthik/llm-orchestrator/internal/orchestrator"
	"github.com/AniruthKarthik/llm-orchestrator/internal/planner"
	"github.com/AniruthKarthik/llm-orchestrator/internal/queue"
	"github.com/AniruthKarthik/llm-orchestrator/internal/store"
	"github.com/AniruthKarthik/llm-orchestrator/internal/tools"
	"github.com/AniruthKarthik/llm-orchestrator/internal/worker"
)

func TestIntegration(t *testing.T) {
	// Setup
	obs := observability.Default()
	appStore := store.New()
	llmClient := llm.NewMockClient()
	
	embedder := memory.NewMockEmbedder()
	vecStore := memory.NewInMemoryStore(embedder)
	retriever := memory.NewRetriever(vecStore, 5)
	
	p := planner.New(llmClient, planner.WithRetriever(retriever), planner.WithObs(obs))
	taskQueue := queue.New(100)
	dlq := queue.NewMemoryDLQ()
	
	registry := []tools.Tool{
		tools.NewSearchTool(),
		tools.NewReportTool(),
	}
	exec := executor.New(appStore, taskQueue, registry, retriever, obs)
	exec.SetDLQ(dlq)
	
	pool := worker.NewPool(2, taskQueue, exec, obs)
	pool.Start(context.Background())
	defer pool.Stop()
	
	orch := orchestrator.New(appStore, taskQueue, retriever, obs)
	orch.Start(context.Background())
	defer orch.Stop()
	
	exec.SetOrchestrator(orch)
	
	h := api.NewHandler(appStore, p, taskQueue, orch, dlq, obs)
	rl := api.NewRateLimiter(60)
	router := api.NewRouter(h, rl)
	
	ts := httptest.NewServer(router)
	defer ts.Close()

	// 1. Create Job
	goal := "test goal"
	reqBody, _ := json.Marshal(map[string]string{"goal": goal})
	resp, err := http.Post(ts.URL+"/job", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatalf("failed to POST /job: %v", err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d", resp.StatusCode)
	}
	
	var createResp struct {
		ID      string `json:"id"`
		Status  string `json:"status"`
		TraceID string `json:"trace_id"`
	}
	json.NewDecoder(resp.Body).Decode(&createResp)
	jobID := createResp.ID
	if jobID == "" {
		t.Fatal("expected job ID in response")
	}

	// 2. Poll for completion
	var jobState map[string]any
	for i := 0; i < 20; i++ {
		resp, err := http.Get(ts.URL + "/job/" + jobID)
		if err != nil {
			t.Fatalf("failed to GET /job/%s: %v", jobID, err)
		}
		json.NewDecoder(resp.Body).Decode(&jobState)
		if jobState["status"] == "completed" {
			break
		}
		if jobState["status"] == "failed" {
			t.Fatalf("job failed: %v", jobState["error"])
		}
		time.Sleep(500 * time.Millisecond)
	}

	if jobState["status"] != "completed" {
		t.Fatalf("expected status completed, got %v", jobState["status"])
	}

	// 3. Check Metrics
	resp, err = http.Get(ts.URL + "/metrics")
	if err != nil {
		t.Fatalf("failed to GET /metrics: %v", err)
	}
	var metrics map[string]any
	json.NewDecoder(resp.Body).Decode(&metrics)
	if metrics["total_tasks_executed"].(float64) == 0 {
		t.Error("expected total_tasks_executed > 0")
	}
}
