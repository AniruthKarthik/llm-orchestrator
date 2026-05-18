package api

import (
	"encoding/json"
	"net/http"

	"github.com/AniruthKarthik/llm-orchestrator/internal/core"
	"github.com/AniruthKarthik/llm-orchestrator/internal/executor"
	"github.com/AniruthKarthik/llm-orchestrator/internal/store"
)

type Server struct {
	executor *executor.Executor
	store    store.Store
}

func NewServer(e *executor.Executor, s store.Store) *Server {
	return &Server{
		executor: e,
		store:    s,
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /workflows", s.handleCreateWorkflow)
	mux.HandleFunc("GET /workflows/{id}", s.handleGetWorkflow)
	mux.HandleFunc("POST /workflows/{id}/execute", s.handleExecuteWorkflow)
	mux.HandleFunc("POST /workflows/{id}/tasks/{taskID}/approve", s.handleApproveTask)
	return mux
}

type TaskRequest struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	Input        map[string]any `json:"input"`
	Dependencies []string       `json:"dependencies"`
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
