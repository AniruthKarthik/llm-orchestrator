package api

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/AniruthKarthik/llm-orchestrator/internal/agents"
	"github.com/AniruthKarthik/llm-orchestrator/internal/core"
	"github.com/AniruthKarthik/llm-orchestrator/internal/dag"
	"github.com/AniruthKarthik/llm-orchestrator/internal/events"
	"github.com/AniruthKarthik/llm-orchestrator/internal/executor"
	"github.com/AniruthKarthik/llm-orchestrator/internal/providers"
	"github.com/AniruthKarthik/llm-orchestrator/internal/store"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

const maxBodyBytes = 1 << 20 // 1 MiB — reject larger request bodies

// tokenModelStats tracks cumulative token usage per model for this server session.
type tokenModelStats struct {
	Model            string `json:"model"`
	PromptTokens     int    `json:"promptTokens"`
	CompletionTokens int    `json:"completionTokens"`
	TotalTokens      int    `json:"totalTokens"`
}

type Server struct {
	executor      *executor.Executor
	store         store.Store
	eb            *events.EventBus
	allowedOrigin string // CORS origin, "*" by default
	apiKey        string // optional — empty means no auth
	storageMode   string // "memory" or "postgres"

	tokenMu    sync.RWMutex
	tokenStats map[string]*tokenModelStats // keyed by model name
}

func NewServer(e *executor.Executor, s store.Store, eb *events.EventBus, storageMode string) *Server {
	origin := os.Getenv("ALLOWED_ORIGIN")
	if origin == "" {
		origin = "*"
	}
	srv := &Server{
		executor:      e,
		store:         s,
		eb:            eb,
		allowedOrigin: origin,
		apiKey:        os.Getenv("API_KEY"),
		storageMode:   storageMode,
		tokenStats:    make(map[string]*tokenModelStats),
	}

	// Subscribe to token usage events to accumulate running totals.
	srv.eb.Subscribe(events.TaskTokenUsage, func(ev events.Event) {
		model, _ := ev.Payload["model"].(string)
		if model == "" {
			model = "unknown"
		}
		toInt := func(v any) int {
			switch n := v.(type) {
			case int:
				return n
			case float64:
				return int(n)
			}
			return 0
		}
		prompt := toInt(ev.Payload["prompt_tokens"])
		completion := toInt(ev.Payload["completion_tokens"])
		total := toInt(ev.Payload["total_tokens"])

		srv.tokenMu.Lock()
		defer srv.tokenMu.Unlock()
		if _, ok := srv.tokenStats[model]; !ok {
			srv.tokenStats[model] = &tokenModelStats{Model: model}
		}
		srv.tokenStats[model].PromptTokens += prompt
		srv.tokenStats[model].CompletionTokens += completion
		srv.tokenStats[model].TotalTokens += total
	})

	return srv
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	// Health (unauthenticated — used by load-balancer probes)
	mux.HandleFunc("GET /api/v1/health", s.handleHealth)

	// Meta
	mux.HandleFunc("GET /api/v1/meta/providers", s.handleListProviders)

	// Config
	mux.HandleFunc("GET /api/v1/config/compose", s.handleGetCompose)
	mux.HandleFunc("PUT /api/v1/config/compose", s.handleUpdateCompose)

	// Workflows
	mux.HandleFunc("GET /api/v1/workflows", s.handleListWorkflows)
	mux.HandleFunc("POST /api/v1/workflows", s.handleCreateWorkflow)
	mux.HandleFunc("GET /api/v1/workflows/{id}", s.handleGetWorkflow)
	mux.HandleFunc("GET /api/v1/workflows/{id}/plan", s.handleGetWorkflowPlan)
	mux.HandleFunc("GET /api/v1/workflows/{id}/events", s.handleListWorkflowEvents)
	mux.HandleFunc("PUT /api/v1/workflows/{id}", s.handleUpdateWorkflow)
	mux.HandleFunc("DELETE /api/v1/workflows/{id}", s.handleDeleteWorkflow)
	mux.HandleFunc("POST /api/v1/workflows/{id}/execute", s.handleExecuteWorkflow)
	mux.HandleFunc("POST /api/v1/workflows/{id}/tasks/{taskID}/approve", s.handleApproveTask)

	// Resources
	mux.HandleFunc("GET /api/v1/agents", s.handleListAgents)
	mux.HandleFunc("GET /api/v1/artifacts", s.handleListArtifacts)
	mux.HandleFunc("GET /api/v1/queues", s.handleListQueues)
	mux.HandleFunc("GET /api/v1/metrics", s.handleGetMetrics)
	mux.HandleFunc("GET /api/v1/events", s.handleListEvents)

	// WebSocket
	mux.HandleFunc("GET /api/v1/ws", s.handleWebSocket)

	// Static UI (production mode — ui/dist must be present)
	// In dev, the Vite dev server handles the UI; this is skipped when the dir is absent.
	uiDir := "ui/dist"
	if _, err := os.Stat(uiDir); err == nil {
		fs := http.FileServer(http.Dir(uiDir))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// Serve the SPA's index.html for all non-file routes (client-side routing)
			if _, err := os.Stat(uiDir + r.URL.Path); os.IsNotExist(err) {
				http.ServeFile(w, r, uiDir+"/index.html")
				return
			}
			fs.ServeHTTP(w, r)
		})
	}

	return s.loggingMiddleware(s.enableCORS(s.authMiddleware(s.bodyLimitMiddleware(mux))))
}

