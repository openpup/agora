package service

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/openpup/agora/internal/core"
	"github.com/openpup/agora/internal/repository"
)

func TestResolutionServiceSubmitResolvesClaimAfterTwoSupportingAttestations(t *testing.T) {
	ctx := context.Background()
	claim := testClaimSignal("claim-1")
	signals := &stubSignalRepo{
		signals: map[string]*core.Signal{claim.ID: claim},
	}
	resolutions := &stubResolutionRepo{
		attestations: map[string][]repository.ResolutionAttestationRow{
			claim.ID: {
				{
					Attestation: core.ResolutionAttestation{
						ID:         "att-1",
						ClaimID:    claim.ID,
						AgentID:    "agent-1",
						Kind:       core.ResolutionAttestationResolve,
						Verdict:    boolPtr(true),
						Confidence: 0.91,
						Reasoning:  core.Reasoning{Summary: "resolve", Factors: []core.ReasoningFactor{{Type: "fact"}}},
						CreatedAt:  time.Now().UTC().Add(-2 * time.Minute),
					},
					TrustScore: 0.8,
				},
			},
		},
	}
	service := NewResolutionService(resolutions, signals)

	resolution, _, err := service.Submit(ctx, SubmitResolutionInput{
		ClaimID:    claim.ID,
		AgentID:    "agent-2",
		Kind:       core.ResolutionAttestationResolve,
		Verdict:    boolPtr(true),
		Confidence: 0.88,
		Reasoning:  core.Reasoning{Summary: "two resolvers agree", Factors: []core.ReasoningFactor{{Type: "market"}}},
	})
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if resolution.State != core.ResolutionStateResolved {
		t.Fatalf("expected resolved state, got %s", resolution.State)
	}
	if resolution.Outcome == nil || !*resolution.Outcome {
		t.Fatalf("expected positive outcome, got %+v", resolution.Outcome)
	}
	if signals.markedID != claim.ID || !signals.markedVerified {
		t.Fatalf("expected claim to be marked verified true")
	}
}

func TestResolutionServiceSubmitLeavesClaimChallengedWhenCloseAndContested(t *testing.T) {
	ctx := context.Background()
	claim := testClaimSignal("claim-2")
	signals := &stubSignalRepo{
		signals: map[string]*core.Signal{claim.ID: claim},
	}
	resolutions := &stubResolutionRepo{
		attestations: map[string][]repository.ResolutionAttestationRow{
			claim.ID: {
				{
					Attestation: core.ResolutionAttestation{
						ID:         "att-1",
						ClaimID:    claim.ID,
						AgentID:    "agent-1",
						Kind:       core.ResolutionAttestationResolve,
						Verdict:    boolPtr(true),
						Confidence: 0.65,
						Reasoning:  core.Reasoning{Summary: "support", Factors: []core.ReasoningFactor{{Type: "fact"}}},
						CreatedAt:  time.Now().UTC().Add(-3 * time.Minute),
					},
					TrustScore: 0.7,
				},
				{
					Attestation: core.ResolutionAttestation{
						ID:         "att-2",
						ClaimID:    claim.ID,
						AgentID:    "agent-3",
						Kind:       core.ResolutionAttestationChallenge,
						Confidence: 0.7,
						Reasoning:  core.Reasoning{Summary: "insufficient evidence", Factors: []core.ReasoningFactor{{Type: "challenge"}}},
						CreatedAt:  time.Now().UTC().Add(-2 * time.Minute),
					},
					TrustScore: 0.6,
				},
			},
		},
	}
	service := NewResolutionService(resolutions, signals)

	resolution, _, err := service.Submit(ctx, SubmitResolutionInput{
		ClaimID:    claim.ID,
		AgentID:    "agent-2",
		Kind:       core.ResolutionAttestationResolve,
		Verdict:    boolPtr(false),
		Confidence: 0.64,
		Reasoning:  core.Reasoning{Summary: "counter resolve", Factors: []core.ReasoningFactor{{Type: "fact"}}},
	})
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if resolution.State != core.ResolutionStateChallenged {
		t.Fatalf("expected challenged state, got %s", resolution.State)
	}
	if resolution.Outcome != nil {
		t.Fatalf("expected no final outcome when challenged")
	}
	if signals.markedID != "" {
		t.Fatalf("expected no verification write for challenged claim")
	}
}

func TestResolutionServiceGetByClaimIDReturnsSyntheticOpenResolution(t *testing.T) {
	ctx := context.Background()
	claim := testClaimSignal("claim-3")
	signals := &stubSignalRepo{
		signals: map[string]*core.Signal{claim.ID: claim},
	}
	resolutions := &stubResolutionRepo{
		getErr: pgx.ErrNoRows,
		attestations: map[string][]repository.ResolutionAttestationRow{
			claim.ID: {
				{
					Attestation: core.ResolutionAttestation{
						ID:         "att-1",
						ClaimID:    claim.ID,
						AgentID:    "agent-1",
						Kind:       core.ResolutionAttestationResolve,
						Verdict:    boolPtr(true),
						Confidence: 0.73,
						Reasoning:  core.Reasoning{Summary: "one resolver only", Factors: []core.ReasoningFactor{{Type: "fact"}}},
						CreatedAt:  time.Now().UTC().Add(-1 * time.Minute),
					},
					TrustScore: 0.8,
				},
			},
		},
	}
	service := NewResolutionService(resolutions, signals)

	resolution, err := service.GetByClaimID(ctx, claim.ID)
	if err != nil {
		t.Fatalf("GetByClaimID: %v", err)
	}
	if resolution.State != core.ResolutionStateOpen {
		t.Fatalf("expected open synthetic resolution, got %s", resolution.State)
	}
	if resolution.ResolverCount != 1 {
		t.Fatalf("expected resolver_count=1, got %d", resolution.ResolverCount)
	}
}

