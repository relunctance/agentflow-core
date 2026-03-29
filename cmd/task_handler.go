package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/agentflow/agentflow-core/api"
	"github.com/agentflow/agentflow-core/internal/bus"
	"github.com/agentflow/agentflow-core/internal/storage"
	"github.com/agentflow/agentflow-core/pkg/models"
	"github.com/google/uuid"
)

// TaskHandler handles task CRUD HTTP endpoints
type TaskHandler struct {
	store *storage.SQLiteStore
	bus   *bus.Bus
}

// NewTaskHandler creates a new TaskHandler
func NewTaskHandler(store *storage.SQLiteStore, b *bus.Bus) *TaskHandler {
	return &TaskHandler{store: store, bus: b}
}

// RegisterRoutes registers task routes on the given ServeMux
func (h *TaskHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/tasks", h.CreateTask)
	mux.HandleFunc("GET /api/tasks", h.ListTasks)
	mux.HandleFunc("GET /api/tasks/{id}", h.GetTask)
	mux.HandleFunc("PUT /api/tasks/{id}", h.UpdateTask)
	mux.HandleFunc("DELETE /api/tasks/{id}", h.DeleteTask)
}

// CreateTaskRequest is the request body for creating a task
type CreateTaskRequest struct {
	AgentID     string `json:"agent_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Priority    int    `json:"priority"`
}

// UpdateTaskRequest is the request body for updating a task
type UpdateTaskRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Priority    int    `json:"priority"`
	Result      string `json:"result"`
}

// CreateTask handles POST /api/tasks
func (h *TaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.WriteError(w, http.StatusMethodNotAllowed, api.ErrCodeBadRequest, "Method not allowed")
		return
	}

	var req CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeInvalidJSON, "Invalid JSON: "+err.Error())
		return
	}

	if req.AgentID == "" {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeMissingField, "agent_id is required")
		return
	}
	if req.Title == "" {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeMissingField, "title is required")
		return
	}

	ctx := r.Context()

	// Verify agent exists
	agent, err := h.store.AgentRepository().GetByID(ctx, req.AgentID)
	if err != nil {
		log.Printf("Failed to get agent: %v", err)
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternalError, "Failed to verify agent")
		return
	}
	if agent == nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeNotFound, "Agent not found: "+req.AgentID)
		return
	}

	task := &models.Task{
		ID:          uuid.New().String(),
		AgentID:     req.AgentID,
		Title:       req.Title,
		Description: req.Description,
		Status:      "pending",
		Priority:    req.Priority,
		Result:      "{}",
	}

	if err := h.store.TaskRepository().Create(ctx, task); err != nil {
		log.Printf("Failed to create task: %v", err)
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternalError, "Failed to create task")
		return
	}

	// Publish task created event
	if h.bus != nil {
		h.bus.Publish(bus.TopicTaskCreated, bus.TaskCreatedPayload{
			TaskID:  task.ID,
			AgentID: task.AgentID,
			Title:   task.Title,
		})
	}

	api.WriteSuccess(w, http.StatusCreated, task)
}

// ListTasks handles GET /api/tasks with optional filtering
func (h *TaskHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse pagination parameters
	page := 1
	pageSize := 20
	if p := r.URL.Query().Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if ps := r.URL.Query().Get("page_size"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil && v > 0 && v <= 100 {
			pageSize = v
		}
	}

	// Filter by agent_id if provided
	var tasks []*models.Task
	var err error

	if agentID := r.URL.Query().Get("agent_id"); agentID != "" {
		tasks, err = h.store.TaskRepository().GetByAgentID(ctx, agentID)
	} else {
		tasks, err = h.store.TaskRepository().GetAll(ctx)
	}

	if err != nil {
		log.Printf("Failed to list tasks: %v", err)
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternalError, "Failed to list tasks")
		return
	}

	total := len(tasks)

	// Apply pagination
	offset := (page - 1) * pageSize
	if offset >= total && total > 0 {
		page = 1
		offset = 0
	}
	end := offset + pageSize
	if end > total {
		end = total
	}

	pageData := tasks
	if offset < total {
		pageData = tasks[offset:end]
	}

	api.WritePaginated(w, pageData, page, pageSize, total)
}

// GetTask handles GET /api/tasks/{id}
func (h *TaskHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	ctx := r.Context()

	task, err := h.store.TaskRepository().GetByID(ctx, taskID)
	if err != nil {
		log.Printf("Failed to get task: %v", err)
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternalError, "Failed to get task")
		return
	}
	if task == nil {
		api.WriteError(w, http.StatusNotFound, api.ErrCodeNotFound, "Task not found")
		return
	}

	api.WriteSuccess(w, http.StatusOK, task)
}

// UpdateTask handles PUT /api/tasks/{id}
func (h *TaskHandler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		api.WriteError(w, http.StatusMethodNotAllowed, api.ErrCodeBadRequest, "Method not allowed")
		return
	}

	taskID := r.PathValue("id")
	ctx := r.Context()

	task, err := h.store.TaskRepository().GetByID(ctx, taskID)
	if err != nil {
		log.Printf("Failed to get task: %v", err)
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternalError, "Failed to get task")
		return
	}
	if task == nil {
		api.WriteError(w, http.StatusNotFound, api.ErrCodeNotFound, "Task not found")
		return
	}

	var req UpdateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeInvalidJSON, "Invalid JSON: "+err.Error())
		return
	}

	// Apply updates
	if req.Title != "" {
		task.Title = req.Title
	}
	if req.Description != "" {
		task.Description = req.Description
	}
	if req.Status != "" {
		task.Status = req.Status
	}
	if req.Priority != 0 {
		task.Priority = req.Priority
	}
	if req.Result != "" {
		task.Result = req.Result
	}

	if err := h.store.TaskRepository().Update(ctx, task); err != nil {
		log.Printf("Failed to update task: %v", err)
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternalError, "Failed to update task")
		return
	}

	// Publish task updated event
	if h.bus != nil {
		h.bus.Publish(bus.TopicTaskUpdated, bus.TaskUpdatedPayload{
			TaskID: task.ID,
			Status: task.Status,
		})
	}

	api.WriteSuccess(w, http.StatusOK, task)
}

// DeleteTask handles DELETE /api/tasks/{id}
func (h *TaskHandler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		api.WriteError(w, http.StatusMethodNotAllowed, api.ErrCodeBadRequest, "Method not allowed")
		return
	}

	taskID := r.PathValue("id")
	ctx := r.Context()

	task, err := h.store.TaskRepository().GetByID(ctx, taskID)
	if err != nil {
		log.Printf("Failed to get task: %v", err)
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternalError, "Failed to get task")
		return
	}
	if task == nil {
		api.WriteError(w, http.StatusNotFound, api.ErrCodeNotFound, "Task not found")
		return
	}

	if err := h.store.TaskRepository().Delete(ctx, taskID); err != nil {
		log.Printf("Failed to delete task: %v", err)
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternalError, "Failed to delete task")
		return
	}

	api.WriteSuccess(w, http.StatusOK, map[string]string{"message": "Task deleted"})
}
