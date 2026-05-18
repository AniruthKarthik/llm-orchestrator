package agents

import (
	"context"
	"testing"
)

func TestPlanner_GenerateAndCompile(t *testing.T) {
	ar := NewAgentRegistry()
	plannerAgent := &Agent{
		ID:   "planner-1",
		Name: "Global Planner",
		Role: RolePlanner,
	}
	ar.Register(plannerAgent)

	planner := NewPlanner(ar)
	
	ctx := context.Background()
	objective := "Analyze the impact of AI on software engineering"
	
	plan, err := planner.GeneratePlan(ctx, "planner-1", objective)
	if err != nil {
		t.Fatalf("GeneratePlan failed: %v", err)
	}

	if len(plan.Tasks) == 0 {
		t.Fatal("Generated plan has no tasks")
	}

	workflow := planner.CompilePlan(plan)
	
	if workflow.Name != plan.Name {
		t.Errorf("Expected workflow name %s, got %s", plan.Name, workflow.Name)
	}

	if len(workflow.Tasks) != len(plan.Tasks) {
		t.Errorf("Expected %d tasks, got %d", len(plan.Tasks), len(workflow.Tasks))
	}

	// Verify dependencies
	task2, err := workflow.GetTask("task-2")
	if err != nil {
		t.Fatalf("Failed to get task-2: %v", err)
	}
	
	if len(task2.Dependencies) != 1 || task2.Dependencies[0] != "task-1" {
		t.Errorf("Task 2 dependencies incorrect: %v", task2.Dependencies)
	}
}
