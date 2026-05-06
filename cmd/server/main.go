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
)

func main() {
	addr := envOrDefault("SERVER_ADDR", ":8080")

	// Dependencies
	st := store.New()
	q := queue.New(100)
	llmClient := llm.NewMockClient()

	p := planner.New(llmClient)

	// Tools registry
	registry := []tools.Tool{
		tools.NewSearchTool(),
		tools.NewFetchTool(),
		tools.NewSummarizeTool(llmClient),
		tools.NewReportTool(),
	}

	exec := executor.New(st, q, registry)

	h := api.NewHandler(st, q, p)

	router := api.NewRouter(h)

	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 90 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start Executor worker loop
	go exec.Start(ctx)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("[server] listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("[server] ListenAndServe: %v", err)
		}
	}()

	<-stop
	log.Println("[server] shutdown signal received")
	cancel() // Stop executor loop

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("[server] graceful shutdown failed: %v", err)
	}

	log.Println("[server] stopped cleanly")
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
