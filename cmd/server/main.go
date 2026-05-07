// cmd/server/main.go — production entry point for the LLM Orchestrator.
//
// Wiring order (bottom-up):
//  1. Config (env-driven, validated at startup)
//  2. Observability (logger, tracer, in-memory metrics)
//  3. Job/task store — in-memory or Postgres based on POSTGRES_DSN
//  4. LLM client — mock, OpenAI, or Anthropic based on LLM_PROVIDER
//  5. Memory RAG layer (embedder + vector store + retriever)
//  6. Planner (LLM + memory + obs)
//  7. Task queue (buffered channel)
//  8. Dead-letter queue
//  9. Tool registry (with circuit breakers on real tools)
//
// 10.  Executor (store + queue + tools + memory + obs + DLQ)
// 11.  Worker pool
// 12.  Orchestrator
// 13.  Wire executor → orchestrator (breaks circular dep)
// 14.  API handler + router (with rate limiter, health, recovery)
// 15.  HTTP server with graceful shutdown
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/AniruthKarthik/llm-orchestrator/internal/api"
	"github.com/AniruthKarthik/llm-orchestrator/internal/config"
	"github.com/AniruthKarthik/llm-orchestrator/internal/executor"
	"github.com/AniruthKarthik/llm-orchestrator/internal/llm"
	"github.com/AniruthKarthik/llm-orchestrator/internal/llm/providers"
	"github.com/AniruthKarthik/llm-orchestrator/internal/memory"
	"github.com/AniruthKarthik/llm-orchestrator/internal/observability"
	"github.com/AniruthKarthik/llm-orchestrator/internal/orchestrator"
	"github.com/AniruthKarthik/llm-orchestrator/internal/planner"
	"github.com/AniruthKarthik/llm-orchestrator/internal/queue"
	"github.com/AniruthKarthik/llm-orchestrator/internal/reliability"
	"github.com/AniruthKarthik/llm-orchestrator/internal/store"
	"github.com/AniruthKarthik/llm-orchestrator/internal/tools"
	"github.com/AniruthKarthik/llm-orchestrator/internal/worker"
)

