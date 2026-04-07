package service

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/openpup/agora/internal/core"
	"github.com/openpup/agora/internal/repository"
)

func TestTrustServiceRecalculateBuildsMultiDimensionalTrust(t *testing.T) {
	ctx := context.Background()
	rootID := "claim-root"
	counterID := "counter-1"
	challengeID := "challenger-1"

	signals := []core.Signal{
		{
			ID:      rootID,
			AgentID: "agent-claim",
			Domain:  "finance.us_stock",
			Kind:    core.SignalKindClaim,
			Claim: core.Claim{
				Confidence: 0.8,
				Structured: map[string]any{"ticker": "NVDA", "direction": "bullish", "market": "us_stock"},
			},
			CreatedAt: time.Now().UTC().Add(-3 * time.Hour),
		},
		{
			ID:       counterID,
			AgentID:  "agent-counter",
			ParentID: &rootID,
			Domain:   "finance.us_stock",
			Kind:     core.SignalKindCounter,
			Claim: core.Claim{
				Confidence: 0.6,
				Structured: map[string]any{"ticker": "NVDA", "direction": "bearish", "market": "us_stock"},
			},
			CreatedAt: time.Now().UTC().Add(-2 * time.Hour),
		},
		{
			ID:      "claim-challenged",
			AgentID: challengeID,
			Domain:  "finance.us_stock",
			Kind:    core.SignalKindClaim,
			Claim: core.Claim{
				Confidence: 0.7,
				Structured: map[string]any{"ticker": "AMD", "direction": "bearish", "market": "us_stock"},
			},
			CreatedAt: time.Now().UTC().Add(-90 * time.Minute),
		},
	}

	resolutions := []core.ClaimResolution{
		{
			ClaimID: rootID,
			Domain:  "finance.us_stock",
			State:   core.ResolutionStateResolved,
			Outcome: boolPtr(true),
		},
		{
			ClaimID:        "claim-challenged",
			Domain:         "finance.us_stock",
			State:          core.ResolutionStateChallenged,
			ChallengeCount: 1,
		},
	}

	attestations := []repository.ResolutionAttestationRow{
		{
			Attestation: core.ResolutionAttestation{
				ID:      "resolve-1",
				ClaimID: rootID,
				AgentID: "agent-resolver",
				Kind:    core.ResolutionAttestationResolve,
				Verdict: boolPtr(true),
			},
			TrustScore: 0.8,
		},
		{
			Attestation: core.ResolutionAttestation{
				ID:      "challenge-1",
				ClaimID: "claim-challenged",
				AgentID: "agent-challenger",
				Kind:    core.ResolutionAttestationChallenge,
			},
			TrustScore: 0.7,
		},
	}

	agents := &trustAgentRepoStub{}
	signalRepo := &trustSignalRepoStub{signals: signals}
	resolutionRepo := &trustResolutionRepoStub{resolutions: resolutions, attestations: attestations}
	service := NewTrustService(agents, signalRepo, resolutionRepo)

	if err := service.Recalculate(ctx); err != nil {
		t.Fatalf("Recalculate: %v", err)
	}

	claimRecord := agents.trackRecord("agent-claim", "finance.us_stock")
	if claimRecord == nil || claimRecord.TotalClaims != 1 || claimRecord.CorrectClaims != 1 {
		t.Fatalf("expected claim stats to be recorded, got %+v", claimRecord)
	}

	counterRecord := agents.trackRecord("agent-counter", "finance.us_stock")
	if counterRecord == nil || counterRecord.TotalCounters != 1 || counterRecord.CorrectCounters != 0 {
		t.Fatalf("expected counter stats to be recorded, got %+v", counterRecord)
	}

	resolverProfile := agents.profile("agent-resolver")
	if resolverProfile.ResolverTrust <= 0.5 {
		t.Fatalf("expected resolver trust to increase, got %+v", resolverProfile)
	}

	challengerRecord := agents.trackRecord("agent-challenger", "finance.us_stock")
	if challengerRecord == nil || challengerRecord.TotalChallenges != 1 || challengerRecord.SuccessfulChallenges != 1 {
		t.Fatalf("expected successful challenge stats, got %+v", challengerRecord)
	}
}

type trustAgentRepoStub struct {
	records  map[string]core.AgentTrackRecord
	profiles map[string]core.AgentTrustProfile
	scores   map[string]float64
}

