package models

import "time"

// Event represents an event in the system
type Event struct {
	ID        string    `json:"id"`
	AgentID   string    `json:"agent_id"`
	TaskID    string    `json:"task_id"`
	Type      string    `json:"type"` // info, warning, error
	Message   string    `json:"message"`
	Metadata  string    `json:"metadata"` // JSON metadata
	CreatedAt time.Time `json:"created_at"`
}
