package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/AniruthKarthik/llm-orchestrator/internal/agents"
	"github.com/AniruthKarthik/llm-orchestrator/internal/config"
	"github.com/AniruthKarthik/llm-orchestrator/internal/core"
	"github.com/AniruthKarthik/llm-orchestrator/internal/dsl"
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

func registerProviders(sm secrets.SecretManager) {
	groqKey, err := sm.Get("GROQ_API_KEY")
	if err == nil {
		providers.Register(groq.NewGroqProvider(groqKey))
		log.Println("Registered Groq provider")
	} else {
		// Fallback for local MVP testing
		providers.Register(&DummyProvider{})
		log.Println("GROQ_API_KEY not found. Registered Dummy provider as 'groq'")
	}
	
	if key, err := sm.Get("OPENAI_API_KEY"); err == nil {
		providers.Register(openai.NewOpenAIProvider(key))
		log.Println("Registered OpenAI provider")
	}
	if key, err := sm.Get("ANTHROPIC_API_KEY"); err == nil {
		providers.Register(anthropic.NewAnthropicProvider(key))
		log.Println("Registered Anthropic provider")
	}
	if key, err := sm.Get("GEMINI_API_KEY"); err == nil {
		providers.Register(gemini.NewGeminiProvider(key))
		log.Println("Registered Gemini provider")
	}
}

// DummyProvider for local MVP testing without API keys
type DummyProvider struct{}
func (p *DummyProvider) Name() string { return "groq" }
func (p *DummyProvider) Capabilities() providers.Capabilities { return providers.Capabilities{} }
func (p *DummyProvider) Stream(ctx context.Context, req providers.GenerateRequest) (<-chan providers.StreamChunk, <-chan error) { return nil, nil }
func (p *DummyProvider) Generate(ctx context.Context, req providers.GenerateRequest) (*providers.GenerateResponse, error) {
	return &providers.GenerateResponse{
		Content: `{"joke": "Why do programmers prefer dark mode? Because light attracts bugs!"}`,
	}, nil
}

func main() {
	// 1. Load Configuration (this also loads .env)
	_ = config.Load()

	if len(os.Args) < 2 {
		fmt.Println("Usage: orch <workflow.yaml>")
		os.Exit(1)
	}

	yamlFile := os.Args[1]
	data, err := os.ReadFile(yamlFile)
	if err != nil {
		fmt.Printf("Failed to read file: %v\n", err)
		os.Exit(1)
	}

	// 1. Parse YAML
	parser := &dsl.YAMLParser{}
	def, err := parser.Parse(data)
	if err != nil {
		fmt.Printf("Failed to parse YAML: %v\n", err)
		os.Exit(1)
	}

	// 2. Setup providers
	sm := secrets.NewEnvSecretManager()
	registerProviders(sm)

	// 3. Compile to Workflow
	compiler := &dsl.Compiler{}
	workflow, err := compiler.Compile(def)
	if err != nil {
		fmt.Printf("Failed to compile workflow: %v\n", err)
		os.Exit(1)
	}

	// 3. Setup core dependencies
	memoryStore := store.NewMemoryStore()
	eventBus := events.NewEventBus(10)
	wr := executor.NewWorkerRegistry()
	ar := agents.NewAgentRegistry()
	art := core.NewArtifactRegistry()
	mem := core.NewMemoryRegistry()
	tr := core.NewToolRegistry()
	tp := core.NewToolPolicy()

	// Register agents for testing
	ar.Register(&agents.Agent{
		ID:           "researcher-1",
		Name:         "Test Researcher",
		Role:         agents.RoleResearcher,
	})
	ar.Register(&agents.Agent{
		ID:           "comedian-1",
		Name:         "Golang Comedian",
		Role:         agents.RoleExecutor,
		Model:        "llama-3.1-8b-instant", // Updated from decommissioned llama3-8b-8192
		Provider:     "groq",
		SystemPrompt: "You are a witty stand-up comedian. You must output your response as valid JSON with a single key 'joke' containing your joke string.",
	})

	// 5. Create Executor
	exec := executor.NewExecutor(wr, ar, art, mem, tr, tp, eventBus, memoryStore)
	exec.UseTaskMiddleware(executor.PanicRecoveryMiddleware)
	exec.UseTaskMiddleware(executor.OutputValidationMiddleware)

	// 5. Start Supervisor
	sup := executor.NewSupervisor(memoryStore, exec, 30*time.Second)
	sup.Start()
	defer sup.Stop()

	// Listen to events
	auditLogger := observer.NewAuditLogger(memoryStore)
	eventBus.Subscribe(events.TaskStarted, auditLogger.Handle)
	eventBus.Subscribe(events.TaskCompleted, auditLogger.Handle)
	eventBus.Subscribe(events.TaskFailed, auditLogger.Handle)

	eventBus.Subscribe(events.TaskStarted, func(e events.Event) {
		fmt.Printf("[Orch] Task Started: %s\n", e.TaskID)
	})
	eventBus.Subscribe(events.TaskCompleted, func(e events.Event) {
		fmt.Printf("[Orch] Task Completed: %s\n", e.TaskID)
	})
	eventBus.Subscribe(events.TaskFailed, func(e events.Event) {
		fmt.Printf("[Orch] Task Failed: %s\n", e.TaskID)
	})

	fmt.Printf("Executing Workflow: %s (%s)\n", workflow.Name, workflow.ID)
	
	// 6. Execute Workflow
	if err := exec.Execute(workflow); err != nil {
		fmt.Printf("Execution failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Execution completed successfully!")
	time.Sleep(100 * time.Millisecond)
}
