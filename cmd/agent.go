package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/agentflow/agentflow-core/internal/bus"
	"github.com/agentflow/agentflow-core/internal/storage"
	"github.com/agentflow/agentflow-core/pkg/models"
	"github.com/google/uuid"
)

// AgentHandler handles agent registration and heartbeat HTTP endpoints
type AgentHandler struct {
	store  *storage.SQLiteStore
	bus    *bus.Bus
	mu     sync.RWMutex
	liveness map[string]time.Time // track last heartbeat per agent
	stopCh  chan struct{}
}

func NewAgentHandler(store *storage.SQLiteStore, b *bus.Bus) *AgentHandler {
	h := &AgentHandler{
		store:    store,
		bus:      b,
		liveness: make(map[string]time.Time),
		stopCh:   make(chan struct{}),
	}
	// Start offline checker goroutine
	go h.checkLiveness()
	return h
}

// RegisterRoutes registers agent routes on the given ServeMux
func (h *AgentHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/agents", h.Register)
	mux.HandleFunc("POST /api/agents/{id}/heartbeat", h.Heartbeat)
	mux.HandleFunc("GET /api/agents", h.ListAgents)
	mux.HandleFunc("GET /api/agents/{id}", h.GetAgent)
}

// Register handles POST /api/agents
// Request body: {"name": "agent-1", "model": "gpt-4", "provider": "openai", "config": "{}"}
func (h *AgentHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name     string `json:"name"`
		Model    string `json:"model"`
		Provider string `json:"provider"`
		Config   string `json:"config"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.Model == "" || req.Provider == "" {
		http.Error(w, "name, model, provider are required", http.StatusBadRequest)
		return
	}
	if req.Config == "" {
		req.Config = "{}"
	}

	agent := &models.Agent{
		ID:       uuid.New().String(),
		Name:     req.Name,
		Model:    req.Model,
		Provider: req.Provider,
		Status:   "active",
		Config:   req.Config,
	}

	ctx := r.Context()
	if err := h.store.AgentRepository().Create(ctx, agent); err != nil {
		log.Printf("Failed to create agent: %v", err)
		http.Error(w, "Failed to create agent", http.StatusInternalServerError)
		return
	}

	// Publish registration event to bus
	if h.bus != nil {
		h.bus.Publish(bus.TopicAgentRegistered, bus.AgentRegisteredPayload{
			AgentID:  agent.ID,
			Name:     agent.Name,
			Model:    agent.Model,
			Provider: agent.Provider,
		})
	}

	// Track initial heartbeat time
	h.mu.Lock()
	h.liveness[agent.ID] = time.Now()
	h.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(agent)
}

// Heartbeat handles POST /api/agents/{id}/heartbeat
func (h *AgentHandler) Heartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	agentID := r.PathValue("id")
	if agentID == "" {
		http.Error(w, "Agent ID required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Get the agent
	agent, err := h.store.AgentRepository().GetByID(ctx, agentID)
	if err != nil {
		log.Printf("Failed to get agent: %v", err)
		http.Error(w, "Failed to get agent", http.StatusInternalServerError)
		return
	}
	if agent == nil {
		http.Error(w, "Agent not found", http.StatusNotFound)
		return
	}

	// Update heartbeat timestamp and mark online
	agent.Status = "active"
	if err := h.store.AgentRepository().Update(ctx, agent); err != nil {
		log.Printf("Failed to update agent: %v", err)
		http.Error(w, "Failed to update agent", http.StatusInternalServerError)
		return
	}

	h.mu.Lock()
	h.liveness[agentID] = time.Now()
	h.mu.Unlock()

	// Publish heartbeat event to bus
	if h.bus != nil {
		h.bus.Publish(bus.TopicAgentHeartbeat, bus.HeartbeatPayload{
			AgentID: agentID,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// ListAgents handles GET /api/agents
func (h *AgentHandler) ListAgents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	agents, err := h.store.AgentRepository().GetAll(ctx)
	if err != nil {
		log.Printf("Failed to list agents: %v", err)
		http.Error(w, "Failed to list agents", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agents)
}

// GetAgent handles GET /api/agents/{id}
func (h *AgentHandler) GetAgent(w http.ResponseWriter, r *http.Request) {
	agentID := r.PathValue("id")
	ctx := r.Context()
	agent, err := h.store.AgentRepository().GetByID(ctx, agentID)
	if err != nil {
		http.Error(w, "Failed to get agent", http.StatusInternalServerError)
		return
	}
	if agent == nil {
		http.Error(w, "Agent not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agent)
}

// checkLiveness periodically checks for agents that have missed heartbeats
// and marks them as offline if no heartbeat received in 30 seconds
func (h *AgentHandler) checkLiveness() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			h.markStaleOffline()
		case <-h.stopCh:
			return
		}
	}
}

func (h *AgentHandler) markStaleOffline() {
	h.mu.Lock()
	defer h.mu.Unlock()

	ctx := context.Background()
	threshold := 30 * time.Second
	now := time.Now()

	for agentID, lastHB := range h.liveness {
		if now.Sub(lastHB) > threshold {
			agent, err := h.store.AgentRepository().GetByID(ctx, agentID)
			if err != nil || agent == nil {
				continue
			}
			if agent.Status != "offline" {
				agent.Status = "offline"
				if err := h.store.AgentRepository().Update(ctx, agent); err != nil {
					log.Printf("Failed to mark agent %s offline: %v", agentID, err)
					continue
				}
				log.Printf("Agent %s marked offline (no heartbeat for >30s)", agentID)

				// Publish offline event
				if h.bus != nil {
					h.bus.Publish(bus.TopicAgentOffline, bus.AgentOfflinePayload{
						AgentID: agentID,
					})
				}
			}
			// Remove from liveness map so we don't keep trying
			delete(h.liveness, agentID)
		}
	}
}

// Stop stops the liveness checker
func (h *AgentHandler) Stop() {
	close(h.stopCh)
}
