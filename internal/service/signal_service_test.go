package service

import (
	"testing"
	"time"

	"github.com/openpup/agora/internal/core"
)

func TestValidateSignalInputClaimRequiresConfidence(t *testing.T) {
	input := CreateSignalInput{
		Domain: "finance.us_stock",
		Kind:   core.SignalKindClaim,
		Claim: core.Claim{
			Statement:  "NVDA will rise",
			Structured: map[string]any{"ticker": "NVDA", "direction": "bullish"},
			Confidence: 0,
		},
		Reasoning: core.Reasoning{
			Factors: []core.ReasoningFactor{{Type: "technical"}},
			Summary: "signal thesis",
		},
	}

	if err := validateSignalInput(input); err == nil {
		t.Fatalf("expected validation error for missing confidence")
	}
}

func TestValidateSignalInputCounterRequiresDisagreement(t *testing.T) {
	parentID := "parent-1"
	input := CreateSignalInput{
		ParentID: &parentID,
		Domain:   "finance.us_stock",
		Kind:     core.SignalKindCounter,
		Claim: core.Claim{
			Statement:  "NVDA will not rise",
			Structured: map[string]any{"ticker": "NVDA", "direction": "bearish"},
			Confidence: 0.72,
		},
		Reasoning: core.Reasoning{
			Factors: []core.ReasoningFactor{{Type: "technical"}},
			Summary: "counter thesis",
		},
	}

	if err := validateSignalInput(input); err == nil {
		t.Fatalf("expected validation error for missing disagreement points")
	}
}

func TestDecodeCursorRoundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	cursorPayload := core.SignalListCursor{CreatedAt: now, ID: "signal-1"}
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
