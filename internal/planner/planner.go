package planner

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/AniruthKarthik/llm-orchestrator/internal/llm"
	"github.com/AniruthKarthik/llm-orchestrator/internal/models"
)

const maxPlanRetries = 3

type Planner struct {
	llm llm.Client
}

// New creates a new Planner with the provided LLM client.
func New(client llm.Client) *Planner {
	return &Planner{llm: client}
}

// Plan uses the LLM to decompose a goal into a series of executable tasks.
func (p *Planner) Plan(ctx context.Context, goal string) ([]models.Task, error) {
	prompt := buildPrompt(goal)
	var lastErr error

	for attempt := 1; attempt <= maxPlanRetries; attempt++ {
		log.Printf("[planner] planning attempt %d/%d for goal: %q", attempt, maxPlanRetries, goal)

		resp, err := p.llm.SendPrompt(ctx, prompt)
		if err != nil {
			lastErr = fmt.Errorf("LLM call failed on attempt %d: %w", attempt, err)
			log.Printf("[planner] %v", lastErr)

			if ctx.Err() != nil {
				return nil, lastErr
			}
			continue
		}

		tasks, err := parseTasks(resp.Content)
		if err != nil {
			lastErr = fmt.Errorf("parse/validate failed on attempt %d: %w", attempt, err)
			log.Printf("[planner] %v", lastErr)
			continue
		}

		log.Printf("[planner] produced %d tasks for goal %q", len(tasks), goal)
		return tasks, nil
	}

	return nil, fmt.Errorf("planner exhausted %d attempts,last error: %w", maxPlanRetries, lastErr)
}

// buildPrompt constructs the instruction sent to the LLM for task planning.
func buildPrompt(goal string) string {

	return fmt.Sprintf(`You are a task planning engine.
Given a high-level goal, decompose it into an ordered list of atomic tasks.
 
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
 
Goal: %s`, goal)
}

// parseTasks converts the raw LLM response into a validated slice of Task models.
func parseTasks(raw json.RawMessage) ([]models.Task, error) {
	if !json.Valid(raw) {
		return nil, fmt.Errorf("LLM content is invalid")
	}

	var tasks []models.Task
	if err := json.Unmarshal(raw, &tasks); err != nil {
		return nil, fmt.Errorf("cannot Unmarshal task array: %w", err)
	}

	if len(tasks) == 0 {
		return nil, fmt.Errorf("LLM returned an empty task list")
	}

	seen := make(map[string]struct{}, len(tasks))

	for i := range tasks {
		if err := validateTask(i, &tasks[i], seen); err != nil {
			return nil, err
		}

		seen[tasks[i].ID] = struct{}{}
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
func validateTask(index int, t *models.Task, seen map[string]struct{}) error {
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
