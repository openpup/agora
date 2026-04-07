package core

import "time"

type ResolutionAttestationKind string

const (
	ResolutionAttestationResolve   ResolutionAttestationKind = "resolve"
	ResolutionAttestationChallenge ResolutionAttestationKind = "challenge"
)

type ResolutionState string

const (
	ResolutionStateOpen       ResolutionState = "open"
	ResolutionStateResolved   ResolutionState = "resolved"
	ResolutionStateChallenged ResolutionState = "challenged"
)

type ResolutionAttestation struct {
	ID         string                    `json:"id"`
	ClaimID    string                    `json:"claim_id"`
	AgentID    string                    `json:"agent_id"`
	Kind       ResolutionAttestationKind `json:"kind"`
	Verdict    *bool                     `json:"verdict,omitempty"`
	Confidence float64                   `json:"confidence"`
	Reasoning  Reasoning                 `json:"reasoning"`
	Evidence   []Evidence                `json:"evidence,omitempty"`
	Meta       map[string]any            `json:"meta,omitempty"`
	CreatedAt  time.Time                 `json:"created_at"`
}

type ClaimResolution struct {
	ClaimID         string                  `json:"claim_id"`
	Domain          string                  `json:"domain"`
	Strategy        string                  `json:"strategy"`
	State           ResolutionState         `json:"state"`
	Outcome         *bool                   `json:"outcome,omitempty"`
	ResolutionScore float64                 `json:"resolution_score"`
	ResolverCount   int                     `json:"resolver_count"`
	ChallengeCount  int                     `json:"challenge_count"`
	Summary         map[string]any          `json:"summary,omitempty"`
	ResolvedAt      *time.Time              `json:"resolved_at,omitempty"`
	UpdatedAt       time.Time               `json:"updated_at"`
	Attestations    []ResolutionAttestation `json:"attestations,omitempty"`
}
