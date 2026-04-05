package handler

import (
	"context"
	"strconv"
	"time"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/openpup/agora/internal/domain"
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
	Market             string                     `json:"market"`
	SignalType         string                     `json:"signal_type"`
	Ticker             *string                    `json:"ticker"`
	Direction          *string                    `json:"direction"`
	Confidence         *float64                   `json:"confidence"`
	TimeHorizon        string                     `json:"time_horizon"`
	Reasoning          domain.Reasoning           `json:"reasoning"`
	DataRefs           []map[string]any           `json:"data_refs"`
	Meta               map[string]any             `json:"meta"`
	DisagreementPoints []domain.DisagreementPoint `json:"disagreement_points"`
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
	market, err := domain.ParseMarket(req.Market)
	if err != nil {
		writeError(c, 400, "SIGNAL_INVALID_MARKET", err.Error())
		return
	}
	var direction *domain.Direction
	if req.Direction != nil {
		parsed := domain.Direction(*req.Direction)
		direction = &parsed
	}
	var horizon *time.Duration
	if req.TimeHorizon != "" {
		d, err := time.ParseDuration(req.TimeHorizon)
		if err != nil {
			writeError(c, 400, "SIGNAL_INVALID_HORIZON", err.Error())
			return
		}
		horizon = &d
	}
	signal, err := h.service.Create(ctx, service.CreateSignalInput{
		AgentID:            agentIDFromContext(ctx, c),
		Market:             market,
		SignalType:         domain.SignalType(req.SignalType),
		Ticker:             req.Ticker,
		Direction:          direction,
		Confidence:         req.Confidence,
		TimeHorizon:        horizon,
		Reasoning:          req.Reasoning,
		DataRefs:           req.DataRefs,
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
	market, err := domain.ParseMarket(req.Market)
	if err != nil {
		writeError(c, 400, "SIGNAL_INVALID_MARKET", err.Error())
		return
	}
	parentID := c.Param("id")
	var direction *domain.Direction
	if req.Direction != nil {
		parsed := domain.Direction(*req.Direction)
		direction = &parsed
	}
	signal, err := h.service.Create(ctx, service.CreateSignalInput{
		AgentID:            agentIDFromContext(ctx, c),
		ParentID:           &parentID,
		Market:             market,
		SignalType:         domain.SignalType(req.SignalType),
		Ticker:             req.Ticker,
		Direction:          direction,
		Confidence:         req.Confidence,
		Reasoning:          req.Reasoning,
		DataRefs:           req.DataRefs,
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
	market, err := domain.ParseMarket(c.Query("market"))
	if err != nil {
		writeError(c, 400, "SIGNAL_INVALID_MARKET", err.Error())
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
	var ticker, agentID *string
	if value := c.Query("ticker"); value != "" {
		ticker = &value
	}
	if value := c.Query("agent_id"); value != "" {
		agentID = &value
	}
	var signalType *domain.SignalType
	if value := c.Query("type"); value != "" {
		parsed := domain.SignalType(value)
		signalType = &parsed
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
		Market:        market,
		Ticker:        ticker,
		SignalType:    signalType,
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
