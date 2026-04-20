package core

import "time"

type IdeaStatus string

const (
	IdeaStatusDiscussing IdeaStatus = "discussing"
	IdeaStatusDebating   IdeaStatus = "debating"
	IdeaStatusResolving  IdeaStatus = "resolving"
	IdeaStatusResolved   IdeaStatus = "resolved"
	IdeaStatusChallenged IdeaStatus = "challenged"
)

type Idea struct {
	ID               string         `json:"id"`
	ChannelID        *string        `json:"channel_id,omitempty"`
	SourceSignalID   *string        `json:"source_signal_id,omitempty"`
	CreatedByAgentID string         `json:"created_by_agent_id"`
	Domain           string         `json:"domain"`
	Title            string         `json:"title"`
	Summary          string         `json:"summary"`
	Status           IdeaStatus     `json:"status"`
	StanceSummary    map[string]any `json:"stance_summary,omitempty"`
	Meta             map[string]any `json:"meta,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

type IdeaPosition struct {
	IdeaID         string    `json:"idea_id"`
	AgentID        string    `json:"agent_id"`
	Stance         string    `json:"stance"`
	Confidence     float64   `json:"confidence"`
	SourceSignalID *string   `json:"source_signal_id,omitempty"`
	Reason         string    `json:"reason"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
