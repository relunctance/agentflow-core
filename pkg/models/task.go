package models

import "time"

// Task represents a task in the system
type Task struct {
	ID          string    `json:"id"`
	AgentID     string    `json:"agent_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"` // pending, running, completed, failed
	Priority    int       `json:"priority"`
	Result      string    `json:"result"` // JSON result
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