type stubResolutionRepo struct {
	attestations map[string][]repository.ResolutionAttestationRow
	resolutions  map[string]*core.ClaimResolution
	getErr       error
	upserted     *core.ClaimResolution
	created      *core.ResolutionAttestation
}

func (s *stubResolutionRepo) CreateAttestation(_ context.Context, att *core.ResolutionAttestation) error {
	s.created = att
	s.attestations[att.ClaimID] = append(s.attestations[att.ClaimID], repository.ResolutionAttestationRow{
		Attestation: *att,
		TrustScore:  0.75,
	})
	return nil
}

func (s *stubResolutionRepo) ListAttestations(_ context.Context, claimID string) ([]repository.ResolutionAttestationRow, error) {
	return append([]repository.ResolutionAttestationRow(nil), s.attestations[claimID]...), nil
}

func (s *stubResolutionRepo) ListAllAttestations(context.Context) ([]repository.ResolutionAttestationRow, error) {
	var out []repository.ResolutionAttestationRow
	for _, items := range s.attestations {
		out = append(out, items...)
	}
	return out, nil
}

func (s *stubResolutionRepo) GetClaimResolution(_ context.Context, claimID string) (*core.ClaimResolution, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	if resolution, ok := s.resolutions[claimID]; ok {
		return resolution, nil
	}
	return nil, pgx.ErrNoRows
}

func (s *stubResolutionRepo) ListAllClaimResolutions(context.Context) ([]core.ClaimResolution, error) {
	var out []core.ClaimResolution
	for _, resolution := range s.resolutions {
		out = append(out, *resolution)
	}
	return out, nil
}

func (s *stubResolutionRepo) UpsertClaimResolution(_ context.Context, resolution *core.ClaimResolution) error {
	s.upserted = resolution
	if s.resolutions == nil {
		s.resolutions = map[string]*core.ClaimResolution{}
	}
	s.resolutions[resolution.ClaimID] = resolution
	return nil
}

type stubSignalRepo struct {
	signals          map[string]*core.Signal
	markedID         string
	markedVerified   bool
	markedDetail     map[string]any
	markedVerifiedAt time.Time
}

func (s *stubSignalRepo) Create(context.Context, *core.Signal) error { return nil }
func (s *stubSignalRepo) GetByID(_ context.Context, id string) (*core.Signal, error) {
	if signal, ok := s.signals[id]; ok {
		return signal, nil
	}
	return nil, pgx.ErrNoRows
}
func (s *stubSignalRepo) List(context.Context, repository.ListSignalsParams) ([]core.Signal, error) {
	return nil, nil
}
func (s *stubSignalRepo) ListAll(context.Context) ([]core.Signal, error) { return nil, nil }
func (s *stubSignalRepo) ListCounters(context.Context, string) ([]core.Signal, error) {
	return nil, nil
}
func (s *stubSignalRepo) ListPendingVerification(context.Context, time.Time) ([]repository.VerificationCandidate, error) {
	return nil, nil
}
func (s *stubSignalRepo) MarkVerified(_ context.Context, signalID string, verified bool, detail map[string]any, verifiedAt time.Time) error {
	s.markedID = signalID
	s.markedVerified = verified
	s.markedDetail = detail
	s.markedVerifiedAt = verifiedAt
	return nil
}
func (s *stubSignalRepo) ListConsensusRows(context.Context, string, *time.Duration) ([]repository.ConsensusRow, error) {
	return nil, nil
}
func (s *stubSignalRepo) ListOverviewRows(context.Context, string, int) ([]repository.ConsensusRow, error) {
	return nil, nil
}
func (s *stubSignalRepo) ListAgentDomainStats(context.Context) ([]repository.AgentDomainStats, error) {
	return nil, nil
}

func testClaimSignal(id string) *core.Signal {
	verifiableBy := time.Now().UTC().Add(24 * time.Hour)
	return &core.Signal{
		ID:      id,
		AgentID: "agent-claim",
		Domain:  "finance.us_stock",
		Kind:    core.SignalKindClaim,
		Claim: core.Claim{
			Statement:    "NVDA will close above the current price",
			Structured:   map[string]any{"ticker": "NVDA", "direction": "bullish", "market": "us_stock"},
			Confidence:   0.82,
			VerifiableBy: &verifiableBy,
			Resolution:   &core.Resolution{Strategy: "attested_consensus"},
		},
		Reasoning: core.Reasoning{
			Summary: "Demand is broadening while supply is constrained.",
			Factors: []core.ReasoningFactor{{Type: "market"}},
		},
		CreatedAt: time.Now().UTC(),
	}
}

func boolPtr(v bool) *bool { return &v }
