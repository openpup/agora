package handler

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/openpup/agora/internal/domain"
	"github.com/openpup/agora/internal/service"
)

type MarketDataHandler struct {
	service *service.MarketDataService
}

func NewMarketDataHandler(service *service.MarketDataService) *MarketDataHandler {
	return &MarketDataHandler{service: service}
}

func (h *MarketDataHandler) Get(ctx context.Context, c *app.RequestContext) {
	market, err := domain.ParseMarket(c.Query("market"))
	if err != nil {
		writeError(c, 400, "MARKET_DATA_INVALID_MARKET", err.Error())
		return
	}
	from, err := parseOptionalTime(c.Query("from"))
	if err != nil {
		writeError(c, 400, "MARKET_DATA_INVALID_FROM", err.Error())
		return
	}
	to, err := parseOptionalTime(c.Query("to"))
	if err != nil {
		writeError(c, 400, "MARKET_DATA_INVALID_TO", err.Error())
		return
	}
	data, err := h.service.List(ctx, domain.MarketDataQuery{
		Ticker:   c.Param("ticker"),
		Market:   market,
		Interval: c.DefaultQuery("interval", "1d"),
		From:     from,
		To:       to,
	})
	if err != nil {
		writeError(c, 500, "MARKET_DATA_FAILED", err.Error())
		return
	}
	c.JSON(200, map[string]any{"ticker": c.Param("ticker"), "market": market, "data": data})
}
