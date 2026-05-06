package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/AniruthKarthik/llm-orchestrator/internal/models"
	"github.com/AniruthKarthik/llm-orchestrator/internal/observability"
	"github.com/AniruthKarthik/llm-orchestrator/internal/orchestrator"
	"github.com/AniruthKarthik/llm-orchestrator/internal/planner"
	"github.com/AniruthKarthik/llm-orchestrator/internal/queue"
	"github.com/AniruthKarthik/llm-orchestrator/internal/store"
	"github.com/AniruthKarthik/llm-orchestrator/internal/utils"
)

type Handler struct {
	store store.Store
	plan  *planner.Planner
	queue queue.Queue
	orch  *orchestrator.Orchestrator
	obs   *observability.Obs
}

// NewHandler creates a new instance of the API handler with its dependencies.
func NewHandler(
	s store.Store,
	p *planner.Planner,
	q queue.Queue,
	orch *orchestrator.Orchestrator,
	obs *observability.Obs,
) *Handler {
	return &Handler{store: s, plan: p, queue: q, orch: orch, obs: obs}
}

type createJobRequest struct {
	Goal string `json:"goal"`
}

type createJobResponse struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	TraceID string `json:"trace_id,omitempty"`
}

// CreateJob handles the POST /job request to start a new high-level goal.
func (h *Handler) CreateJob(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.obs.Tracer.Start(r.Context(), "api.CreateJob")
	defer span.End(ctx)

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
	ctx = observability.WithJobID(ctx, jobID)
	h.obs.Log.Info(ctx, "api: creating job", observability.F("goal", req.Goal))

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
		span.SetError(err)
		utils.WriteError(w, http.StatusInternalServerError, "failed to save job")
		return
	}

	planCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	tasks, err := h.plan.Plan(planCtx, req.Goal)
	if err != nil {
		h.obs.Log.Error(ctx, "api: planning failed", observability.F("err", err.Error()))
		span.SetError(err)
		_ = h.store.SetJobError(jobID, fmt.Sprintf("planning failed: %v", err))
		utils.WriteJSON(w, http.StatusAccepted, createJobResponse{
			ID:      jobID,
			Status:  string(models.JobStatusFailed),
			TraceID: span.TraceID(),
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

	h.orch.TrackJob(jobID)
	h.orch.Publish(orchestrator.Event{
		Type:  orchestrator.EventJobQueued,
		JobID: jobID,
	})

	h.obs.Log.Info(ctx, "api: job queued",
		observability.F("task_count", len(tasks)),
		observability.F("trace_id", span.TraceID()),
	)

	utils.WriteJSON(w, http.StatusAccepted, createJobResponse{
		ID:      jobID,
		Status:  string(models.JobStatusQueued),
		TraceID: span.TraceID(),
	})
}

// GetJob handles the GET /job/{id} request to retrieve the current state of a job.
func (h *Handler) GetJob(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.obs.Tracer.Start(r.Context(), "api.GetJob")
	defer span.End(ctx)

	id := jobIDFromPath(r.URL.Path)
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "missing job id in path")
		return
	}
	ctx = observability.WithJobID(ctx, id)

	job, err := h.store.GetJob(id)
	if err != nil {
		span.SetError(err)
		utils.WriteError(w, http.StatusNotFound, fmt.Sprintf("job %q not found", id))
		return
	}

	h.obs.Log.Info(ctx, "api: get job", observability.F("status", string(job.Status)))
	utils.WriteJSON(w, http.StatusOK, job)
}

// GetJobResult handles the GET /job/{id}/result request to retrieve the final output if available.
func (h *Handler) GetJobResult(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.obs.Tracer.Start(r.Context(), "api.GetJobResult")
	defer span.End(ctx)

	path := strings.TrimSuffix(r.URL.Path, "/result")
	id := jobIDFromPath(path)
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "missing job id in path")
		return
	}
	ctx = observability.WithJobID(ctx, id)

	job, err := h.store.GetJob(id)
	if err != nil {
		span.SetError(err)
		utils.WriteError(w, http.StatusNotFound, fmt.Sprintf("job %q not found", id))
		return
	}

	if job.Status != models.JobStatusCompleted {
		h.obs.Log.Info(ctx, "api: result not yet available",
			observability.F("status", string(job.Status)))
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

// GetMetrics handles the GET /metrics request to retrieve a snapshot of the system telemetry.
func (h *Handler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	snapshot := h.obs.Metrics.Snapshot()
	utils.WriteJSON(w, http.StatusOK, snapshot)
}

// jobIDFromPath extracts the job ID from a URL path.
func jobIDFromPath(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 2 {
		return ""
	}
	return parts[len(parts)-1]
}

// newID generates a random UUID-like string.
func newID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("newID: crypto/rand failed: %v", err))
	}
	h := hex.EncodeToString(b)
	return fmt.Sprintf("%s-%s-%s-%s-%s", h[0:8], h[8:12], h[12:16], h[16:20], h[20:32])
}
