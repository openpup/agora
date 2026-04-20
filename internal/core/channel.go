package core

import "time"

type ChannelKind string

const (
	ChannelKindDomain ChannelKind = "domain"
	ChannelKindTopic  ChannelKind = "topic"
	ChannelKindSystem ChannelKind = "system"
)

type Channel struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Slug        string         `json:"slug"`
	Domain      string         `json:"domain"`
	Kind        ChannelKind    `json:"kind"`
	Description string         `json:"description"`
	Meta        map[string]any `json:"meta,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

type ChannelMessageKind string

const (
	ChannelMessageKindChat     ChannelMessageKind = "chat"
	ChannelMessageKindQuestion ChannelMessageKind = "question"
	ChannelMessageKindAnswer   ChannelMessageKind = "answer"
	ChannelMessageKindProtocol ChannelMessageKind = "protocol"
	ChannelMessageKindSystem   ChannelMessageKind = "system"
)

type ChannelMessage struct {
	ID        string             `json:"id"`
	ChannelID string             `json:"channel_id"`
	IdeaID    *string            `json:"idea_id,omitempty"`
	AgentID   string             `json:"agent_id"`
	Kind      ChannelMessageKind `json:"kind"`
	Intent    string             `json:"intent"`
	Body      string             `json:"body"`
	Refs      []CrossRef         `json:"refs,omitempty"`
	Meta      map[string]any     `json:"meta,omitempty"`
	CreatedAt time.Time          `json:"created_at"`
}
