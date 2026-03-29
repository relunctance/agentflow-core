package storage

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/agentflow/agentflow-core/pkg/models"
)

// SQLiteStore holds the database connection and repositories
type SQLiteStore struct {
	db     *sql.DB
	agents *AgentRepository
	tasks  *TaskRepository
	events *EventRepository
}

// NewSQLiteStore creates a new SQLite store with automatic migrations
func NewSQLiteStore(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite3", path+"?_foreign_keys=on")
	if err != nil {
		return nil, err
	}

	store := &SQLiteStore{db: db}
	if err := store.migrate(); err != nil {
		db.Close()
		return nil, err
	}

	store.agents = &AgentRepository{db: db}
	store.tasks = &TaskRepository{db: db}
	store.events = &EventRepository{db: db}

	return store, nil
}

// migrate creates all necessary tables
func (s *SQLiteStore) migrate() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS agents (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			model TEXT NOT NULL,
			provider TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'active',
			config TEXT NOT NULL DEFAULT '{}',
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS tasks (
			id TEXT PRIMARY KEY,
			agent_id TEXT NOT NULL,
			title TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'pending',
			priority INTEGER NOT NULL DEFAULT 0,
			result TEXT NOT NULL DEFAULT '{}',
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			FOREIGN KEY (agent_id) REFERENCES agents(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS events (
			id TEXT PRIMARY KEY,
			agent_id TEXT NOT NULL,
			task_id TEXT NOT NULL DEFAULT '',
			type TEXT NOT NULL DEFAULT 'info',
			message TEXT NOT NULL,
			metadata TEXT NOT NULL DEFAULT '{}',
			created_at DATETIME NOT NULL,
			FOREIGN KEY (agent_id) REFERENCES agents(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_agent_id ON tasks(agent_id)`,
		`CREATE INDEX IF NOT EXISTS idx_events_agent_id ON events(agent_id)`,
		`CREATE INDEX IF NOT EXISTS idx_events_task_id ON events(task_id)`,
	}

	for _, m := range migrations {
		if _, err := s.db.Exec(m); err != nil {
			return err
		}
	}
	return nil
}

// Close closes the database connection
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// AgentRepository returns the Agent repository
func (s *SQLiteStore) AgentRepository() models.AgentRepository {
	return s.agents
}

// TaskRepository returns the Task repository
func (s *SQLiteStore) TaskRepository() models.TaskRepository {
	return s.tasks
}

// EventRepository returns the Event repository
func (s *SQLiteStore) EventRepository() models.EventRepository {
	return s.events
}

// AgentRepository implements models.AgentRepository using SQLite
type AgentRepository struct {
	db *sql.DB
}

func (r *AgentRepository) Create(ctx context.Context, agent *models.Agent) error {
	agent.CreatedAt = time.Now()
	agent.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO agents (id, name, model, provider, status, config, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		agent.ID, agent.Name, agent.Model, agent.Provider, agent.Status, agent.Config, agent.CreatedAt, agent.UpdatedAt)
	return err
}

func (r *AgentRepository) GetByID(ctx context.Context, id string) (*models.Agent, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, name, model, provider, status, config, created_at, updated_at FROM agents WHERE id = ?`, id)
	agent := &models.Agent{}
	err := row.Scan(&agent.ID, &agent.Name, &agent.Model, &agent.Provider, &agent.Status, &agent.Config, &agent.CreatedAt, &agent.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return agent, err
}

func (r *AgentRepository) GetAll(ctx context.Context) ([]*models.Agent, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, model, provider, status, config, created_at, updated_at FROM agents ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agents []*models.Agent
	for rows.Next() {
		agent := &models.Agent{}
		if err := rows.Scan(&agent.ID, &agent.Name, &agent.Model, &agent.Provider, &agent.Status, &agent.Config, &agent.CreatedAt, &agent.UpdatedAt); err != nil {
			return nil, err
		}
		agents = append(agents, agent)
	}
	return agents, rows.Err()
}

func (r *AgentRepository) Update(ctx context.Context, agent *models.Agent) error {
	agent.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx,
		`UPDATE agents SET name = ?, model = ?, provider = ?, status = ?, config = ?, updated_at = ? WHERE id = ?`,
		agent.Name, agent.Model, agent.Provider, agent.Status, agent.Config, agent.UpdatedAt, agent.ID)
	return err
}

func (r *AgentRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM agents WHERE id = ?`, id)
	return err
}

// TaskRepository implements models.TaskRepository using SQLite
type TaskRepository struct {
	db *sql.DB
}

func (r *TaskRepository) Create(ctx context.Context, task *models.Task) error {
	task.CreatedAt = time.Now()
	task.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO tasks (id, agent_id, title, description, status, priority, result, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		task.ID, task.AgentID, task.Title, task.Description, task.Status, task.Priority, task.Result, task.CreatedAt, task.UpdatedAt)
	return err
}

func (r *TaskRepository) GetByID(ctx context.Context, id string) (*models.Task, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, agent_id, title, description, status, priority, result, created_at, updated_at FROM tasks WHERE id = ?`, id)
	task := &models.Task{}
	err := row.Scan(&task.ID, &task.AgentID, &task.Title, &task.Description, &task.Status, &task.Priority, &task.Result, &task.CreatedAt, &task.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return task, err
}

func (r *TaskRepository) GetByAgentID(ctx context.Context, agentID string) ([]*models.Task, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, agent_id, title, description, status, priority, result, created_at, updated_at FROM tasks WHERE agent_id = ? ORDER BY created_at DESC`, agentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*models.Task
	for rows.Next() {
		task := &models.Task{}
		if err := rows.Scan(&task.ID, &task.AgentID, &task.Title, &task.Description, &task.Status, &task.Priority, &task.Result, &task.CreatedAt, &task.UpdatedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, rows.Err()
}

func (r *TaskRepository) GetAll(ctx context.Context) ([]*models.Task, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, agent_id, title, description, status, priority, result, created_at, updated_at FROM tasks ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*models.Task
	for rows.Next() {
		task := &models.Task{}
		if err := rows.Scan(&task.ID, &task.AgentID, &task.Title, &task.Description, &task.Status, &task.Priority, &task.Result, &task.CreatedAt, &task.UpdatedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, rows.Err()
}

func (r *TaskRepository) Update(ctx context.Context, task *models.Task) error {
	task.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx,
		`UPDATE tasks SET agent_id = ?, title = ?, description = ?, status = ?, priority = ?, result = ?, updated_at = ? WHERE id = ?`,
		task.AgentID, task.Title, task.Description, task.Status, task.Priority, task.Result, task.UpdatedAt, task.ID)
	return err
}

func (r *TaskRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM tasks WHERE id = ?`, id)
	return err
}

// EventRepository implements models.EventRepository using SQLite
type EventRepository struct {
	db *sql.DB
}

func (r *EventRepository) Create(ctx context.Context, event *models.Event) error {
	event.CreatedAt = time.Now()
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO events (id, agent_id, task_id, type, message, metadata, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		event.ID, event.AgentID, event.TaskID, event.Type, event.Message, event.Metadata, event.CreatedAt)
	return err
}

func (r *EventRepository) GetByID(ctx context.Context, id string) (*models.Event, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, agent_id, task_id, type, message, metadata, created_at FROM events WHERE id = ?`, id)
	event := &models.Event{}
	err := row.Scan(&event.ID, &event.AgentID, &event.TaskID, &event.Type, &event.Message, &event.Metadata, &event.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return event, err
}

func (r *EventRepository) GetByAgentID(ctx context.Context, agentID string) ([]*models.Event, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, agent_id, task_id, type, message, metadata, created_at FROM events WHERE agent_id = ? ORDER BY created_at DESC`, agentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*models.Event
	for rows.Next() {
		event := &models.Event{}
		if err := rows.Scan(&event.ID, &event.AgentID, &event.TaskID, &event.Type, &event.Message, &event.Metadata, &event.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func (r *EventRepository) GetByTaskID(ctx context.Context, taskID string) ([]*models.Event, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, agent_id, task_id, type, message, metadata, created_at FROM events WHERE task_id = ? ORDER BY created_at DESC`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*models.Event
	for rows.Next() {
		event := &models.Event{}
		if err := rows.Scan(&event.ID, &event.AgentID, &event.TaskID, &event.Type, &event.Message, &event.Metadata, &event.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func (r *EventRepository) GetAll(ctx context.Context) ([]*models.Event, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, agent_id, task_id, type, message, metadata, created_at FROM events ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*models.Event
	for rows.Next() {
		event := &models.Event{}
		if err := rows.Scan(&event.ID, &event.AgentID, &event.TaskID, &event.Type, &event.Message, &event.Metadata, &event.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func (r *EventRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM events WHERE id = ?`, id)
	return err
}
