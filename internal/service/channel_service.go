package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/openpup/agora/internal/core"
	pkgerrors "github.com/openpup/agora/internal/pkg/errors"
	"github.com/openpup/agora/internal/repository"
)

type CreateChannelInput struct {
	Name        string
	Slug        string
	Domain      string
	Kind        core.ChannelKind
	Description string
	Meta        map[string]any
}

type CreateChannelMessageInput struct {
	ChannelID string
	AgentID   string
	Kind      core.ChannelMessageKind
	Intent    string
	Body      string
	Refs      []core.CrossRef
	Meta      map[string]any
}

type ChannelService struct {
	repo repository.ChannelRepository
}

func NewChannelService(repo repository.ChannelRepository) *ChannelService {
	return &ChannelService{repo: repo}
}

func (s *ChannelService) CreateChannel(ctx context.Context, input CreateChannelInput) (*core.Channel, error) {
	if input.Name == "" || input.Slug == "" || input.Domain == "" {
		return nil, fmt.Errorf("channel_service.CreateChannel required fields: %w", pkgerrors.ErrInvalidInput)
	}
	now := time.Now().UTC()
	kind := input.Kind
	if kind == "" {
		kind = core.ChannelKindDomain
	}
	channel := &core.Channel{
		ID:          uuid.NewString(),
		Name:        input.Name,
		Slug:        input.Slug,
		Domain:      input.Domain,
		Kind:        kind,
		Description: input.Description,
		Meta:        input.Meta,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.repo.CreateChannel(ctx, channel); err != nil {
		return nil, fmt.Errorf("channel_service.CreateChannel persist: %w", err)
	}
	return channel, nil
}

func (s *ChannelService) ListChannels(ctx context.Context, params repository.ListChannelsParams) ([]core.Channel, error) {
	channels, err := s.repo.ListChannels(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("channel_service.ListChannels: %w", err)
	}
	return channels, nil
}

func (s *ChannelService) CreateMessage(ctx context.Context, input CreateChannelMessageInput) (*core.ChannelMessage, error) {
	if input.ChannelID == "" || input.AgentID == "" || strings.TrimSpace(input.Body) == "" {
		return nil, fmt.Errorf("channel_service.CreateMessage required fields: %w", pkgerrors.ErrInvalidInput)
	}
	body := strings.TrimSpace(input.Body)
	if len(body) > 4000 {
		return nil, fmt.Errorf("channel_service.CreateMessage body too long: %w", pkgerrors.ErrInvalidInput)
	}
	kind := input.Kind
	if kind == "" {
		kind = core.ChannelMessageKindChat
	}
	intent := strings.TrimSpace(input.Intent)
	if intent == "" {
		intent = "discuss"
	}
	if _, err := s.repo.GetChannelByID(ctx, input.ChannelID); err != nil {
		return nil, fmt.Errorf("channel_service.CreateMessage channel: %w", err)
	}
	message := &core.ChannelMessage{
		ID:        uuid.NewString(),
		ChannelID: input.ChannelID,
		AgentID:   input.AgentID,
		Kind:      kind,
		Intent:    intent,
		Body:      body,
		Refs:      input.Refs,
		Meta:      input.Meta,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.repo.CreateMessage(ctx, message); err != nil {
		return nil, fmt.Errorf("channel_service.CreateMessage persist: %w", err)
	}
	return message, nil
}

func (s *ChannelService) ListMessages(ctx context.Context, params repository.ListChannelMessagesParams) ([]core.ChannelMessage, error) {
	if params.ChannelID == "" {
		return nil, fmt.Errorf("channel_service.ListMessages channel: %w", pkgerrors.ErrInvalidInput)
	}
	messages, err := s.repo.ListMessages(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("channel_service.ListMessages: %w", err)
	}
	return messages, nil
}
