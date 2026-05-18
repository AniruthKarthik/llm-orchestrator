package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"
	"github.com/AniruthKarthik/llm-orchestrator/internal/agents"
	"github.com/AniruthKarthik/llm-orchestrator/internal/core"
	"github.com/AniruthKarthik/llm-orchestrator/internal/events"
	"github.com/AniruthKarthik/llm-orchestrator/internal/executor"
	"github.com/AniruthKarthik/llm-orchestrator/internal/providers"
	"github.com/AniruthKarthik/llm-orchestrator/internal/store"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Server struct {
	executor *executor.Executor
	store    store.Store
	eb       *events.EventBus
}

func NewServer(e *executor.Executor, s store.Store, eb *events.EventBus) *Server {
	return &Server{
		executor: e,
		store:    s,
		eb:       eb,
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	// Health
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
	mux.HandleFunc("PUT /api/v1/workflows/{id}", s.handleUpdateWorkflow)
	mux.HandleFunc("DELETE /api/v1/workflows/{id}", s.handleDeleteWorkflow)
	mux.HandleFunc("POST /api/v1/workflows/{id}/execute", s.handleExecuteWorkflow)
	mux.HandleFunc("POST /api/v1/workflows/{id}/tasks/{taskID}/approve", s.handleApproveTask)

	// Resources
	mux.HandleFunc("GET /api/v1/agents", s.handleListAgents)
	mux.HandleFunc("GET /api/v1/artifacts", s.handleListArtifacts)
	mux.HandleFunc("GET /api/v1/queues", s.handleListQueues)
	mux.HandleFunc("GET /api/v1/metrics", s.handleGetMetrics)

	// WebSocket
	mux.HandleFunc("GET /api/v1/ws", s.handleWebSocket)

	return s.loggingMiddleware(s.enableCORS(mux))
}

// --- Middleware ---

func (s *Server) loggingMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		h.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func (s *Server) enableCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		h.ServeHTTP(w, r)
	})
}

// --- Helpers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("writeJSON encode error: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// --- Handlers ---

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
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
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer c.Close()

	ch := make(chan events.Event, 256)

	eventTypes := []events.EventType{
		events.TaskStarted,
		events.TaskCompleted,
		events.TaskFailed,
		events.WorkflowStarted,
		events.WorkflowCompleted,
		events.WorkflowFailed,
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
				log.Printf("WebSocket write error: %v", err)
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
		log.Printf("listWorkflows error: %v", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if records == nil {
		records = []store.WorkflowRecord{}
	}
	writeJSON(w, http.StatusOK, records)
}

type TaskRequest struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	Input        map[string]any `json:"input"`
	Dependencies []string       `json:"dependencies"`
	Provider     string         `json:"provider"`
	Model        string         `json:"model"`
	SystemPrompt string         `json:"systemPrompt"`
	RequiresApproval bool       `json:"requiresApproval"`
	MaxRetries   int            `json:"maxRetries"`
	TimeoutMs    int            `json:"timeoutMs"`
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
		log.Printf("createWorkflow save error: %v", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	for _, tr := range req.Tasks {
		if tr.ID == "" {
			tr.ID = fmt.Sprintf("task-%d", time.Now().UnixNano())
		}
		input := tr.Input
		if input == nil {
			input = map[string]any{}
		}
		if tr.SystemPrompt != "" {
			input["system_prompt"] = tr.SystemPrompt
		}
		task := core.NewTask(tr.ID, workflow.ID, tr.Name, tr.Description, input, tr.Dependencies)
		task.Provider = tr.Provider
		task.Model = tr.Model
		if tr.TimeoutMs > 0 {
			task.Timeout = time.Duration(tr.TimeoutMs) * time.Millisecond
		}

		if ag := s.buildAgentForTask(req.ID, tr); ag != nil {
			task.AgentID = ag.ID
			s.executor.GetAgentRegistry().Register(ag)
		}

		_ = workflow.AddTask(task)
		if err := s.store.SaveTask(store.TaskToRecord(workflow.ID, task)); err != nil {
			log.Printf("createWorkflow saveTask error: %v", err)
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	writeJSON(w, http.StatusCreated, workflow)
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
		log.Printf("getWorkflow tasks error: %v", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	workflow := store.RecordToWorkflow(record, tasks)
	writeJSON(w, http.StatusOK, workflow)
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
	if err := s.store.UpdateWorkflow(record); err != nil {
		log.Printf("updateWorkflow error: %v", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	for _, tr := range req.Tasks {
		if tr.ID == "" {
			tr.ID = fmt.Sprintf("task-%d", time.Now().UnixNano())
		}
		input := tr.Input
		if input == nil {
			input = map[string]any{}
		}
		if tr.SystemPrompt != "" {
			input["system_prompt"] = tr.SystemPrompt
		}
		task := core.NewTask(tr.ID, id, tr.Name, tr.Description, input, tr.Dependencies)
		task.Provider = tr.Provider
		task.Model = tr.Model
		if tr.TimeoutMs > 0 {
			task.Timeout = time.Duration(tr.TimeoutMs) * time.Millisecond
		}

		if ag := s.buildAgentForTask(id, tr); ag != nil {
			task.AgentID = ag.ID
			s.executor.GetAgentRegistry().Register(ag)
		}

		_, lookupErr := s.store.GetTask(id, tr.ID)
		if lookupErr != nil {
			if err := s.store.SaveTask(store.TaskToRecord(id, task)); err != nil {
				log.Printf("updateWorkflow saveTask error: %v", err)
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
		} else {
			if err := s.store.UpdateTask(store.TaskToRecord(id, task)); err != nil {
				log.Printf("updateWorkflow updateTask error: %v", err)
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) handleDeleteWorkflow(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.store.DeleteWorkflow(id); err != nil {
		log.Printf("deleteWorkflow error: %v", err)
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
		log.Printf("executeWorkflow tasks error: %v", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	workflow := store.RecordToWorkflow(record, tasks)

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
			log.Printf("workflow %s execution error: %v", id, err)
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
		log.Printf("approveTask update error: %v", err)
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
		log.Printf("listAgents store error: %v", err)
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
		log.Printf("listArtifacts store error: %v", err)
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
		log.Printf("listQueues pending error: %v", err)
	}
	running, err := s.store.ListTasksByStatus(string(core.TaskRunning))
	if err != nil {
		log.Printf("listQueues running error: %v", err)
	}
	waiting, err := s.store.ListTasksByStatus(string(core.TaskWaitingForApproval))
	if err != nil {
		log.Printf("listQueues waiting error: %v", err)
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

	writeJSON(w, http.StatusOK, map[string]any{
		"activeWorkflows":    active,
		"completedWorkflows": completed,
		"failedWorkflows":    failed,
		"pendingWorkflows":   pending,
		"totalWorkflows":     len(workflows),
		"tasksInQueue":       queueDepth,
		"providersOnline":    len(providerList),
		"providers":          providerList,
	})
}
