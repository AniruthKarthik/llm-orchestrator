package dag

import (
	"github.com/AniruthKarthik/llm-orchestrator/internal/core"
)

type TopologicalPlanner struct {
	validator *Validator
}

func NewTopologicalPlanner() *TopologicalPlanner {
	return &TopologicalPlanner{
		validator: NewValidator(),
	}
}

func (p *TopologicalPlanner) BuildExecutionPlan(
	workflow *core.Workflow,
) (*ExecutionPlan, error) {
	if err := p.validator.Validate(workflow); err != nil {
		return nil, err
	}

	tasks := workflow.GetTasks()
	plan := NewExecutionPlan(workflow.ID)
	indegree := p.buildIndegreeMap(tasks)
	adjacency := p.buildAdjacencyList(tasks)
	processed := make(map[string]bool)

	level := 0
	for {
		currentLevelTasks := p.collectZeroIndegree(indegree, processed)
		if len(currentLevelTasks) == 0 {
			break
		}

		stage := ExecutionStage{
			Level:   level,
			TaskIDs: currentLevelTasks,
		}
		plan.AddStage(stage)

		for _, taskID := range currentLevelTasks {
			processed[taskID] = true
			p.decrementNeighbors(taskID, adjacency, indegree)
		}
		level++
	}

	return plan, nil
}

func (p *TopologicalPlanner) buildIndegreeMap(
	tasks map[string]*core.Task,
) map[string]int {
	indegree := make(map[string]int)
	for id, task := range tasks {
		if _, exists := indegree[id]; !exists {
			indegree[id] = 0
		}
		indegree[id] = len(task.Dependencies)
	}
	return indegree
}

func (p *TopologicalPlanner) buildAdjacencyList(
	tasks map[string]*core.Task,
) map[string][]string {
	adjacency := make(map[string][]string)
	for id, task := range tasks {
		for _, dep := range task.Dependencies {
			adjacency[dep] = append(adjacency[dep], id)
		}
	}
	return adjacency
}

func (p *TopologicalPlanner) collectZeroIndegree(
	indegree map[string]int,
	processed map[string]bool,
) []string {
	var tasks []string
	for id, count := range indegree {
		if count == 0 && !processed[id] {
			tasks = append(tasks, id)
		}
	}
	return tasks
}

func (p *TopologicalPlanner) decrementNeighbors(
	taskID string,
	adjacency map[string][]string,
	indegree map[string]int,
) {
	for _, neighbor := range adjacency[taskID] {
		indegree[neighbor]--
	}
}
