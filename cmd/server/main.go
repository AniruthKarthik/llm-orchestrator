package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AniruthKarthik/llm-orchestrator/internal/agents"
	"github.com/AniruthKarthik/llm-orchestrator/internal/api"
	"github.com/AniruthKarthik/llm-orchestrator/internal/config"
	"github.com/AniruthKarthik/llm-orchestrator/internal/core"
	"github.com/AniruthKarthik/llm-orchestrator/internal/events"
	"github.com/AniruthKarthik/llm-orchestrator/internal/executor"
	"github.com/AniruthKarthik/llm-orchestrator/internal/observer"
	"github.com/AniruthKarthik/llm-orchestrator/internal/providers"
	"github.com/AniruthKarthik/llm-orchestrator/internal/providers/anthropic"
	"github.com/AniruthKarthik/llm-orchestrator/internal/providers/gemini"
	"github.com/AniruthKarthik/llm-orchestrator/internal/providers/groq"
	"github.com/AniruthKarthik/llm-orchestrator/internal/providers/openai"
	"github.com/AniruthKarthik/llm-orchestrator/internal/secrets"
	"github.com/AniruthKarthik/llm-orchestrator/internal/store"
)

func main() {
	// -- Structured logging (JSON in production, text in dev) --
	logLevel := slog.LevelInfo
	if os.Getenv("LOG_LEVEL") == "debug" {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
	slog.SetDefault(logger)

	slog.Info("llm-orchestrator starting")

	// 1. Load Configuration
	cfg := config.Load()
	sm := secrets.NewEnvSecretManager()

	// 2. Initialize Store
	var s store.Store
	if cfg.Database.URL != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		pgStore, err := store.NewPostgresStore(ctx, cfg.Database.URL, cfg.Database.MigrationsPath)
		if err != nil {
			slog.Warn("failed to connect to PostgreSQL, falling back to MemoryStore", "error", err)
			s = store.NewMemoryStore()
		} else {
			slog.Info("connected to PostgreSQL")
			s = pgStore
		}
	} else {
		slog.Info("DATABASE_URL not set, using in-memory store (data will not persist)")
		s = store.NewMemoryStore()
	}

	// 3. Register LLM Providers
	registered := registerProviders(sm)
	if registered == 0 {
		slog.Warn("no LLM providers registered — set GROQ_API_KEY, OPENAI_API_KEY, ANTHROPIC_API_KEY, or GEMINI_API_KEY")
	}

	// 4. Initialize Core Components
	eb := events.NewEventBus(1)
	wr := executor.NewWorkerRegistry()
	ar := agents.NewAgentRegistry()
	art := core.NewArtifactRegistry()
	mem := core.NewMemoryRegistry()
	tr := core.NewToolRegistry()
	tp := core.NewToolPolicy()

	exec := executor.NewExecutor(wr, ar, art, mem, tr, tp, eb, s)
	exec.UseTaskMiddleware(executor.PanicRecoveryMiddleware)
	exec.UseTaskMiddleware(executor.OutputValidationMiddleware)

	// 5. Start Supervisor (recovers interrupted workflows after restart)
	sup := executor.NewSupervisor(s, exec, 30*time.Second)
	sup.Start()
	defer sup.Stop()

	// 6. Wire Audit Logger to EventBus
	al := observer.NewAuditLogger(s)
	for _, eventType := range []events.EventType{
		events.WorkflowStarted,
		events.WorkflowCompleted,
		events.WorkflowFailed,
		events.TaskStarted,
		events.TaskWaitingForApproval,
		events.TaskRetried,
		events.TaskCompleted,
		events.TaskFailed,
		events.TaskTokenUsage,
		events.StageStarted,
		events.StageCompleted,
		events.StageFailed,
	} {
		eb.Subscribe(eventType, al.Handle)
	}

	// 7. Build HTTP Server
	srv := api.NewServer(exec, s, eb)
	handler := srv.Routes()

	httpServer := &http.Server{
		Addr:         cfg.Server.Port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second, // longer for streaming responses
		IdleTimeout:  120 * time.Second,
	}

	// 8. Start HTTP server in background
	serverErr := make(chan error, 1)
	go func() {
		slog.Info("HTTP server listening", "addr", cfg.Server.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// 9. Graceful shutdown on SIGINT / SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		slog.Error("server error", "error", err)
		os.Exit(1)
	case sig := <-quit:
		slog.Info("shutdown signal received", "signal", sig.String())
	}

	slog.Info("shutting down gracefully (30s timeout)...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		slog.Error("forced shutdown", "error", err)
	} else {
		slog.Info("server stopped cleanly")
	}
}

func registerProviders(sm secrets.SecretManager) int {
	registered := 0

	if key, err := sm.Get("GROQ_API_KEY"); err == nil {
		providers.Register(groq.NewGroqProvider(key))
		slog.Info("provider registered", "provider", "groq")
		registered++
	}

	if key, err := sm.Get("OPENAI_API_KEY"); err == nil {
		providers.Register(openai.NewOpenAIProvider(key))
		slog.Info("provider registered", "provider", "openai")
		registered++
	}

	if key, err := sm.Get("ANTHROPIC_API_KEY"); err == nil {
		providers.Register(anthropic.NewAnthropicProvider(key))
		slog.Info("provider registered", "provider", "anthropic")
		registered++
	}

	if key, err := sm.Get("GEMINI_API_KEY"); err == nil {
		providers.Register(gemini.NewGeminiProvider(key))
		slog.Info("provider registered", "provider", "gemini")
		registered++
	}

	fmt.Sprintf("registered %d provider(s)", registered) // suppress unused warning
	return registered
}
