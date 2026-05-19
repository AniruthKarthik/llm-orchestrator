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
	agents      map[string]AgentRecord
	artifacts   map[string]ArtifactRecord

	mutex sync.RWMutex
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		workflows:   make(map[string]WorkflowRecord),
		tasks:       make(map[string]map[string]TaskRecord),
		checkpoints: make(map[string][]CheckpointRecord),
		agents:      make(map[string]AgentRecord),
		artifacts:   make(map[string]ArtifactRecord),
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
		// Workflow exists but has no tasks yet — return empty slice, not error
		return []TaskRecord{}, nil
	}

	tasks := make([]TaskRecord, 0, len(workflowTasks))

	for _, task := range workflowTasks {
		tasks = append(tasks, deepCopyTask(task))
	}

	return tasks, nil
}

func (m *MemoryStore) ListTasksByStatus(
	status string,
) ([]TaskRecord, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	list := make([]TaskRecord, 0)
	for _, workflowTasks := range m.tasks {
		for _, task := range workflowTasks {
			if task.Status == status {
				list = append(list, deepCopyTask(task))
			}
		}
	}

	return list, nil
}

func (m *MemoryStore) DeleteTask(workflowID string, taskID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	workflowTasks, exists := m.tasks[workflowID]
	if !exists {
		return fmt.Errorf("workflow tasks not found: %s", workflowID)
	}

	if _, exists := workflowTasks[taskID]; !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}

	delete(workflowTasks, taskID)
	return nil
}

func deepCopyTask(t TaskRecord) TaskRecord {
	cp := t
	cp.Input = core.DeepCopyMap(t.Input)
	cp.Output = core.DeepCopyMap(t.Output)
	cp.Dependencies = core.DeepCopyStringSlice(t.Dependencies)
	cp.OutputSchema = core.DeepCopyStringMap(t.OutputSchema)
	return cp
}

func (m *MemoryStore) SaveAgent(agent AgentRecord) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	cp := agent
	cp.Tools = core.DeepCopyStringSlice(agent.Tools)
	cp.Config = core.DeepCopyMap(agent.Config)

	m.agents[agent.ID] = cp
	return nil
}

func (m *MemoryStore) GetAgent(agentID string) (AgentRecord, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	agent, exists := m.agents[agentID]
	if !exists {
		return AgentRecord{}, fmt.Errorf("agent not found: %s", agentID)
	}

	cp := agent
	cp.Tools = core.DeepCopyStringSlice(agent.Tools)
	cp.Config = core.DeepCopyMap(agent.Config)

	return cp, nil
}

func (m *MemoryStore) ListAgents() ([]AgentRecord, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	list := make([]AgentRecord, 0, len(m.agents))
	for _, agent := range m.agents {
		cp := agent
		cp.Tools = core.DeepCopyStringSlice(agent.Tools)
		cp.Config = core.DeepCopyMap(agent.Config)
		list = append(list, cp)
	}

	return list, nil
}

func (m *MemoryStore) SaveArtifact(artifact ArtifactRecord) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	cp := artifact
	cp.Data = core.DeepCopyValue(artifact.Data)
	cp.Metadata = core.DeepCopyMap(artifact.Metadata)

	m.artifacts[artifact.ID] = cp
	return nil
}

func (m *MemoryStore) GetArtifact(artifactID string) (ArtifactRecord, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	artifact, exists := m.artifacts[artifactID]
	if !exists {
		return ArtifactRecord{}, fmt.Errorf("artifact not found: %s", artifactID)
	}

	cp := artifact
	cp.Data = core.DeepCopyValue(artifact.Data)
	cp.Metadata = core.DeepCopyMap(artifact.Metadata)

	return cp, nil
}

func (m *MemoryStore) ListArtifactsByWorkflow(workflowID string) ([]ArtifactRecord, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	list := make([]ArtifactRecord, 0)
	for _, artifact := range m.artifacts {
		if artifact.WorkflowID == workflowID {
			cp := artifact
			cp.Data = core.DeepCopyValue(artifact.Data)
			cp.Metadata = core.DeepCopyMap(artifact.Metadata)
			list = append(list, cp)
		}
	}

	return list, nil
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

func (m *MemoryStore) ListWorkflows() ([]WorkflowRecord, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	workflows := make([]WorkflowRecord, 0, len(m.workflows))
	for _, w := range m.workflows {
		workflows = append(workflows, w)
	}

	return workflows, nil
}

func (m *MemoryStore) ListAllArtifacts() ([]ArtifactRecord, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	list := make([]ArtifactRecord, 0, len(m.artifacts))
	for _, artifact := range m.artifacts {
		cp := artifact
		cp.Data = core.DeepCopyValue(artifact.Data)
		cp.Metadata = core.DeepCopyMap(artifact.Metadata)
		list = append(list, cp)
	}
	return list, nil
}

func (m *MemoryStore) DeleteWorkflow(workflowID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.workflows[workflowID]; !exists {
		return fmt.Errorf("workflow not found: %s", workflowID)
	}

	delete(m.workflows, workflowID)
	delete(m.tasks, workflowID)
	delete(m.checkpoints, workflowID)
	return nil
}
