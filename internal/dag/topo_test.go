package dag

import (
	"reflect"
	"sort"
	"testing"

	"github.com/AniruthKarthik/llm-orchestrator/internal/core"
)

func TestTopologicalPlanner_BuildExecutionPlan(t *testing.T) {
	planner := NewTopologicalPlanner()

	t.Run("Linear dependencies", func(t *testing.T) {
		wf := core.NewWorkflow("wf1", "Linear Workflow", "")
		t1 := core.NewTask("t1", "Task 1", "", nil, nil)
		t2 := core.NewTask("t2", "Task 2", "", nil, []string{"t1"})
		t3 := core.NewTask("t3", "Task 3", "", nil, []string{"t2"})

		wf.AddTask(t1)
		wf.AddTask(t2)
		wf.AddTask(t3)

		plan, err := planner.BuildExecutionPlan(wf)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if plan == nil {
			t.Fatal("plan is nil")
		}
		if len(plan.Stages) != 3 {
			t.Fatalf("expected 3 stages, got %d", len(plan.Stages))
		}

		if !reflect.DeepEqual(plan.Stages[0].TaskIDs, []string{"t1"}) {
			t.Errorf("stage 0: expected [t1], got %v", plan.Stages[0].TaskIDs)
		}
		if !reflect.DeepEqual(plan.Stages[1].TaskIDs, []string{"t2"}) {
			t.Errorf("stage 1: expected [t2], got %v", plan.Stages[1].TaskIDs)
		}
		if !reflect.DeepEqual(plan.Stages[2].TaskIDs, []string{"t3"}) {
			t.Errorf("stage 2: expected [t3], got %v", plan.Stages[2].TaskIDs)
		}
	})

	t.Run("Parallel dependencies", func(t *testing.T) {
		wf := core.NewWorkflow("wf2", "Parallel Workflow", "")
		t1 := core.NewTask("t1", "Task 1", "", nil, nil)
		t2 := core.NewTask("t2", "Task 2", "", nil, nil)
		t3 := core.NewTask("t3", "Task 3", "", nil, []string{"t1", "t2"})

		wf.AddTask(t1)
		wf.AddTask(t2)
		wf.AddTask(t3)

		plan, err := planner.BuildExecutionPlan(wf)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(plan.Stages) != 2 {
			t.Fatalf("expected 2 stages, got %d", len(plan.Stages))
		}

		// Sort IDs for consistent comparison
		stage0 := plan.Stages[0].TaskIDs
		sort.Strings(stage0)
		if !reflect.DeepEqual(stage0, []string{"t1", "t2"}) {
			t.Errorf("stage 0: expected [t1, t2], got %v", stage0)
		}
		if !reflect.DeepEqual(plan.Stages[1].TaskIDs, []string{"t3"}) {
			t.Errorf("stage 1: expected [t3], got %v", plan.Stages[1].TaskIDs)
		}
	})

	t.Run("Complex dependencies", func(t *testing.T) {
		wf := core.NewWorkflow("wf3", "Complex Workflow", "")
		t1 := core.NewTask("t1", "Task 1", "", nil, nil)
		t2 := core.NewTask("t2", "Task 2", "", nil, []string{"t1"})
		t3 := core.NewTask("t3", "Task 3", "", nil, []string{"t1"})
		t4 := core.NewTask("t4", "Task 4", "", nil, []string{"t2", "t3"})
		t5 := core.NewTask("t5", "Task 5", "", nil, nil)

		wf.AddTask(t1)
		wf.AddTask(t2)
		wf.AddTask(t3)
		wf.AddTask(t4)
		wf.AddTask(t5)

		plan, err := planner.BuildExecutionPlan(wf)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(plan.Stages) != 3 {
			t.Fatalf("expected 3 stages, got %d", len(plan.Stages))
		}

		stage0 := plan.Stages[0].TaskIDs
		sort.Strings(stage0)
		if !reflect.DeepEqual(stage0, []string{"t1", "t5"}) {
			t.Errorf("stage 0: expected [t1, t5], got %v", stage0)
		}

		stage1 := plan.Stages[1].TaskIDs
		sort.Strings(stage1)
		if !reflect.DeepEqual(stage1, []string{"t2", "t3"}) {
			t.Errorf("stage 1: expected [t2, t3], got %v", stage1)
		}

		if !reflect.DeepEqual(plan.Stages[2].TaskIDs, []string{"t4"}) {
			t.Errorf("stage 2: expected [t4], got %v", plan.Stages[2].TaskIDs)
		}
	})

	t.Run("Empty workflow", func(t *testing.T) {
		wf := core.NewWorkflow("wf4", "Empty Workflow", "")
		plan, err := planner.BuildExecutionPlan(wf)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(plan.Stages) != 0 {
			t.Errorf("expected 0 stages, got %d", len(plan.Stages))
		}
	})
}
