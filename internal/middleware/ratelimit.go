package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/redis/go-redis/v9"
)

func RateLimit(redis *redis.Client, perMinute int) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		agentIDValue, ok := c.Get(AgentIDKey)
		if !ok {
			c.Next(ctx)
			return
		}
		agentID, _ := agentIDValue.(string)
		key := fmt.Sprintf("ratelimit:%s:%s", agentID, time.Now().UTC().Format("200601021504"))
		count, err := redis.Incr(ctx, key).Result()
		if err != nil {
			writeError(c, 500, "RATE_LIMIT_ERROR", "failed to check rate limit")
			c.Abort()
			return
		}
		if count == 1 {
			_ = redis.Expire(ctx, key, time.Minute).Err()
		}
		if count > int64(perMinute) {
			writeError(c, 429, "RATE_LIMIT_EXCEEDED", "rate limit exceeded")
			c.Abort()
			return
		}
		c.Next(ctx)
	}
}
