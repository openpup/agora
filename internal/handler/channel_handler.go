package handler

import (
	"context"
	"strconv"
	"time"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/openpup/agora/internal/core"
	"github.com/openpup/agora/internal/repository"
	"github.com/openpup/agora/internal/service"
)

type ChannelHandler struct {
	service     *service.ChannelService
	idempotency *service.IdempotencyService
}

func NewChannelHandler(service *service.ChannelService, idempotency *service.IdempotencyService) *ChannelHandler {
	return &ChannelHandler{service: service, idempotency: idempotency}
}

type createChannelRequest struct {
	Name        string         `json:"name"`
	Slug        string         `json:"slug"`
	Domain      string         `json:"domain"`
	Kind        string         `json:"kind"`
	Description string         `json:"description"`
	Meta        map[string]any `json:"meta"`
}

type createChannelMessageRequest struct {
	Kind   string          `json:"kind"`
	Intent string          `json:"intent"`
	Body   string          `json:"body"`
	Refs   []core.CrossRef `json:"refs"`
	Meta   map[string]any  `json:"meta"`
}

func (h *ChannelHandler) CreateChannel(ctx context.Context, c *app.RequestContext) {
	key := idempotencyKey(c, agentIDFromContext(ctx, c))
	if serveIdempotentResponse(ctx, c, h.idempotency, key) {
		return
	}
	var req createChannelRequest
	if err := c.BindAndValidate(&req); err != nil {
		writeError(c, 400, "CHANNEL_INVALID", err.Error())
		return
	}
	channel, err := h.service.CreateChannel(ctx, service.CreateChannelInput{
		Name:        req.Name,
		Slug:        req.Slug,
		Domain:      req.Domain,
		Kind:        core.ChannelKind(req.Kind),
		Description: req.Description,
		Meta:        req.Meta,
	})
	if err != nil {
		writeError(c, 400, "CHANNEL_CREATE_FAILED", err.Error())
		return
	}
	response := map[string]any{"channel": channel}
	if err := storeIdempotentResponse(ctx, h.idempotency, key, response); err != nil {
		writeError(c, 500, "IDEMPOTENCY_STORE_FAILED", err.Error())
		return
	}
	c.JSON(201, response)
}

func (h *ChannelHandler) ListChannels(ctx context.Context, c *app.RequestContext) {
	limit := parseBoundedLimit(c.Query("limit"), 50, 100)
	channels, err := h.service.ListChannels(ctx, repository.ListChannelsParams{
		Domain: c.Query("domain"),
		Limit:  limit,
	})
	if err != nil {
		writeError(c, 500, "CHANNEL_LIST_FAILED", err.Error())
		return
	}
	c.JSON(200, map[string]any{"channels": channels})
}

func (h *ChannelHandler) CreateMessage(ctx context.Context, c *app.RequestContext) {
	key := idempotencyKey(c, agentIDFromContext(ctx, c))
	if serveIdempotentResponse(ctx, c, h.idempotency, key) {
		return
	}
	var req createChannelMessageRequest
	if err := c.BindAndValidate(&req); err != nil {
		writeError(c, 400, "CHANNEL_MESSAGE_INVALID", err.Error())
		return
	}
	message, err := h.service.CreateMessage(ctx, service.CreateChannelMessageInput{
		ChannelID: c.Param("id"),
		AgentID:   agentIDFromContext(ctx, c),
		Kind:      core.ChannelMessageKind(req.Kind),
		Intent:    req.Intent,
		Body:      req.Body,
		Refs:      req.Refs,
		Meta:      req.Meta,
	})
	if err != nil {
		writeError(c, 400, "CHANNEL_MESSAGE_CREATE_FAILED", err.Error())
		return
	}
	response := map[string]any{"message": message}
	if err := storeIdempotentResponse(ctx, h.idempotency, key, response); err != nil {
		writeError(c, 500, "IDEMPOTENCY_STORE_FAILED", err.Error())
		return
	}
	c.JSON(201, response)
}

func (h *ChannelHandler) ListMessages(ctx context.Context, c *app.RequestContext) {
	var before *time.Time
	if raw := c.Query("before"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			writeError(c, 400, "CHANNEL_MESSAGE_INVALID_BEFORE", err.Error())
			return
		}
		before = &parsed
	}
	messages, err := h.service.ListMessages(ctx, repository.ListChannelMessagesParams{
		ChannelID: c.Param("id"),
		Before:    before,
		Limit:     parseBoundedLimit(c.Query("limit"), 100, 200),
	})
	if err != nil {
		writeError(c, 500, "CHANNEL_MESSAGE_LIST_FAILED", err.Error())
		return
	}
	c.JSON(200, map[string]any{"messages": messages})
}

func parseBoundedLimit(raw string, fallback, max int) int {
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		return fallback
	}
	if parsed > max {
		return max
	}
	return parsed
}
