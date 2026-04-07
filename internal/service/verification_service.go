package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/openpup/agora/internal/core"
	"github.com/openpup/agora/internal/domainplugin"
	"github.com/openpup/agora/internal/pubsub"
	"github.com/openpup/agora/internal/repository"
)

type VerificationService struct {
	signals  repository.SignalRepository
	registry *domainplugin.Registry
	publisher SignalPublisher
}

func NewVerificationService(signals repository.SignalRepository, registry *domainplugin.Registry, publisher SignalPublisher) *VerificationService {
	return &VerificationService{signals: signals, registry: registry, publisher: publisher}
}

func (s *VerificationService) VerifyExpired(ctx context.Context, cutoff time.Time) error {
	pending, err := s.signals.ListPendingVerification(ctx, cutoff)
	if err != nil {
		return fmt.Errorf("verification_service.VerifyExpired list pending: %w", err)
	}
	for _, candidate := range pending {
		plugin, err := s.registry.Get(candidate.Domain)
		if err != nil {
			// No plugin for this domain, skip
			continue
		}
		signal := core.Signal{
			ID:      candidate.SignalID,
			Domain:  candidate.Domain,
			Kind:    candidate.Kind,
			Claim: core.Claim{
				Structured:   candidate.Structured,
				Confidence:   candidate.Confidence,
				VerifiableBy: &candidate.VerifiableBy,
			},
			CreatedAt: candidate.CreatedAt,
		}
		verified, detail, err := plugin.Verify(ctx, signal)
		if err != nil {
			return fmt.Errorf("verification_service.VerifyExpired verify %s: %w", candidate.SignalID, err)
		}
		if err := s.signals.MarkVerified(ctx, candidate.SignalID, verified, detail, cutoff); err != nil {
			return fmt.Errorf("verification_service.VerifyExpired mark verified: %w", err)
		}
		payload, _ := json.Marshal(map[string]any{
			"signal_id": candidate.SignalID,
			"verified":  verified,
			"detail":    detail,
		})
		if err := s.publisher.PublishSignal(ctx, pubsub.SignalVerifiedSubject(candidate.Domain), payload); err != nil {
			return fmt.Errorf("verification_service.VerifyExpired publish: %w", err)
		}
	}
	return nil
}
