package handler

import (
	"context"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/openpup/agora/internal/service"
)

type AgentHandler struct {
	service     *service.AgentService
	idempotency *service.IdempotencyService
}

func NewAgentHandler(service *service.AgentService, idempotency *service.IdempotencyService) *AgentHandler {
	return &AgentHandler{service: service, idempotency: idempotency}
}

type registerAgentRequest struct {
	Name         string         `json:"name"`
	Capabilities []string       `json:"capabilities"`
	DataSources  []string       `json:"data_sources"`
	Metadata     map[string]any `json:"metadata"`
}

func (h *AgentHandler) Register(ctx context.Context, c *app.RequestContext) {
	key := idempotencyKey(c, "anonymous")
	if serveIdempotentResponse(ctx, c, h.idempotency, key) {
		return
	}
	var req registerAgentRequest
	if err := c.BindAndValidate(&req); err != nil {
		writeError(c, 400, "AGENT_INVALID", err.Error())
		return
	}
	agent, apiKey, err := h.service.Register(ctx, req.Name, req.Capabilities, req.DataSources, req.Metadata)
	if err != nil {
		writeError(c, 500, "AGENT_REGISTER_FAILED", err.Error())
		return
	}
	response := map[string]any{"agent_id": agent.ID, "api_key": apiKey}
	if err := storeIdempotentResponse(ctx, h.idempotency, key, response); err != nil {
		writeError(c, 500, "IDEMPOTENCY_STORE_FAILED", err.Error())
		return
	}
	c.JSON(201, response)
}

func (h *AgentHandler) Me(ctx context.Context, c *app.RequestContext) {
	agent, err := h.service.GetMe(ctx, agentIDFromContext(ctx, c))
	if err != nil {
		writeError(c, 404, "AGENT_NOT_FOUND", err.Error())
		return
	}
	c.JSON(200, agent)
}

type patchAgentRequest struct {
	Name         string         `json:"name"`
	Capabilities []string       `json:"capabilities"`
	DataSources  []string       `json:"data_sources"`
	Metadata     map[string]any `json:"metadata"`
}

func (h *AgentHandler) PatchMe(ctx context.Context, c *app.RequestContext) {
	var req patchAgentRequest
	if err := c.BindAndValidate(&req); err != nil {
		writeError(c, 400, "AGENT_INVALID_PATCH", err.Error())
		return
	}
	agent, err := h.service.Update(ctx, agentIDFromContext(ctx, c), req.Name, req.Capabilities, req.DataSources, req.Metadata)
	if err != nil {
		writeError(c, 500, "AGENT_PATCH_FAILED", err.Error())
		return
	}
	c.JSON(200, agent)
}

func (h *AgentHandler) TrackRecord(ctx context.Context, c *app.RequestContext) {
	records, err := h.service.GetTrackRecord(ctx, c.Param("id"))
	if err != nil {
		writeError(c, 500, "TRACK_RECORD_FAILED", err.Error())
		return
	}
	c.JSON(200, map[string]any{"agent_id": c.Param("id"), "records": records})
}

func (h *AgentHandler) ListPublic(ctx context.Context, c *app.RequestContext) {
	limit := 20
	if raw := c.Query("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	agents, err := h.service.ListPublic(ctx, limit)
	if err != nil {
		writeError(c, 500, "AGENT_LIST_FAILED", err.Error())
		return
	}
	c.JSON(200, map[string]any{"agents": agents})
}