// --- Middleware ---

// responseWriter wraps http.ResponseWriter to capture the status code for logging.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hj, ok := rw.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("webserver doesn't support hijacking")
	}
	return hj.Hijack()
}

func (s *Server) loggingMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := newResponseWriter(w)
		h.ServeHTTP(rw, r)
		slog.Info("http",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.statusCode,
			"latency_ms", time.Since(start).Milliseconds(),
			"remote", r.RemoteAddr,
		)
	})
}

func (s *Server) enableCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", s.allowedOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		h.ServeHTTP(w, r)
	})
}

func (s *Server) authMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Auth is opt-in: skip if API_KEY is not configured
		if s.apiKey == "" {
			h.ServeHTTP(w, r)
			return
		}
		// Health check is always public
		if r.URL.Path == "/api/v1/health" {
			h.ServeHTTP(w, r)
			return
		}
		// Accept X-API-Key header or Authorization: Bearer <key>
		key := r.Header.Get("X-API-Key")
		if key == "" {
			bearer := r.Header.Get("Authorization")
			if len(bearer) > 7 && bearer[:7] == "Bearer " {
				key = bearer[7:]
			}
		}
		if key != s.apiKey {
			writeError(w, http.StatusUnauthorized, "invalid or missing API key")
			return
		}
		h.ServeHTTP(w, r)
	})
}

func (s *Server) bodyLimitMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
			r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		}
		h.ServeHTTP(w, r)
	})
}

// --- Helpers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("writeJSON encode", "error", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// --- Handlers ---

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":      "ok",
		"storageMode": s.storageMode,
	})
}

func (s *Server) handleGetCompose(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile("docker-compose.yaml")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			data, err = os.ReadFile("docker-compose.yaml.example")
			if err != nil {
				writeError(w, http.StatusNotFound, "docker-compose.yaml not found")
				return
			}
		} else {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write(data)
}