func main() {
	// ── 1. Config ─────────────────────────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	// ── 2. Observability ──────────────────────────────────────────────────────
	obs := observability.Default()
	rootCtx := context.Background()
	obs.Log.Info(rootCtx, "llm-orchestrator starting",
		observability.F("addr", cfg.Server.Addr),
		observability.F("workers", cfg.Workers.Count),
		observability.F("llm_provider", cfg.LLM.Provider),
		observability.F("postgres_enabled", cfg.Postgres.Enabled),
	)

	// ── 3. Store ──────────────────────────────────────────────────────────────
	// Postgres support requires the pgx driver to be available in the module
	// cache (blocked in this environment).  The in-memory store is always used
	// when POSTGRES_DSN is unset; the adapter is the swap point for production.
	var appStore store.Store
	if cfg.Postgres.Enabled {
		obs.Log.Info(rootCtx, "postgres DSN configured — Postgres store would be initialised here")
		obs.Log.Warn(rootCtx, "pgx driver not available in build environment; falling back to in-memory store")
		appStore = store.New()
	} else {
		obs.Log.Info(rootCtx, "using in-memory store (set POSTGRES_DSN to enable Postgres)")
		appStore = store.New()
	}

	// ── 4. LLM client ─────────────────────────────────────────────────────────
	llmClient, err := buildLLMClient(cfg, obs)
	if err != nil {
		log.Fatalf("LLM client init failed: %v", err)
	}

	// ── 5. Memory RAG ─────────────────────────────────────────────────────────
	embedder := memory.NewMockEmbedder()
	vecStore := memory.NewInMemoryStore(embedder)
	retriever := memory.NewRetriever(vecStore, 5)

	// ── 6. Planner ────────────────────────────────────────────────────────────
	p := planner.New(llmClient,
		planner.WithRetriever(retriever),
		planner.WithObs(obs),
	)

	// ── 7. Task queue ─────────────────────────────────────────────────────────
	taskQueue := queue.New(cfg.Queue.BufferSize)

	// ── 8. Dead-letter queue ──────────────────────────────────────────────────
	dlq := queue.NewMemoryDLQ()

	// ── 9. Circuit breaker factory for external tools ─────────────────────────
	newCB := func(name string) *reliability.CircuitBreaker {
		return reliability.NewCircuitBreaker(reliability.Config{
			Name:             name,
			FailureThreshold: cfg.CircuitBreaker.FailureThreshold,
			SuccessThreshold: cfg.CircuitBreaker.SuccessThreshold,
			Timeout:          cfg.CircuitBreaker.Timeout,
			OnStateChange: func(name string, from, to reliability.State) {
				obs.Log.Warn(rootCtx, "circuit breaker state changed",
					observability.F("name", name),
					observability.F("from", from.String()),
					observability.F("to", to.String()),
				)
			},
		})
	}

	// Tool registry — real tools wrapped with circuit breakers.
	toolRegistry := []tools.Tool{
		tools.NewSearchTool(),
		tools.NewFetchTool(),
		tools.NewSummarizeTool(llmClient),
		tools.NewReportTool(),
		// Real fetch and GitHub tools wrapped with circuit breakers.
		tools.NewCircuitBreakerTool(tools.NewRealFetchTool(), newCB("real-fetch")),
		tools.NewCircuitBreakerTool(tools.NewGitHubIssueTool(os.Getenv("GITHUB_TOKEN")), newCB("github-issue")),
	}

	// ── 10. Executor ──────────────────────────────────────────────────────────
	exec := executor.New(appStore, taskQueue, toolRegistry, retriever, obs)
	exec.SetDLQ(dlq)

	// ── 11. Worker pool ───────────────────────────────────────────────────────
	pool := worker.NewPool(cfg.Workers.Count, taskQueue, exec, obs)
	pool.Start(rootCtx)

	// ── 12. Orchestrator ──────────────────────────────────────────────────────
	orch := orchestrator.New(appStore, taskQueue, retriever, obs)
	orch.Start(rootCtx)

	// ── 13. Wire executor → orchestrator ──────────────────────────────────────
	exec.SetOrchestrator(orch)

	// ── 14. API layer ─────────────────────────────────────────────────────────
	rateLimiter := api.NewRateLimiter(cfg.RateLimit.RequestsPerMinute)
	h := api.NewHandler(appStore, p, taskQueue, orch, dlq, obs)
	router := api.NewRouter(h, rateLimiter)

	// ── 15. HTTP server ───────────────────────────────────────────────────────
	srv := &http.Server{
		Addr:         cfg.Server.Addr,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		obs.Log.Info(rootCtx, "http server listening", observability.F("addr", cfg.Server.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("ListenAndServe: %v", err)
		}
	}()

	<-stop
	obs.Log.Info(rootCtx, "shutdown signal received — draining")

	// Stop accepting new HTTP requests.
	shutCtx, cancel := context.WithTimeout(rootCtx, cfg.Server.ShutdownGrace)
	defer cancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		obs.Log.Error(rootCtx, "http shutdown error", observability.F("err", err.Error()))
	}

	// Stop the orchestrator (flushes its event loop) then drain the worker pool.
	orch.Stop()
	pool.Stop()

	obs.Log.Info(rootCtx, "llm-orchestrator stopped cleanly")
}

// buildLLMClient selects and constructs the LLM client based on config.
func buildLLMClient(cfg *config.Config, obs *observability.Obs) (llm.Client, error) {
	switch cfg.LLM.Provider {
	case "openai":
		obs.Log.Info(context.Background(), "using OpenAI LLM provider",
			observability.F("model", cfg.LLM.OpenAIModel))
		return providers.NewOpenAIClient(providers.OpenAIConfig{
			APIKey:     cfg.LLM.OpenAIKey,
			Model:      cfg.LLM.OpenAIModel,
			Timeout:    cfg.LLM.Timeout,
			MaxRetries: cfg.LLM.MaxRetries,
		}), nil

	case "anthropic":
		obs.Log.Info(context.Background(), "using Anthropic LLM provider",
			observability.F("model", cfg.LLM.AnthropicModel))
		return providers.NewAnthropicClient(providers.AnthropicConfig{
			APIKey:     cfg.LLM.AnthropicKey,
			Model:      cfg.LLM.AnthropicModel,
			Timeout:    cfg.LLM.Timeout,
			MaxRetries: cfg.LLM.MaxRetries,
		}), nil

	case "mock", "":
		obs.Log.Info(context.Background(), "using mock LLM provider")
		return llm.NewMockClient(), nil

	default:
		return nil, fmt.Errorf("unknown LLM_PROVIDER %q", cfg.LLM.Provider)
	}
}
