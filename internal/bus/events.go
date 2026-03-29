package bus

import (
	"encoding/json"
	"time"
)

// Topic constants
const (
	TopicAgentRegistered = "agent.registered"
	TopicAgentHeartbeat  = "agent.heartbeat"
	TopicAgentOffline    = "agent.offline"
	TopicAgentOnline     = "agent.online"
	TopicTaskCreated     = "task.created"
	TopicTaskUpdated     = "task.updated"
	TopicTaskDeleted     = "task.deleted"
	TopicEventLogged     = "event.logged"
)

// Event represents a domain event on the bus
type Event struct {
	Topic   string          `json:"topic"`
	Payload json.RawMessage `json:"payload"`
	Time    time.Time       `json:"time"`
}

// AgentRegisteredPayload is the payload for agent registration events
type AgentRegisteredPayload struct {
	AgentID  string `json:"agent_id"`
	Name     string `json:"name"`
	Model    string `json:"model"`
	Provider string `json:"provider"`
}

// HeartbeatPayload is the payload for heartbeat events
type HeartbeatPayload struct {
	AgentID string `json:"agent_id"`
}

// AgentOfflinePayload is the payload for agent offline events
type AgentOfflinePayload struct {
	AgentID string `json:"agent_id"`
}

// TaskCreatedPayload is the payload for task creation events
type TaskCreatedPayload struct {
	TaskID  string `json:"task_id"`
	AgentID string `json:"agent_id"`
	Title   string `json:"title"`
}

// TaskUpdatedPayload is the payload for task update events
type TaskUpdatedPayload struct {
	TaskID string `json:"task_id"`
	Status string `json:"status"`
}

// EventLoggedPayload is the payload for event log events
type EventLoggedPayload struct {
	EventID  string `json:"event_id"`
	AgentID  string `json:"agent_id"`
	TaskID   string `json:"task_id"`
	Type     string `json:"type"`
	Message  string `json:"message"`
	Metadata string `json:"metadata"`
}
