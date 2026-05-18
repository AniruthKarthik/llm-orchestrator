package dag

import (
	"errors"
	"testing"

	"github.com/AniruthKarthik/llm-orchestrator/internal/core"
)

func TestValidator_Validate(t *testing.T) {
	v := NewValidator()

	t.Run("Valid workflow", func(t *testing.T) {
		w := core.NewWorkflow("w1", "test", "")
		t1 := core.NewTask("t1", w.ID, "task 1", "", nil, nil)
		t2 := core.NewTask("t2", w.ID, "task 2", "", nil, []string{"t1"})
		w.AddTask(t1)
		w.AddTask(t2)

		if err := v.Validate(w); err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("Unknown dependency", func(t *testing.T) {
		w := core.NewWorkflow("w1", "test", "")
		t1 := core.NewTask("t1", w.ID, "task 1", "", nil, []string{"unknown"})
		w.AddTask(t1)

		err := v.Validate(w)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, ErrTaskNotFound) {
			t.Errorf("expected ErrTaskNotFound, got %v", err)
		}
	})

	t.Run("Self dependency", func(t *testing.T) {
		w := core.NewWorkflow("w1", "test", "")
		t1 := core.NewTask("t1", w.ID, "task 1", "", nil, []string{"t1"})
		w.AddTask(t1)

		err := v.Validate(w)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, ErrSelfDependency) {
			t.Errorf("expected ErrSelfDependency, got %v", err)
		}
	})

	t.Run("Circular dependency (direct)", func(t *testing.T) {
		w := core.NewWorkflow("w1", "test", "")
		t1 := core.NewTask("t1", w.ID, "task 1", "", nil, []string{"t2"})
		t2 := core.NewTask("t2", w.ID, "task 2", "", nil, []string{"t1"})
		w.AddTask(t1)
		w.AddTask(t2)

		err := v.Validate(w)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, ErrCircularDependency) {
			t.Errorf("expected ErrCircularDependency, got %v", err)
		}
	})

	t.Run("Circular dependency (indirect)", func(t *testing.T) {
		w := core.NewWorkflow("w1", "test", "")
		t1 := core.NewTask("t1", w.ID, "task 1", "", nil, []string{"t2"})
		t2 := core.NewTask("t2", w.ID, "task 2", "", nil, []string{"t3"})
		t3 := core.NewTask("t3", w.ID, "task 3", "", nil, []string{"t1"})
		w.AddTask(t1)
		w.AddTask(t2)
		w.AddTask(t3)

		err := v.Validate(w)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, ErrCircularDependency) {
			t.Errorf("expected ErrCircularDependency, got %v", err)
		}
	})
}
