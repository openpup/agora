package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openpup/agora/internal/domain"
)

type SubscriptionRepository interface {
	Create(context.Context, *domain.Subscription) error
	ListByAgent(context.Context, string) ([]domain.Subscription, error)
	ListActive(context.Context) ([]domain.Subscription, error)
	Delete(context.Context, string, string) error
}

type PGSubscriptionRepository struct {
	pool *pgxpool.Pool
}

func NewPGSubscriptionRepository(pool *pgxpool.Pool) *PGSubscriptionRepository {
	return &PGSubscriptionRepository{pool: pool}
}

func (r *PGSubscriptionRepository) Create(ctx context.Context, sub *domain.Subscription) error {
	filter, _ := json.Marshal(sub.Filter)
	_, err := r.pool.Exec(ctx, `
		INSERT INTO subscriptions (id, agent_id, filter, delivery, webhook_url, nats_subject, active, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
	`, sub.ID, sub.AgentID, filter, sub.Delivery, sub.WebhookURL, sub.NATSSubject, sub.Active, sub.CreatedAt, sub.UpdatedAt)
	if err != nil {
		return fmt.Errorf("subscription_repo.Create: %w", err)
	}
	return nil
}

func (r *PGSubscriptionRepository) ListByAgent(ctx context.Context, agentID string) ([]domain.Subscription, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, agent_id, filter, delivery, webhook_url, nats_subject, active, created_at, updated_at
		FROM subscriptions WHERE agent_id=$1 ORDER BY created_at DESC
	`, agentID)
	if err != nil {
		return nil, fmt.Errorf("subscription_repo.ListByAgent: %w", err)
	}
	defer rows.Close()
	return scanSubscriptions(rows)
}

func (r *PGSubscriptionRepository) ListActive(ctx context.Context) ([]domain.Subscription, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, agent_id, filter, delivery, webhook_url, nats_subject, active, created_at, updated_at
		FROM subscriptions WHERE active=TRUE ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("subscription_repo.ListActive: %w", err)
	}
	defer rows.Close()
	return scanSubscriptions(rows)
}

func (r *PGSubscriptionRepository) Delete(ctx context.Context, id, agentID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM subscriptions WHERE id=$1 AND agent_id=$2`, id, agentID)
	if err != nil {
		return fmt.Errorf("subscription_repo.Delete: %w", err)
	}
	return nil
}

type pgxRows interface {
	Next() bool
	Scan(...any) error
}

func scanSubscriptions(rows pgxRows) ([]domain.Subscription, error) {
	var out []domain.Subscription
	for rows.Next() {
		var sub domain.Subscription
		var filter []byte
		if err := rows.Scan(&sub.ID, &sub.AgentID, &filter, &sub.Delivery, &sub.WebhookURL, &sub.NATSSubject, &sub.Active, &sub.CreatedAt, &sub.UpdatedAt); err != nil {
			return nil, fmt.Errorf("subscription_repo.scanSubscriptions scan: %w", err)
		}
		if err := json.Unmarshal(filter, &sub.Filter); err != nil {
			return nil, fmt.Errorf("subscription_repo.scanSubscriptions filter: %w", err)
		}
		out = append(out, sub)
	}
	return out, nil
}
