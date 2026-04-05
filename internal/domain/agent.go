package domain

import "time"

type AgentStatus string

const (
	AgentStatusActive    AgentStatus = "active"
	AgentStatusSuspended AgentStatus = "suspended"
	AgentStatusRevoked   AgentStatus = "revoked"
)

type Agent struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	APIKeyHash   string         `json:"-"`
	Capabilities []string       `json:"capabilities"`
	DataSources  []string       `json:"data_sources"`
	TrustScore   float64        `json:"trust_score"`
	Metadata     map[string]any `json:"metadata"`
	Status       AgentStatus    `json:"status"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

type AgentTrackRecord struct {
	AgentID            string    `json:"agent_id"`
	Market             Market    `json:"market"`
	TotalPredictions   int       `json:"total_predictions"`
	CorrectPredictions int       `json:"correct_predictions"`
	Accuracy           float64   `json:"accuracy"`
	AvgConfidence      float64   `json:"avg_confidence"`
	LastCalculatedAt   time.Time `json:"last_calculated_at"`
}
