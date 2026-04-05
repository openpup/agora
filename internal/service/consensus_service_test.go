package service

import (
	"testing"
	"time"

	"github.com/openpup/agora/internal/domain"
	"github.com/openpup/agora/internal/repository"
)

func TestComputeConsensus(t *testing.T) {
	now := time.Now().UTC()
	rows := []repository.ConsensusRow{
		{Ticker: "NVDA", Direction: "bullish", Confidence: 0.8, TrustScore: 0.9, SignalID: "1", AgentID: "a1", CreatedAt: now},
		{Ticker: "NVDA", Direction: "bullish", Confidence: 0.6, TrustScore: 0.7, SignalID: "2", AgentID: "a2", CreatedAt: now.Add(-time.Minute)},
		{Ticker: "NVDA", Direction: "bearish", Confidence: 0.5, TrustScore: 0.4, SignalID: "3", AgentID: "a3", CreatedAt: now.Add(-2 * time.Minute)},
	}

	detail := computeConsensus(domain.MarketUSStock, "NVDA", rows)

	if detail.WeightedDirection != "bullish" {
		t.Fatalf("expected bullish direction, got %s", detail.WeightedDirection)
	}
	if detail.BullishCount != 2 || detail.BearishCount != 1 {
		t.Fatalf("unexpected direction counts: %+v", detail)
	}
	if len(detail.TopSignals) != 3 {
		t.Fatalf("expected 3 top signals, got %d", len(detail.TopSignals))
	}
	if detail.TopSignals[0].SignalID != "1" {
		t.Fatalf("expected strongest signal first, got %s", detail.TopSignals[0].SignalID)
	}
}
