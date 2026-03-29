package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/agentflow/agentflow-core/internal/bus"
)

// TestOfflineDetector_MarksAgentOfflineAfter30s verifies that an agent
// without heartbeat for 30 seconds is automatically marked offline.
func TestOfflineDetector_MarksAgentOfflineAfter30s(t *testing.T) {
	store, eventBus, cleanup := setupTest(t)
	defer cleanup()

	handler := NewAgentHandler(store, eventBus)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	// Subscribe to offline events
	subCh, unsub := eventBus.Subscribe(bus.TopicAgentOffline)
	defer unsub()

	// Register an agent (this starts tracking it in liveness map)
	body := map[string]string{
		"name":     "offline-test-agent",
		"model":    "gpt-4",
		"provider": "openai",
	}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/agents", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("Failed to register agent: %d", rec.Code)
	}

	var agent map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&agent)
	agentID := agent["id"].(string)

	// Verify agent is active
	ctx := context.Background()
	dbAgent, _ := store.AgentRepository().GetByID(ctx, agentID)
	if dbAgent.Status != "active" {
		t.Errorf("Expected agent status 'active', got %s", dbAgent.Status)
	}

	// Simulate time passage by directly manipulating liveness map
	// (bypass the 30s wait)
	handler.mu.Lock()
	handler.liveness[agentID] = time.Now().Add(-31 * time.Second)
	handler.mu.Unlock()

	// Trigger a manual check
	handler.markStaleOffline()

	// Verify agent is now offline
	dbAgent, _ = store.AgentRepository().GetByID(ctx, agentID)
	if dbAgent.Status != "offline" {
		t.Errorf("Expected agent status 'offline', got %s", dbAgent.Status)
	}

	// Verify offline event was published
	select {
	case evt := <-subCh:
		payload := map[string]string{}
		json.Unmarshal(evt.Payload, &payload)
		if payload["agent_id"] != agentID {
			t.Errorf("Event agent_id mismatch: %s != %s", payload["agent_id"], agentID)
		}
	default:
		t.Error("Expected offline event to be published")
	}
}

// TestOfflineDetector_AgentWithHeartbeatStaysActive verifies that an agent
// that sends heartbeat before 30s threshold stays active.
func TestOfflineDetector_AgentWithHeartbeatStaysActive(t *testing.T) {
	store, eventBus, cleanup := setupTest(t)
	defer cleanup()

	handler := NewAgentHandler(store, eventBus)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	// Register an agent
	regBody := map[string]string{
		"name":     "active-agent",
		"model":    "claude-3",
		"provider": "anthropic",
	}
	regBytes, _ := json.Marshal(regBody)
	regReq := httptest.NewRequest(http.MethodPost, "/api/agents", bytes.NewReader(regBytes))
	regReq.Header.Set("Content-Type", "application/json")
	regRec := httptest.NewRecorder()
	mux.ServeHTTP(regRec, regReq)

	var agent map[string]interface{}
	json.NewDecoder(regRec.Body).Decode(&agent)
	agentID := agent["id"].(string)

	// Subscribe to offline events (should NOT receive any)
	subCh, unsub := eventBus.Subscribe(bus.TopicAgentOffline)
	defer unsub()

	// Simulate approaching threshold (29s elapsed)
	handler.mu.Lock()
	handler.liveness[agentID] = time.Now().Add(-29 * time.Second)
	handler.mu.Unlock()

	handler.markStaleOffline()

	// Verify agent is still active
	ctx := context.Background()
	dbAgent, _ := store.AgentRepository().GetByID(ctx, agentID)
	if dbAgent.Status != "active" {
		t.Errorf("Expected agent status 'active', got %s", dbAgent.Status)
	}

	// Verify NO offline event was published
	select {
	case <-subCh:
		t.Error("Should not have received offline event for active agent")
	default:
		// expected
	}
}