func (s *trustAgentRepoStub) Create(context.Context, *core.Agent) error { return nil }
func (s *trustAgentRepoStub) GetByID(context.Context, string) (*core.Agent, error) {
	return nil, pgx.ErrNoRows
}
func (s *trustAgentRepoStub) GetByAPIKeyHash(context.Context, string) (*core.Agent, error) {
	return nil, pgx.ErrNoRows
}
func (s *trustAgentRepoStub) ListPublic(context.Context, int) ([]core.Agent, error) { return nil, nil }
func (s *trustAgentRepoStub) ListTrackRecords(context.Context, string) ([]core.AgentTrackRecord, error) {
	return nil, nil
}
func (s *trustAgentRepoStub) Update(context.Context, *core.Agent) error { return nil }
func (s *trustAgentRepoStub) UpdateTrustProfile(_ context.Context, agentID string, trustScore float64, profile core.AgentTrustProfile) error {
	if s.profiles == nil {
		s.profiles = map[string]core.AgentTrustProfile{}
	}
	if s.scores == nil {
		s.scores = map[string]float64{}
	}
	s.profiles[agentID] = profile
	s.scores[agentID] = trustScore
	return nil
}
func (s *trustAgentRepoStub) UpsertTrackRecord(_ context.Context, rec core.AgentTrackRecord) error {
	if s.records == nil {
		s.records = map[string]core.AgentTrackRecord{}
	}
	s.records[rec.AgentID+"|"+rec.Domain] = rec
	return nil
}
func (s *trustAgentRepoStub) trackRecord(agentID, domain string) *core.AgentTrackRecord {
	rec, ok := s.records[agentID+"|"+domain]
	if !ok {
		return nil
	}
	return &rec
}
func (s *trustAgentRepoStub) profile(agentID string) core.AgentTrustProfile {
	return s.profiles[agentID]
}

type trustSignalRepoStub struct {
	signals []core.Signal
}

func (s *trustSignalRepoStub) Create(context.Context, *core.Signal) error { return nil }
func (s *trustSignalRepoStub) GetByID(context.Context, string) (*core.Signal, error) {
	return nil, pgx.ErrNoRows
}
func (s *trustSignalRepoStub) List(context.Context, repository.ListSignalsParams) ([]core.Signal, error) {
	return nil, nil
}
func (s *trustSignalRepoStub) ListAll(context.Context) ([]core.Signal, error) { return s.signals, nil }
func (s *trustSignalRepoStub) ListCounters(context.Context, string) ([]core.Signal, error) {
	return nil, nil
}
func (s *trustSignalRepoStub) ListPendingVerification(context.Context, time.Time) ([]repository.VerificationCandidate, error) {
	return nil, nil
}
func (s *trustSignalRepoStub) MarkVerified(context.Context, string, bool, map[string]any, time.Time) error {
	return nil
}
func (s *trustSignalRepoStub) ListConsensusRows(context.Context, string, *time.Duration) ([]repository.ConsensusRow, error) {
	return nil, nil
}
func (s *trustSignalRepoStub) ListOverviewRows(context.Context, string, int) ([]repository.ConsensusRow, error) {
	return nil, nil
}
func (s *trustSignalRepoStub) ListAgentDomainStats(context.Context) ([]repository.AgentDomainStats, error) {
	return nil, nil
}

type trustResolutionRepoStub struct {
	resolutions  []core.ClaimResolution
	attestations []repository.ResolutionAttestationRow
}

func (s *trustResolutionRepoStub) CreateAttestation(context.Context, *core.ResolutionAttestation) error {
	return nil
}
func (s *trustResolutionRepoStub) ListAttestations(context.Context, string) ([]repository.ResolutionAttestationRow, error) {
	return nil, nil
}
func (s *trustResolutionRepoStub) ListAllAttestations(context.Context) ([]repository.ResolutionAttestationRow, error) {
	return s.attestations, nil
}
func (s *trustResolutionRepoStub) GetClaimResolution(context.Context, string) (*core.ClaimResolution, error) {
	return nil, pgx.ErrNoRows
}
func (s *trustResolutionRepoStub) ListAllClaimResolutions(context.Context) ([]core.ClaimResolution, error) {
	return s.resolutions, nil
}
func (s *trustResolutionRepoStub) UpsertClaimResolution(context.Context, *core.ClaimResolution) error {
	return nil
}
