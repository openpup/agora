package pubsub

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

type Publisher struct {
	js nats.JetStreamContext
}

func NewPublisher(js nats.JetStreamContext) *Publisher {
	return &Publisher{js: js}
}

func (p *Publisher) EnsureStreams() error {
	if _, err := p.js.AddStream(&nats.StreamConfig{
		Name:      "SIGNALS",
		Subjects:  []string{"signals.*.*.*", "agents.trust.updated.*"},
		Retention: nats.LimitsPolicy,
		MaxAge:    30 * 24 * time.Hour,
	}); err != nil && err != nats.ErrStreamNameAlreadyInUse {
		return fmt.Errorf("pubsub.Publisher.EnsureStreams signals: %w", err)
	}
	if _, err := p.js.AddStream(&nats.StreamConfig{
		Name:      "MARKET_DATA",
		Subjects:  []string{"market.data.*.*"},
		Retention: nats.LimitsPolicy,
		MaxAge:    7 * 24 * time.Hour,
	}); err != nil && err != nats.ErrStreamNameAlreadyInUse {
		return fmt.Errorf("pubsub.Publisher.EnsureStreams market data: %w", err)
	}
	return nil
}

func (p *Publisher) PublishSignal(ctx context.Context, subject string, payload []byte) error {
	if _, err := p.js.PublishMsg(&nats.Msg{Subject: subject, Data: payload}); err != nil {
		return fmt.Errorf("pubsub.Publisher.PublishSignal: %w", err)
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}
