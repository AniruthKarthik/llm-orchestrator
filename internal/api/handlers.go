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
	planner *planner.Planner
	queue   queue.Queue
}

func NewHandler(s store.Store, p *planner.Planner, q queue.Queue) *Handler {
	return &Handler{store: s, planner: p, queue: q}
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
		utils.WriteError(w, http.StatusInternalServerError, "failed to save job")
		return
	}

	log.Printf("[handler] created job %s for goal: %q", jobID, req.Goal)

	planCtx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	tasks, err := h.planner.Plan(planCtx, req.Goal)
	if err != nil {
		log.Printf("[handler] planning failed for job %s: %v", jobID, err)
		_ = h.store.SetJobError(jobID, fmt.Sprintf("planning failed: %v", err))
		utils.WriteJSON(w, http.StatusAccepted, createJobResponse{
			ID:     jobID,
			Status: string(models.JobStatusFailed),
		})
		return
	}

	for i := range tasks {
		tasks[i].JobID = jobID
	}

	job.Tasks = tasks
	job.Status = models.JobStatusQueued
	job.UpdatedAt = time.Now().UTC()
	if err := h.store.SaveJob(job); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "failed to persist tasks")
		return
	}

	for _, t := range tasks {
		if err := h.queue.Enqueue(t); err != nil {
			log.Printf("[handler] enqueue failed for task %s: %v", t.ID, err)
		}
	}

	log.Printf("[handler] job %s queued with %d tasks", jobID, len(tasks))
	utils.WriteJSON(w, http.StatusAccepted, createJobResponse{
		ID:     jobID,
		Status: string(models.JobStatusQueued),
	})
}

func (h *Handler) GetJob(w http.ResponseWriter, r *http.Request) {
	id := jobIDFromPath(r.URL.Path)
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
	path := strings.TrimSuffix(r.URL.Path, "/result")
	id := jobIDFromPath(path)
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

func jobIDFromPath(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 2 {
		return ""
	}
	return parts[len(parts)-1]
}

func newID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("newID: crypto/rand failed: %v", err))
	}
	h := hex.EncodeToString(b)
	return fmt.Sprintf("%s-%s-%s-%s-%s", h[0:8], h[8:12], h[12:16], h[16:20], h[20:32])
}