// TestOfflineDetector_MultipleAgents verifies offline detection works
// correctly with multiple agents at different stages.
func TestOfflineDetector_MultipleAgents(t *testing.T) {
	store, eventBus, cleanup := setupTest(t)
	defer cleanup()

	handler := NewAgentHandler(store, eventBus)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	ctx := context.Background()
	offlineCh, unsub := eventBus.Subscribe(bus.TopicAgentOffline)
	defer unsub()

	// Register 3 agents
	agentIDs := make([]string, 3)
	for i := 0; i < 3; i++ {
		regBody := map[string]string{
			"name":     "multi-agent",
			"model":    "gpt-4",
			"provider": "openai",
		}
		regBytes, _ := json.Marshal(regBody)
		regReq := httptest.NewRequest(http.MethodPost, "/api/agents", bytes.NewReader(regBytes))
		regReq.Header.Set("Content-Type", "application/json")
		regRec := httptest.NewRecorder()
		mux.ServeHTTP(regRec, regReq)

		var agent map[string]interface{}
		json.NewDecoder(regRec.Body).Decode(&agent)
		agentIDs[i] = agent["id"].(string)
	}

	// Agent 0: stale (35s ago) -> should go offline
	// Agent 1: stale (35s ago) -> should go offline
	// Agent 2: fresh (5s ago) -> should stay active
	handler.mu.Lock()
	handler.liveness[agentIDs[0]] = time.Now().Add(-35 * time.Second)
	handler.liveness[agentIDs[1]] = time.Now().Add(-35 * time.Second)
	handler.liveness[agentIDs[2]] = time.Now().Add(-5 * time.Second)
	handler.mu.Unlock()

	handler.markStaleOffline()

	// Verify correct offline/active states
	dbAgent0, _ := store.AgentRepository().GetByID(ctx, agentIDs[0])
	dbAgent1, _ := store.AgentRepository().GetByID(ctx, agentIDs[1])
	dbAgent2, _ := store.AgentRepository().GetByID(ctx, agentIDs[2])

	if dbAgent0.Status != "offline" {
		t.Errorf("Agent0: expected 'offline', got %s", dbAgent0.Status)
	}
	if dbAgent1.Status != "offline" {
		t.Errorf("Agent1: expected 'offline', got %s", dbAgent1.Status)
	}
	if dbAgent2.Status != "active" {
		t.Errorf("Agent2: expected 'active', got %s", dbAgent2.Status)
	}

	// Should receive exactly 2 offline events
	offlineCount := 0
	timeout := time.After(500 * time.Millisecond)
	for {
		select {
		case <-offlineCh:
			offlineCount++
			if offlineCount > 2 {
				t.Errorf("Received more offline events than expected")
				return
			}
		case <-timeout:
			if offlineCount != 2 {
				t.Errorf("Expected 2 offline events, got %d", offlineCount)
			}
			return
		}
	}
}

// TestOfflineDetector_AgentGoesOnlineAfterReconnect verifies that an offline
// agent can come back online after sending a new heartbeat.
func TestOfflineDetector_AgentGoesOnlineAfterReconnect(t *testing.T) {
	store, eventBus, cleanup := setupTest(t)
	defer cleanup()

	handler := NewAgentHandler(store, eventBus)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	// Register an agent
	regBody := map[string]string{
		"name":     "reconnect-agent",
		"model":    "gpt-4",
		"provider": "openai",
	}
	regBytes, _ := json.Marshal(regBody)
	regReq := httptest.NewRequest(http.MethodPost, "/api/agents", bytes.NewReader(regBytes))
	regReq.Header.Set("Content-Type", "application/json")
	regRec := httptest.NewRecorder()
	mux.ServeHTTP(regRec, regReq)

	var agent map[string]interface{}
	json.NewDecoder(regRec.Body).Decode(&agent)
	agentID := agent["id"].(string)

	// Force it offline
	handler.mu.Lock()
	handler.liveness[agentID] = time.Now().Add(-31 * time.Second)
	handler.mu.Unlock()
	handler.markStaleOffline()

	ctx := context.Background()
	dbAgent, _ := store.AgentRepository().GetByID(ctx, agentID)
	if dbAgent.Status != "offline" {
		t.Fatalf("Expected offline, got %s", dbAgent.Status)
	}

	// Agent reconnects with heartbeat
	hbReq := httptest.NewRequest(http.MethodPost, "/api/agents/"+agentID+"/heartbeat", nil)
	hbRec := httptest.NewRecorder()
	mux.ServeHTTP(hbRec, hbReq)

	if hbRec.Code != http.StatusOK {
		t.Fatalf("Heartbeat failed: %d", hbRec.Code)
	}

	// Verify agent is back active
	dbAgent, _ = store.AgentRepository().GetByID(ctx, agentID)
	if dbAgent.Status != "active" {
		t.Errorf("Expected 'active' after reconnect, got %s", dbAgent.Status)
	}
}
