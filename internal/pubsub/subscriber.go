package pubsub

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go"
)

type Subscriber struct {
	conn *nats.Conn
}

func NewSubscriber(conn *nats.Conn) *Subscriber {
	return &Subscriber{conn: conn}
}

func (s *Subscriber) Subscribe(ctx context.Context, subject string, handler func(*nats.Msg)) error {
	sub, err := s.conn.Subscribe(subject, handler)
	if err != nil {
		return fmt.Errorf("pubsub.Subscriber.Subscribe: %w", err)
	}
	go func() {
		<-ctx.Done()
		_ = sub.Unsubscribe()
	}()
	return nil
}
