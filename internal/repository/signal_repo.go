package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openpup/agora/internal/core"
)

type ListSignalsParams struct {
	Domain        string
	Kind          *core.SignalKind
	AgentID       *string
	MinConfidence *float64
	Since         *time.Time
	Cursor        *core.SignalListCursor
	Limit         int
}

type ConsensusRow struct {
	SignalID   string
	AgentID    string
	Structured map[string]any
	Confidence float64
	TrustScore float64
	CreatedAt  time.Time
}

type VerificationCandidate struct {
	SignalID     string
	Domain       string
	Kind         core.SignalKind
	Structured   map[string]any
	Confidence   float64
	VerifiableBy time.Time
	CreatedAt    time.Time
}

type AgentDomainStats struct {
	AgentID            string
	Domain             string
	TotalPredictions   int
	CorrectPredictions int
	AvgConfidence      float64
}

type SignalRepository interface {
	Create(context.Context, *core.Signal) error
	GetByID(context.Context, string) (*core.Signal, error)
	List(context.Context, ListSignalsParams) ([]core.Signal, error)
	ListAll(context.Context) ([]core.Signal, error)
	ListCounters(context.Context, string) ([]core.Signal, error)
	ListPendingVerification(context.Context, time.Time) ([]VerificationCandidate, error)
	MarkVerified(context.Context, string, bool, map[string]any, time.Time) error
	ListConsensusRows(context.Context, string, *time.Duration) ([]ConsensusRow, error)
	ListOverviewRows(context.Context, string, int) ([]ConsensusRow, error)
	ListAgentDomainStats(context.Context) ([]AgentDomainStats, error)
}

type PGSignalRepository struct {
	pool *pgxpool.Pool
}

func NewPGSignalRepository(pool *pgxpool.Pool) *PGSignalRepository {
	return &PGSignalRepository{pool: pool}
}

func (r *PGSignalRepository) Create(ctx context.Context, signal *core.Signal) error {
	reasoning, _ := json.Marshal(signal.Reasoning)
	evidence, _ := json.Marshal(signal.Evidence)
	disagreement, _ := json.Marshal(signal.DisagreementPoints)
	refs, _ := json.Marshal(signal.Refs)
	meta, _ := json.Marshal(signal.Meta)
	structured, _ := json.Marshal(signal.Claim.Structured)
	var resolution []byte
	if signal.Claim.Resolution != nil {
		resolution, _ = json.Marshal(signal.Claim.Resolution)
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO signals
		(id, agent_id, parent_id, domain, kind, statement, structured, confidence, verifiable_by, resolution, reasoning, evidence, disagreement, refs, meta, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
	`, signal.ID, signal.AgentID, signal.ParentID, signal.Domain, signal.Kind,
		signal.Claim.Statement, structured, signal.Claim.Confidence, signal.Claim.VerifiableBy, resolution,
		reasoning, evidence, disagreement, refs, meta, signal.CreatedAt)
	if err != nil {
		return fmt.Errorf("signal_repo.Create: %w", err)
	}
	return nil
}

func (r *PGSignalRepository) GetByID(ctx context.Context, id string) (*core.Signal, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, agent_id, parent_id, domain, kind, statement, structured, confidence, verifiable_by, resolution,
			reasoning, evidence, disagreement, refs, meta, verified, verified_at, verification_detail, created_at
		FROM signals WHERE id=$1
	`, id)
	signal, err := scanSignal(row)
	if err != nil {
		return nil, fmt.Errorf("signal_repo.GetByID: %w", err)
	}
	return signal, nil
}

func scanSignal(row pgx.Row) (*core.Signal, error) {
	var s core.Signal
	var structured, reasoning, evidence, disagreement, refs, meta, verificationRaw, resolution []byte
	if err := row.Scan(
		&s.ID, &s.AgentID, &s.ParentID, &s.Domain, &s.Kind,
		&s.Claim.Statement, &structured, &s.Claim.Confidence, &s.Claim.VerifiableBy, &resolution,
		&reasoning, &evidence, &disagreement, &refs, &meta,
		&s.Verified, &s.VerifiedAt, &verificationRaw, &s.CreatedAt,
	); err != nil {
		return nil, err
	}
	_ = json.Unmarshal(structured, &s.Claim.Structured)
	if len(resolution) > 0 {
		s.Claim.Resolution = &core.Resolution{}
		_ = json.Unmarshal(resolution, s.Claim.Resolution)
	}
	_ = json.Unmarshal(reasoning, &s.Reasoning)
	_ = json.Unmarshal(evidence, &s.Evidence)
	_ = json.Unmarshal(disagreement, &s.DisagreementPoints)
	_ = json.Unmarshal(refs, &s.Refs)
	_ = json.Unmarshal(meta, &s.Meta)
	_ = json.Unmarshal(verificationRaw, &s.VerificationDetail)
	return &s, nil
}

func (r *PGSignalRepository) List(ctx context.Context, params ListSignalsParams) ([]core.Signal, error) {
	args := []any{params.Domain}
	where := []string{"domain=$1"}
	argPos := 2
	if params.Kind != nil {
		where = append(where, fmt.Sprintf("kind=$%d", argPos))
		args = append(args, *params.Kind)
		argPos++
	}
	if params.AgentID != nil {
		where = append(where, fmt.Sprintf("agent_id=$%d", argPos))
		args = append(args, *params.AgentID)
		argPos++
	}
	if params.MinConfidence != nil {
		where = append(where, fmt.Sprintf("confidence >= $%d", argPos))
		args = append(args, *params.MinConfidence)
		argPos++
	}
	if params.Since != nil {
		where = append(where, fmt.Sprintf("created_at >= $%d", argPos))
		args = append(args, *params.Since)
		argPos++
	}
	if params.Cursor != nil {
		where = append(where, fmt.Sprintf("(created_at, id) < ($%d, $%d)", argPos, argPos+1))
		args = append(args, params.Cursor.CreatedAt, params.Cursor.ID)
		argPos += 2
	}
	args = append(args, params.Limit)
	query := fmt.Sprintf(`
		SELECT id, agent_id, parent_id, domain, kind, statement, structured, confidence, verifiable_by, resolution,
			reasoning, evidence, disagreement, refs, meta, verified, verified_at, verification_detail, created_at
		FROM signals WHERE %s ORDER BY created_at DESC, id DESC LIMIT $%d
	`, strings.Join(where, " AND "), argPos)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("signal_repo.List: %w", err)
	}
	defer rows.Close()
	var out []core.Signal
	for rows.Next() {
		s, err := scanSignal(rows)
		if err != nil {
			return nil, fmt.Errorf("signal_repo.List scan: %w", err)
		}
		out = append(out, *s)
	}
	return out, nil
}

