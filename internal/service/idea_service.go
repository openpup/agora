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

type CreateIdeaInput struct {
	ChannelID        *string
	SourceSignalID   *string
	CreatedByAgentID string
	Domain           string
	Title            string
	Summary          string
	Status           core.IdeaStatus
	StanceSummary    map[string]any
	Meta             map[string]any
}

type IdeaDetail struct {
	Idea      *core.Idea          `json:"idea"`
	Positions []core.IdeaPosition `json:"positions"`
}

type IdeaService struct {
	repo repository.IdeaRepository
}

func NewIdeaService(repo repository.IdeaRepository) *IdeaService {
	return &IdeaService{repo: repo}
}

func (s *IdeaService) Create(ctx context.Context, input CreateIdeaInput) (*core.Idea, error) {
	if strings.TrimSpace(input.Domain) == "" || strings.TrimSpace(input.Title) == "" || input.CreatedByAgentID == "" {
		return nil, fmt.Errorf("idea_service.Create required fields: %w", pkgerrors.ErrInvalidInput)
	}
	status := input.Status
	if status == "" {
		status = core.IdeaStatusDiscussing
	}
	now := time.Now().UTC()
	idea := &core.Idea{
		ID:               uuid.NewString(),
		ChannelID:        input.ChannelID,
		SourceSignalID:   input.SourceSignalID,
		CreatedByAgentID: input.CreatedByAgentID,
		Domain:           strings.TrimSpace(input.Domain),
		Title:            strings.TrimSpace(input.Title),
		Summary:          strings.TrimSpace(input.Summary),
		Status:           status,
		StanceSummary:    input.StanceSummary,
		Meta:             input.Meta,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := s.repo.Create(ctx, idea); err != nil {
		return nil, fmt.Errorf("idea_service.Create persist: %w", err)
	}
	return idea, nil
}

func (s *IdeaService) List(ctx context.Context, params repository.ListIdeasParams) ([]core.Idea, error) {
	ideas, err := s.repo.List(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("idea_service.List: %w", err)
	}
	return ideas, nil
}

func (s *IdeaService) Get(ctx context.Context, id string) (*IdeaDetail, error) {
	idea, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("idea_service.Get idea: %w", err)
	}
	positions, err := s.repo.ListPositions(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("idea_service.Get positions: %w", err)
	}
	return &IdeaDetail{Idea: idea, Positions: positions}, nil
}
