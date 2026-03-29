package models

import "context"

// AgentRepository defines the interface for Agent data access
type AgentRepository interface {
	Create(ctx context.Context, agent *Agent) error
	GetByID(ctx context.Context, id string) (*Agent, error)
	GetAll(ctx context.Context) ([]*Agent, error)
	Update(ctx context.Context, agent *Agent) error
	Delete(ctx context.Context, id string) error
}

// TaskRepository defines the interface for Task data access
type TaskRepository interface {
	Create(ctx context.Context, task *Task) error
	GetByID(ctx context.Context, id string) (*Task, error)
	GetByAgentID(ctx context.Context, agentID string) ([]*Task, error)
	GetAll(ctx context.Context) ([]*Task, error)
	Update(ctx context.Context, task *Task) error
	Delete(ctx context.Context, id string) error
}

// EventRepository defines the interface for Event data access
type EventRepository interface {
	Create(ctx context.Context, event *Event) error
	GetByID(ctx context.Context, id string) (*Event, error)
	GetByAgentID(ctx context.Context, agentID string) ([]*Event, error)
	GetByTaskID(ctx context.Context, taskID string) ([]*Event, error)
	GetAll(ctx context.Context) ([]*Event, error)
	Delete(ctx context.Context, id string) error
}
