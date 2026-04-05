package handler

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type HealthHandler struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

func NewHealthHandler(db *pgxpool.Pool, redis *redis.Client) *HealthHandler {
	return &HealthHandler{db: db, redis: redis}
}

func (h *HealthHandler) Healthz(ctx context.Context, c *app.RequestContext) {
	if err := h.db.Ping(ctx); err != nil {
		writeError(c, 503, "DB_UNAVAILABLE", err.Error())
		return
	}
	if err := h.redis.Ping(ctx).Err(); err != nil {
		writeError(c, 503, "REDIS_UNAVAILABLE", err.Error())
		return
	}
	c.JSON(200, map[string]string{"status": "ok"})
}
