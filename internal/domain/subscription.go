package domain

import "time"

type DeliveryMethod string

const (
	DeliveryWebSocket DeliveryMethod = "websocket"
	DeliveryWebhook   DeliveryMethod = "webhook"
	DeliveryNATS      DeliveryMethod = "nats"
)

type Subscription struct {
	ID          string         `json:"id"`
	AgentID     string         `json:"agent_id"`
	Filter      SignalFilter   `json:"filter"`
	Delivery    DeliveryMethod `json:"delivery"`
	WebhookURL  *string        `json:"webhook_url,omitempty"`
	NATSSubject *string        `json:"nats_subject,omitempty"`
	Active      bool           `json:"active"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}
