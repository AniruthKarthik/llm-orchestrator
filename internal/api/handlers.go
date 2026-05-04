package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/AniruthKarthik/llm-orchestrator/internal/models"
	"github.com/AniruthKarthik/llm-orchestrator/internal/planner"
	"github.com/AniruthKarthik/llm-orchestrator/internal/utils"
)

type jobStore struct {
	mu   sync.RWMutex
	jobs map[string]*models.Job
}

func newJobStore() *jobStore {
	return &jobStore{jobs: make(map[string]*models.Job)}
}

func (s *jobStore) set(job *models.Job) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs[job.ID] = job
}

func (s *jobStore) get(id string) (*models.Job, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	j, ok := s.jobs[id]
	return j, ok
}

type Handler struct {
	store   *jobStore
	planner *planner.Planner
}

func NewHandler(p *planner.Planner) *Handler {
	return &Handler{
		store:   newJobStore(),
		planner: p,
	}
}

type createJobRequest struct {
	Goal string `json:"goal"`
}

type createJobResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

func (h *Handler) CreateJob(w http.ResponseWriter, r *http.Request) {
	var req createJobRequest
	if err := utils.DecodeJSON(r, &req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	req.Goal = strings.TrimSpace(req.Goal)
	if req.Goal == "" {
		utils.WriteError(w, http.StatusBadRequest, "'goal' must not be empty")
		return
	}

	jobID := newID()
	now := time.Now().UTC()
	job := &models.Job{
		ID:        jobID,
		Goal:      req.Goal,
		Status:    models.JobStatusPlanning,
		Tasks:     []models.Task{},
		CreatedAt: now,
		UpdatedAt: now,
	}
	h.store.set(job)

	log.Printf("[handler] created job %s for goal: %q", jobID, req.Goal)

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	tasks, err := h.planner.Plan(ctx, req.Goal)
	if err != nil {
		log.Printf("[handler] planning failed for job %s: %v", jobID, err)
		h.updateJob(jobID, func(j *models.Job) {
			j.Status = models.JobStatusFailed
			j.Error = fmt.Sprintf("planning failed: %v", err)
			j.UpdatedAt = time.Now().UTC()
		})
		utils.WriteJSON(w, http.StatusAccepted, createJobResponse{ID: jobID, Status: string(models.JobStatusFailed)})
		return
	}

	h.updateJob(jobID, func(j *models.Job) {
		j.Tasks = tasks
		j.Status = models.JobStatusQueued
		j.UpdatedAt = time.Now().UTC()
	})

	utils.WriteJSON(w, http.StatusAccepted, createJobResponse{ID: jobID, Status: string(models.JobStatusQueued)})
}

func (h *Handler) GetJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "missing job id in path")
		return
	}

	job, ok := h.store.get(id)
	if !ok {
		utils.WriteError(w, http.StatusNotFound, fmt.Sprintf("job %q not found", id))
		return
	}

	utils.WriteJSON(w, http.StatusOK, job)
}

func (h *Handler) GetJobResult(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "missing job id in path")
		return
	}

	job, ok := h.store.get(id)
	if !ok {
		utils.WriteError(w, http.StatusNotFound, fmt.Sprintf("job %q not found", id))
		return
	}

	if job.Status != models.JobStatusCompleted {
		utils.WriteJSON(w, http.StatusAccepted, map[string]any{
			"id":     job.ID,
			"status": job.Status,
			"result": nil,
			"note":   "result not yet available; poll GET /job/{id} for status",
		})
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]any{
		"id":     job.ID,
		"status": job.Status,
		"result": job.Result,
	})
}

// ---- helpers ----------------------------------------------------------------

func (h *Handler) updateJob(id string, fn func(*models.Job)) {
	h.store.mu.Lock()
	defer h.store.mu.Unlock()
	if j, ok := h.store.jobs[id]; ok {
		fn(j)
	}
}

// newID generates a cryptographically random 16-byte hex string used as a Job
// ID.  Using crypto/rand avoids the need for an external UUID library.
func newID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("newID: crypto/rand failed: %v", err))
	}
	// Format as UUID-like string: 8-4-4-4-12
	h := hex.EncodeToString(b)
	return fmt.Sprintf("%s-%s-%s-%s-%s", h[0:8], h[8:12], h[12:16], h[16:20], h[20:32])
}
