package core

import "time"

type AgentStatus string

const (
	AgentStatusActive    AgentStatus = "active"
	AgentStatusSuspended AgentStatus = "suspended"
	AgentStatusRevoked   AgentStatus = "revoked"
)

type Agent struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	APIKeyHash   string            `json:"-"`
	Capabilities []string          `json:"capabilities"`
	DataSources  []string          `json:"data_sources"`
	TrustScore   float64           `json:"trust_score"`
	TrustProfile AgentTrustProfile `json:"trust_profile"`
	Metadata     map[string]any    `json:"metadata"`
	Status       AgentStatus       `json:"status"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

type AgentTrustProfile struct {
	ClaimTrust     float64 `json:"claim_trust"`
	CounterTrust   float64 `json:"counter_trust"`
	ResolverTrust  float64 `json:"resolver_trust"`
	ChallengeTrust float64 `json:"challenge_trust"`
}

type AgentTrackRecord struct {
	AgentID              string    `json:"agent_id"`
	Domain               string    `json:"domain"`
	TotalPredictions     int       `json:"total_predictions"`
	CorrectPredictions   int       `json:"correct_predictions"`
	Accuracy             float64   `json:"accuracy"`
	TotalClaims          int       `json:"total_claims"`
	CorrectClaims        int       `json:"correct_claims"`
	ClaimAccuracy        float64   `json:"claim_accuracy"`
	TotalCounters        int       `json:"total_counters"`
	CorrectCounters      int       `json:"correct_counters"`
	CounterAccuracy      float64   `json:"counter_accuracy"`
	TotalResolutions     int       `json:"total_resolutions"`
	AlignedResolutions   int       `json:"aligned_resolutions"`
	ResolutionAccuracy   float64   `json:"resolution_accuracy"`
	TotalChallenges      int       `json:"total_challenges"`
	SuccessfulChallenges int       `json:"successful_challenges"`
	ChallengeAccuracy    float64   `json:"challenge_accuracy"`
	ClaimTrust           float64   `json:"claim_trust"`
	CounterTrust         float64   `json:"counter_trust"`
	ResolverTrust        float64   `json:"resolver_trust"`
	ChallengeTrust       float64   `json:"challenge_trust"`
	AvgConfidence        float64   `json:"avg_confidence"`
	LastCalculatedAt     time.Time `json:"last_calculated_at"`
}
