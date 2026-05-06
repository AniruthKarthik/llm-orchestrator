package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/AniruthKarthik/llm-orchestrator/internal/models"
	"github.com/AniruthKarthik/llm-orchestrator/internal/planner"
	"github.com/AniruthKarthik/llm-orchestrator/internal/queue"
	"github.com/AniruthKarthik/llm-orchestrator/internal/store"
	"github.com/AniruthKarthik/llm-orchestrator/internal/utils"
)

type Handler struct {
	store   store.Store
	queue   queue.Queue
	planner *planner.Planner
}

func NewHandler(s store.Store, q queue.Queue, p *planner.Planner) *Handler {
	return &Handler{
		store:   s,
		queue:   q,
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
	if err := h.store.SaveJob(job); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, fmt.Sprintf("failed to save job: %v", err))
		return
	}

	log.Printf("[handler] created job %s for goal: %q", jobID, req.Goal)

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	tasks, err := h.planner.Plan(ctx, req.Goal)
	if err != nil {
		log.Printf("[handler] planning failed for job %s: %v", jobID, err)
		if err := h.store.SetJobError(jobID, fmt.Sprintf("planning failed: %v", err)); err != nil {
			log.Printf("[handler] failed to update job status: %v", err)
		}
		utils.WriteJSON(w, http.StatusAccepted, createJobResponse{ID: jobID, Status: string(models.JobStatusFailed)})
		return
	}

	// Set JobID for each task and update job in store
	for i := range tasks {
		tasks[i].JobID = jobID
	}

	if err := h.store.UpdateJobStatus(jobID, models.JobStatusQueued); err != nil {
		log.Printf("[handler] failed to update job status: %v", err)
	}

	// Update tasks in job
	if err := h.store.UpdateJobTasks(jobID, tasks); err != nil {
		log.Printf("[handler] failed to update job tasks: %v", err)
	}

	// Enqueue tasks
	for _, task := range tasks {
		if err := h.queue.Enqueue(task); err != nil {
			log.Printf("[handler] failed to enqueue task %s: %v", task.ID, err)
		}
	}

	utils.WriteJSON(w, http.StatusAccepted, createJobResponse{ID: jobID, Status: string(models.JobStatusQueued)})
}

func (h *Handler) GetJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "missing job id in path")
		return
	}

	job, err := h.store.GetJob(id)
	if err != nil {
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

	job, err := h.store.GetJob(id)
	if err != nil {
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
