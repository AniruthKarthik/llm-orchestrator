// cmd/server/main.go — entry point for the LLM Orchestrator.
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
	"github.com/AniruthKarthik/llm-orchestrator/internal/planner"
	"github.com/AniruthKarthik/llm-orchestrator/internal/queue"
	"github.com/AniruthKarthik/llm-orchestrator/internal/store"
	"github.com/AniruthKarthik/llm-orchestrator/internal/tools"
	"github.com/AniruthKarthik/llm-orchestrator/internal/worker"
)

// main initializes the system dependencies, starts the worker pool, and launches the HTTP server.
func main() {
	addr := envOrDefault("SERVER_ADDR", ":8080")
	workerCount := 4
	memStore := store.New()
	llmClient := llm.NewMockClient()
	p := planner.New(llmClient)
	taskQueue := queue.New(0)
	toolRegistry := []tools.Tool{
		tools.NewSearchTool(),
		tools.NewFetchTool(),
		tools.NewSummarizeTool(llmClient),
		tools.NewReportTool(),
	}
	exec := executor.New(memStore, taskQueue, toolRegistry)
	pool := worker.NewPool(workerCount, taskQueue, exec)
	pool.Start(context.Background())
	h := api.NewHandler(memStore, p, taskQueue)
	router := api.NewRouter(h)
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
		log.Printf("[server] listening on %s (%d workers)", addr, workerCount)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("[server] ListenAndServe: %v", err)
		}
	}()

	<-stop
	log.Println("[server] shutdown signal received")

	shutCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		log.Printf("[server] HTTP shutdown error: %v", err)
	}

	pool.Stop()

	log.Println("[server] stopped cleanly")
}

// envOrDefault retrieves an environment variable or returns a fallback value if not set.
func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