func (s *Server) handleUpdateCompose(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if req.Content == "" {
		writeError(w, http.StatusBadRequest, "content must not be empty")
		return
	}

	if err := os.WriteFile("docker-compose.yaml", []byte(req.Content), 0644); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "saved"})
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("WebSocket upgrade", "error", err)
		return
	}
	defer c.Close()

	ch := make(chan events.Event, 256)

	eventTypes := []events.EventType{
		events.TaskStarted,
		events.TaskWaitingForApproval,
		events.TaskRetried,
		events.TaskCompleted,
		events.TaskFailed,
		events.TaskTokenUsage,
		events.WorkflowStarted,
		events.WorkflowCompleted,
		events.WorkflowFailed,
		events.StageStarted,
		events.StageCompleted,
		events.StageFailed,
	}

	// Register subscriptions and track IDs for clean removal
	type subEntry struct {
		etype events.EventType
		id    events.SubscriptionID
	}
	subs := make([]subEntry, 0, len(eventTypes))
	for _, et := range eventTypes {
		id := s.eb.Subscribe(et, func(e events.Event) {
			select {
			case ch <- e:
			default:
				// Client is slow; drop this event rather than block
			}
		})
		subs = append(subs, subEntry{etype: et, id: id})
	}

	// Ensure subscriptions are cleaned up when the connection drops
	defer func() {
		for _, sub := range subs {
			s.eb.Unsubscribe(sub.etype, sub.id)
		}
	}()

	// Use a done channel to stop reading from ch when WS closes
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, _, err := c.ReadMessage()
			if err != nil {
				return
			}
		}
	}()

	for {
		select {
		case <-done:
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			if err := c.WriteJSON(event); err != nil {
				slog.Error("WebSocket write", "error", err)
				return
			}
		}
	}
}

func (s *Server) handleListProviders(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, providers.List(r.Context()))
}

