package service

import (
	"context"
	"fmt"
	"time"

	"github.com/openpup/agora/internal/domain"
	"github.com/openpup/agora/internal/repository"
)

type MarketDataSource interface {
	FetchCandles(context.Context, domain.Market, string, time.Time, time.Time, string) ([]domain.Candle, error)
}

type MarketDataService struct {
	repo repository.MarketDataRepository
}

func NewMarketDataService(repo repository.MarketDataRepository) *MarketDataService {
	return &MarketDataService{repo: repo}
}

func (s *MarketDataService) List(ctx context.Context, query domain.MarketDataQuery) ([]domain.Candle, error) {
	candles, err := s.repo.ListCandles(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("market_data_service.List: %w", err)
	}
	return candles, nil
}

func (s *MarketDataService) Upsert(ctx context.Context, candles []domain.Candle) error {
	if err := s.repo.UpsertCandles(ctx, candles); err != nil {
		return fmt.Errorf("market_data_service.Upsert: %w", err)
	}
	return nil
}
