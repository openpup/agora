package handler

import (
	"context"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/openpup/agora/internal/core"
	"github.com/openpup/agora/internal/repository"
	"github.com/openpup/agora/internal/service"
)

type SignalHandler struct {
	service     *service.SignalService
	idempotency *service.IdempotencyService
}

func NewSignalHandler(service *service.SignalService, idempotency *service.IdempotencyService) *SignalHandler {
	return &SignalHandler{service: service, idempotency: idempotency}
}

type createSignalRequest struct {
	Domain             string                     `json:"domain"`
	Kind               string                     `json:"kind"`
	Claim              core.Claim                 `json:"claim"`
	Reasoning          core.Reasoning             `json:"reasoning"`
	Evidence           []core.Evidence            `json:"evidence"`
	Refs               []core.CrossRef            `json:"refs"`
	Meta               map[string]any             `json:"meta"`
	DisagreementPoints []core.DisagreementPoint   `json:"disagreement_points"`
}

func (h *SignalHandler) Create(ctx context.Context, c *app.RequestContext) {
	key := idempotencyKey(c, agentIDFromContext(ctx, c))
	if serveIdempotentResponse(ctx, c, h.idempotency, key) {
		return
	}
	var req createSignalRequest
	if err := c.BindAndValidate(&req); err != nil {
		writeError(c, 400, "SIGNAL_INVALID", err.Error())
		return
	}
	if req.Domain == "" {
		writeError(c, 400, "SIGNAL_INVALID_DOMAIN", "domain is required")
		return
	}
	signal, err := h.service.Create(ctx, service.CreateSignalInput{
		AgentID:            agentIDFromContext(ctx, c),
		Domain:             req.Domain,
		Kind:               core.SignalKind(req.Kind),
		Claim:              req.Claim,
		Reasoning:          req.Reasoning,
		Evidence:           req.Evidence,
		Refs:               req.Refs,
		Meta:               req.Meta,
		DisagreementPoints: req.DisagreementPoints,
	})
	if err != nil {
		writeError(c, 400, "SIGNAL_CREATE_FAILED", err.Error())
		return
	}
	response := map[string]any{"signal_id": signal.ID, "created_at": signal.CreatedAt}
	if err := storeIdempotentResponse(ctx, h.idempotency, key, response); err != nil {
		writeError(c, 500, "IDEMPOTENCY_STORE_FAILED", err.Error())
		return
	}
	c.JSON(201, response)
}

func (h *SignalHandler) CreateCounter(ctx context.Context, c *app.RequestContext) {
	key := idempotencyKey(c, agentIDFromContext(ctx, c))
	if serveIdempotentResponse(ctx, c, h.idempotency, key) {
		return
	}
	var req createSignalRequest
	if err := c.BindAndValidate(&req); err != nil {
		writeError(c, 400, "SIGNAL_INVALID", err.Error())
		return
	}
	if req.Domain == "" {
		writeError(c, 400, "SIGNAL_INVALID_DOMAIN", "domain is required")
		return
	}
	parentID := c.Param("id")
	signal, err := h.service.Create(ctx, service.CreateSignalInput{
		AgentID:            agentIDFromContext(ctx, c),
		ParentID:           &parentID,
		Domain:             req.Domain,
		Kind:               core.SignalKind(req.Kind),
		Claim:              req.Claim,
		Reasoning:          req.Reasoning,
		Evidence:           req.Evidence,
		Refs:               req.Refs,
		Meta:               req.Meta,
		DisagreementPoints: req.DisagreementPoints,
	})
	if err != nil {
		writeError(c, 400, "COUNTER_SIGNAL_CREATE_FAILED", err.Error())
		return
	}
	response := map[string]any{"signal_id": signal.ID, "created_at": signal.CreatedAt}
	if err := storeIdempotentResponse(ctx, h.idempotency, key, response); err != nil {
		writeError(c, 500, "IDEMPOTENCY_STORE_FAILED", err.Error())
		return
	}
	c.JSON(201, response)
}

func (h *SignalHandler) Get(ctx context.Context, c *app.RequestContext) {
	signal, err := h.service.GetByID(ctx, c.Param("id"))
	if err != nil {
		writeError(c, 404, "SIGNAL_NOT_FOUND", err.Error())
		return
	}
	c.JSON(200, signal)
}

func (h *SignalHandler) List(ctx context.Context, c *app.RequestContext) {
	domain := c.Query("domain")
	if domain == "" {
		writeError(c, 400, "SIGNAL_INVALID_DOMAIN", "domain query parameter is required")
		return
	}
	limit := 50
	if raw := c.Query("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}
	cursor, err := service.DecodeCursor(c.Query("cursor"))
	if err != nil {
		writeError(c, 400, "SIGNAL_INVALID_CURSOR", err.Error())
		return
	}
	var agentID *string
	if value := c.Query("agent_id"); value != "" {
		agentID = &value
	}
	var kind *core.SignalKind
	if value := c.Query("kind"); value != "" {
		parsed := core.SignalKind(value)
		kind = &parsed
	}
	var minConfidence *float64
	if value := c.Query("min_confidence"); value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil {
			minConfidence = &parsed
		}
	}
	since, err := parseOptionalTime(c.Query("since"))
	if err != nil {
		writeError(c, 400, "SIGNAL_INVALID_SINCE", err.Error())
		return
	}
	signals, nextCursor, err := h.service.List(ctx, repository.ListSignalsParams{
		Domain:        domain,
		Kind:          kind,
		AgentID:       agentID,
		MinConfidence: minConfidence,
		Since:         since,
		Cursor:        cursor,
		Limit:         limit,
	})
	if err != nil {
		writeError(c, 500, "SIGNAL_LIST_FAILED", err.Error())
		return
	}
	c.JSON(200, map[string]any{"signals": signals, "next_cursor": nextCursor})
}
