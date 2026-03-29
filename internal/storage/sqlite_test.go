package storage

import (
	"context"
	"os"
	"testing"

	"github.com/agentflow/agentflow-core/pkg/models"
)

func setupTestDB(t *testing.T) (*SQLiteStore, func()) {
	tmpfile, err := os.CreateTemp("", "test_agentflow_*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpfile.Close()

	store, err := NewSQLiteStore(tmpfile.Name())
	if err != nil {
		os.Remove(tmpfile.Name())
		t.Fatalf("failed to create store: %v", err)
	}

	cleanup := func() {
		store.Close()
		os.Remove(tmpfile.Name())
	}
	return store, cleanup
}

func TestAgentRepository_CRUD(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()
	repo := store.agents

	// Create
	agent := &models.Agent{
		ID:       "agent-1",
		Name:     "Test Agent",
		Model:    "gpt-4",
		Provider: "openai",
		Status:   "active",
		Config:   `{"temperature":0.7}`,
	}
	if err := repo.Create(ctx, agent); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// GetByID
	got, err := repo.GetByID(ctx, "agent-1")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got.Name != "Test Agent" {
		t.Errorf("expected name 'Test Agent', got '%s'", got.Name)
	}

	// GetAll
	all, err := repo.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}
	if len(all) != 1 {
		t.Errorf("expected 1 agent, got %d", len(all))
	}

	// Update
	agent.Name = "Updated Agent"
	if err := repo.Update(ctx, agent); err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	got, _ = repo.GetByID(ctx, "agent-1")
	if got.Name != "Updated Agent" {
		t.Errorf("expected 'Updated Agent', got '%s'", got.Name)
	}

	// Delete
	if err := repo.Delete(ctx, "agent-1"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	got, _ = repo.GetByID(ctx, "agent-1")
	if got != nil {
		t.Errorf("expected nil after delete, got %v", got)
	}
}

func TestTaskRepository_CRUD(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()
	agentRepo := store.agents
	taskRepo := store.tasks

	// Create agent first
	agent := &models.Agent{ID: "agent-1", Name: "Test", Model: "gpt-4", Provider: "openai"}
	agentRepo.Create(ctx, agent)

	// Create
	task := &models.Task{
		ID:          "task-1",
		AgentID:     "agent-1",
		Title:       "Test Task",
		Description: "A test task",
		Status:      "pending",
		Priority:    1,
	}
	if err := taskRepo.Create(ctx, task); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// GetByID
	got, err := taskRepo.GetByID(ctx, "task-1")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got.Title != "Test Task" {
		t.Errorf("expected 'Test Task', got '%s'", got.Title)
	}

	// GetByAgentID
	tasks, err := taskRepo.GetByAgentID(ctx, "agent-1")
	if err != nil {
		t.Fatalf("GetByAgentID failed: %v", err)
	}
	if len(tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(tasks))
	}

	// Update
	task.Status = "completed"
	if err := taskRepo.Update(ctx, task); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Delete
	if err := taskRepo.Delete(ctx, "task-1"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
}

func TestEventRepository_CRUD(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()
	agentRepo := store.agents
	eventRepo := store.events

	// Create agent first
	agent := &models.Agent{ID: "agent-1", Name: "Test", Model: "gpt-4", Provider: "openai"}
	agentRepo.Create(ctx, agent)

	// Create
	event := &models.Event{
		ID:       "event-1",
		AgentID:  "agent-1",
		TaskID:   "task-1",
		Type:     "info",
		Message:  "Test event",
		Metadata: `{"key":"value"}`,
	}
	if err := eventRepo.Create(ctx, event); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// GetByID
	got, err := eventRepo.GetByID(ctx, "event-1")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got.Message != "Test event" {
		t.Errorf("expected 'Test event', got '%s'", got.Message)
	}

	// GetByAgentID
	events, err := eventRepo.GetByAgentID(ctx, "agent-1")
	if err != nil {
		t.Fatalf("GetByAgentID failed: %v", err)
	}
	if len(events) != 1 {
		t.Errorf("expected 1 event, got %d", len(events))
	}

	// GetByTaskID
	events, err = eventRepo.GetByTaskID(ctx, "task-1")
	if err != nil {
		t.Fatalf("GetByTaskID failed: %v", err)
	}
	if len(events) != 1 {
		t.Errorf("expected 1 event, got %d", len(events))
	}

	// Delete
	if err := eventRepo.Delete(ctx, "event-1"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
}

func TestNewSQLiteStore(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "test_agentflow_new_*.db")
	if err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	store, err := NewSQLiteStore(tmpfile.Name())
	if err != nil {
		t.Fatalf("NewSQLiteStore failed: %v", err)
	}
	if store == nil {
		t.Fatal("expected non-nil store")
	}
	if store.db == nil {
		t.Fatal("expected non-nil db")
	}
	store.Close()
}
