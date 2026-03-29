package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/agentflow/agentflow-core/internal/bus"
	"github.com/agentflow/agentflow-core/internal/storage"
)

func setupTest(t *testing.T) (*storage.SQLiteStore, *bus.Bus, func()) {
	tmpfile, err := os.CreateTemp("", "agentflow_test_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpfile.Close()

	store, err := storage.NewSQLiteStore(tmpfile.Name())
	if err != nil {
		os.Remove(tmpfile.Name())
		t.Fatalf("Failed to create store: %v", err)
	}

	eventBus := bus.New()

	cleanup := func() {
		store.Close()
		eventBus.Close()
		os.Remove(tmpfile.Name())
	}

	return store, eventBus, cleanup
}

func TestAgentRegister(t *testing.T) {
	store, eventBus, cleanup := setupTest(t)
	defer cleanup()

	handler := NewAgentHandler(store, eventBus)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	// Subscribe to registration events
	subCh, unsub := eventBus.Subscribe(bus.TopicAgentRegistered)
	defer unsub()

	// Register an agent
	body := map[string]string{
		"name":     "test-agent",
		"model":    "gpt-4",
		"provider": "openai",
	}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/agents", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var agent map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&agent); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if agent["id"] == "" {
		t.Error("Expected agent ID to be set")
	}
	if agent["status"] != "active" {
		t.Errorf("Expected status 'active', got %v", agent["status"])
	}

	// Verify event was published
	select {
	case evt := <-subCh:
		payload := map[string]string{}
		json.Unmarshal(evt.Payload, &payload)
		if payload["agent_id"] != agent["id"] {
			t.Errorf("Event agent_id mismatch: %s != %s", payload["agent_id"], agent["id"])
		}
	default:
		t.Error("Expected registration event to be published")
	}
}

func TestAgentHeartbeat(t *testing.T) {
	store, eventBus, cleanup := setupTest(t)
	defer cleanup()

	handler := NewAgentHandler(store, eventBus)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	// First register an agent
	regBody := map[string]string{
		"name":     "hb-agent",
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

	// Subscribe to heartbeat events
	subCh, unsub := eventBus.Subscribe(bus.TopicAgentHeartbeat)
	defer unsub()

	// Send heartbeat
	hbReq := httptest.NewRequest(http.MethodPost, "/api/agents/"+agentID+"/heartbeat", nil)
	hbRec := httptest.NewRecorder()
	mux.ServeHTTP(hbRec, hbReq)

	if hbRec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", hbRec.Code, hbRec.Body.String())
	}

	// Verify heartbeat event
	select {
	case evt := <-subCh:
		payload := map[string]string{}
		json.Unmarshal(evt.Payload, &payload)
		if payload["agent_id"] != agentID {
			t.Errorf("Heartbeat agent_id mismatch: %s != %s", payload["agent_id"], agentID)
		}
	default:
		t.Error("Expected heartbeat event to be published")
	}
}

func TestAgentHeartbeatNotFound(t *testing.T) {
	store, eventBus, cleanup := setupTest(t)
	defer cleanup()

	handler := NewAgentHandler(store, eventBus)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/agents/nonexistent/heartbeat", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rec.Code)
	}
}

func TestAgentList(t *testing.T) {
	store, eventBus, cleanup := setupTest(t)
	defer cleanup()

	handler := NewAgentHandler(store, eventBus)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	// Register two agents
	for i := 0; i < 2; i++ {
		body := map[string]string{
			"name":     "list-agent",
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
	}

	// List agents
	req := httptest.NewRequest(http.MethodGet, "/api/agents", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", rec.Code)
	}

	var agents []interface{}
	if err := json.NewDecoder(rec.Body).Decode(&agents); err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}

	if len(agents) != 2 {
		t.Errorf("Expected 2 agents, got %d", len(agents))
	}
}

func TestBusPublishSubscribe(t *testing.T) {
	b := bus.New()
	defer b.Close()

	ch, unsub := b.Subscribe(bus.TopicAgentRegistered)
	defer unsub()

	b.Publish(bus.TopicAgentRegistered, bus.AgentRegisteredPayload{
		AgentID:  "test-123",
		Name:     "test-agent",
		Model:    "gpt-4",
		Provider: "openai",
	})

	select {
	case evt := <-ch:
		if evt.Topic != bus.TopicAgentRegistered {
			t.Errorf("Expected topic %s, got %s", bus.TopicAgentRegistered, evt.Topic)
		}
	case <-ch:
		t.Error("Timed out waiting for event")
	}
}

func TestBusUnsubscribe(t *testing.T) {
	b := bus.New()
	defer b.Close()

	ch, unsub := b.Subscribe(bus.TopicAgentHeartbeat)
	unsub() // unsubscribe immediately

	// This should not block since there are no subscribers
	b.Publish(bus.TopicAgentHeartbeat, bus.HeartbeatPayload{AgentID: "x"})
}
