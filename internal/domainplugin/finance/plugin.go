package finance

import (
	"context"
	"fmt"
	"time"

	"github.com/openpup/agora/internal/core"
)

type Plugin struct {
	marketData *MarketDataRepo
}

func NewPlugin(marketData *MarketDataRepo) *Plugin {
	return &Plugin{marketData: marketData}
}

func (p *Plugin) Name() string { return "finance" }

func (p *Plugin) Definition() core.DomainDef {
	return core.DomainDef{
		ID:        "finance",
		Name:      "Finance",
		Namespace: "finance",
		ClaimSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"ticker":    map[string]any{"type": "string"},
				"direction": map[string]any{"type": "string", "enum": []string{"bullish", "bearish", "neutral"}},
				"threshold": map[string]any{"type": "number"},
			},
			"required": []string{"ticker", "direction"},
		},
		Resolution: core.Resolution{
			Strategy: "price_comparison",
		},
		Status:    "active",
		CreatedAt: time.Now().UTC(),
	}
}

func (p *Plugin) ValidateClaim(structured map[string]any) error {
	if _, ok := structured["ticker"]; !ok {
		return fmt.Errorf("finance: ticker is required")
	}
	dir, ok := structured["direction"]
	if !ok {
		return fmt.Errorf("finance: direction is required")
	}
	dirStr, _ := dir.(string)
	if dirStr != "bullish" && dirStr != "bearish" && dirStr != "neutral" {
		return fmt.Errorf("finance: direction must be bullish, bearish, or neutral")
	}
	return nil
}

func (p *Plugin) Verify(ctx context.Context, signal core.Signal) (bool, map[string]any, error) {
	ticker, _ := signal.Claim.Structured["ticker"].(string)
	direction, _ := signal.Claim.Structured["direction"].(string)
	if ticker == "" || direction == "" {
		return false, nil, fmt.Errorf("finance: invalid claim structured data")
	}

	startPrice, err := p.marketData.GetClosestPrice(ctx, ticker, ">=", signal.CreatedAt)
	if err != nil {
		return false, nil, err
	}
	if signal.Claim.VerifiableBy == nil {
		return false, nil, fmt.Errorf("finance: verifiable_by is required for verification")
	}
	endPrice, err := p.marketData.GetClosestPrice(ctx, ticker, ">=", *signal.Claim.VerifiableBy)
	if err != nil {
		return false, nil, err
	}

	var correct bool
	switch direction {
	case "bullish":
		correct = endPrice > startPrice
	case "bearish":
		correct = endPrice < startPrice
	default:
		correct = false
	}

	detail := map[string]any{
		"start_price": startPrice,
		"end_price":   endPrice,
		"delta":       endPrice - startPrice,
	}
	return correct, detail, nil
}

func (p *Plugin) ResolveConsensus(signals []core.Signal) (map[string]any, error) {
	var bullish, bearish, neutral int
	var bullSum, bearSum float64

	type sigSummary struct {
		SignalID   string  `json:"signal_id"`
		AgentID   string  `json:"agent_id"`
		Direction string  `json:"direction"`
		Score     float64 `json:"score"`
	}
	var top []sigSummary

	for _, s := range signals {
		dir, _ := s.Claim.Structured["direction"].(string)
		switch dir {
		case "bullish":
			bullish++
			bullSum += s.Claim.Confidence
		case "bearish":
			bearish++
			bearSum += s.Claim.Confidence
		default:
			neutral++
		}
		top = append(top, sigSummary{
			SignalID:   s.ID,
			AgentID:   s.AgentID,
			Direction: dir,
			Score:     s.Claim.Confidence,
		})
	}

	total := bullish + bearish + neutral
	consensus := 0.0
	direction := "neutral"
	if total > 0 {
		consensus = float64(bullish-bearish) / float64(total)
		if consensus > 0 {
			direction = "bullish"
		} else if consensus < 0 {
			direction = "bearish"
		}
	}

	avgBull := 0.0
	if bullish > 0 {
		avgBull = bullSum / float64(bullish)
	}
	avgBear := 0.0
	if bearish > 0 {
		avgBear = bearSum / float64(bearish)
	}

	if len(top) > 5 {
		top = top[:5]
	}

	return map[string]any{
		"bullish_count":          bullish,
		"bearish_count":          bearish,
		"neutral_count":          neutral,
		"avg_bullish_confidence": avgBull,
		"avg_bearish_confidence": avgBear,
		"weighted_consensus":     consensus,
		"weighted_direction":     direction,
		"top_signals":            top,
	}, nil
}
