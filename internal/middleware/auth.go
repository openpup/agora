package middleware

import (
	"context"
	"fmt"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/openpup/agora/internal/service"
)

const AgentIDKey = "agent_id"

func Auth(authService *service.AuthService, headerName string) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		apiKey := string(c.Request.Header.Peek(headerName))
		if apiKey == "" {
			writeError(c, 401, "AUTH_MISSING", "missing agent key")
			c.Abort()
			return
		}
		agent, err := authService.Authenticate(ctx, apiKey)
		if err != nil {
			writeError(c, 401, "AUTH_INVALID", fmt.Sprintf("invalid agent key: %v", err))
			c.Abort()
			return
		}
		c.Set(AgentIDKey, agent.ID)
		c.Next(ctx)
	}
}