func (s *Server) handleListWorkflows(w http.ResponseWriter, r *http.Request) {
	records, err := s.store.ListWorkflows()
	if err != nil {
		slog.Error("listWorkflows", "error", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if records == nil {
		records = []store.WorkflowRecord{}
	}
	// Enrich each record with task count
	for i, rec := range records {
		if tasks, err := s.store.GetWorkflowTasks(rec.ID); err == nil {
			records[i].TaskCount = len(tasks)
		}
	}
	writeJSON(w, http.StatusOK, records)
}

type TaskRequest struct {
	ID               string         `json:"id"`
	Name             string         `json:"name"`
	Description      string         `json:"description"`
	Input            map[string]any `json:"input"`
	Dependencies     []string       `json:"dependencies"`
	Provider         string         `json:"provider"`
	Model            string         `json:"model"`
	SystemPrompt     string         `json:"systemPrompt"`
	RequiresApproval bool           `json:"requiresApproval"`
	MaxRetries       int            `json:"maxRetries"`
	TimeoutMs        int            `json:"timeoutMs"`
}

type CreateWorkflowRequest struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Tasks       []TaskRequest `json:"tasks"`
}

func (req *CreateWorkflowRequest) validate() error {
	if req.Name == "" {
		return errors.New("workflow name is required")
	}
	return nil
}

func (s *Server) buildAgentForTask(wfID string, tr TaskRequest) *agents.Agent {
	if tr.Provider == "" || tr.Model == "" {
		return nil
	}
	agentID := fmt.Sprintf("agent-%s-%s-%s", wfID, tr.ID, tr.Provider)
	return &agents.Agent{
		ID:           agentID,
		Name:         fmt.Sprintf("Agent for %s", tr.Name),
		Provider:     tr.Provider,
		Model:        tr.Model,
		Role:         agents.RoleExecutor,
		SystemPrompt: tr.SystemPrompt,
	}
}

func taskFromRequest(workflowID string, tr TaskRequest) *core.Task {
	input := tr.Input
	if input == nil {
		input = map[string]any{}
	}
	if tr.SystemPrompt != "" {
		input["system_prompt"] = tr.SystemPrompt
	}

	task := core.NewTask(tr.ID, workflowID, tr.Name, tr.Description, input, tr.Dependencies)
	task.Provider = tr.Provider
	task.Model = tr.Model
	task.RequiresApproval = tr.RequiresApproval
	if tr.MaxRetries > 0 {
		task.RetryPolicy = &core.RetryPolicy{
			MaxRetries: tr.MaxRetries,
			Strategy:   core.RetryStrategyImmediate,
		}
	}
	if tr.TimeoutMs > 0 {
		task.Timeout = time.Duration(tr.TimeoutMs) * time.Millisecond
	}
	return task
}

func validateExecutableWorkflow(workflow *core.Workflow, tasks []store.TaskRecord) error {
	if len(tasks) == 0 {
		return errors.New("workflow has no tasks to execute")
	}
	if _, err := dag.NewTopologicalPlanner().BuildExecutionPlan(workflow); err != nil {
		return fmt.Errorf("invalid workflow DAG: %w", err)
	}
	for _, task := range tasks {
		if task.Provider == "" {
			continue
		}
		if task.Model == "" {
			return fmt.Errorf("task %s has provider %q but no model", task.ID, task.Provider)
		}
		if _, err := providers.Get(task.Provider); err != nil {
			return fmt.Errorf("task %s uses unavailable provider %q: %w", task.ID, task.Provider, err)
		}
	}
	return nil
}

func (s *Server) handleCreateWorkflow(w http.ResponseWriter, r *http.Request) {
	var req CreateWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if err := req.validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.ID == "" {
		req.ID = fmt.Sprintf("wf-%d", time.Now().UnixMilli())
	}

	workflow := core.NewWorkflow(req.ID, req.Name, req.Description)
	if err := s.store.SaveWorkflow(store.WorkflowToRecord(workflow)); err != nil {
		slog.Error("createWorkflow save", "error", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	for _, tr := range req.Tasks {
		if tr.ID == "" {
			tr.ID = fmt.Sprintf("task-%d", time.Now().UnixNano())
		}
		task := taskFromRequest(workflow.ID, tr)

		if ag := s.buildAgentForTask(req.ID, tr); ag != nil {
			task.AgentID = ag.ID
			s.executor.GetAgentRegistry().Register(ag)
			_ = s.store.SaveAgent(store.AgentToRecord(ag))
		}

		_ = workflow.AddTask(task)
		if err := s.store.SaveTask(store.TaskToRecord(workflow.ID, task)); err != nil {
			slog.Error("createWorkflow saveTask", "error", err)
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	// Return the persisted WorkflowRecord (properly camelCase JSON)
	saved, err := s.store.GetWorkflow(req.ID)
	if err != nil {
		// Fallback: return what we have
		saved = store.WorkflowToRecord(workflow)
	}
	saved.TaskCount = len(req.Tasks)
	writeJSON(w, http.StatusCreated, saved)
}

func (s *Server) handleGetWorkflow(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	record, err := s.store.GetWorkflow(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "workflow not found")
		return
	}

	tasks, err := s.store.GetWorkflowTasks(id)
	if err != nil {
		slog.Error("getWorkflow tasks", "error", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Build a response map with camelCase fields and tasks map keyed by ID
	tasksMap := make(map[string]store.TaskRecord, len(tasks))
	for _, t := range tasks {
		tasksMap[t.ID] = t
	}
	record.TaskCount = len(tasks)

	type WorkflowResponse struct {
		store.WorkflowRecord
		Tasks map[string]store.TaskRecord `json:"tasks"`
	}
	writeJSON(w, http.StatusOK, WorkflowResponse{
		WorkflowRecord: record,
		Tasks:          tasksMap,
	})
}

type executionPlanNode struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Status           string `json:"status"`
	Attempt          int    `json:"attempt"`
	MaxRetries       int    `json:"maxRetries"`
	Provider         string `json:"provider,omitempty"`
	Model            string `json:"model,omitempty"`
	RequiresApproval bool   `json:"requiresApproval,omitempty"`
}

type executionPlanEdge struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
}

type executionPlanResponse struct {
	WorkflowID string               `json:"workflowId"`
	Stages     []dag.ExecutionStage `json:"stages"`
	Nodes      []executionPlanNode  `json:"nodes"`
	Edges      []executionPlanEdge  `json:"edges"`
}

func (s *Server) handleGetWorkflowPlan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	record, err := s.store.GetWorkflow(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "workflow not found")
		return
	}

	tasks, err := s.store.GetWorkflowTasks(id)
	if err != nil {
		slog.Error("getWorkflowPlan tasks", "error", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	workflow := store.RecordToWorkflow(record, tasks)
	plan, err := dag.NewTopologicalPlanner().BuildExecutionPlan(workflow)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid workflow DAG: "+err.Error())
		return
	}

	nodes := make([]executionPlanNode, 0, len(tasks))
	edges := make([]executionPlanEdge, 0)
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].ID < tasks[j].ID
	})
	for _, task := range tasks {
		maxRetries := 0
		if task.RetryPolicy != nil {
			maxRetries = task.RetryPolicy.MaxRetries
		}
		nodes = append(nodes, executionPlanNode{
			ID:               task.ID,
			Name:             task.Name,
			Status:           task.Status,
			Attempt:          task.Attempt,
			MaxRetries:       maxRetries,
			Provider:         task.Provider,
			Model:            task.Model,
			RequiresApproval: task.RequiresApproval,
		})
		for _, depID := range task.Dependencies {
			edges = append(edges, executionPlanEdge{
				ID:     fmt.Sprintf("e-%s-%s", depID, task.ID),
				Source: depID,
				Target: task.ID,
			})
		}
	}
	sort.Slice(edges, func(i, j int) bool {
		return edges[i].ID < edges[j].ID
	})

	writeJSON(w, http.StatusOK, executionPlanResponse{
		WorkflowID: id,
		Stages:     plan.Stages,
		Nodes:      nodes,
		Edges:      edges,
	})
}

