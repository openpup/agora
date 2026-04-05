package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openpup/agora/internal/domain"
)

type AgentRepository interface {
	Create(context.Context, *domain.Agent) error
	GetByID(context.Context, string) (*domain.Agent, error)
	GetByAPIKeyHash(context.Context, string) (*domain.Agent, error)
	ListTrackRecords(context.Context, string) ([]domain.AgentTrackRecord, error)
	Update(context.Context, *domain.Agent) error
	UpdateTrustScore(context.Context, string, float64) error
	UpsertTrackRecord(context.Context, domain.AgentTrackRecord) error
}

type PGAgentRepository struct {
	pool *pgxpool.Pool
}

func NewPGAgentRepository(pool *pgxpool.Pool) *PGAgentRepository {
	return &PGAgentRepository{pool: pool}
}

func (r *PGAgentRepository) Create(ctx context.Context, agent *domain.Agent) error {
	capabilities, _ := json.Marshal(agent.Capabilities)
	dataSources, _ := json.Marshal(agent.DataSources)
	metadata, _ := json.Marshal(agent.Metadata)
	_, err := r.pool.Exec(ctx, `
		INSERT INTO agents (id, name, api_key_hash, capabilities, data_sources, trust_score, metadata, status, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
	`, agent.ID, agent.Name, agent.APIKeyHash, capabilities, dataSources, agent.TrustScore, metadata, agent.Status, agent.CreatedAt, agent.UpdatedAt)
	if err != nil {
		return fmt.Errorf("agent_repo.Create: %w", err)
	}
	return nil
}

func (r *PGAgentRepository) GetByID(ctx context.Context, id string) (*domain.Agent, error) {
	return r.getOne(ctx, "SELECT id, name, api_key_hash, capabilities, data_sources, trust_score, metadata, status, created_at, updated_at FROM agents WHERE id=$1", id)
}

func (r *PGAgentRepository) GetByAPIKeyHash(ctx context.Context, apiKeyHash string) (*domain.Agent, error) {
	return r.getOne(ctx, "SELECT id, name, api_key_hash, capabilities, data_sources, trust_score, metadata, status, created_at, updated_at FROM agents WHERE api_key_hash=$1", apiKeyHash)
}

func (r *PGAgentRepository) getOne(ctx context.Context, query string, arg string) (*domain.Agent, error) {
	row := r.pool.QueryRow(ctx, query, arg)
	var agent domain.Agent
	var capabilities []byte
	var dataSources []byte
	var metadata []byte
	if err := row.Scan(&agent.ID, &agent.Name, &agent.APIKeyHash, &capabilities, &dataSources, &agent.TrustScore, &metadata, &agent.Status, &agent.CreatedAt, &agent.UpdatedAt); err != nil {
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

func (r *PGAgentRepository) ListTrackRecords(ctx context.Context, agentID string) ([]domain.AgentTrackRecord, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT agent_id, market, total_predictions, correct_predictions, accuracy, avg_confidence, last_calculated_at
		FROM agent_track_records WHERE agent_id=$1 ORDER BY market
	`, agentID)
	if err != nil {
		return nil, fmt.Errorf("agent_repo.ListTrackRecords: %w", err)
	}
	defer rows.Close()

	var out []domain.AgentTrackRecord
	for rows.Next() {
		var rec domain.AgentTrackRecord
		if err := rows.Scan(&rec.AgentID, &rec.Market, &rec.TotalPredictions, &rec.CorrectPredictions, &rec.Accuracy, &rec.AvgConfidence, &rec.LastCalculatedAt); err != nil {
			return nil, fmt.Errorf("agent_repo.ListTrackRecords scan: %w", err)
		}
		out = append(out, rec)
	}
	return out, nil
}

func (r *PGAgentRepository) Update(ctx context.Context, agent *domain.Agent) error {
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

func (r *PGAgentRepository) UpdateTrustScore(ctx context.Context, agentID string, trustScore float64) error {
	_, err := r.pool.Exec(ctx, `UPDATE agents SET trust_score=$2, updated_at=NOW() WHERE id=$1`, agentID, trustScore)
	if err != nil {
		return fmt.Errorf("agent_repo.UpdateTrustScore: %w", err)
	}
	return nil
}

func (r *PGAgentRepository) UpsertTrackRecord(ctx context.Context, rec domain.AgentTrackRecord) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO agent_track_records (agent_id, market, total_predictions, correct_predictions, accuracy, avg_confidence, last_calculated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT (agent_id, market) DO UPDATE SET
			total_predictions=EXCLUDED.total_predictions,
			correct_predictions=EXCLUDED.correct_predictions,
			accuracy=EXCLUDED.accuracy,
			avg_confidence=EXCLUDED.avg_confidence,
			last_calculated_at=EXCLUDED.last_calculated_at
	`, rec.AgentID, rec.Market, rec.TotalPredictions, rec.CorrectPredictions, rec.Accuracy, rec.AvgConfidence, rec.LastCalculatedAt)
	if err != nil {
		return fmt.Errorf("agent_repo.UpsertTrackRecord: %w", err)
	}
	return nil
}
