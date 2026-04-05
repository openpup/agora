package handler

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/hertz-contrib/websocket"

	"github.com/openpup/agora/internal/domain"
	"github.com/openpup/agora/internal/service"
)

type WSHandler struct {
	auth          *service.AuthService
	subscriptions *service.SubscriptionService
	upgrader      websocket.HertzUpgrader
	mu            sync.RWMutex
	connections   map[string]*websocket.Conn
}

func NewWSHandler(auth *service.AuthService, subscriptions *service.SubscriptionService) *WSHandler {
	return &WSHandler{
		auth:          auth,
		subscriptions: subscriptions,
		upgrader: websocket.HertzUpgrader{
			CheckOrigin: func(_ *app.RequestContext) bool { return true },
		},
		connections: map[string]*websocket.Conn{},
	}
}

func (h *WSHandler) Stream(ctx context.Context, c *app.RequestContext) {
	apiKey := string(c.Request.Header.Peek("X-Agent-Key"))
	agent, err := h.auth.Authenticate(ctx, apiKey)
	if err != nil {
		writeError(c, 401, "AUTH_INVALID", err.Error())
		return
	}
	err = h.upgrader.Upgrade(c, func(conn *websocket.Conn) {
		h.mu.Lock()
		h.connections[agent.ID] = conn
		h.mu.Unlock()
		defer func() {
			h.mu.Lock()
			delete(h.connections, agent.ID)
			h.mu.Unlock()
			_ = conn.Close()
		}()
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if string(message) == `{"action":"ping"}` || string(message) == `{"action": "ping"}` {
				_ = conn.WriteJSON(map[string]any{"type": "pong"})
			}
		}
	})
	if err != nil {
		writeError(c, 500, "WS_UPGRADE_FAILED", err.Error())
	}
}

func (h *WSHandler) PushMatching(ctx context.Context, signal domain.Signal) error {
	matches, err := h.subscriptions.Match(ctx, signal)
	if err != nil {
		return err
	}
	payload, _ := json.Marshal(map[string]any{"type": "signal", "data": signal})
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, sub := range matches {
		if conn, ok := h.connections[sub.AgentID]; ok {
			if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
				return err
			}
		}
	}
	return nil
}
