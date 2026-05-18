package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

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
	mux.HandleFunc("GET /api/v1/meta/providers", s.handleListProviders)
	mux.HandleFunc("GET /api/v1/config/compose", s.handleGetCompose)
	mux.HandleFunc("PUT /api/v1/config/compose", s.handleUpdateCompose)
	mux.HandleFunc("GET /api/v1/workflows", s.handleListWorkflows)
	mux.HandleFunc("POST /api/v1/workflows", s.handleCreateWorkflow)
	mux.HandleFunc("GET /api/v1/workflows/{id}", s.handleGetWorkflow)
	mux.HandleFunc("PUT /api/v1/workflows/{id}", s.handleUpdateWorkflow)
	mux.HandleFunc("POST /api/v1/workflows/{id}/execute", s.handleExecuteWorkflow)
	mux.HandleFunc("POST /api/v1/workflows/{id}/tasks/{taskID}/approve", s.handleApproveTask)
	mux.HandleFunc("GET /api/v1/ws", s.handleWebSocket)

	return s.enableCORS(mux)
}

func (s *Server) handleGetCompose(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile("docker-compose.yaml")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			data, err = os.ReadFile("docker-compose.yaml.example")
			if err != nil {
				http.Error(w, "docker-compose.yaml.example not found", http.StatusNotFound)
				return
			}
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/x-yaml")
	w.Write(data)
}

func (s *Server) handleUpdateCompose(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := os.WriteFile("docker-compose.yaml", []byte(req.Content), 0644); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()

	ch := make(chan events.Event, 100)
	// Subscribe to all events for this UI connection
	// In production, might want a specific subscriber ID
	s.eb.Subscribe(events.TaskStarted, func(e events.Event) { ch <- e })
	s.eb.Subscribe(events.TaskCompleted, func(e events.Event) { ch <- e })
	s.eb.Subscribe(events.TaskFailed, func(e events.Event) { ch <- e })
	s.eb.Subscribe(events.WorkflowStarted, func(e events.Event) { ch <- e })
	s.eb.Subscribe(events.WorkflowCompleted, func(e events.Event) { ch <- e })
	s.eb.Subscribe(events.WorkflowFailed, func(e events.Event) { ch <- e })

	for event := range ch {
		err := c.WriteJSON(event)
		if err != nil {
			log.Println("write:", err)
			break
		}
	}
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

func (s *Server) handleListProviders(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(providers.List(r.Context()))
}

func (s *Server) handleListWorkflows(w http.ResponseWriter, r *http.Request) {
	records, err := s.store.ListWorkflows()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(records)
}

type TaskRequest struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	Input        map[string]any `json:"input"`
	Dependencies []string       `json:"dependencies"`
	Provider     string         `json:"provider"`
	Model        string         `json:"model"`
}

type CreateWorkflowRequest struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Tasks       []TaskRequest `json:"tasks"`
}

func (s *Server) handleCreateWorkflow(w http.ResponseWriter, r *http.Request) {
	var req CreateWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	workflow := core.NewWorkflow(req.ID, req.Name, req.Description)
	if err := s.store.SaveWorkflow(store.WorkflowToRecord(workflow)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for _, tr := range req.Tasks {
		task := core.NewTask(tr.ID, workflow.ID, tr.Name, tr.Description, tr.Input, tr.Dependencies)
		task.Provider = tr.Provider
		task.Model = tr.Model
		
		// If provider and model are specified, we create a dynamic agent ID
		if tr.Provider != "" && tr.Model != "" {
			agentID := fmt.Sprintf("agent-%s-%s-%s", workflow.ID, tr.ID, tr.Provider)
			task.AgentID = agentID
			
			// Register a temporary agent for this task
			// In a real system, you might want to persist these agents or handle them differently
			s.executor.GetAgentRegistry().Register(&agents.Agent{
				ID:       agentID,
				Name:     fmt.Sprintf("Agent for %s", tr.Name),
				Provider: tr.Provider,
				Model:    tr.Model,
				Role:     agents.RoleExecutor,
			})
		}
		
		_ = workflow.AddTask(task)
		if err := s.store.SaveTask(store.TaskToRecord(workflow.ID, task)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(workflow)
}

func (s *Server) handleGetWorkflow(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	record, err := s.store.GetWorkflow(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	tasks, err := s.store.GetWorkflowTasks(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	workflow := store.RecordToWorkflow(record, tasks)
	json.NewEncoder(w).Encode(workflow)
}

func (s *Server) handleUpdateWorkflow(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req CreateWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	record, err := s.store.GetWorkflow(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	record.Name = req.Name
	record.Description = req.Description
	if err := s.store.UpdateWorkflow(record); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for _, tr := range req.Tasks {
		task := core.NewTask(tr.ID, id, tr.Name, tr.Description, tr.Input, tr.Dependencies)
		task.Provider = tr.Provider
		task.Model = tr.Model
		
		if tr.Provider != "" && tr.Model != "" {
			agentID := fmt.Sprintf("agent-%s-%s-%s", id, tr.ID, tr.Provider)
			task.AgentID = agentID
			s.executor.GetAgentRegistry().Register(&agents.Agent{
				ID:       agentID,
				Name:     fmt.Sprintf("Agent for %s", tr.Name),
				Provider: tr.Provider,
				Model:    tr.Model,
				Role:     agents.RoleExecutor,
			})
		}
		
		// Try to get it first, update if exists, else save
		_, err := s.store.GetTask(id, tr.ID)
		if err != nil {
			if err := s.store.SaveTask(store.TaskToRecord(id, task)); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			if err := s.store.UpdateTask(store.TaskToRecord(id, task)); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleExecuteWorkflow(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	record, err := s.store.GetWorkflow(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	tasks, err := s.store.GetWorkflowTasks(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	workflow := store.RecordToWorkflow(record, tasks)

	// Execute asynchronously
	go func() {
		_ = s.executor.Execute(workflow)
	}()

	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleApproveTask(w http.ResponseWriter, r *http.Request) {
	wfID := r.PathValue("id")
	taskID := r.PathValue("taskID")

	rec, err := s.store.GetTask(wfID, taskID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if rec.Status != string(core.TaskWaitingForApproval) {
		http.Error(w, "task is not waiting for approval", http.StatusBadRequest)
		return
	}

	rec.Status = string(core.TaskRunning)
	if err := s.store.UpdateTask(rec); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
