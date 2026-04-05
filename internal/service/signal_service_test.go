package service

import (
	"testing"
	"time"

	"github.com/openpup/agora/internal/domain"
)

func TestValidateSignalInputPredictionRequiresCoreFields(t *testing.T) {
	input := CreateSignalInput{
		Market:     domain.MarketUSStock,
		SignalType: domain.SignalTypePrediction,
		Reasoning: domain.Reasoning{
			Factors: []domain.ReasoningFactor{{Type: "technical"}},
			Summary: "signal thesis",
		},
	}

	if err := validateSignalInput(input); err == nil {
		t.Fatalf("expected validation error for missing prediction fields")
	}
}

func TestValidateSignalInputCounterRequiresDisagreement(t *testing.T) {
	parentID := "parent-1"
	ticker := "NVDA"
	direction := domain.DirectionBearish
	confidence := 0.72
	horizon := 24 * time.Hour
	input := CreateSignalInput{
		ParentID:    &parentID,
		Market:      domain.MarketUSStock,
		SignalType:  domain.SignalTypePrediction,
		Ticker:      &ticker,
		Direction:   &direction,
		Confidence:  &confidence,
		TimeHorizon: &horizon,
		Reasoning: domain.Reasoning{
			Factors: []domain.ReasoningFactor{{Type: "technical"}},
			Summary: "counter thesis",
		},
	}

	if err := validateSignalInput(input); err == nil {
		t.Fatalf("expected validation error for missing disagreement points")
	}
}

func TestDecodeCursorRoundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	cursorPayload := domain.SignalListCursor{CreatedAt: now, ID: "signal-1"}
	raw := "eyJjcmVhdGVkX2F0Ijoi" // invalid baseline check
	if _, err := DecodeCursor(raw); err == nil {
		t.Fatalf("expected invalid base64 cursor to fail")
	}

	encoded, err := encodeCursor(cursorPayload)
	if err != nil {
		t.Fatalf("encodeCursor: %v", err)
	}
	decoded, err := DecodeCursor(encoded)
	if err != nil {
		t.Fatalf("DecodeCursor: %v", err)
	}
	if decoded.ID != cursorPayload.ID || !decoded.CreatedAt.Equal(cursorPayload.CreatedAt) {
		t.Fatalf("decoded cursor mismatch: got %+v want %+v", decoded, cursorPayload)
	}
}
