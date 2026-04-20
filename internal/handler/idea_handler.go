package handler

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/openpup/agora/internal/core"
	"github.com/openpup/agora/internal/repository"
	"github.com/openpup/agora/internal/service"
)

type IdeaHandler struct {
	service     *service.IdeaService
	idempotency *service.IdempotencyService
}

func NewIdeaHandler(service *service.IdeaService, idempotency *service.IdempotencyService) *IdeaHandler {
	return &IdeaHandler{service: service, idempotency: idempotency}
}

type createIdeaRequest struct {
	ChannelID      *string        `json:"channel_id"`
	SourceSignalID *string        `json:"source_signal_id"`
	Domain         string         `json:"domain"`
	Title          string         `json:"title"`
	Summary        string         `json:"summary"`
	Status         string         `json:"status"`
	StanceSummary  map[string]any `json:"stance_summary"`
	Meta           map[string]any `json:"meta"`
}

func (h *IdeaHandler) Create(ctx context.Context, c *app.RequestContext) {
	key := idempotencyKey(c, agentIDFromContext(ctx, c))
	if serveIdempotentResponse(ctx, c, h.idempotency, key) {
		return
	}
	var req createIdeaRequest
	if err := c.BindAndValidate(&req); err != nil {
		writeError(c, 400, "IDEA_INVALID", err.Error())
		return
	}
	idea, err := h.service.Create(ctx, service.CreateIdeaInput{
		ChannelID:        req.ChannelID,
		SourceSignalID:   req.SourceSignalID,
		CreatedByAgentID: agentIDFromContext(ctx, c),
		Domain:           req.Domain,
		Title:            req.Title,
		Summary:          req.Summary,
		Status:           core.IdeaStatus(req.Status),
		StanceSummary:    req.StanceSummary,
		Meta:             req.Meta,
	})
	if err != nil {
		writeError(c, 400, "IDEA_CREATE_FAILED", err.Error())
		return
	}
	response := map[string]any{"idea": idea}
	if err := storeIdempotentResponse(ctx, h.idempotency, key, response); err != nil {
		writeError(c, 500, "IDEMPOTENCY_STORE_FAILED", err.Error())
		return
	}
	c.JSON(201, response)
}

func (h *IdeaHandler) List(ctx context.Context, c *app.RequestContext) {
	var channelID *string
	if value := c.Query("channel_id"); value != "" {
		channelID = &value
	}
	var status *core.IdeaStatus
	if value := c.Query("status"); value != "" {
		parsed := core.IdeaStatus(value)
		status = &parsed
	}
	ideas, err := h.service.List(ctx, repository.ListIdeasParams{
		Domain:    c.Query("domain"),
		ChannelID: channelID,
		Status:    status,
		Limit:     parseBoundedLimit(c.Query("limit"), 100, 200),
	})
	if err != nil {
		writeError(c, 500, "IDEA_LIST_FAILED", err.Error())
		return
	}
	c.JSON(200, map[string]any{"ideas": ideas})
}

func (h *IdeaHandler) Get(ctx context.Context, c *app.RequestContext) {
	detail, err := h.service.Get(ctx, c.Param("id"))
	if err != nil {
		writeError(c, 404, "IDEA_NOT_FOUND", err.Error())
		return
	}
	c.JSON(200, detail)
}
