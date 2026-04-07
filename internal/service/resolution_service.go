package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/openpup/agora/internal/core"
	"github.com/openpup/agora/internal/repository"
)

type ResolutionService struct {
	resolutions repository.ResolutionRepository
	signals     repository.SignalRepository
}

type SubmitResolutionInput struct {
	ClaimID    string
	AgentID    string
	Kind       core.ResolutionAttestationKind
	Verdict    *bool
	Confidence float64
	Reasoning  core.Reasoning
	Evidence   []core.Evidence
	Meta       map[string]any
}

func NewResolutionService(resolutions repository.ResolutionRepository, signals repository.SignalRepository) *ResolutionService {
	return &ResolutionService{resolutions: resolutions, signals: signals}
}

func (s *ResolutionService) Submit(ctx context.Context, input SubmitResolutionInput) (*core.ClaimResolution, *core.ResolutionAttestation, error) {
	if err := validateResolutionInput(input); err != nil {
		return nil, nil, err
	}
	claim, err := s.signals.GetByID(ctx, input.ClaimID)
	if err != nil {
		return nil, nil, fmt.Errorf("resolution_service.Submit claim: %w", err)
	}
	if claim.Kind != core.SignalKindClaim {
		return nil, nil, fmt.Errorf("resolution_service.Submit: claim_id must reference a claim")
	}

	att := &core.ResolutionAttestation{
		ID:         uuid.NewString(),
		ClaimID:    input.ClaimID,
		AgentID:    input.AgentID,
		Kind:       input.Kind,
		Verdict:    input.Verdict,
		Confidence: input.Confidence,
		Reasoning:  input.Reasoning,
		Evidence:   input.Evidence,
		Meta:       input.Meta,
		CreatedAt:  time.Now().UTC(),
	}
	if err := s.resolutions.CreateAttestation(ctx, att); err != nil {
		return nil, nil, fmt.Errorf("resolution_service.Submit create attestation: %w", err)
	}

	resolution, err := s.recompute(ctx, claim)
	if err != nil {
		return nil, nil, fmt.Errorf("resolution_service.Submit recompute: %w", err)
	}
	return resolution, att, nil
}

func (s *ResolutionService) GetByClaimID(ctx context.Context, claimID string) (*core.ClaimResolution, error) {
	resolution, err := s.resolutions.GetClaimResolution(ctx, claimID)
	if err == nil {
		attestations, attErr := s.resolutions.ListAttestations(ctx, claimID)
		if attErr == nil {
			resolution.Attestations = mapAttestations(attestations)
		}
		return resolution, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("resolution_service.GetByClaimID: %w", err)
	}
	claim, claimErr := s.signals.GetByID(ctx, claimID)
	if claimErr != nil {
		return nil, fmt.Errorf("resolution_service.GetByClaimID claim: %w", claimErr)
	}
	attestations, attErr := s.resolutions.ListAttestations(ctx, claimID)
	if attErr != nil {
		return nil, fmt.Errorf("resolution_service.GetByClaimID attestations: %w", attErr)
	}
	return &core.ClaimResolution{
		ClaimID:        claim.ID,
		Domain:         claim.Domain,
		Strategy:       resolutionStrategy(claim),
		State:          core.ResolutionStateOpen,
		ResolverCount:  countKind(attestations, core.ResolutionAttestationResolve),
		ChallengeCount: countKind(attestations, core.ResolutionAttestationChallenge),
		Summary:        map[string]any{},
		UpdatedAt:      time.Now().UTC(),
		Attestations:   mapAttestations(attestations),
	}, nil
}

