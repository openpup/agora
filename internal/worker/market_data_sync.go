package worker

import (
	"context"

	"go.uber.org/zap"
)

// MarketDataSync is a placeholder for domain-specific market data synchronization.
// In the new architecture, domain plugins own their own background workers.
type MarketDataSync struct {
	logger *zap.Logger
}

func NewMarketDataSync(logger *zap.Logger) *MarketDataSync {
	return &MarketDataSync{logger: logger}
}

func (w *MarketDataSync) Run(ctx context.Context) {
	_ = ctx
}
