package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openpup/agora/internal/domain"
)

type ListSignalsParams struct {
	Market        domain.Market
	Ticker        *string
	SignalType    *domain.SignalType
	AgentID       *string
	MinConfidence *float64
	Since         *time.Time
	Cursor        *domain.SignalListCursor
	Limit         int
}

type ConsensusRow struct {
	Ticker     string
	Direction  string
	Confidence float64
	TrustScore float64
	SignalID   string
	AgentID    string
	CreatedAt  time.Time
}

type VerificationCandidate struct {
	SignalID  string
	Ticker    string
	Market    domain.Market
	Direction domain.Direction
	CreatedAt time.Time
	ExpiresAt time.Time
}

type AgentMarketStats struct {
	AgentID            string
	Market             domain.Market
	TotalPredictions   int
	CorrectPredictions int
	AvgConfidence      float64
}

type SignalRepository interface {
	Create(context.Context, *domain.Signal) error
	GetByID(context.Context, string) (*domain.Signal, error)
	List(context.Context, ListSignalsParams) ([]domain.Signal, error)
	ListCounters(context.Context, string) ([]domain.Signal, error)
	ListPendingVerification(context.Context, time.Time) ([]VerificationCandidate, error)
	MarkVerified(context.Context, string, bool, map[string]any, time.Time) error
	ListConsensusRows(context.Context, domain.Market, string, *time.Duration) ([]ConsensusRow, error)
	ListOverviewRows(context.Context, domain.Market, int) ([]ConsensusRow, error)
	ListAgentMarketStats(context.Context) ([]AgentMarketStats, error)
}

type PGSignalRepository struct {
	pool *pgxpool.Pool
}

func NewPGSignalRepository(pool *pgxpool.Pool) *PGSignalRepository {
	return &PGSignalRepository{pool: pool}
}

func (r *PGSignalRepository) Create(ctx context.Context, signal *domain.Signal) error {
	reasoning, _ := json.Marshal(signal.Reasoning)
	dataRefs, _ := json.Marshal(signal.DataRefs)
	meta, _ := json.Marshal(signal.Meta)
	disagreement, _ := json.Marshal(signal.DisagreementPoints)
	_, err := r.pool.Exec(ctx, `
		INSERT INTO signals
		(id, agent_id, parent_id, market, signal_type, ticker, direction, confidence, time_horizon, expires_at, reasoning, data_refs, meta, verification_detail, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
	`, signal.ID, signal.AgentID, signal.ParentID, signal.Market, signal.SignalType, signal.Ticker, signal.Direction, signal.Confidence, durationToInterval(signal.TimeHorizon), signal.ExpiresAt, mergeReasoning(reasoning, disagreement), dataRefs, meta, nil, signal.CreatedAt)
	if err != nil {
		return fmt.Errorf("signal_repo.Create: %w", err)
	}
	return nil
}

func durationToInterval(d *time.Duration) *string {
	if d == nil {
		return nil
	}
	s := d.String()
	return &s
}

func mergeReasoning(reasoning, disagreement []byte) []byte {
	var payload map[string]any
	_ = json.Unmarshal(reasoning, &payload)
	var points []map[string]any
	_ = json.Unmarshal(disagreement, &points)
	if len(points) > 0 {
		payload["disagreement_points"] = points
	}
	out, _ := json.Marshal(payload)
	return out
}

func (r *PGSignalRepository) GetByID(ctx context.Context, id string) (*domain.Signal, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, agent_id, parent_id, market, signal_type, ticker, direction, confidence, time_horizon::text, expires_at, reasoning, data_refs, meta, verified, verified_at, verification_detail, created_at
		FROM signals WHERE id=$1
	`, id)
	signal, err := scanSignal(row)
	if err != nil {
		return nil, fmt.Errorf("signal_repo.GetByID: %w", err)
	}
	return signal, nil
}

func scanSignal(row pgx.Row) (*domain.Signal, error) {
	var s domain.Signal
	var direction *string
	var confidence *float64
	var horizon *string
	var reasoningRaw, dataRefsRaw, metaRaw []byte
	var verificationRaw []byte
	if err := row.Scan(&s.ID, &s.AgentID, &s.ParentID, &s.Market, &s.SignalType, &s.Ticker, &direction, &confidence, &horizon, &s.ExpiresAt, &reasoningRaw, &dataRefsRaw, &metaRaw, &s.Verified, &s.VerifiedAt, &verificationRaw, &s.CreatedAt); err != nil {
		return nil, err
	}
	if direction != nil {
		parsed := domain.Direction(*direction)
		s.Direction = &parsed
	}
	s.Confidence = confidence
	if horizon != nil && *horizon != "" {
		if d, err := time.ParseDuration(strings.ReplaceAll(*horizon, " ", "")); err == nil {
			s.TimeHorizon = &d
		}
	}
	var reasoningPayload map[string]json.RawMessage
	if err := json.Unmarshal(reasoningRaw, &reasoningPayload); err == nil {
		_ = json.Unmarshal(reasoningPayload["factors"], &s.Reasoning.Factors)
		_ = json.Unmarshal(reasoningPayload["summary"], &s.Reasoning.Summary)
		_ = json.Unmarshal(reasoningPayload["disagreement_points"], &s.DisagreementPoints)
	}
	_ = json.Unmarshal(dataRefsRaw, &s.DataRefs)
	_ = json.Unmarshal(metaRaw, &s.Meta)
	_ = json.Unmarshal(verificationRaw, &s.VerificationDetail)
	return &s, nil
}

func (r *PGSignalRepository) List(ctx context.Context, params ListSignalsParams) ([]domain.Signal, error) {
	args := []any{params.Market}
	where := []string{"market=$1"}
	argPos := 2
	if params.Ticker != nil {
		where = append(where, fmt.Sprintf("ticker=$%d", argPos))
		args = append(args, *params.Ticker)
		argPos++
	}
	if params.SignalType != nil {
		where = append(where, fmt.Sprintf("signal_type=$%d", argPos))
		args = append(args, *params.SignalType)
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
		SELECT id, agent_id, parent_id, market, signal_type, ticker, direction, confidence, time_horizon::text, expires_at, reasoning, data_refs, meta, verified, verified_at, verification_detail, created_at
		FROM signals WHERE %s ORDER BY created_at DESC, id DESC LIMIT $%d
	`, strings.Join(where, " AND "), argPos)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("signal_repo.List: %w", err)
	}
	defer rows.Close()
	var out []domain.Signal
	for rows.Next() {
		s, err := scanSignal(rows)
		if err != nil {
			return nil, fmt.Errorf("signal_repo.List scan: %w", err)
		}
		out = append(out, *s)
	}
	return out, nil
}

