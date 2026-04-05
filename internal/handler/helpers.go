package handler

import (
	"context"
	"encoding/json"
	"time"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/openpup/agora/internal/middleware"
	pkgerrors "github.com/openpup/agora/internal/pkg/errors"
	"github.com/openpup/agora/internal/service"
)

func requestIDFromContext(c *app.RequestContext) string {
	value, ok := c.Get(middleware.RequestIDKey)
	if !ok {
		return ""
	}
	requestID, _ := value.(string)
	return requestID
}

func writeError(c *app.RequestContext, status int, code, message string) {
	c.JSON(status, pkgerrors.ErrorResponse{
		Error: pkgerrors.ErrorBody{
			Code:      code,
			Message:   message,
			RequestID: requestIDFromContext(c),
		},
	})
}

func parseOptionalTime(raw string) (*time.Time, error) {
	if raw == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func agentIDFromContext(_ context.Context, c *app.RequestContext) string {
	value, _ := c.Get(middleware.AgentIDKey)
	agentID, _ := value.(string)
	return agentID
}

func idempotencyKey(c *app.RequestContext, actorID string) string {
	value := string(c.Request.Header.Peek("Idempotency-Key"))
	if value == "" {
		return ""
	}
	return actorID + ":" + c.FullPath() + ":" + value
}

func serveIdempotentResponse(ctx context.Context, c *app.RequestContext, svc *service.IdempotencyService, key string) bool {
	if svc == nil || key == "" {
		return false
	}
	payload, found, err := svc.Get(ctx, key)
	if err != nil || !found {
		return false
	}
	c.Response.Header.SetContentType("application/json")
	c.SetStatusCode(201)
	c.Write(payload)
	return true
}

func storeIdempotentResponse(ctx context.Context, svc *service.IdempotencyService, key string, payload any) error {
	if svc == nil || key == "" {
		return nil
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return svc.Store(ctx, key, raw)
}