func (s *ResolutionService) recompute(ctx context.Context, claim *core.Signal) (*core.ClaimResolution, error) {
	rows, err := s.resolutions.ListAttestations(ctx, claim.ID)
	if err != nil {
		return nil, fmt.Errorf("resolution_service.recompute list: %w", err)
	}

	var supportWeight float64
	var rejectWeight float64
	var resolverCount int
	var challengeCount int
	for _, row := range rows {
		switch row.Attestation.Kind {
		case core.ResolutionAttestationResolve:
			resolverCount++
			weight := math.Max(row.TrustScore, 0.2) * math.Max(row.Attestation.Confidence, 0.1)
			if row.Attestation.Verdict != nil && *row.Attestation.Verdict {
				supportWeight += weight
			} else if row.Attestation.Verdict != nil {
				rejectWeight += weight
			}
		case core.ResolutionAttestationChallenge:
			challengeCount++
		}
	}

	var outcome *bool
	state := core.ResolutionStateOpen
	score := 0.0
	totalWeight := supportWeight + rejectWeight
	if totalWeight > 0 {
		score = (supportWeight - rejectWeight) / totalWeight
		if resolverCount >= 2 {
			v := supportWeight >= rejectWeight
			outcome = &v
			if challengeCount > 0 && math.Abs(score) < 0.20 {
				state = core.ResolutionStateChallenged
				outcome = nil
			} else {
				state = core.ResolutionStateResolved
			}
		}
	}

	var resolvedAt *time.Time
	if state == core.ResolutionStateResolved {
		now := time.Now().UTC()
		resolvedAt = &now
	}
	resolution := &core.ClaimResolution{
		ClaimID:         claim.ID,
		Domain:          claim.Domain,
		Strategy:        resolutionStrategy(claim),
		State:           state,
		Outcome:         outcome,
		ResolutionScore: score,
		ResolverCount:   resolverCount,
		ChallengeCount:  challengeCount,
		Summary: map[string]any{
			"support_weight": supportWeight,
			"reject_weight":  rejectWeight,
		},
		ResolvedAt:   resolvedAt,
		UpdatedAt:    time.Now().UTC(),
		Attestations: mapAttestations(rows),
	}
	if err := s.resolutions.UpsertClaimResolution(ctx, resolution); err != nil {
		return nil, fmt.Errorf("resolution_service.recompute upsert: %w", err)
	}
	if state == core.ResolutionStateResolved && outcome != nil {
		detail := map[string]any{
			"strategy":         resolution.Strategy,
			"resolution_score": resolution.ResolutionScore,
			"resolver_count":   resolverCount,
			"challenge_count":  challengeCount,
			"support_weight":   supportWeight,
			"reject_weight":    rejectWeight,
		}
		if err := s.signals.MarkVerified(ctx, claim.ID, *outcome, detail, *resolvedAt); err != nil {
			return nil, fmt.Errorf("resolution_service.recompute mark verified: %w", err)
		}
	}
	return resolution, nil
}

func validateResolutionInput(input SubmitResolutionInput) error {
	if input.ClaimID == "" || input.AgentID == "" {
		return fmt.Errorf("resolution_service.validateResolutionInput: claim_id and agent_id are required")
	}
	if input.Confidence <= 0 {
		return fmt.Errorf("resolution_service.validateResolutionInput: confidence must be greater than zero")
	}
	if input.Reasoning.Summary == "" || len(input.Reasoning.Factors) == 0 {
		return fmt.Errorf("resolution_service.validateResolutionInput: reasoning is required")
	}
	if input.Kind == core.ResolutionAttestationResolve && input.Verdict == nil {
		return fmt.Errorf("resolution_service.validateResolutionInput: resolve verdict is required")
	}
	return nil
}

func resolutionStrategy(claim *core.Signal) string {
	if claim.Claim.Resolution != nil && claim.Claim.Resolution.Strategy != "" {
		return claim.Claim.Resolution.Strategy
	}
	return "attested_consensus"
}

func mapAttestations(rows []repository.ResolutionAttestationRow) []core.ResolutionAttestation {
	out := make([]core.ResolutionAttestation, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.Attestation)
	}
	return out
}

func countKind(rows []repository.ResolutionAttestationRow, kind core.ResolutionAttestationKind) int {
	total := 0
	for _, row := range rows {
		if row.Attestation.Kind == kind {
			total++
		}
	}
	return total
}
