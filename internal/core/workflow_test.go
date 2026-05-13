package core

import (
	"testing"
)

func TestWorkflow_ReadyTasks(t *testing.T) {
	w := NewWorkflow("w1", "test", "")
	
	t1 := NewTask("t1", "task 1", "", nil, nil)
	t2 := NewTask("t2", "task 2", "", nil, []string{"t1"})
	t3 := NewTask("t3", "task 3", "", nil, []string{"t1"})
	t4 := NewTask("t4", "task 4", "", nil, []string{"t2", "t3"})
	
	w.AddTask(t1)
	w.AddTask(t2)
	w.AddTask(t3)
	w.AddTask(t4)

	// Initially, only t1 should be ready
	ready := w.ReadyTasks()
	if len(ready) != 1 || ready[0].ID != "t1" {
		t.Errorf("expected only t1 to be ready, got %v", ready)
	}

	// Complete t1
	t1.Status = TaskRunning
	t1.Complete(nil)

	// Now t2 and t3 should be ready
	ready = w.ReadyTasks()
	if len(ready) != 2 {
		t.Errorf("expected 2 tasks to be ready, got %d", len(ready))
	}
	
	foundT2, foundT3 := false, false
	for _, task := range ready {
		if task.ID == "t2" {
			foundT2 = true
		}
		if task.ID == "t3" {
			foundT3 = true
		}
	}
	if !foundT2 || !foundT3 {
		t.Errorf("expected t2 and t3 to be ready, got %v", ready)
	}

	// Complete t2
	t2.Status = TaskRunning
	t2.Complete(nil)

	// Still only t3 should be ready (t4 needs t3 too)
	ready = w.ReadyTasks()
	if len(ready) != 1 || ready[0].ID != "t3" {
		t.Errorf("expected only t3 to be ready, got %v", ready)
	}

	// Complete t3
	t3.Status = TaskRunning
	t3.Complete(nil)

	// Now t4 should be ready
	ready = w.ReadyTasks()
	if len(ready) != 1 || ready[0].ID != "t4" {
		t.Errorf("expected only t4 to be ready, got %v", ready)
	}
}
