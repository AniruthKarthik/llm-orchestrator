package planner

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/AniruthKarthik/llm-orchestrator/internal/llm"
	"github.com/AniruthKarthik/llm-orchestrator/internal/memory"
	"github.com/AniruthKarthik/llm-orchestrator/internal/models"
	"github.com/AniruthKarthik/llm-orchestrator/internal/observability"
)

const maxPlanRetries = 3

type Planner struct {
	llm       llm.Client
	retriever *memory.Retriever
	obs       *observability.Obs
}

// New creates a new Planner with the provided LLM client and options.
func New(client llm.Client, opts ...Option) *Planner {
	p := &Planner{
		llm: client,
		obs: &observability.Obs{
			Log:     observability.NoopLogger{},
			Tracer:  observability.NoopTracer{},
			Metrics: observability.NoopMetrics{},
		},
	}
	for _, o := range opts {
		o(p)
	}
	return p
}

type Option func(*Planner)

// WithRetriever configures the planner to use a memory retriever.
func WithRetriever(r *memory.Retriever) Option {
	return func(p *Planner) { p.retriever = r }
}

// WithObs configures the planner to use a specific observability bundle.
func WithObs(obs *observability.Obs) Option {
	return func(p *Planner) { p.obs = obs }
}

// Plan uses the LLM to decompose a goal into a series of executable tasks.
func (p *Planner) Plan(ctx context.Context, goal string) ([]models.Task, error) {
	spanCtx, span := p.obs.Tracer.Start(ctx, "planner.Plan")
	defer span.End(spanCtx)

	p.obs.Log.Info(spanCtx, "planner started",
		observability.F("goal", goal),
	)

	var memContext string
	if p.retriever != nil {
		var err error
		memContext, err = p.retriever.RetrieveContext(spanCtx, goal)
		if err != nil {
			p.obs.Log.Warn(spanCtx, "planner: memory retrieval failed",
				observability.F("err", err.Error()))
		} else if memContext != "" {
			p.obs.Log.Debug(spanCtx, "planner: injecting memory context",
				observability.F("context_len", len(memContext)))
		}
	}

	prompt := buildPrompt(goal, memContext)
	var lastErr error

	for attempt := 1; attempt <= maxPlanRetries; attempt++ {
		p.obs.Log.Info(spanCtx, "planner LLM attempt",
			observability.F("attempt", attempt),
			observability.F("max_attempts", maxPlanRetries),
		)

		resp, err := p.llm.SendPrompt(spanCtx, prompt)
		if err != nil {
			lastErr = fmt.Errorf("LLM call failed on attempt %d: %w", attempt, err)
			p.obs.Log.Warn(spanCtx, "planner: LLM call failed", observability.F("err", lastErr.Error()))
			if spanCtx.Err() != nil {
				span.SetError(lastErr)
				return nil, lastErr
			}
			continue
		}

		tasks, err := parseTasks(resp.Content)
		if err != nil {
			lastErr = fmt.Errorf("parse/validate failed on attempt %d: %w", attempt, err)
			p.obs.Log.Warn(spanCtx, "planner: parse failed", observability.F("err", lastErr.Error()))
			continue
		}

		p.obs.Log.Info(spanCtx, "planner produced tasks",
			observability.F("task_count", len(tasks)),
		)
		return tasks, nil
	}

	span.SetError(lastErr)
	return nil, fmt.Errorf("planner exhausted %d attempts, last error: %w", maxPlanRetries, lastErr)
}

// buildPrompt constructs the instruction sent to the LLM for task planning.
func buildPrompt(goal, memContext string) string {
	contextBlock := ""
	if memContext != "" {
		contextBlock = "\n\nPRIOR CONTEXT FROM MEMORY (use if relevant):\n" + memContext + "\n"
	}
	return fmt.Sprintf(`You are a task planning engine.
Given a high-level goal, decompose it into an ordered list of atomic tasks.%s
STRICT OUTPUT RULES:
- Respond with ONLY a valid JSON array.
- No explanations, no markdown, no code fences.
- Each element MUST conform to this schema:
  {
    "id":         string   (unique, e.g. "task-001"),
    "type":       string   (e.g. "research", "generate", "validate"),
    "payload":    object   (arbitrary key-value relevant to the task),
    "status":     string   (always "pending" for new tasks),
    "depends_on": []string (IDs of tasks that must complete first; [] if none),
    "result":     null,
    "retries":    0
  }
 
Goal: %s`, contextBlock, goal)
}

// parseTasks converts the raw LLM response into a validated slice of Task models.
func parseTasks(raw json.RawMessage) ([]models.Task, error) {
	if !json.Valid(raw) {
		return nil, fmt.Errorf("LLM content is not valid JSON")
	}
	var tasks []models.Task
	if err := json.Unmarshal(raw, &tasks); err != nil {
		return nil, fmt.Errorf("cannot unmarshal task array: %w", err)
	}
	if len(tasks) == 0 {
		return nil, fmt.Errorf("LLM returned an empty task list")
	}
	seen := make(map[string]struct{}, len(tasks))
	for i, t := range tasks {
		if err := validateTask(i, t, seen); err != nil {
			return nil, err
		}
		seen[t.ID] = struct{}{}
	}
	for _, t := range tasks {
		for _, dep := range t.DependsOn {
			if _, ok := seen[dep]; !ok {
				return nil, fmt.Errorf("task %q references unknown dependency %q", t.ID, dep)
			}
		}
	}
	return tasks, nil
}

// validateTask ensures an individual task has all required fields and a unique ID.
func validateTask(index int, t models.Task, seen map[string]struct{}) error {
	if t.ID == "" {
		return fmt.Errorf("task[%d]: missing required field 'id'", index)
	}
	if _, dup := seen[t.ID]; dup {
		return fmt.Errorf("task[%d]: duplicate id %q", index, t.ID)
	}
	if t.Type == "" {
		return fmt.Errorf("task %q: missing required field 'type'", t.ID)
	}
	if t.Status == "" {
		return fmt.Errorf("task %q: missing required field 'status'", t.ID)
	}
	if t.DependsOn == nil {
		t.DependsOn = []string{}
	}
	return nil
}
