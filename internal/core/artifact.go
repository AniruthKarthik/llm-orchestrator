package core

import (
	"fmt"
	"sync"
	"time"
)

// ArtifactType defines the kind of data stored in an artifact.
type ArtifactType string

const (
	ArtifactTypeText ArtifactType = "TEXT"
	ArtifactTypeCode ArtifactType = "CODE"
	ArtifactTypeFile ArtifactType = "FILE"
	ArtifactTypeJSON ArtifactType = "JSON"
)

// Artifact represents a piece of data produced or consumed by a task.
type Artifact struct {
	ID         string
	WorkflowID string
	TaskID     string // The task that produced this artifact
	Name       string
	Type       ArtifactType
	Data       any
	Metadata   map[string]any
	CreatedAt  time.Time
}

// ArtifactRegistry manages artifacts in memory.
type ArtifactRegistry struct {
	artifacts map[string]*Artifact
	mu        sync.RWMutex
}

func NewArtifactRegistry() *ArtifactRegistry {
	return &ArtifactRegistry{
		artifacts: make(map[string]*Artifact),
	}
}

func (r *ArtifactRegistry) Register(a *Artifact) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.artifacts[a.ID]; exists {
		return fmt.Errorf("artifact already exists: %s", a.ID)
	}
	r.artifacts[a.ID] = a
	return nil
}

func (r *ArtifactRegistry) Get(id string) (*Artifact, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, exists := r.artifacts[id]
	return a, exists
}

func (r *ArtifactRegistry) ListByWorkflow(workflowID string) []*Artifact {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := make([]*Artifact, 0)
	for _, a := range r.artifacts {
		if a.WorkflowID == workflowID {
			list = append(list, a)
		}
	}
	return list
}

func (r *ArtifactRegistry) ListByTask(taskID string) []*Artifact {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := make([]*Artifact, 0)
	for _, a := range r.artifacts {
		if a.TaskID == taskID {
			list = append(list, a)
		}
	}
	return list
}
