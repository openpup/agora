package worker

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/openpup/agora/internal/service"
)

type TrustCalculator struct {
	service  *service.TrustService
	interval time.Duration
	logger   *zap.Logger
}

func NewTrustCalculator(service *service.TrustService, interval time.Duration, logger *zap.Logger) *TrustCalculator {
	return &TrustCalculator{service: service, interval: interval, logger: logger}
}

func (w *TrustCalculator) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.service.Recalculate(ctx); err != nil {
				w.logger.Error("trust recalculation failed", zap.Error(err))
			}
		}
	}
}
