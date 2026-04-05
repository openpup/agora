package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/openpup/agora/internal/domain"
	"github.com/openpup/agora/internal/pubsub"
	"github.com/openpup/agora/internal/repository"
)

type VerificationService struct {
	signals    repository.SignalRepository
	marketData repository.MarketDataRepository
	publisher  SignalPublisher
}

func NewVerificationService(signals repository.SignalRepository, marketData repository.MarketDataRepository, publisher SignalPublisher) *VerificationService {
	return &VerificationService{signals: signals, marketData: marketData, publisher: publisher}
}

func (s *VerificationService) VerifyExpired(ctx context.Context, cutoff time.Time) error {
	pending, err := s.signals.ListPendingVerification(ctx, cutoff)
	if err != nil {
		return fmt.Errorf("verification_service.VerifyExpired list pending: %w", err)
	}
	for _, candidate := range pending {
		startPrice, err := s.marketData.GetClosestPrice(ctx, candidate.Market, candidate.Ticker, ">=", candidate.CreatedAt.Format(time.RFC3339))
		if err != nil {
			return fmt.Errorf("verification_service.VerifyExpired start price: %w", err)
		}
		endPrice, err := s.marketData.GetClosestPrice(ctx, candidate.Market, candidate.Ticker, ">=", candidate.ExpiresAt.Format(time.RFC3339))
		if err != nil {
			return fmt.Errorf("verification_service.VerifyExpired end price: %w", err)
		}
		verified := isCorrect(candidate.Direction, startPrice, endPrice)
		detail := map[string]any{
			"start_price": startPrice,
			"end_price":   endPrice,
			"delta":       endPrice - startPrice,
		}
		if err := s.signals.MarkVerified(ctx, candidate.SignalID, verified, detail, cutoff); err != nil {
			return fmt.Errorf("verification_service.VerifyExpired mark verified: %w", err)
		}
		payload, _ := jsonMarshal(map[string]any{
			"signal_id": candidate.SignalID,
			"verified":  verified,
			"detail":    detail,
		})
		if err := s.publisher.PublishSignal(ctx, pubsub.SignalVerifiedSubject(candidate.Market, candidate.Ticker), payload); err != nil {
			return fmt.Errorf("verification_service.VerifyExpired publish: %w", err)
		}
	}
	return nil
}

func isCorrect(direction domain.Direction, startPrice, endPrice float64) bool {
	switch direction {
	case domain.DirectionBullish:
		return endPrice > startPrice
	case domain.DirectionBearish:
		return endPrice < startPrice
	default:
		return endPrice == startPrice
	}
}

func jsonMarshal(v any) ([]byte, error) {
	return json.Marshal(v)
}
