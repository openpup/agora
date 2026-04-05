package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/openpup/agora/internal/domain"
	"github.com/openpup/agora/internal/repository"
)

type ConsensusDetail struct {
	Ticker               string          `json:"ticker"`
	Market               domain.Market   `json:"market"`
	BullishCount         int             `json:"bullish_count"`
	BearishCount         int             `json:"bearish_count"`
	NeutralCount         int             `json:"neutral_count"`
	AvgBullishConfidence float64         `json:"avg_bullish_confidence"`
	AvgBearishConfidence float64         `json:"avg_bearish_confidence"`
	WeightedConsensus    float64         `json:"weighted_consensus"`
	WeightedDirection    string          `json:"weighted_direction"`
	TopSignals           []SignalSummary `json:"top_signals"`
	UpdatedAt            time.Time       `json:"updated_at"`
}

type SignalSummary struct {
	SignalID  string    `json:"signal_id"`
	AgentID   string    `json:"agent_id"`
	Direction string    `json:"direction"`
	Score     float64   `json:"score"`
	CreatedAt time.Time `json:"created_at"`
}

type ConsensusOverview struct {
	Market      domain.Market     `json:"market"`
	TopBullish  []ConsensusTicker `json:"top_bullish"`
	TopBearish  []ConsensusTicker `json:"top_bearish"`
	MostDebated []ConsensusTicker `json:"most_debated"`
}

type ConsensusTicker struct {
	Ticker            string  `json:"ticker"`
	WeightedConsensus float64 `json:"weighted_consensus"`
	SignalCount       int     `json:"signal_count"`
}

type ConsensusService struct {
	repo  repository.SignalRepository
	cache *redis.Client
}

func NewConsensusService(repo repository.SignalRepository, cache *redis.Client) *ConsensusService {
	return &ConsensusService{repo: repo, cache: cache}
}

func (s *ConsensusService) GetTickerConsensus(ctx context.Context, market domain.Market, ticker string, horizon *time.Duration) (*ConsensusDetail, error) {
	cacheKey := fmt.Sprintf("consensus:%s:%s", market, ticker)
	if cached, err := s.cache.Get(ctx, cacheKey).Bytes(); err == nil {
		var detail ConsensusDetail
		if json.Unmarshal(cached, &detail) == nil {
			return &detail, nil
		}
	}
	rows, err := s.repo.ListConsensusRows(ctx, market, ticker, horizon)
	if err != nil {
		return nil, fmt.Errorf("consensus_service.GetTickerConsensus rows: %w", err)
	}
	detail := computeConsensus(market, ticker, rows)
	payload, _ := json.Marshal(detail)
	_ = s.cache.Set(ctx, cacheKey, payload, 30*time.Second).Err()
	return detail, nil
}

func computeConsensus(market domain.Market, ticker string, rows []repository.ConsensusRow) *ConsensusDetail {
	detail := &ConsensusDetail{
		Ticker:    ticker,
		Market:    market,
		UpdatedAt: time.Now().UTC(),
	}
	var bullSum, bearSum float64
	var bullCount, bearCount int
	for _, row := range rows {
		score := row.Confidence * row.TrustScore
		switch row.Direction {
		case string(domain.DirectionBullish):
			detail.BullishCount++
			bullSum += row.Confidence
			bullCount++
		case string(domain.DirectionBearish):
			detail.BearishCount++
			bearSum += row.Confidence
			bearCount++
		default:
			detail.NeutralCount++
		}
		detail.TopSignals = append(detail.TopSignals, SignalSummary{
			SignalID:  row.SignalID,
			AgentID:   row.AgentID,
			Direction: row.Direction,
			Score:     score,
			CreatedAt: row.CreatedAt,
		})
	}
	if bullCount > 0 {
		detail.AvgBullishConfidence = bullSum / float64(bullCount)
	}
	if bearCount > 0 {
		detail.AvgBearishConfidence = bearSum / float64(bearCount)
	}
	sort.Slice(detail.TopSignals, func(i, j int) bool { return detail.TopSignals[i].Score > detail.TopSignals[j].Score })
	if len(detail.TopSignals) > 5 {
		detail.TopSignals = detail.TopSignals[:5]
	}
	totalWeight := float64(detail.BullishCount + detail.BearishCount + detail.NeutralCount)
	if totalWeight > 0 {
		detail.WeightedConsensus = float64(detail.BullishCount-detail.BearishCount) / totalWeight
	}
	switch {
	case detail.WeightedConsensus > 0:
		detail.WeightedDirection = string(domain.DirectionBullish)
	case detail.WeightedConsensus < 0:
		detail.WeightedDirection = string(domain.DirectionBearish)
	default:
		detail.WeightedDirection = string(domain.DirectionNeutral)
	}
	return detail
}

func (s *ConsensusService) GetOverview(ctx context.Context, market domain.Market) (*ConsensusOverview, error) {
	rows, err := s.repo.ListOverviewRows(ctx, market, 10)
	if err != nil {
		return nil, fmt.Errorf("consensus_service.GetOverview: %w", err)
	}
	buckets := map[string][]repository.ConsensusRow{}
	for _, row := range rows {
		buckets[row.Ticker] = append(buckets[row.Ticker], row)
	}
	var tickers []ConsensusTicker
	for ticker, grouped := range buckets {
		detail := computeConsensus(market, ticker, grouped)
		tickers = append(tickers, ConsensusTicker{
			Ticker: ticker, WeightedConsensus: detail.WeightedConsensus, SignalCount: len(grouped),
		})
	}
	sort.Slice(tickers, func(i, j int) bool { return tickers[i].WeightedConsensus > tickers[j].WeightedConsensus })
	overview := &ConsensusOverview{Market: market}
	for _, item := range tickers {
		if item.WeightedConsensus >= 0 && len(overview.TopBullish) < 10 {
			overview.TopBullish = append(overview.TopBullish, item)
		}
		if item.WeightedConsensus < 0 && len(overview.TopBearish) < 10 {
			overview.TopBearish = append(overview.TopBearish, item)
		}
	}
	sort.Slice(tickers, func(i, j int) bool { return tickers[i].SignalCount > tickers[j].SignalCount })
	if len(tickers) > 10 {
		overview.MostDebated = tickers[:10]
	} else {
		overview.MostDebated = tickers
	}
	return overview, nil
}
