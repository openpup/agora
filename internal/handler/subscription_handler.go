package handler

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/openpup/agora/internal/core"
	"github.com/openpup/agora/internal/service"
)

type SubscriptionHandler struct {
	service     *service.SubscriptionService
	idempotency *service.IdempotencyService
}

func NewSubscriptionHandler(service *service.SubscriptionService, idempotency *service.IdempotencyService) *SubscriptionHandler {
	return &SubscriptionHandler{service: service, idempotency: idempotency}
}

type createSubscriptionRequest struct {
	Filter      core.SignalFilter `json:"filter"`
	Delivery    string            `json:"delivery"`
	WebhookURL  *string           `json:"webhook_url"`
	NATSSubject *string           `json:"nats_subject"`
}

func (h *SubscriptionHandler) Create(ctx context.Context, c *app.RequestContext) {
	key := idempotencyKey(c, agentIDFromContext(ctx, c))
	if serveIdempotentResponse(ctx, c, h.idempotency, key) {
		return
	}
	var req createSubscriptionRequest
	if err := c.BindAndValidate(&req); err != nil {
		writeError(c, 400, "SUBSCRIPTION_INVALID", err.Error())
		return
	}
	sub, err := h.service.Create(ctx, agentIDFromContext(ctx, c), req.Filter, core.DeliveryMethod(req.Delivery), req.WebhookURL, req.NATSSubject)
	if err != nil {
		writeError(c, 400, "SUBSCRIPTION_CREATE_FAILED", err.Error())
		return
	}
	response := map[string]any{"subscription_id": sub.ID}
	if err := storeIdempotentResponse(ctx, h.idempotency, key, response); err != nil {
		writeError(c, 500, "IDEMPOTENCY_STORE_FAILED", err.Error())
		return
	}
	c.JSON(201, response)
}

func (h *SubscriptionHandler) List(ctx context.Context, c *app.RequestContext) {
	subs, err := h.service.List(ctx, agentIDFromContext(ctx, c))
	if err != nil {
		writeError(c, 500, "SUBSCRIPTION_LIST_FAILED", err.Error())
		return
	}
	c.JSON(200, map[string]any{"subscriptions": subs})
}

func (h *SubscriptionHandler) Delete(ctx context.Context, c *app.RequestContext) {
	if err := h.service.Delete(ctx, c.Param("id"), agentIDFromContext(ctx, c)); err != nil {
		writeError(c, 500, "SUBSCRIPTION_DELETE_FAILED", err.Error())
		return
	}
	c.SetStatusCode(204)
}
