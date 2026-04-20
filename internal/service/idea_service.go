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

type SubmitIdeaPositionInput struct {
	IdeaID         string
	AgentID        string
	Stance         string
	Confidence     float64
	SourceSignalID *string
	Reason         string
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

func (s *IdeaService) SubmitPosition(ctx context.Context, input SubmitIdeaPositionInput) (*core.Idea, *core.IdeaPosition, error) {
	if input.IdeaID == "" || input.AgentID == "" || strings.TrimSpace(input.Stance) == "" {
		return nil, nil, fmt.Errorf("idea_service.SubmitPosition required fields: %w", pkgerrors.ErrInvalidInput)
	}
	stance := strings.TrimSpace(input.Stance)
	if stance != "support" && stance != "oppose" && stance != "neutral" {
		return nil, nil, fmt.Errorf("idea_service.SubmitPosition stance: %w", pkgerrors.ErrInvalidInput)
	}
	if input.Confidence <= 0 || input.Confidence > 1 {
		return nil, nil, fmt.Errorf("idea_service.SubmitPosition confidence: %w", pkgerrors.ErrInvalidInput)
	}
	idea, err := s.repo.GetByID(ctx, input.IdeaID)
	if err != nil {
		return nil, nil, fmt.Errorf("idea_service.SubmitPosition idea: %w", err)
	}
	now := time.Now().UTC()
	position := &core.IdeaPosition{
		IdeaID:         input.IdeaID,
		AgentID:        input.AgentID,
		Stance:         stance,
		Confidence:     input.Confidence,
		SourceSignalID: input.SourceSignalID,
		Reason:         strings.TrimSpace(input.Reason),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.repo.UpsertPosition(ctx, position); err != nil {
		return nil, nil, fmt.Errorf("idea_service.SubmitPosition persist: %w", err)
	}
	summary, err := s.recomputeStanceSummary(ctx, input.IdeaID)
	if err != nil {
		return nil, nil, err
	}
	status := nextStatusForStance(idea.Status, summary)
	if err := s.repo.UpdateLifecycle(ctx, input.IdeaID, status, input.SourceSignalID, summary); err != nil {
		return nil, nil, fmt.Errorf("idea_service.SubmitPosition lifecycle: %w", err)
	}
	idea, err = s.repo.GetByID(ctx, input.IdeaID)
	if err != nil {
		return nil, nil, fmt.Errorf("idea_service.SubmitPosition reload: %w", err)
	}
	return idea, position, nil
}

func (s *IdeaService) UpdateLifecycle(ctx context.Context, ideaID string, status core.IdeaStatus, sourceSignalID *string) (*core.Idea, error) {
	if ideaID == "" || status == "" {
		return nil, fmt.Errorf("idea_service.UpdateLifecycle required fields: %w", pkgerrors.ErrInvalidInput)
	}
	summary, err := s.recomputeStanceSummary(ctx, ideaID)
	if err != nil {
		return nil, err
	}
	if err := s.repo.UpdateLifecycle(ctx, ideaID, status, sourceSignalID, summary); err != nil {
		return nil, fmt.Errorf("idea_service.UpdateLifecycle persist: %w", err)
	}
	idea, err := s.repo.GetByID(ctx, ideaID)
	if err != nil {
		return nil, fmt.Errorf("idea_service.UpdateLifecycle reload: %w", err)
	}
	return idea, nil
}

func (s *IdeaService) recomputeStanceSummary(ctx context.Context, ideaID string) (map[string]any, error) {
	positions, err := s.repo.ListPositions(ctx, ideaID)
	if err != nil {
		return nil, fmt.Errorf("idea_service.recomputeStanceSummary positions: %w", err)
	}
	summary := map[string]any{"support": 0, "oppose": 0, "neutral": 0}
	for _, position := range positions {
		switch position.Stance {
		case "support":
			summary["support"] = summary["support"].(int) + 1
		case "oppose":
			summary["oppose"] = summary["oppose"].(int) + 1
		default:
			summary["neutral"] = summary["neutral"].(int) + 1
		}
	}
	return summary, nil
}

func nextStatusForStance(current core.IdeaStatus, summary map[string]any) core.IdeaStatus {
	if current == core.IdeaStatusResolved || current == core.IdeaStatusResolving || current == core.IdeaStatusChallenged {
		return current
	}
	if asInt(summary["oppose"]) > 0 && asInt(summary["support"]) > 0 {
		return core.IdeaStatusDebating
	}
	return current
}

func asInt(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case float64:
		return int(typed)
	default:
		return 0
	}
}