func (s *Server) handleListWorkflowEvents(w http.ResponseWriter, r *http.Request) {
	workflowID := r.PathValue("id")
	events, err := s.store.ListEvents(workflowID, 500)
	if err != nil {
		slog.Error("listWorkflowEvents", "workflow_id", workflowID, "error", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if events == nil {
		events = []store.EventRecord{}
	}
	writeJSON(w, http.StatusOK, events)
}

func (s *Server) handleListEvents(w http.ResponseWriter, r *http.Request) {
	workflowID := r.URL.Query().Get("workflowId")
	events, err := s.store.ListEvents(workflowID, 500)
	if err != nil {
		slog.Error("listEvents", "workflow_id", workflowID, "error", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if events == nil {
		events = []store.EventRecord{}
	}
	writeJSON(w, http.StatusOK, events)
}

func (s *Server) handleUpdateWorkflow(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req CreateWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	record, err := s.store.GetWorkflow(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "workflow not found")
		return
	}

	record.Name = req.Name
	record.Description = req.Description
	record.Status = string(core.WorkflowPending)
	record.StartedAt = nil
	record.FinishedAt = nil
	if err := s.store.UpdateWorkflow(record); err != nil {
		slog.Error("updateWorkflow", "error", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	seen := make(map[string]bool, len(req.Tasks))
	for _, tr := range req.Tasks {
		if tr.ID == "" {
			tr.ID = fmt.Sprintf("task-%d", time.Now().UnixNano())
		}
		seen[tr.ID] = true
		task := taskFromRequest(id, tr)

		if ag := s.buildAgentForTask(id, tr); ag != nil {
			task.AgentID = ag.ID
			s.executor.GetAgentRegistry().Register(ag)
			_ = s.store.SaveAgent(store.AgentToRecord(ag))
		}

		_, lookupErr := s.store.GetTask(id, tr.ID)
		if lookupErr != nil {
			if err := s.store.SaveTask(store.TaskToRecord(id, task)); err != nil {
				slog.Error("updateWorkflow saveTask", "error", err)
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
		} else {
			if err := s.store.UpdateTask(store.TaskToRecord(id, task)); err != nil {
				slog.Error("updateWorkflow updateTask", "error", err)
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
		}
	}

	existingTasks, err := s.store.GetWorkflowTasks(id)
	if err != nil {
		slog.Error("updateWorkflow existing tasks", "error", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	for _, existing := range existingTasks {
		if !seen[existing.ID] {
			if err := s.store.DeleteTask(id, existing.ID); err != nil {
				slog.Error("updateWorkflow deleteTask", "task_id", existing.ID, "error", err)
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
		}
	}

	// Return updated record
	updated, err := s.store.GetWorkflow(id)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]string{"status": "updated", "id": id})
		return
	}
	if tasks, err := s.store.GetWorkflowTasks(id); err == nil {
		updated.TaskCount = len(tasks)
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) handleDeleteWorkflow(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.store.DeleteWorkflow(id); err != nil {
		slog.Error("deleteWorkflow", "error", err)
		writeError(w, http.StatusNotFound, "workflow not found or could not be deleted")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleExecuteWorkflow(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	record, err := s.store.GetWorkflow(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "workflow not found")
		return
	}

	tasks, err := s.store.GetWorkflowTasks(id)
	if err != nil {
		slog.Error("executeWorkflow tasks", "error", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	workflow := store.RecordToWorkflow(record, tasks)
	
	// Reset workflow state for re-execution
	workflow.Status = core.WorkflowPending
	workflow.StartedAt = nil
	workflow.FinishedAt = nil
	for _, task := range workflow.Tasks {
		task.Status = core.TaskPending
		task.StartedAt = nil
		task.FinishedAt = nil
		task.Output = nil
		task.Error = ""
	}

	if err := validateExecutableWorkflow(workflow, tasks); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Re-register agents from task metadata so they survive a server restart
	for _, task := range workflow.Tasks {
		if task.AgentID != "" && task.Provider != "" && task.Model != "" {
			if _, found := s.executor.GetAgentRegistry().Get(task.AgentID); !found {
				s.executor.GetAgentRegistry().Register(&agents.Agent{
					ID:       task.AgentID,
					Name:     fmt.Sprintf("Agent for %s", task.Name),
					Provider: task.Provider,
					Model:    task.Model,
					Role:     agents.RoleExecutor,
				})
			}
		}
	}

	go func() {
		if err := s.executor.Execute(workflow); err != nil {
			slog.Error("workflow execution error", "workflow_id", id, "error", err)
			if rec, getErr := s.store.GetWorkflow(id); getErr == nil && rec.Status != string(core.WorkflowFailed) {
				rec.Status = string(core.WorkflowFailed)
				now := time.Now()
				rec.FinishedAt = &now
				_ = s.store.UpdateWorkflow(rec)
				s.eb.Publish(events.Event{
					Type:       events.WorkflowFailed,
					WorkflowID: id,
					Timestamp:  now,
					Payload: map[string]any{
						"error": err.Error(),
					},
				})
			}
		}
	}()

	writeJSON(w, http.StatusAccepted, map[string]string{"status": "executing", "workflowId": id})
}

func (s *Server) handleApproveTask(w http.ResponseWriter, r *http.Request) {
	wfID := r.PathValue("id")
	taskID := r.PathValue("taskID")

	rec, err := s.store.GetTask(wfID, taskID)
	if err != nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}

	if rec.Status != string(core.TaskWaitingForApproval) {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("task is not waiting for approval (current status: %s)", rec.Status))
		return
	}

	rec.Status = string(core.TaskRunning)
	if err := s.store.UpdateTask(rec); err != nil {
		slog.Error("approveTask update", "error", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "approved"})
}

func (s *Server) handleListAgents(w http.ResponseWriter, r *http.Request) {
	// Use the persistent store — not the in-memory agent registry — so agents
	// survive a server restart.
	records, err := s.store.ListAgents()
	if err != nil {
		slog.Error("listAgents store", "error", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Merge with in-memory registry (dynamic agents registered at runtime but
	// not yet persisted). The in-memory registry is the authoritative source
	// for agents that were just registered in this session.
	inMem := s.executor.GetAgentRegistry().List()
	storeIDs := make(map[string]bool, len(records))
	for _, r := range records {
		storeIDs[r.ID] = true
	}
	for _, a := range inMem {
		if !storeIDs[a.ID] {
			records = append(records, store.AgentRecord{
				ID:       a.ID,
				Name:     a.Name,
				Role:     string(a.Role),
				Model:    a.Model,
				Provider: a.Provider,
				Tools:    a.Tools,
			})
		}
	}

	if records == nil {
		records = []store.AgentRecord{}
	}
	writeJSON(w, http.StatusOK, records)
}

func (s *Server) handleListArtifacts(w http.ResponseWriter, r *http.Request) {
	// Use the persistent store — not the in-memory artifact registry — so
	// artifacts survive a server restart.
	records, err := s.store.ListAllArtifacts()
	if err != nil {
		slog.Error("listArtifacts store", "error", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Also merge with in-memory registry for artifacts generated in this session
	inMem := s.executor.GetArtifactRegistry().ListAll()
	storeIDs := make(map[string]bool, len(records))
	for _, r := range records {
		storeIDs[r.ID] = true
	}
	for _, a := range inMem {
		if !storeIDs[a.ID] {
			records = append(records, store.ArtifactRecord{
				ID:         a.ID,
				WorkflowID: a.WorkflowID,
				TaskID:     a.TaskID,
				Name:       a.Name,
				Type:       string(a.Type),
				Data:       a.Data,
				Metadata:   a.Metadata,
				CreatedAt:  a.CreatedAt,
			})
		}
	}

	if records == nil {
		records = []store.ArtifactRecord{}
	}
	writeJSON(w, http.StatusOK, records)
}

func (s *Server) handleListQueues(w http.ResponseWriter, r *http.Request) {
	pending, err := s.store.ListTasksByStatus(string(core.TaskPending))
	if err != nil {
		slog.Error("listQueues pending", "error", err)
	}
	running, err := s.store.ListTasksByStatus(string(core.TaskRunning))
	if err != nil {
		slog.Error("listQueues running", "error", err)
	}
	waiting, err := s.store.ListTasksByStatus(string(core.TaskWaitingForApproval))
	if err != nil {
		slog.Error("listQueues waiting", "error", err)
	}

	all := append(pending, running...)
	all = append(all, waiting...)
	if all == nil {
		all = []store.TaskRecord{}
	}
	writeJSON(w, http.StatusOK, all)
}

func (s *Server) handleGetMetrics(w http.ResponseWriter, r *http.Request) {
	workflows, _ := s.store.ListWorkflows()

	var active, completed, failed, pending int
	for _, wf := range workflows {
		switch wf.Status {
		case string(core.WorkflowRunning):
			active++
		case string(core.WorkflowCompleted):
			completed++
		case string(core.WorkflowFailed):
			failed++
		case string(core.WorkflowPending):
			pending++
		}
	}

	queueDepth := 0
	if q, err := s.store.ListTasksByStatus(string(core.TaskPending)); err == nil {
		queueDepth += len(q)
	}
	if q, err := s.store.ListTasksByStatus(string(core.TaskRunning)); err == nil {
		queueDepth += len(q)
	}

	providerList := providers.List(r.Context())

	// Collect per-model token stats
	s.tokenMu.RLock()
	tokenList := make([]*tokenModelStats, 0, len(s.tokenStats))
	var totalPrompt, totalCompletion, totalTokens int
	for _, v := range s.tokenStats {
		tokenList = append(tokenList, v)
		totalPrompt += v.PromptTokens
		totalCompletion += v.CompletionTokens
		totalTokens += v.TotalTokens
	}
	s.tokenMu.RUnlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"activeWorkflows":    active,
		"completedWorkflows": completed,
		"failedWorkflows":    failed,
		"pendingWorkflows":   pending,
		"totalWorkflows":     len(workflows),
		"tasksInQueue":       queueDepth,
		"providersOnline":    len(providerList),
		"providers":          providerList,
		"tokenUsage": map[string]any{
			"totalPromptTokens":     totalPrompt,
			"totalCompletionTokens": totalCompletion,
			"totalTokens":           totalTokens,
			"byModel":               tokenList,
		},
	})
}
