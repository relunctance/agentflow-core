package models

import "time"

// Agent represents an agent in the system
type Agent struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Model     string    `json:"model"`
	Provider  string    `json:"provider"`
	Status    string    `json:"status"` // active, inactive, error
	Config    string    `json:"config"` // JSON config
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
