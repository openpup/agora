package domain

import "time"

type SignalType string
type Direction string

const (
	SignalTypePrediction SignalType = "prediction"
	SignalTypeAnalysis   SignalType = "analysis"
	SignalTypeAlert      SignalType = "alert"
	SignalTypeDataShare  SignalType = "data_share"
)

const (
	DirectionBullish Direction = "bullish"
	DirectionBearish Direction = "bearish"
	DirectionNeutral Direction = "neutral"
)

type Signal struct {
	ID                 string              `json:"id"`
	AgentID            string              `json:"agent_id"`
	ParentID           *string             `json:"parent_id,omitempty"`
	Market             Market              `json:"market"`
	SignalType         SignalType          `json:"signal_type"`
	Ticker             *string             `json:"ticker,omitempty"`
	Direction          *Direction          `json:"direction,omitempty"`
	Confidence         *float64            `json:"confidence,omitempty"`
	TimeHorizon        *time.Duration      `json:"time_horizon,omitempty"`
	ExpiresAt          *time.Time          `json:"expires_at,omitempty"`
	Reasoning          Reasoning           `json:"reasoning"`
	DataRefs           []map[string]any    `json:"data_refs"`
	Meta               map[string]any      `json:"meta"`
	Verified           *bool               `json:"verified,omitempty"`
	VerifiedAt         *time.Time          `json:"verified_at,omitempty"`
	VerificationDetail map[string]any      `json:"verification_detail,omitempty"`
	DisagreementPoints []DisagreementPoint `json:"disagreement_points,omitempty"`
	CreatedAt          time.Time           `json:"created_at"`
	CounterSignals     []Signal            `json:"counter_signals,omitempty"`
}

type Reasoning struct {
	Factors []ReasoningFactor `json:"factors"`
	Summary string            `json:"summary"`
}

type ReasoningFactor struct {
	Type           string         `json:"type"`
	Indicator      string         `json:"indicator,omitempty"`
	Value          any            `json:"value,omitempty"`
	Interpretation string         `json:"interpretation,omitempty"`
	Meta           map[string]any `json:"meta,omitempty"`
}

type DisagreementPoint struct {
	OriginalFactor string         `json:"original_factor"`
	Counter        string         `json:"counter"`
	Evidence       map[string]any `json:"evidence"`
}

type SignalFilter struct {
	Market        Market       `json:"market"`
	Tickers       []string     `json:"tickers,omitempty"`
	SignalTypes   []SignalType `json:"signal_types,omitempty"`
	MinConfidence *float64     `json:"min_confidence,omitempty"`
	MinTrustScore *float64     `json:"min_trust_score,omitempty"`
}

type SignalListCursor struct {
	CreatedAt time.Time `json:"created_at"`
	ID        string    `json:"id"`
}
