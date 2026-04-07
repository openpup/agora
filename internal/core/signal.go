package core

import "time"

type SignalKind string

const (
	SignalKindClaim   SignalKind = "claim"
	SignalKindCounter SignalKind = "counter"
	SignalKindData    SignalKind = "data"
	SignalKindQuery   SignalKind = "query"
)

type Signal struct {
	ID                 string              `json:"id"`
	AgentID            string              `json:"agent_id"`
	ParentID           *string             `json:"parent_id,omitempty"`
	Domain             string              `json:"domain"`
	Kind               SignalKind          `json:"kind"`
	Claim              Claim               `json:"claim"`
	Reasoning          Reasoning           `json:"reasoning"`
	Evidence           []Evidence          `json:"evidence,omitempty"`
	DisagreementPoints []DisagreementPoint `json:"disagreement_points,omitempty"`
	Refs               []CrossRef          `json:"refs,omitempty"`
	Verified           *bool               `json:"verified,omitempty"`
	VerifiedAt         *time.Time          `json:"verified_at,omitempty"`
	VerificationDetail map[string]any      `json:"verification_detail,omitempty"`
	Meta               map[string]any      `json:"meta,omitempty"`
	CreatedAt          time.Time           `json:"created_at"`
	CounterSignals     []Signal            `json:"counter_signals,omitempty"`
}

type Claim struct {
	Statement    string         `json:"statement"`
	Structured   map[string]any `json:"structured"`
	Confidence   float64        `json:"confidence"`
	VerifiableBy *time.Time     `json:"verifiable_by,omitempty"`
	Resolution   *Resolution    `json:"resolution,omitempty"`
}

type Resolution struct {
	Strategy string         `json:"strategy"`
	Params   map[string]any `json:"params,omitempty"`
}

type Evidence struct {
	Type string         `json:"type"`
	Ref  string         `json:"ref"`
	Meta map[string]any `json:"meta,omitempty"`
}

type CrossRef struct {
	Domain   string `json:"domain"`
	SignalID string `json:"signal_id"`
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
	Domain        string       `json:"domain"`
	Kinds         []SignalKind `json:"kinds,omitempty"`
	MinConfidence *float64     `json:"min_confidence,omitempty"`
	MinTrustScore *float64     `json:"min_trust_score,omitempty"`
}

type SignalListCursor struct {
	CreatedAt time.Time `json:"created_at"`
	ID        string    `json:"id"`
}
