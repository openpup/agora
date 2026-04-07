package handler

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/openpup/agora/internal/core"
	"github.com/openpup/agora/internal/service"
)

type ResolutionHandler struct {
	service     *service.ResolutionService
	idempotency *service.IdempotencyService
}

func NewResolutionHandler(service *service.ResolutionService, idempotency *service.IdempotencyService) *ResolutionHandler {
	return &ResolutionHandler{service: service, idempotency: idempotency}
}

type submitResolutionRequest struct {
	Kind       string          `json:"kind"`
	Verdict    *bool           `json:"verdict"`
	Confidence float64         `json:"confidence"`
	Reasoning  core.Reasoning  `json:"reasoning"`
	Evidence   []core.Evidence `json:"evidence"`
	Meta       map[string]any  `json:"meta"`
}

func (h *ResolutionHandler) Submit(ctx context.Context, c *app.RequestContext) {
	key := idempotencyKey(c, agentIDFromContext(ctx, c))
	if serveIdempotentResponse(ctx, c, h.idempotency, key) {
		return
	}
	var req submitResolutionRequest
	if err := c.BindAndValidate(&req); err != nil {
		writeError(c, 400, "RESOLUTION_INVALID", err.Error())
		return
	}
	resolution, attestation, err := h.service.Submit(ctx, service.SubmitResolutionInput{
		ClaimID:    c.Param("id"),
		AgentID:    agentIDFromContext(ctx, c),
		Kind:       core.ResolutionAttestationKind(req.Kind),
		Verdict:    req.Verdict,
		Confidence: req.Confidence,
		Reasoning:  req.Reasoning,
		Evidence:   req.Evidence,
		Meta:       req.Meta,
	})
	if err != nil {
		writeError(c, 400, "RESOLUTION_SUBMIT_FAILED", err.Error())
		return
	}
	response := map[string]any{
		"attestation": attestation,
		"resolution":  resolution,
	}
	if err := storeIdempotentResponse(ctx, h.idempotency, key, response); err != nil {
		writeError(c, 500, "IDEMPOTENCY_STORE_FAILED", err.Error())
		return
	}
	c.JSON(201, response)
}

func (h *ResolutionHandler) GetByClaimID(ctx context.Context, c *app.RequestContext) {
	resolution, err := h.service.GetByClaimID(ctx, c.Param("id"))
	if err != nil {
		writeError(c, 404, "RESOLUTION_NOT_FOUND", err.Error())
		return
	}
	c.JSON(200, resolution)
}
