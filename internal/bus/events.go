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
