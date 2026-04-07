package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/openpup/agora/internal/repository"
)

type ConsensusResult struct {
	Domain    string         `json:"domain"`
	Consensus map[string]any `json:"consensus"`
	UpdatedAt time.Time      `json:"updated_at"`
}

type ConsensusService struct {
	repo  repository.SignalRepository
	cache *redis.Client
}

func NewConsensusService(repo repository.SignalRepository, cache *redis.Client) *ConsensusService {
	return &ConsensusService{repo: repo, cache: cache}
}

func (s *ConsensusService) GetConsensus(ctx context.Context, domain string, horizon *time.Duration) (*ConsensusResult, error) {
	cacheKey := fmt.Sprintf("consensus:%s", domain)
	if cached, err := s.cache.Get(ctx, cacheKey).Bytes(); err == nil {
		var result ConsensusResult
		if json.Unmarshal(cached, &result) == nil {
			return &result, nil
		}
	}
	rows, err := s.repo.ListConsensusRows(ctx, domain, horizon)
	if err != nil {
		return nil, fmt.Errorf("consensus_service.GetConsensus rows: %w", err)
	}
	result := &ConsensusResult{
		Domain: domain,
		Consensus: map[string]any{
			"signal_count": len(rows),
		},
		UpdatedAt: time.Now().UTC(),
	}
	payload, _ := json.Marshal(result)
	_ = s.cache.Set(ctx, cacheKey, payload, 30*time.Second).Err()
	return result, nil
}

func (s *ConsensusService) GetOverview(ctx context.Context, domain string) (*ConsensusResult, error) {
	rows, err := s.repo.ListOverviewRows(ctx, domain, 10)
	if err != nil {
		return nil, fmt.Errorf("consensus_service.GetOverview: %w", err)
	}
	result := &ConsensusResult{
		Domain: domain,
		Consensus: map[string]any{
			"signal_count": len(rows),
		},
		UpdatedAt: time.Now().UTC(),
	}
	return result, nil
}
