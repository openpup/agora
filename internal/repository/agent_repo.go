package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openpup/agora/internal/core"
)

type AgentRepository interface {
	Create(context.Context, *core.Agent) error
	GetByID(context.Context, string) (*core.Agent, error)
	GetByAPIKeyHash(context.Context, string) (*core.Agent, error)
	ListPublic(context.Context, int) ([]core.Agent, error)
	ListTrackRecords(context.Context, string) ([]core.AgentTrackRecord, error)
	Update(context.Context, *core.Agent) error
	UpdateTrustProfile(context.Context, string, float64, core.AgentTrustProfile) error
	UpsertTrackRecord(context.Context, core.AgentTrackRecord) error
}

type PGAgentRepository struct {
	pool *pgxpool.Pool
}

func NewPGAgentRepository(pool *pgxpool.Pool) *PGAgentRepository {
	return &PGAgentRepository{pool: pool}
}

func (r *PGAgentRepository) Create(ctx context.Context, agent *core.Agent) error {
	capabilities, _ := json.Marshal(agent.Capabilities)
	dataSources, _ := json.Marshal(agent.DataSources)
	metadata, _ := json.Marshal(agent.Metadata)
	_, err := r.pool.Exec(ctx, `
		INSERT INTO agents (id, name, api_key_hash, capabilities, data_sources, trust_score, claim_trust, counter_trust, resolver_trust, challenge_trust, metadata, status, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
	`, agent.ID, agent.Name, agent.APIKeyHash, capabilities, dataSources, agent.TrustScore, agent.TrustProfile.ClaimTrust, agent.TrustProfile.CounterTrust, agent.TrustProfile.ResolverTrust, agent.TrustProfile.ChallengeTrust, metadata, agent.Status, agent.CreatedAt, agent.UpdatedAt)
	if err != nil {
		return fmt.Errorf("agent_repo.Create: %w", err)
	}
	return nil
}

func (r *PGAgentRepository) GetByID(ctx context.Context, id string) (*core.Agent, error) {
	return r.getOne(ctx, "SELECT id, name, api_key_hash, capabilities, data_sources, trust_score, claim_trust, counter_trust, resolver_trust, challenge_trust, metadata, status, created_at, updated_at FROM agents WHERE id=$1", id)
}

func (r *PGAgentRepository) GetByAPIKeyHash(ctx context.Context, apiKeyHash string) (*core.Agent, error) {
	return r.getOne(ctx, "SELECT id, name, api_key_hash, capabilities, data_sources, trust_score, claim_trust, counter_trust, resolver_trust, challenge_trust, metadata, status, created_at, updated_at FROM agents WHERE api_key_hash=$1", apiKeyHash)
}

func (r *PGAgentRepository) ListPublic(ctx context.Context, limit int) ([]core.Agent, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, api_key_hash, capabilities, data_sources, trust_score, claim_trust, counter_trust, resolver_trust, challenge_trust, metadata, status, created_at, updated_at
		FROM agents
		WHERE status = 'active'
		ORDER BY trust_score DESC, created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("agent_repo.ListPublic: %w", err)
	}
	defer rows.Close()

	var out []core.Agent
	for rows.Next() {
		var agent core.Agent
		var capabilities []byte
		var dataSources []byte
		var metadata []byte
		if err := rows.Scan(&agent.ID, &agent.Name, &agent.APIKeyHash, &capabilities, &dataSources, &agent.TrustScore, &agent.TrustProfile.ClaimTrust, &agent.TrustProfile.CounterTrust, &agent.TrustProfile.ResolverTrust, &agent.TrustProfile.ChallengeTrust, &metadata, &agent.Status, &agent.CreatedAt, &agent.UpdatedAt); err != nil {
			return nil, fmt.Errorf("agent_repo.ListPublic scan: %w", err)
		}
		if err := json.Unmarshal(capabilities, &agent.Capabilities); err != nil {
			return nil, fmt.Errorf("agent_repo.ListPublic capabilities: %w", err)
		}
		if err := json.Unmarshal(dataSources, &agent.DataSources); err != nil {
			return nil, fmt.Errorf("agent_repo.ListPublic data sources: %w", err)
		}
		if err := json.Unmarshal(metadata, &agent.Metadata); err != nil {
			return nil, fmt.Errorf("agent_repo.ListPublic metadata: %w", err)
		}
		out = append(out, agent)
	}
	return out, nil
}

func (r *PGAgentRepository) getOne(ctx context.Context, query string, arg string) (*core.Agent, error) {
	row := r.pool.QueryRow(ctx, query, arg)
	var agent core.Agent
	var capabilities []byte
	var dataSources []byte
	var metadata []byte
	if err := row.Scan(&agent.ID, &agent.Name, &agent.APIKeyHash, &capabilities, &dataSources, &agent.TrustScore, &agent.TrustProfile.ClaimTrust, &agent.TrustProfile.CounterTrust, &agent.TrustProfile.ResolverTrust, &agent.TrustProfile.ChallengeTrust, &metadata, &agent.Status, &agent.CreatedAt, &agent.UpdatedAt); err != nil {
		return nil, fmt.Errorf("agent_repo.getOne: %w", err)
	}
	if err := json.Unmarshal(capabilities, &agent.Capabilities); err != nil {
		return nil, fmt.Errorf("agent_repo.getOne capabilities: %w", err)
	}
	if err := json.Unmarshal(dataSources, &agent.DataSources); err != nil {
		return nil, fmt.Errorf("agent_repo.getOne data sources: %w", err)
	}
	if err := json.Unmarshal(metadata, &agent.Metadata); err != nil {
		return nil, fmt.Errorf("agent_repo.getOne metadata: %w", err)
	}
	return &agent, nil
}

