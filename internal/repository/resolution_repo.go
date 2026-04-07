package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openpup/agora/internal/core"
)

type ResolutionAttestationRow struct {
	Attestation core.ResolutionAttestation
	TrustScore  float64
}

type ResolutionRepository interface {
	CreateAttestation(context.Context, *core.ResolutionAttestation) error
	ListAttestations(context.Context, string) ([]ResolutionAttestationRow, error)
	ListAllAttestations(context.Context) ([]ResolutionAttestationRow, error)
	GetClaimResolution(context.Context, string) (*core.ClaimResolution, error)
	ListAllClaimResolutions(context.Context) ([]core.ClaimResolution, error)
	UpsertClaimResolution(context.Context, *core.ClaimResolution) error
}

type PGResolutionRepository struct {
	pool *pgxpool.Pool
}

func NewPGResolutionRepository(pool *pgxpool.Pool) *PGResolutionRepository {
	return &PGResolutionRepository{pool: pool}
}

func (r *PGResolutionRepository) CreateAttestation(ctx context.Context, att *core.ResolutionAttestation) error {
	reasoning, _ := json.Marshal(att.Reasoning)
	evidence, _ := json.Marshal(att.Evidence)
	meta, _ := json.Marshal(att.Meta)
	_, err := r.pool.Exec(ctx, `
		INSERT INTO resolution_attestations (id, claim_id, agent_id, kind, verdict, confidence, reasoning, evidence, meta, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
	`, att.ID, att.ClaimID, att.AgentID, att.Kind, att.Verdict, att.Confidence, reasoning, evidence, meta, att.CreatedAt)
	if err != nil {
		return fmt.Errorf("resolution_repo.CreateAttestation: %w", err)
	}
	return nil
}

func (r *PGResolutionRepository) ListAttestations(ctx context.Context, claimID string) ([]ResolutionAttestationRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT ra.id, ra.claim_id, ra.agent_id, ra.kind, ra.verdict, ra.confidence, ra.reasoning, ra.evidence, ra.meta, ra.created_at, a.trust_score
		FROM resolution_attestations ra
		JOIN agents a ON a.id = ra.agent_id
		WHERE ra.claim_id = $1
		ORDER BY ra.created_at ASC
	`, claimID)
	if err != nil {
		return nil, fmt.Errorf("resolution_repo.ListAttestations: %w", err)
	}
	defer rows.Close()
	return scanAttestationRows(rows)
}

func (r *PGResolutionRepository) ListAllAttestations(ctx context.Context) ([]ResolutionAttestationRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT ra.id, ra.claim_id, ra.agent_id, ra.kind, ra.verdict, ra.confidence, ra.reasoning, ra.evidence, ra.meta, ra.created_at, a.trust_score
		FROM resolution_attestations ra
		JOIN agents a ON a.id = ra.agent_id
		ORDER BY ra.created_at ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("resolution_repo.ListAllAttestations: %w", err)
	}
	defer rows.Close()
	return scanAttestationRows(rows)
}

func scanAttestationRows(rows interface {
	Next() bool
	Scan(...any) error
	Close()
}) ([]ResolutionAttestationRow, error) {
	var out []ResolutionAttestationRow
	for rows.Next() {
		var row ResolutionAttestationRow
		var reasoning, evidence, meta []byte
		if err := rows.Scan(
			&row.Attestation.ID,
			&row.Attestation.ClaimID,
			&row.Attestation.AgentID,
			&row.Attestation.Kind,
			&row.Attestation.Verdict,
			&row.Attestation.Confidence,
			&reasoning,
			&evidence,
			&meta,
			&row.Attestation.CreatedAt,
			&row.TrustScore,
		); err != nil {
			return nil, fmt.Errorf("resolution_repo.ListAttestations scan: %w", err)
		}
		_ = json.Unmarshal(reasoning, &row.Attestation.Reasoning)
		_ = json.Unmarshal(evidence, &row.Attestation.Evidence)
		_ = json.Unmarshal(meta, &row.Attestation.Meta)
		out = append(out, row)
	}
	return out, nil
}

func (r *PGResolutionRepository) GetClaimResolution(ctx context.Context, claimID string) (*core.ClaimResolution, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT claim_id, domain, strategy, state, outcome, resolution_score, resolver_count, challenge_count, summary, resolved_at, updated_at
		FROM claim_resolutions
		WHERE claim_id = $1
	`, claimID)
	var resolution core.ClaimResolution
	var summary []byte
	if err := row.Scan(
		&resolution.ClaimID,
		&resolution.Domain,
		&resolution.Strategy,
		&resolution.State,
		&resolution.Outcome,
		&resolution.ResolutionScore,
		&resolution.ResolverCount,
		&resolution.ChallengeCount,
		&summary,
		&resolution.ResolvedAt,
		&resolution.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("resolution_repo.GetClaimResolution: %w", err)
	}
	_ = json.Unmarshal(summary, &resolution.Summary)
	return &resolution, nil
}

func (r *PGResolutionRepository) ListAllClaimResolutions(ctx context.Context) ([]core.ClaimResolution, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT claim_id, domain, strategy, state, outcome, resolution_score, resolver_count, challenge_count, summary, resolved_at, updated_at
		FROM claim_resolutions
		ORDER BY updated_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("resolution_repo.ListAllClaimResolutions: %w", err)
	}
	defer rows.Close()
	var out []core.ClaimResolution
	for rows.Next() {
		var resolution core.ClaimResolution
		var summary []byte
		if err := rows.Scan(
			&resolution.ClaimID,
			&resolution.Domain,
			&resolution.Strategy,
			&resolution.State,
			&resolution.Outcome,
			&resolution.ResolutionScore,
			&resolution.ResolverCount,
			&resolution.ChallengeCount,
			&summary,
			&resolution.ResolvedAt,
			&resolution.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("resolution_repo.ListAllClaimResolutions scan: %w", err)
		}
		_ = json.Unmarshal(summary, &resolution.Summary)
		out = append(out, resolution)
	}
	return out, nil
}

func (r *PGResolutionRepository) UpsertClaimResolution(ctx context.Context, resolution *core.ClaimResolution) error {
	summary, _ := json.Marshal(resolution.Summary)
	_, err := r.pool.Exec(ctx, `
		INSERT INTO claim_resolutions (claim_id, domain, strategy, state, outcome, resolution_score, resolver_count, challenge_count, summary, resolved_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT (claim_id) DO UPDATE SET
			domain = EXCLUDED.domain,
			strategy = EXCLUDED.strategy,
			state = EXCLUDED.state,
			outcome = EXCLUDED.outcome,
			resolution_score = EXCLUDED.resolution_score,
			resolver_count = EXCLUDED.resolver_count,
			challenge_count = EXCLUDED.challenge_count,
			summary = EXCLUDED.summary,
			resolved_at = EXCLUDED.resolved_at,
			updated_at = EXCLUDED.updated_at
	`, resolution.ClaimID, resolution.Domain, resolution.Strategy, resolution.State, resolution.Outcome, resolution.ResolutionScore, resolution.ResolverCount, resolution.ChallengeCount, summary, resolution.ResolvedAt, resolution.UpdatedAt)
	if err != nil {
		return fmt.Errorf("resolution_repo.UpsertClaimResolution: %w", err)
	}
	return nil
}

var _ time.Time
