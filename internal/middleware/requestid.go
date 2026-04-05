package middleware

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/google/uuid"
)

const RequestIDKey = "request_id"

func RequestID() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		requestID := string(c.Request.Header.Peek("X-Request-Id"))
		if requestID == "" {
			requestID = "req_" + uuid.NewString()
		}
		c.Set(RequestIDKey, requestID)
		c.Response.Header.Set("X-Request-Id", requestID)
		c.Next(ctx)
	}
}
