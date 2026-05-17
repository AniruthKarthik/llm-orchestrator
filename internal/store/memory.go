package store

import (
	"fmt"
	"sync"

	"github.com/AniruthKarthik/llm-orchestrator/internal/core"
)

type MemoryStore struct {
	workflows   map[string]WorkflowRecord
	tasks       map[string]map[string]TaskRecord
	checkpoints map[string][]CheckpointRecord

	mutex sync.RWMutex
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		workflows:   make(map[string]WorkflowRecord),
		tasks:       make(map[string]map[string]TaskRecord),
		checkpoints: make(map[string][]CheckpointRecord),
	}
}

func (m *MemoryStore) SaveWorkflow(
	workflow WorkflowRecord,
) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.workflows[workflow.ID] = workflow

	if _, exists := m.tasks[workflow.ID]; !exists {
		m.tasks[workflow.ID] = make(map[string]TaskRecord)
	}

	return nil
}

func (m *MemoryStore) UpdateWorkflow(
	workflow WorkflowRecord,
) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.workflows[workflow.ID]; !exists {
		return fmt.Errorf(
			"workflow not found: %s",
			workflow.ID,
		)
	}

	m.workflows[workflow.ID] = workflow

	return nil
}

func (m *MemoryStore) GetWorkflow(
	workflowID string,
) (WorkflowRecord, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	workflow, exists := m.workflows[workflowID]
	if !exists {
		return WorkflowRecord{}, fmt.Errorf(
			"workflow not found: %s",
			workflowID,
		)
	}

	return workflow, nil
}

func (m *MemoryStore) SaveTask(
	task TaskRecord,
) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.workflows[task.WorkflowID]; !exists {
		return fmt.Errorf(
			"workflow not found: %s",
			task.WorkflowID,
		)
	}

	if _, exists := m.tasks[task.WorkflowID]; !exists {
		m.tasks[task.WorkflowID] = make(map[string]TaskRecord)
	}

	m.tasks[task.WorkflowID][task.ID] = deepCopyTask(task)

	return nil
}

func (m *MemoryStore) UpdateTask(
	task TaskRecord,
) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	workflowTasks, exists := m.tasks[task.WorkflowID]
	if !exists {
		return fmt.Errorf(
			"workflow tasks not found: %s",
			task.WorkflowID,
		)
	}

	if _, exists := workflowTasks[task.ID]; !exists {
		return fmt.Errorf(
			"task not found: %s",
			task.ID,
		)
	}

	workflowTasks[task.ID] = deepCopyTask(task)

	return nil
}

func (m *MemoryStore) GetTask(
	workflowID string,
	taskID string,
) (TaskRecord, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	workflowTasks, exists := m.tasks[workflowID]
	if !exists {
		return TaskRecord{}, fmt.Errorf(
			"workflow tasks not found: %s",
			workflowID,
		)
	}

	task, exists := workflowTasks[taskID]
	if !exists {
		return TaskRecord{}, fmt.Errorf(
			"task not found: %s",
			taskID,
		)
	}

	return deepCopyTask(task), nil
}

func (m *MemoryStore) GetWorkflowTasks(
	workflowID string,
) ([]TaskRecord, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	workflowTasks, exists := m.tasks[workflowID]
	if !exists {
		return nil, fmt.Errorf(
			"workflow tasks not found: %s",
			workflowID,
		)
	}

	tasks := make([]TaskRecord, 0, len(workflowTasks))

	for _, task := range workflowTasks {
		tasks = append(tasks, deepCopyTask(task))
	}

	return tasks, nil
}

func deepCopyTask(t TaskRecord) TaskRecord {
	cp := t
	cp.Input = core.DeepCopyMap(t.Input)
	cp.Output = core.DeepCopyMap(t.Output)
	cp.Dependencies = core.DeepCopyStringSlice(t.Dependencies)
	return cp
}

func (m *MemoryStore) SaveCheckpoint(checkpoint CheckpointRecord) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	cp := checkpoint
	if checkpoint.StateData != nil {
		cp.StateData = make([]byte, len(checkpoint.StateData))
		copy(cp.StateData, checkpoint.StateData)
	}

	m.checkpoints[checkpoint.WorkflowID] = append(m.checkpoints[checkpoint.WorkflowID], cp)
	return nil
}

func (m *MemoryStore) GetLatestCheckpoint(workflowID string) (CheckpointRecord, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	checkpoints, exists := m.checkpoints[workflowID]
	if !exists || len(checkpoints) == 0 {
		return CheckpointRecord{}, fmt.Errorf("no checkpoints found for workflow: %s", workflowID)
	}

	cp := checkpoints[len(checkpoints)-1]
	if cp.StateData != nil {
		dataCopy := make([]byte, len(cp.StateData))
		copy(dataCopy, cp.StateData)
		cp.StateData = dataCopy
	}

	return cp, nil
}