func (r *PGSignalRepository) ListAll(ctx context.Context) ([]core.Signal, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, agent_id, parent_id, domain, kind, statement, structured, confidence, verifiable_by, resolution,
			reasoning, evidence, disagreement, refs, meta, verified, verified_at, verification_detail, created_at
		FROM signals
		ORDER BY created_at DESC, id DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("signal_repo.ListAll: %w", err)
	}
	defer rows.Close()
	var out []core.Signal
	for rows.Next() {
		s, err := scanSignal(rows)
		if err != nil {
			return nil, fmt.Errorf("signal_repo.ListAll scan: %w", err)
		}
		out = append(out, *s)
	}
	return out, nil
}

func (r *PGSignalRepository) ListCounters(ctx context.Context, parentID string) ([]core.Signal, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, agent_id, parent_id, domain, kind, statement, structured, confidence, verifiable_by, resolution,
			reasoning, evidence, disagreement, refs, meta, verified, verified_at, verification_detail, created_at
		FROM signals WHERE parent_id=$1 ORDER BY created_at ASC
	`, parentID)
	if err != nil {
		return nil, fmt.Errorf("signal_repo.ListCounters: %w", err)
	}
	defer rows.Close()
	var out []core.Signal
	for rows.Next() {
		s, err := scanSignal(rows)
		if err != nil {
			return nil, fmt.Errorf("signal_repo.ListCounters scan: %w", err)
		}
		out = append(out, *s)
	}
	return out, nil
}

func (r *PGSignalRepository) ListPendingVerification(ctx context.Context, cutoff time.Time) ([]VerificationCandidate, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, domain, kind, structured, confidence, verifiable_by, created_at
		FROM signals
		WHERE kind='claim' AND verified IS NULL AND verifiable_by IS NOT NULL AND verifiable_by < $1
	`, cutoff)
	if err != nil {
		return nil, fmt.Errorf("signal_repo.ListPendingVerification: %w", err)
	}
	defer rows.Close()
	var out []VerificationCandidate
	for rows.Next() {
		var row VerificationCandidate
		var structured []byte
		if err := rows.Scan(&row.SignalID, &row.Domain, &row.Kind, &structured, &row.Confidence, &row.VerifiableBy, &row.CreatedAt); err != nil {
			return nil, fmt.Errorf("signal_repo.ListPendingVerification scan: %w", err)
		}
		_ = json.Unmarshal(structured, &row.Structured)
		out = append(out, row)
	}
	return out, nil
}

