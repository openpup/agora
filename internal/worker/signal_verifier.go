package worker

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/openpup/agora/internal/service"
)

type SignalVerifier struct {
	service  *service.VerificationService
	interval time.Duration
	logger   *zap.Logger
}

func NewSignalVerifier(service *service.VerificationService, interval time.Duration, logger *zap.Logger) *SignalVerifier {
	return &SignalVerifier{service: service, interval: interval, logger: logger}
}

func (w *SignalVerifier) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.service.VerifyExpired(ctx, time.Now().UTC()); err != nil {
				w.logger.Error("signal verification failed", zap.Error(err))
			}
		}
	}
}
