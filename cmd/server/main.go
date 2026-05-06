// cmd/server/main.go — entry point for the LLM Orchestrator.
//
// Wiring order (bottom-up, respecting dependencies):
//  1. Observability bundle (logger, tracer, metrics)
//  2. Shared job/task store
//  3. LLM client
//  4. Memory embedder + vector store + retriever
//  5. Planner (LLM + memory retriever + obs)
//  6. Task queue
//  7. Tool registry
//  8. Executor (store + queue + tools + memory + obs)
//  9. Worker pool (queue + executor + obs)
//
// 10.  Orchestrator (store + queue + memory + obs)
// 11.  Wire executor → orchestrator (breaks circular dep)
// 12.  API handler + router
// 13.  HTTP server
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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

func main() {
	addr := envOrDefault("SERVER_ADDR", ":8080")
	workerCount := 4

	// ── 1. Observability ──────────────────────────────────────────────────────
	obs := observability.Default()
	rootCtx := context.Background()
	obs.Log.Info(rootCtx, "llm-orchestrator starting",
		observability.F("addr", addr),
		observability.F("workers", workerCount),
	)

	// ── 2. Shared store ───────────────────────────────────────────────────────
	memStore := store.New()

	// ── 3. LLM client ─────────────────────────────────────────────────────────
	llmClient := llm.NewMockClient()

	// ── 4. Memory (RAG) ───────────────────────────────────────────────────────
	embedder := memory.NewMockEmbedder()
	vecStore := memory.NewInMemoryStore(embedder)
	retriever := memory.NewRetriever(vecStore, 5)

	// ── 5. Planner ────────────────────────────────────────────────────────────
	p := planner.New(llmClient,
		planner.WithRetriever(retriever),
		planner.WithObs(obs),
	)

	// ── 6. Task queue ─────────────────────────────────────────────────────────
	taskQueue := queue.New(0)

	// ── 7. Tool registry ──────────────────────────────────────────────────────
	toolRegistry := []tools.Tool{
		tools.NewSearchTool(),
		tools.NewFetchTool(),
		tools.NewSummarizeTool(llmClient),
		tools.NewReportTool(),
	}

	// ── 8. Executor ───────────────────────────────────────────────────────────
	exec := executor.New(memStore, taskQueue, toolRegistry, retriever, obs)

	// ── 9. Worker pool ────────────────────────────────────────────────────────
	pool := worker.NewPool(workerCount, taskQueue, exec, obs)
	pool.Start(rootCtx)

	// ── 10. Orchestrator ──────────────────────────────────────────────────────
	orch := orchestrator.New(memStore, taskQueue, retriever, obs)
	orch.Start(rootCtx)

	// ── 11. Wire executor → orchestrator (break circular dep) ─────────────────
	exec.SetOrchestrator(orch)

	// ── 12. API handler + router ──────────────────────────────────────────────
	h := api.NewHandler(memStore, p, taskQueue, orch, obs)
	router := api.NewRouter(h)

	// ── 13. HTTP server with graceful shutdown ────────────────────────────────
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 90 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		obs.Log.Info(rootCtx, "http server listening", observability.F("addr", addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("ListenAndServe: %v", err)
		}
	}()

	<-stop
	obs.Log.Info(rootCtx, "shutdown signal received")

	shutCtx, cancel := context.WithTimeout(rootCtx, 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		obs.Log.Error(rootCtx, "http shutdown error", observability.F("err", err.Error()))
	}

	orch.Stop()
	pool.Stop()

	obs.Log.Info(rootCtx, "llm-orchestrator stopped cleanly")
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