func (r *PGSignalRepository) ListCounters(ctx context.Context, parentID string) ([]domain.Signal, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, agent_id, parent_id, market, signal_type, ticker, direction, confidence, time_horizon::text, expires_at, reasoning, data_refs, meta, verified, verified_at, verification_detail, created_at
		FROM signals WHERE parent_id=$1 ORDER BY created_at ASC
	`, parentID)
	if err != nil {
		return nil, fmt.Errorf("signal_repo.ListCounters: %w", err)
	}
	defer rows.Close()
	var out []domain.Signal
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
		SELECT id, ticker, market, direction, created_at, expires_at
		FROM signals
		WHERE signal_type='prediction' AND verified IS NULL AND expires_at IS NOT NULL AND expires_at < $1
	`, cutoff)
	if err != nil {
		return nil, fmt.Errorf("signal_repo.ListPendingVerification: %w", err)
	}
	defer rows.Close()
	var out []VerificationCandidate
	for rows.Next() {
		var row VerificationCandidate
		if err := rows.Scan(&row.SignalID, &row.Ticker, &row.Market, &row.Direction, &row.CreatedAt, &row.ExpiresAt); err != nil {
			return nil, fmt.Errorf("signal_repo.ListPendingVerification scan: %w", err)
		}
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

func (r *PGSignalRepository) ListConsensusRows(ctx context.Context, market domain.Market, ticker string, _ *time.Duration) ([]ConsensusRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT s.ticker, s.direction, COALESCE(s.confidence, 0), a.trust_score, s.id, s.agent_id, s.created_at
		FROM signals s JOIN agents a ON a.id=s.agent_id
		WHERE s.market=$1 AND s.ticker=$2 AND s.signal_type='prediction'
		ORDER BY s.created_at DESC
	`, market, ticker)
	if err != nil {
		return nil, fmt.Errorf("signal_repo.ListConsensusRows: %w", err)
	}
	defer rows.Close()
	var out []ConsensusRow
	for rows.Next() {
		var row ConsensusRow
		if err := rows.Scan(&row.Ticker, &row.Direction, &row.Confidence, &row.TrustScore, &row.SignalID, &row.AgentID, &row.CreatedAt); err != nil {
			return nil, fmt.Errorf("signal_repo.ListConsensusRows scan: %w", err)
		}
		out = append(out, row)
	}
	return out, nil
}

func (r *PGSignalRepository) ListOverviewRows(ctx context.Context, market domain.Market, limit int) ([]ConsensusRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT s.ticker, s.direction, COALESCE(s.confidence, 0), a.trust_score, s.id, s.agent_id, s.created_at
		FROM signals s JOIN agents a ON a.id=s.agent_id
		WHERE s.market=$1 AND s.signal_type='prediction'
		ORDER BY s.created_at DESC
		LIMIT $2
	`, market, limit*25)
	if err != nil {
		return nil, fmt.Errorf("signal_repo.ListOverviewRows: %w", err)
	}
	defer rows.Close()
	var out []ConsensusRow
	for rows.Next() {
		var row ConsensusRow
		if err := rows.Scan(&row.Ticker, &row.Direction, &row.Confidence, &row.TrustScore, &row.SignalID, &row.AgentID, &row.CreatedAt); err != nil {
			return nil, fmt.Errorf("signal_repo.ListOverviewRows scan: %w", err)
		}
		out = append(out, row)
	}
	return out, nil
}

func (r *PGSignalRepository) ListAgentMarketStats(ctx context.Context) ([]AgentMarketStats, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT agent_id, market,
			COUNT(*) FILTER (WHERE signal_type='prediction')::int AS total_predictions,
			COUNT(*) FILTER (WHERE signal_type='prediction' AND verified = TRUE)::int AS correct_predictions,
			COALESCE(AVG(confidence) FILTER (WHERE signal_type='prediction' AND confidence IS NOT NULL), 0)::float8 AS avg_confidence
		FROM signals
		GROUP BY agent_id, market
	`)
	if err != nil {
		return nil, fmt.Errorf("signal_repo.ListAgentMarketStats: %w", err)
	}
	defer rows.Close()
	var out []AgentMarketStats
	for rows.Next() {
		var row AgentMarketStats
		if err := rows.Scan(&row.AgentID, &row.Market, &row.TotalPredictions, &row.CorrectPredictions, &row.AvgConfidence); err != nil {
			return nil, fmt.Errorf("signal_repo.ListAgentMarketStats scan: %w", err)
		}
		out = append(out, row)
	}
	return out, nil
}
