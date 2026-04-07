package handler

import (
	"context"
	"time"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/openpup/agora/internal/service"
)

type ConsensusHandler struct {
	service *service.ConsensusService
}

func NewConsensusHandler(service *service.ConsensusService) *ConsensusHandler {
	return &ConsensusHandler{service: service}
}

func (h *ConsensusHandler) GetConsensus(ctx context.Context, c *app.RequestContext) {
	domain := c.Query("domain")
	if domain == "" {
		writeError(c, 400, "CONSENSUS_INVALID_DOMAIN", "domain query parameter is required")
		return
	}
	var horizon *time.Duration
	if raw := c.Query("time_horizon"); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil {
			writeError(c, 400, "CONSENSUS_INVALID_HORIZON", err.Error())
			return
		}
		horizon = &parsed
	}
	resp, err := h.service.GetConsensus(ctx, domain, horizon)
	if err != nil {
		writeError(c, 500, "CONSENSUS_FAILED", err.Error())
		return
	}
	c.JSON(200, resp)
}

func (h *ConsensusHandler) Overview(ctx context.Context, c *app.RequestContext) {
	domain := c.Query("domain")
	if domain == "" {
		writeError(c, 400, "CONSENSUS_INVALID_DOMAIN", "domain query parameter is required")
		return
	}
	resp, err := h.service.GetOverview(ctx, domain)
	if err != nil {
		writeError(c, 500, "CONSENSUS_OVERVIEW_FAILED", err.Error())
		return
	}
	c.JSON(200, resp)
}
