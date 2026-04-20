package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openpup/agora/internal/core"
)

type ListIdeasParams struct {
	Domain    string
	ChannelID *string
	Status    *core.IdeaStatus
	Limit     int
}

type IdeaRepository interface {
	Create(context.Context, *core.Idea) error
	GetByID(context.Context, string) (*core.Idea, error)
	List(context.Context, ListIdeasParams) ([]core.Idea, error)
	UpsertPosition(context.Context, *core.IdeaPosition) error
	ListPositions(context.Context, string) ([]core.IdeaPosition, error)
}

type PGIdeaRepository struct {
	pool *pgxpool.Pool
}

func NewPGIdeaRepository(pool *pgxpool.Pool) *PGIdeaRepository {
	return &PGIdeaRepository{pool: pool}
}

func (r *PGIdeaRepository) Create(ctx context.Context, idea *core.Idea) error {
	stanceSummary, _ := json.Marshal(idea.StanceSummary)
	meta, _ := json.Marshal(idea.Meta)
	_, err := r.pool.Exec(ctx, `
		INSERT INTO ideas
		(id, channel_id, source_signal_id, created_by_agent_id, domain, title, summary, status, stance_summary, meta, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
	`, idea.ID, idea.ChannelID, idea.SourceSignalID, idea.CreatedByAgentID, idea.Domain, idea.Title, idea.Summary, idea.Status, stanceSummary, meta, idea.CreatedAt, idea.UpdatedAt)
	if err != nil {
		return fmt.Errorf("idea_repo.Create: %w", err)
	}
	return nil
}

func (r *PGIdeaRepository) GetByID(ctx context.Context, id string) (*core.Idea, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, channel_id, source_signal_id, created_by_agent_id, domain, title, summary, status, stance_summary, meta, created_at, updated_at
		FROM ideas WHERE id=$1
	`, id)
	idea, err := scanIdea(row)
	if err != nil {
		return nil, fmt.Errorf("idea_repo.GetByID: %w", err)
	}
	return idea, nil
}

func (r *PGIdeaRepository) List(ctx context.Context, params ListIdeasParams) ([]core.Idea, error) {
	limit := params.Limit
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	args := []any{}
	where := []string{}
	if params.Domain != "" {
		args = append(args, params.Domain)
		where = append(where, fmt.Sprintf("domain=$%d", len(args)))
	}
	if params.ChannelID != nil {
		args = append(args, *params.ChannelID)
		where = append(where, fmt.Sprintf("channel_id=$%d", len(args)))
	}
	if params.Status != nil {
		args = append(args, *params.Status)
		where = append(where, fmt.Sprintf("status=$%d", len(args)))
	}
	whereSQL := "TRUE"
	if len(where) > 0 {
		whereSQL = where[0]
		for _, item := range where[1:] {
			whereSQL += " AND " + item
		}
	}
	args = append(args, limit)
	rows, err := r.pool.Query(ctx, fmt.Sprintf(`
		SELECT id, channel_id, source_signal_id, created_by_agent_id, domain, title, summary, status, stance_summary, meta, created_at, updated_at
		FROM ideas WHERE %s ORDER BY updated_at DESC, created_at DESC LIMIT $%d
	`, whereSQL, len(args)), args...)
	if err != nil {
		return nil, fmt.Errorf("idea_repo.List: %w", err)
	}
	defer rows.Close()

	var out []core.Idea
	for rows.Next() {
		idea, err := scanIdea(rows)
		if err != nil {
			return nil, fmt.Errorf("idea_repo.List scan: %w", err)
		}
		out = append(out, *idea)
	}
	return out, nil
}

func (r *PGIdeaRepository) UpsertPosition(ctx context.Context, position *core.IdeaPosition) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO idea_positions (idea_id, agent_id, stance, confidence, source_signal_id, reason, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (idea_id, agent_id) DO UPDATE SET
			stance=EXCLUDED.stance,
			confidence=EXCLUDED.confidence,
			source_signal_id=EXCLUDED.source_signal_id,
			reason=EXCLUDED.reason,
			updated_at=EXCLUDED.updated_at
	`, position.IdeaID, position.AgentID, position.Stance, position.Confidence, position.SourceSignalID, position.Reason, position.CreatedAt, position.UpdatedAt)
	if err != nil {
		return fmt.Errorf("idea_repo.UpsertPosition: %w", err)
	}
	return nil
}

func (r *PGIdeaRepository) ListPositions(ctx context.Context, ideaID string) ([]core.IdeaPosition, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT idea_id, agent_id, stance, confidence, source_signal_id, reason, created_at, updated_at
		FROM idea_positions WHERE idea_id=$1 ORDER BY updated_at DESC
	`, ideaID)
	if err != nil {
		return nil, fmt.Errorf("idea_repo.ListPositions: %w", err)
	}
	defer rows.Close()

	var out []core.IdeaPosition
	for rows.Next() {
		var position core.IdeaPosition
		if err := rows.Scan(&position.IdeaID, &position.AgentID, &position.Stance, &position.Confidence, &position.SourceSignalID, &position.Reason, &position.CreatedAt, &position.UpdatedAt); err != nil {
			return nil, fmt.Errorf("idea_repo.ListPositions scan: %w", err)
		}
		out = append(out, position)
	}
	return out, nil
}

func scanIdea(row pgx.Row) (*core.Idea, error) {
	var idea core.Idea
	var stanceSummary, meta []byte
	if err := row.Scan(&idea.ID, &idea.ChannelID, &idea.SourceSignalID, &idea.CreatedByAgentID, &idea.Domain, &idea.Title, &idea.Summary, &idea.Status, &stanceSummary, &meta, &idea.CreatedAt, &idea.UpdatedAt); err != nil {
		return nil, err
	}
	_ = json.Unmarshal(stanceSummary, &idea.StanceSummary)
	_ = json.Unmarshal(meta, &idea.Meta)
	return &idea, nil
}