func (r *PGSignalRepository) MarkVerified(ctx context.Context, signalID string, verified bool, detail map[string]any, verifiedAt time.Time) error {
	payload, _ := json.Marshal(detail)
	_, err := r.pool.Exec(ctx, `
		UPDATE signals SET verified=$2, verified_at=$3, verification_detail=$4 WHERE id=$1
	`, signalID, verified, verifiedAt, payload)
	if err != nil {
		return fmt.Errorf("signal_repo.MarkVerified: %w", err)
	}
	return nil
}

func (r *PGSignalRepository) ListConsensusRows(ctx context.Context, domain string, _ *time.Duration) ([]ConsensusRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT s.id, s.agent_id, s.structured, s.confidence, a.trust_score, s.created_at
		FROM signals s JOIN agents a ON a.id=s.agent_id
		WHERE s.domain=$1 AND s.kind='claim'
		ORDER BY s.created_at DESC
	`, domain)
	if err != nil {
		return nil, fmt.Errorf("signal_repo.ListConsensusRows: %w", err)
	}
	defer rows.Close()
	var out []ConsensusRow
	for rows.Next() {
		var row ConsensusRow
		var structured []byte
		if err := rows.Scan(&row.SignalID, &row.AgentID, &structured, &row.Confidence, &row.TrustScore, &row.CreatedAt); err != nil {
			return nil, fmt.Errorf("signal_repo.ListConsensusRows scan: %w", err)
		}
		_ = json.Unmarshal(structured, &row.Structured)
		out = append(out, row)
	}
	return out, nil
}

func (r *PGSignalRepository) ListOverviewRows(ctx context.Context, domain string, limit int) ([]ConsensusRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT s.id, s.agent_id, s.structured, s.confidence, a.trust_score, s.created_at
		FROM signals s JOIN agents a ON a.id=s.agent_id
		WHERE s.domain=$1 AND s.kind='claim'
		ORDER BY s.created_at DESC
		LIMIT $2
	`, domain, limit*25)
	if err != nil {
		return nil, fmt.Errorf("signal_repo.ListOverviewRows: %w", err)
	}
	defer rows.Close()
	var out []ConsensusRow
	for rows.Next() {
		var row ConsensusRow
		var structured []byte
		if err := rows.Scan(&row.SignalID, &row.AgentID, &structured, &row.Confidence, &row.TrustScore, &row.CreatedAt); err != nil {
			return nil, fmt.Errorf("signal_repo.ListOverviewRows scan: %w", err)
		}
		_ = json.Unmarshal(structured, &row.Structured)
		out = append(out, row)
	}
	return out, nil
}

func (r *PGSignalRepository) ListAgentDomainStats(ctx context.Context) ([]AgentDomainStats, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT agent_id, domain,
			COUNT(*) FILTER (WHERE kind='claim')::int AS total_predictions,
			COUNT(*) FILTER (WHERE kind='claim' AND verified = TRUE)::int AS correct_predictions,
			COALESCE(AVG(confidence) FILTER (WHERE kind='claim'), 0)::float8 AS avg_confidence
		FROM signals
		GROUP BY agent_id, domain
	`)
	if err != nil {
		return nil, fmt.Errorf("signal_repo.ListAgentDomainStats: %w", err)
	}
	defer rows.Close()
	var out []AgentDomainStats
	for rows.Next() {
		var row AgentDomainStats
		if err := rows.Scan(&row.AgentID, &row.Domain, &row.TotalPredictions, &row.CorrectPredictions, &row.AvgConfidence); err != nil {
			return nil, fmt.Errorf("signal_repo.ListAgentDomainStats scan: %w", err)
		}
		out = append(out, row)
	}
	return out, nil
}