func (r *PGAgentRepository) ListTrackRecords(ctx context.Context, agentID string) ([]core.AgentTrackRecord, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT agent_id, domain, total_claims, correct_claims, accuracy, total_counters, correct_counters, counter_accuracy,
			total_resolutions, aligned_resolutions, resolution_accuracy,
			total_challenges, successful_challenges, challenge_accuracy,
			claim_trust, counter_trust, resolver_trust, challenge_trust,
			avg_confidence, last_calculated_at
		FROM agent_track_records WHERE agent_id=$1 ORDER BY domain
	`, agentID)
	if err != nil {
		return nil, fmt.Errorf("agent_repo.ListTrackRecords: %w", err)
	}
	defer rows.Close()

	var out []core.AgentTrackRecord
	for rows.Next() {
		var rec core.AgentTrackRecord
		if err := rows.Scan(
			&rec.AgentID,
			&rec.Domain,
			&rec.TotalClaims,
			&rec.CorrectClaims,
			&rec.ClaimAccuracy,
			&rec.TotalCounters,
			&rec.CorrectCounters,
			&rec.CounterAccuracy,
			&rec.TotalResolutions,
			&rec.AlignedResolutions,
			&rec.ResolutionAccuracy,
			&rec.TotalChallenges,
			&rec.SuccessfulChallenges,
			&rec.ChallengeAccuracy,
			&rec.ClaimTrust,
			&rec.CounterTrust,
			&rec.ResolverTrust,
			&rec.ChallengeTrust,
			&rec.AvgConfidence,
			&rec.LastCalculatedAt,
		); err != nil {
			return nil, fmt.Errorf("agent_repo.ListTrackRecords scan: %w", err)
		}
		rec.TotalPredictions = rec.TotalClaims
		rec.CorrectPredictions = rec.CorrectClaims
		rec.Accuracy = rec.ClaimAccuracy
		out = append(out, rec)
	}
	return out, nil
}

func (r *PGAgentRepository) Update(ctx context.Context, agent *core.Agent) error {
	capabilities, _ := json.Marshal(agent.Capabilities)
	dataSources, _ := json.Marshal(agent.DataSources)
	metadata, _ := json.Marshal(agent.Metadata)
	_, err := r.pool.Exec(ctx, `
		UPDATE agents SET name=$2, capabilities=$3, data_sources=$4, metadata=$5, updated_at=$6 WHERE id=$1
	`, agent.ID, agent.Name, capabilities, dataSources, metadata, agent.UpdatedAt)
	if err != nil {
		return fmt.Errorf("agent_repo.Update: %w", err)
	}
	return nil
}

func (r *PGAgentRepository) UpdateTrustProfile(ctx context.Context, agentID string, trustScore float64, profile core.AgentTrustProfile) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE agents
		SET trust_score=$2, claim_trust=$3, counter_trust=$4, resolver_trust=$5, challenge_trust=$6, updated_at=NOW()
		WHERE id=$1
	`, agentID, trustScore, profile.ClaimTrust, profile.CounterTrust, profile.ResolverTrust, profile.ChallengeTrust)
	if err != nil {
		return fmt.Errorf("agent_repo.UpdateTrustProfile: %w", err)
	}
	return nil
}

func (r *PGAgentRepository) UpsertTrackRecord(ctx context.Context, rec core.AgentTrackRecord) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO agent_track_records (
			agent_id, domain, total_claims, correct_claims, accuracy,
			total_counters, correct_counters, counter_accuracy,
			total_resolutions, aligned_resolutions, resolution_accuracy,
			total_challenges, successful_challenges, challenge_accuracy,
			claim_trust, counter_trust, resolver_trust, challenge_trust,
			avg_confidence, last_calculated_at
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20)
		ON CONFLICT (agent_id, domain) DO UPDATE SET
			total_claims=EXCLUDED.total_claims,
			correct_claims=EXCLUDED.correct_claims,
			accuracy=EXCLUDED.accuracy,
			total_counters=EXCLUDED.total_counters,
			correct_counters=EXCLUDED.correct_counters,
			counter_accuracy=EXCLUDED.counter_accuracy,
			total_resolutions=EXCLUDED.total_resolutions,
			aligned_resolutions=EXCLUDED.aligned_resolutions,
			resolution_accuracy=EXCLUDED.resolution_accuracy,
			total_challenges=EXCLUDED.total_challenges,
			successful_challenges=EXCLUDED.successful_challenges,
			challenge_accuracy=EXCLUDED.challenge_accuracy,
			claim_trust=EXCLUDED.claim_trust,
			counter_trust=EXCLUDED.counter_trust,
			resolver_trust=EXCLUDED.resolver_trust,
			challenge_trust=EXCLUDED.challenge_trust,
			avg_confidence=EXCLUDED.avg_confidence,
			last_calculated_at=EXCLUDED.last_calculated_at
	`, rec.AgentID, rec.Domain, rec.TotalClaims, rec.CorrectClaims, rec.ClaimAccuracy, rec.TotalCounters, rec.CorrectCounters, rec.CounterAccuracy, rec.TotalResolutions, rec.AlignedResolutions, rec.ResolutionAccuracy, rec.TotalChallenges, rec.SuccessfulChallenges, rec.ChallengeAccuracy, rec.ClaimTrust, rec.CounterTrust, rec.ResolverTrust, rec.ChallengeTrust, rec.AvgConfidence, rec.LastCalculatedAt)
	if err != nil {
		return fmt.Errorf("agent_repo.UpsertTrackRecord: %w", err)
	}
	return nil
}
