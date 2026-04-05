package worker

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/openpup/agora/internal/domain"
	"github.com/openpup/agora/internal/pubsub"
	"github.com/openpup/agora/internal/service"
)

type MarketDataSync struct {
	source    service.MarketDataSource
	service   *service.MarketDataService
	publisher *pubsub.Publisher
	markets   map[string]struct {
		Market   domain.Market
		Interval time.Duration
	}
	logger *zap.Logger
}

func NewMarketDataSync(source service.MarketDataSource, service *service.MarketDataService, publisher *pubsub.Publisher, logger *zap.Logger) *MarketDataSync {
	return &MarketDataSync{source: source, service: service, publisher: publisher, logger: logger}
}

func (w *MarketDataSync) Run(ctx context.Context) {
	_ = ctx
}
