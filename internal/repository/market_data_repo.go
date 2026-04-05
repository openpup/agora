package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openpup/agora/internal/domain"
)

type MarketDataRepository interface {
	UpsertCandles(context.Context, []domain.Candle) error
	ListCandles(context.Context, domain.MarketDataQuery) ([]domain.Candle, error)
	GetClosestPrice(context.Context, domain.Market, string, string, string) (float64, error)
}

type PGMarketDataRepository struct {
	pool *pgxpool.Pool
}

func NewPGMarketDataRepository(pool *pgxpool.Pool) *PGMarketDataRepository {
	return &PGMarketDataRepository{pool: pool}
}

func (r *PGMarketDataRepository) UpsertCandles(ctx context.Context, candles []domain.Candle) error {
	for _, candle := range candles {
		meta, _ := json.Marshal(candle.Metadata)
		if _, err := r.pool.Exec(ctx, `
			INSERT INTO market_data (time, ticker, market, open, high, low, close, volume, metadata)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
			ON CONFLICT (time, ticker, market) DO UPDATE SET
				open=EXCLUDED.open, high=EXCLUDED.high, low=EXCLUDED.low, close=EXCLUDED.close, volume=EXCLUDED.volume, metadata=EXCLUDED.metadata
		`, candle.Time, candle.Ticker, candle.Market, candle.Open, candle.High, candle.Low, candle.Close, candle.Volume, meta); err != nil {
			return fmt.Errorf("market_data_repo.UpsertCandles: %w", err)
		}
	}
	return nil
}

func (r *PGMarketDataRepository) ListCandles(ctx context.Context, query domain.MarketDataQuery) ([]domain.Candle, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT time, ticker, market, open, high, low, close, volume, metadata
		FROM market_data
		WHERE ticker=$1 AND market=$2
		  AND ($3::timestamptz IS NULL OR time >= $3)
		  AND ($4::timestamptz IS NULL OR time <= $4)
		ORDER BY time ASC
	`, query.Ticker, query.Market, query.From, query.To)
	if err != nil {
		return nil, fmt.Errorf("market_data_repo.ListCandles: %w", err)
	}
	defer rows.Close()
	var out []domain.Candle
	for rows.Next() {
		var candle domain.Candle
		var meta []byte
		if err := rows.Scan(&candle.Time, &candle.Ticker, &candle.Market, &candle.Open, &candle.High, &candle.Low, &candle.Close, &candle.Volume, &meta); err != nil {
			return nil, fmt.Errorf("market_data_repo.ListCandles scan: %w", err)
		}
		_ = json.Unmarshal(meta, &candle.Metadata)
		out = append(out, candle)
	}
	return out, nil
}

func (r *PGMarketDataRepository) GetClosestPrice(ctx context.Context, market domain.Market, ticker string, op string, ref string) (float64, error) {
	query := fmt.Sprintf(`
		SELECT close FROM market_data
		WHERE market=$1 AND ticker=$2 AND time %s $3::timestamptz
		ORDER BY ABS(EXTRACT(EPOCH FROM (time - $3::timestamptz))) ASC
		LIMIT 1
	`, op)
	var close float64
	if err := r.pool.QueryRow(ctx, query, market, ticker, ref).Scan(&close); err != nil {
		return 0, fmt.Errorf("market_data_repo.GetClosestPrice: %w", err)
	}
	return close, nil
}
