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

// EventHandler handles event log HTTP endpoints
type EventHandler struct {
	store *storage.SQLiteStore
	bus   *bus.Bus
}

// NewEventHandler creates a new EventHandler
func NewEventHandler(store *storage.SQLiteStore, b *bus.Bus) *EventHandler {
	return &EventHandler{store: store, bus: b}
}

// RegisterRoutes registers event routes on the given ServeMux
func (h *EventHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/events", h.CreateEvent)
	mux.HandleFunc("GET /api/events", h.ListEvents)
	mux.HandleFunc("GET /api/events/{id}", h.GetEvent)
}

// CreateEventRequest is the request body for creating an event
type CreateEventRequest struct {
	AgentID   string `json:"agent_id"`
	TaskID    string `json:"task_id"`
	Type      string `json:"type"` // info, warning, error
	Message   string `json:"message"`
	Metadata  string `json:"metadata"`
}

// CreateEvent handles POST /api/events
func (h *EventHandler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.WriteError(w, http.StatusMethodNotAllowed, api.ErrCodeBadRequest, "Method not allowed")
		return
	}

	var req CreateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeInvalidJSON, "Invalid JSON: "+err.Error())
		return
	}

	if req.AgentID == "" {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeMissingField, "agent_id is required")
		return
	}
	if req.Message == "" {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeMissingField, "message is required")
		return
	}
	if req.Type == "" {
		req.Type = "info"
	}
	if req.TaskID == "" {
		req.TaskID = ""
	}
	if req.Metadata == "" {
		req.Metadata = "{}"
	}

	// Validate type
	if req.Type != "info" && req.Type != "warning" && req.Type != "error" {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeValidation, "type must be one of: info, warning, error")
		return
	}

	ctx := r.Context()

	event := &models.Event{
		ID:       uuid.New().String(),
		AgentID:  req.AgentID,
		TaskID:   req.TaskID,
		Type:     req.Type,
		Message:  req.Message,
		Metadata: req.Metadata,
	}

	if err := h.store.EventRepository().Create(ctx, event); err != nil {
		log.Printf("Failed to create event: %v", err)
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternalError, "Failed to create event")
		return
	}

	// Publish event logged to bus
	if h.bus != nil {
		h.bus.Publish(bus.TopicEventLogged, bus.EventLoggedPayload{
			EventID:  event.ID,
			AgentID:  event.AgentID,
			TaskID:   event.TaskID,
			Type:     event.Type,
			Message:  event.Message,
			Metadata: event.Metadata,
		})
	}

	api.WriteSuccess(w, http.StatusCreated, event)
}

// ListEvents handles GET /api/events with optional filtering
func (h *EventHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse pagination
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

	var events []*models.Event
	var err error

	// Filter by agent_id or task_id if provided
	if agentID := r.URL.Query().Get("agent_id"); agentID != "" {
		events, err = h.store.EventRepository().GetByAgentID(ctx, agentID)
	} else if taskID := r.URL.Query().Get("task_id"); taskID != "" {
		events, err = h.store.EventRepository().GetByTaskID(ctx, taskID)
	} else {
		events, err = h.store.EventRepository().GetAll(ctx)
	}

	if err != nil {
		log.Printf("Failed to list events: %v", err)
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternalError, "Failed to list events")
		return
	}

	total := len(events)

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

	pageData := events
	if offset < total {
		pageData = events[offset:end]
	}

	api.WritePaginated(w, pageData, page, pageSize, total)
}

// GetEvent handles GET /api/events/{id}
func (h *EventHandler) GetEvent(w http.ResponseWriter, r *http.Request) {
	eventID := r.PathValue("id")
	ctx := r.Context()

	event, err := h.store.EventRepository().GetByID(ctx, eventID)
	if err != nil {
		log.Printf("Failed to get event: %v", err)
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternalError, "Failed to get event")
		return
	}
	if event == nil {
		api.WriteError(w, http.StatusNotFound, api.ErrCodeNotFound, "Event not found")
		return
	}

	api.WriteSuccess(w, http.StatusOK, event)
}
