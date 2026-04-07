package handler

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/openpup/agora/internal/domainplugin/finance"
)

type FinanceMarketDataHandler struct {
	repo *finance.MarketDataRepo
}

func NewFinanceMarketDataHandler(repo *finance.MarketDataRepo) *FinanceMarketDataHandler {
	return &FinanceMarketDataHandler{repo: repo}
}

func (h *FinanceMarketDataHandler) List(ctx context.Context, c *app.RequestContext) {
	domain := c.Query("domain")
	if domain == "" {
		writeError(c, 400, "MARKET_DATA_INVALID_DOMAIN", "domain query parameter is required")
		return
	}
	ticker := c.Query("ticker")
	if ticker == "" {
		writeError(c, 400, "MARKET_DATA_INVALID_TICKER", "ticker query parameter is required")
		return
	}
	market, err := marketFromDomain(domain)
	if err != nil {
		writeError(c, 400, "MARKET_DATA_INVALID_DOMAIN", err.Error())
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
	candles, err := h.repo.ListCandles(ctx, finance.MarketDataQuery{
		Ticker: ticker,
		Market: market,
		From:   from,
		To:     to,
	})
	if err != nil {
		writeError(c, 500, "MARKET_DATA_LIST_FAILED", err.Error())
		return
	}
	c.JSON(200, map[string]any{
		"domain": domain,
		"ticker": ticker,
		"data":   candles,
	})
}

func marketFromDomain(domain string) (string, error) {
	if !strings.HasPrefix(domain, "finance.") {
		return "", fmt.Errorf("unsupported domain %q", domain)
	}
	parts := strings.Split(domain, ".")
	if len(parts) < 2 {
		return "", fmt.Errorf("unsupported domain %q", domain)
	}
	return parts[len(parts)-1], nil
}
