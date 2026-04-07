package finance

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Candle struct {
	Time     time.Time      `json:"time"`
	Ticker   string         `json:"ticker"`
	Market   string         `json:"market"`
	Open     float64        `json:"open"`
	High     float64        `json:"high"`
	Low      float64        `json:"low"`
	Close    float64        `json:"close"`
	Volume   float64        `json:"volume"`
	Metadata map[string]any `json:"metadata"`
}

type MarketDataQuery struct {
	Ticker   string
	Market   string
	Interval string
	From     *time.Time
	To       *time.Time
}

type MarketDataRepo struct {
	pool *pgxpool.Pool
}

func NewMarketDataRepo(pool *pgxpool.Pool) *MarketDataRepo {
	return &MarketDataRepo{pool: pool}
}

func (r *MarketDataRepo) UpsertCandles(ctx context.Context, candles []Candle) error {
	for _, candle := range candles {
		meta, _ := json.Marshal(candle.Metadata)
		if _, err := r.pool.Exec(ctx, `
			INSERT INTO finance_market_data (time, ticker, market, open, high, low, close, volume, metadata)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
			ON CONFLICT (time, ticker, market) DO UPDATE SET
				open=EXCLUDED.open, high=EXCLUDED.high, low=EXCLUDED.low, close=EXCLUDED.close, volume=EXCLUDED.volume, metadata=EXCLUDED.metadata
		`, candle.Time, candle.Ticker, candle.Market, candle.Open, candle.High, candle.Low, candle.Close, candle.Volume, meta); err != nil {
			return fmt.Errorf("finance.MarketDataRepo.UpsertCandles: %w", err)
		}
	}
	return nil
}

func (r *MarketDataRepo) ListCandles(ctx context.Context, query MarketDataQuery) ([]Candle, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT time, ticker, market, open, high, low, close, volume, metadata
		FROM finance_market_data
		WHERE ticker=$1 AND market=$2
		  AND ($3::timestamptz IS NULL OR time >= $3)
		  AND ($4::timestamptz IS NULL OR time <= $4)
		ORDER BY time ASC
	`, query.Ticker, query.Market, query.From, query.To)
	if err != nil {
		return nil, fmt.Errorf("finance.MarketDataRepo.ListCandles: %w", err)
	}
	defer rows.Close()
	var out []Candle
	for rows.Next() {
		var candle Candle
		var meta []byte
		if err := rows.Scan(&candle.Time, &candle.Ticker, &candle.Market, &candle.Open, &candle.High, &candle.Low, &candle.Close, &candle.Volume, &meta); err != nil {
			return nil, fmt.Errorf("finance.MarketDataRepo.ListCandles scan: %w", err)
		}
		_ = json.Unmarshal(meta, &candle.Metadata)
		out = append(out, candle)
	}
	return out, nil
}

func (r *MarketDataRepo) GetClosestPrice(ctx context.Context, ticker string, op string, ref time.Time) (float64, error) {
	query := fmt.Sprintf(`
		SELECT close FROM finance_market_data
		WHERE ticker=$1 AND time %s $2::timestamptz
		ORDER BY ABS(EXTRACT(EPOCH FROM (time - $2::timestamptz))) ASC
		LIMIT 1
	`, op)
	var closePrice float64
	if err := r.pool.QueryRow(ctx, query, ticker, ref).Scan(&closePrice); err != nil {
		return 0, fmt.Errorf("finance.MarketDataRepo.GetClosestPrice: %w", err)
	}
	return closePrice, nil
}
