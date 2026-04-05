package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/openpup/agora/internal/domain"
	"github.com/openpup/agora/internal/repository"
)

type TrustService struct {
	agents  repository.AgentRepository
	signals repository.SignalRepository
}

func NewTrustService(agents repository.AgentRepository, signals repository.SignalRepository) *TrustService {
	return &TrustService{agents: agents, signals: signals}
}

func (s *TrustService) Recalculate(ctx context.Context) error {
	stats, err := s.signals.ListAgentMarketStats(ctx)
	if err != nil {
		return fmt.Errorf("trust_service.Recalculate stats: %w", err)
	}
	maxTotal := 1
	for _, stat := range stats {
		if stat.TotalPredictions > maxTotal {
			maxTotal = stat.TotalPredictions
		}
	}
	for _, stat := range stats {
		accuracy := 0.0
		if stat.TotalPredictions > 0 {
			accuracy = float64(stat.CorrectPredictions) / float64(stat.TotalPredictions)
		}
		trust := 0.5
		if stat.TotalPredictions > 0 {
			trust = accuracy * math.Log(float64(stat.TotalPredictions)+1) / math.Log(float64(maxTotal)+1)
		}
		rec := domain.AgentTrackRecord{
			AgentID:            stat.AgentID,
			Market:             stat.Market,
			TotalPredictions:   stat.TotalPredictions,
			CorrectPredictions: stat.CorrectPredictions,
			Accuracy:           accuracy,
			AvgConfidence:      stat.AvgConfidence,
			LastCalculatedAt:   time.Now().UTC(),
		}
		if err := s.agents.UpsertTrackRecord(ctx, rec); err != nil {
			return fmt.Errorf("trust_service.Recalculate upsert track record: %w", err)
		}
		if err := s.agents.UpdateTrustScore(ctx, stat.AgentID, trust); err != nil {
			return fmt.Errorf("trust_service.Recalculate update trust: %w", err)
		}
	}
	return nil
}
