package mq

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go"

	"github.com/openpup/agora/internal/config"
)

func NewNATS(ctx context.Context, cfg config.NATSConfig) (*nats.Conn, nats.JetStreamContext, error) {
	nc, err := nats.Connect(cfg.URL)
	if err != nil {
		return nil, nil, fmt.Errorf("mq.NewNATS connect: %w", err)
	}
	select {
	case <-ctx.Done():
		nc.Close()
		return nil, nil, ctx.Err()
	default:
	}
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, nil, fmt.Errorf("mq.NewNATS jetstream: %w", err)
	}
	return nc, js, nil
}
