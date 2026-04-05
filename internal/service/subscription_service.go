package service

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"

	"github.com/openpup/agora/internal/domain"
	pkgerrors "github.com/openpup/agora/internal/pkg/errors"
	"github.com/openpup/agora/internal/repository"
)

type SubscriptionService struct {
	repo repository.SubscriptionRepository
}

func NewSubscriptionService(repo repository.SubscriptionRepository) *SubscriptionService {
	return &SubscriptionService{repo: repo}
}

func (s *SubscriptionService) Create(ctx context.Context, agentID string, filter domain.SignalFilter, delivery domain.DeliveryMethod, webhookURL, natsSubject *string) (*domain.Subscription, error) {
	if !filter.Market.Valid() {
		return nil, fmt.Errorf("subscription_service.Create market: %w", pkgerrors.ErrInvalidInput)
	}
	if delivery == domain.DeliveryWebhook && webhookURL == nil {
		return nil, fmt.Errorf("subscription_service.Create webhook: %w", pkgerrors.ErrInvalidInput)
	}
	if delivery == domain.DeliveryNATS && natsSubject == nil {
		return nil, fmt.Errorf("subscription_service.Create nats subject: %w", pkgerrors.ErrInvalidInput)
	}
	now := time.Now().UTC()
	sub := &domain.Subscription{
		ID:          uuid.NewString(),
		AgentID:     agentID,
		Filter:      filter,
		Delivery:    delivery,
		WebhookURL:  webhookURL,
		NATSSubject: natsSubject,
		Active:      true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.repo.Create(ctx, sub); err != nil {
		return nil, fmt.Errorf("subscription_service.Create persist: %w", err)
	}
	return sub, nil
}

func (s *SubscriptionService) List(ctx context.Context, agentID string) ([]domain.Subscription, error) {
	subs, err := s.repo.ListByAgent(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("subscription_service.List: %w", err)
	}
	return subs, nil
}

func (s *SubscriptionService) Delete(ctx context.Context, id, agentID string) error {
	if err := s.repo.Delete(ctx, id, agentID); err != nil {
		return fmt.Errorf("subscription_service.Delete: %w", err)
	}
	return nil
}

func (s *SubscriptionService) Match(ctx context.Context, signal domain.Signal) ([]domain.Subscription, error) {
	subs, err := s.repo.ListActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("subscription_service.Match: %w", err)
	}
	var out []domain.Subscription
	for _, sub := range subs {
		if sub.Filter.Market != signal.Market {
			continue
		}
		if len(sub.Filter.Tickers) > 0 && (signal.Ticker == nil || !slices.Contains(sub.Filter.Tickers, *signal.Ticker)) {
			continue
		}
		if len(sub.Filter.SignalTypes) > 0 && !slices.Contains(sub.Filter.SignalTypes, signal.SignalType) {
			continue
		}
		if sub.Filter.MinConfidence != nil && (signal.Confidence == nil || *signal.Confidence < *sub.Filter.MinConfidence) {
			continue
		}
		out = append(out, sub)
	}
	return out, nil
}
