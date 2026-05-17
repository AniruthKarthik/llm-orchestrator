package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/AniruthKarthik/llm-orchestrator/internal/api"
	"github.com/AniruthKarthik/llm-orchestrator/internal/config"
	"github.com/AniruthKarthik/llm-orchestrator/internal/core"
	"github.com/AniruthKarthik/llm-orchestrator/internal/distributed"
	"github.com/AniruthKarthik/llm-orchestrator/internal/events"
	"github.com/AniruthKarthik/llm-orchestrator/internal/executor"
	"github.com/AniruthKarthik/llm-orchestrator/internal/providers"
	"github.com/AniruthKarthik/llm-orchestrator/internal/providers/anthropic"
	"github.com/AniruthKarthik/llm-orchestrator/internal/providers/gemini"
	"github.com/AniruthKarthik/llm-orchestrator/internal/providers/groq"
	"github.com/AniruthKarthik/llm-orchestrator/internal/providers/openai"
	"github.com/AniruthKarthik/llm-orchestrator/internal/secrets"
	"github.com/AniruthKarthik/llm-orchestrator/internal/store"
	"github.com/redis/go-redis/v9"
)

type DummyWorker struct{}

func (w *DummyWorker) Execute(ctx context.Context, execCtx *executor.ExecutionContext, task *core.Task) (map[string]any, error) {
	log.Printf("Executing task: %s (%s)", task.ID, task.Name)
	return map[string]any{"status": "success", "taskID": task.ID}, nil
}

func main() {
	// 1. Load Configuration
	cfg := config.Load()
	sm := secrets.NewEnvSecretManager()

	// 2. Initialize Store
	var s store.Store
	if cfg.Database.URL != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		pgStore, err := store.NewPostgresStore(ctx, cfg.Database.URL, cfg.Database.MigrationsPath)
		if err != nil {
			log.Printf("Failed to connect to PostgreSQL: %v. Falling back to MemoryStore.", err)
			s = store.NewMemoryStore()
		} else {
			log.Println("Connected to PostgreSQL successfully.")
			s = pgStore
		}
	} else {
		log.Println("No database URL provided. Using MemoryStore.")
		s = store.NewMemoryStore()
	}

	// 3. Register LLM Providers
	registerProviders(sm)

	// 4. Initialize Coordination
	var coord distributed.Coordinator
	if cfg.Redis.URL != "" {
		rdb := redis.NewClient(&redis.Options{
			Addr: cfg.Redis.URL,
		})
		// Verify connection
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := rdb.Ping(ctx).Err(); err != nil {
			log.Printf("Failed to connect to Redis: %v. Distributed coordination will be disabled.", err)
		} else {
			log.Println("Connected to Redis successfully.")
			coord = distributed.NewRedisCoordinator(rdb)
		}
	}

	// 5. Initialize Core Components
	eb := events.NewEventBus(10)
	wr := executor.NewWorkerRegistry()

	// Register a dummy worker for testing
	wr.Register("test-task", &DummyWorker{})

	exec := executor.NewExecutor(wr, eb, s)
	if coord != nil {
		exec.WithCoordinator(coord)
	}
	srv := api.NewServer(exec, s)

	// 5. Start HTTP Server
	handler := srv.Routes()

	fmt.Printf("Starting server on %s\n", cfg.Server.Port)
	if err := http.ListenAndServe(cfg.Server.Port, handler); err != nil {
		log.Fatal(err)
	}
}

func registerProviders(sm secrets.SecretManager) {
	// Groq
	if key, err := sm.Get("GROQ_API_KEY"); err == nil {
		providers.Register(groq.NewGroqProvider(key))
		log.Println("Registered Groq provider")
	}

	// OpenAI
	if key, err := sm.Get("OPENAI_API_KEY"); err == nil {
		providers.Register(openai.NewOpenAIProvider(key))
		log.Println("Registered OpenAI provider")
	}

	// Anthropic
	if key, err := sm.Get("ANTHROPIC_API_KEY"); err == nil {
		providers.Register(anthropic.NewAnthropicProvider(key))
		log.Println("Registered Anthropic provider")
	}

	// Gemini
	if key, err := sm.Get("GEMINI_API_KEY"); err == nil {
		providers.Register(gemini.NewGeminiProvider(key))
		log.Println("Registered Gemini provider")
	}

	if len(providers.List()) == 0 {
		log.Println("Warning: No LLM providers were registered. Check your environment variables.")
	}
}
