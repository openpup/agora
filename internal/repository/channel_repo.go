package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openpup/agora/internal/core"
)

type ListChannelsParams struct {
	Domain string
	Limit  int
}

type ListChannelMessagesParams struct {
	ChannelID string
	Before    *time.Time
	Limit     int
}

type ChannelRepository interface {
	CreateChannel(context.Context, *core.Channel) error
	GetChannelByID(context.Context, string) (*core.Channel, error)
	ListChannels(context.Context, ListChannelsParams) ([]core.Channel, error)
	CreateMessage(context.Context, *core.ChannelMessage) error
	ListMessages(context.Context, ListChannelMessagesParams) ([]core.ChannelMessage, error)
}

type PGChannelRepository struct {
	pool *pgxpool.Pool
}

func NewPGChannelRepository(pool *pgxpool.Pool) *PGChannelRepository {
	return &PGChannelRepository{pool: pool}
}

func (r *PGChannelRepository) CreateChannel(ctx context.Context, channel *core.Channel) error {
	meta, _ := json.Marshal(channel.Meta)
	_, err := r.pool.Exec(ctx, `
		INSERT INTO channels (id, name, slug, domain, kind, description, meta, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
	`, channel.ID, channel.Name, channel.Slug, channel.Domain, channel.Kind, channel.Description, meta, channel.CreatedAt, channel.UpdatedAt)
	if err != nil {
		return fmt.Errorf("channel_repo.CreateChannel: %w", err)
	}
	return nil
}

func (r *PGChannelRepository) GetChannelByID(ctx context.Context, id string) (*core.Channel, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, name, slug, domain, kind, description, meta, created_at, updated_at
		FROM channels WHERE id=$1
	`, id)
	channel, err := scanChannel(row)
	if err != nil {
		return nil, fmt.Errorf("channel_repo.GetChannelByID: %w", err)
	}
	return channel, nil
}

func (r *PGChannelRepository) ListChannels(ctx context.Context, params ListChannelsParams) ([]core.Channel, error) {
	limit := params.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var rows pgx.Rows
	var err error
	if params.Domain != "" {
		rows, err = r.pool.Query(ctx, `
			SELECT id, name, slug, domain, kind, description, meta, created_at, updated_at
			FROM channels WHERE domain=$1 ORDER BY kind, slug LIMIT $2
		`, params.Domain, limit)
	} else {
		rows, err = r.pool.Query(ctx, `
			SELECT id, name, slug, domain, kind, description, meta, created_at, updated_at
			FROM channels ORDER BY domain, kind, slug LIMIT $1
		`, limit)
	}
	if err != nil {
		return nil, fmt.Errorf("channel_repo.ListChannels: %w", err)
	}
	defer rows.Close()

	var out []core.Channel
	for rows.Next() {
		channel, err := scanChannel(rows)
		if err != nil {
			return nil, fmt.Errorf("channel_repo.ListChannels scan: %w", err)
		}
		out = append(out, *channel)
	}
	return out, nil
}

func (r *PGChannelRepository) CreateMessage(ctx context.Context, message *core.ChannelMessage) error {
	refs, _ := json.Marshal(message.Refs)
	meta, _ := json.Marshal(message.Meta)
	_, err := r.pool.Exec(ctx, `
		INSERT INTO channel_messages (id, channel_id, agent_id, kind, intent, body, refs, meta, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
	`, message.ID, message.ChannelID, message.AgentID, message.Kind, message.Intent, message.Body, refs, meta, message.CreatedAt)
	if err != nil {
		return fmt.Errorf("channel_repo.CreateMessage: %w", err)
	}
	return nil
}

func (r *PGChannelRepository) ListMessages(ctx context.Context, params ListChannelMessagesParams) ([]core.ChannelMessage, error) {
	limit := params.Limit
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	args := []any{params.ChannelID}
	where := "channel_id=$1"
	if params.Before != nil {
		args = append(args, *params.Before)
		where += " AND created_at < $2"
	}
	args = append(args, limit)
	rows, err := r.pool.Query(ctx, fmt.Sprintf(`
		SELECT id, channel_id, agent_id, kind, intent, body, refs, meta, created_at
		FROM channel_messages WHERE %s ORDER BY created_at DESC, id DESC LIMIT $%d
	`, where, len(args)), args...)
	if err != nil {
		return nil, fmt.Errorf("channel_repo.ListMessages: %w", err)
	}
	defer rows.Close()

	var out []core.ChannelMessage
	for rows.Next() {
		message, err := scanChannelMessage(rows)
		if err != nil {
			return nil, fmt.Errorf("channel_repo.ListMessages scan: %w", err)
		}
		out = append(out, *message)
	}
	return out, nil
}

func scanChannel(row pgx.Row) (*core.Channel, error) {
	var channel core.Channel
	var meta []byte
	if err := row.Scan(&channel.ID, &channel.Name, &channel.Slug, &channel.Domain, &channel.Kind, &channel.Description, &meta, &channel.CreatedAt, &channel.UpdatedAt); err != nil {
		return nil, err
	}
	_ = json.Unmarshal(meta, &channel.Meta)
	return &channel, nil
}

func scanChannelMessage(row pgx.Row) (*core.ChannelMessage, error) {
	var message core.ChannelMessage
	var refs, meta []byte
	if err := row.Scan(&message.ID, &message.ChannelID, &message.AgentID, &message.Kind, &message.Intent, &message.Body, &refs, &meta, &message.CreatedAt); err != nil {
		return nil, err
	}
	_ = json.Unmarshal(refs, &message.Refs)
	_ = json.Unmarshal(meta, &message.Meta)
	return &message, nil
}
