package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/openpup/agora/internal/core"
	pkgerrors "github.com/openpup/agora/internal/pkg/errors"
	"github.com/openpup/agora/internal/pubsub"
	"github.com/openpup/agora/internal/repository"
)

type SignalPublisher interface {
	PublishSignal(context.Context, string, []byte) error
}

type CreateSignalInput struct {
	AgentID            string
	ParentID           *string
	Domain             string
	Kind               core.SignalKind
	Claim              core.Claim
	Reasoning          core.Reasoning
	Evidence           []core.Evidence
	Refs               []core.CrossRef
	Meta               map[string]any
	DisagreementPoints []core.DisagreementPoint
}

type SignalService struct {
	repo      repository.SignalRepository
	publisher SignalPublisher
}

func NewSignalService(repo repository.SignalRepository, publisher SignalPublisher) *SignalService {
	return &SignalService{repo: repo, publisher: publisher}
}

func (s *SignalService) Create(ctx context.Context, input CreateSignalInput) (*core.Signal, error) {
	if err := validateSignalInput(input); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	signal := &core.Signal{
		ID:                 uuid.NewString(),
		AgentID:            input.AgentID,
		ParentID:           input.ParentID,
		Domain:             input.Domain,
		Kind:               input.Kind,
		Claim:              input.Claim,
		Reasoning:          input.Reasoning,
		Evidence:           input.Evidence,
		Refs:               input.Refs,
		Meta:               input.Meta,
		DisagreementPoints: input.DisagreementPoints,
		CreatedAt:          now,
	}
	if err := s.repo.Create(ctx, signal); err != nil {
		return nil, fmt.Errorf("signal_service.Create persist: %w", err)
	}
	payload, _ := json.Marshal(signal)
	subject := pubsub.SignalPublishedSubject(signal.Domain, string(signal.Kind))
	if input.ParentID != nil {
		subject = pubsub.SignalCounteredSubject(signal.Domain)
	}
	if err := s.publisher.PublishSignal(ctx, subject, payload); err != nil {
		return nil, fmt.Errorf("signal_service.Create publish: %w", err)
	}
	return signal, nil
}

func validateSignalInput(input CreateSignalInput) error {
	if input.Domain == "" {
		return fmt.Errorf("signal_service.validateSignalInput domain: %w", pkgerrors.ErrInvalidInput)
	}
	if len(input.Reasoning.Factors) == 0 || input.Reasoning.Summary == "" {
		return fmt.Errorf("signal_service.validateSignalInput reasoning: %w", pkgerrors.ErrInvalidInput)
	}
	if input.Kind == core.SignalKindClaim {
		if input.Claim.Confidence <= 0 {
			return fmt.Errorf("signal_service.validateSignalInput claim confidence: %w", pkgerrors.ErrInvalidInput)
		}
	}
	if input.ParentID != nil && len(input.DisagreementPoints) == 0 {
		return fmt.Errorf("signal_service.validateSignalInput disagreement points: %w", pkgerrors.ErrInvalidInput)
	}
	return nil
}

func (s *SignalService) GetByID(ctx context.Context, signalID string) (*core.Signal, error) {
	signal, err := s.repo.GetByID(ctx, signalID)
	if err != nil {
		return nil, fmt.Errorf("signal_service.GetByID get signal: %w", err)
	}
	counters, err := s.repo.ListCounters(ctx, signalID)
	if err != nil {
		return nil, fmt.Errorf("signal_service.GetByID counters: %w", err)
	}
	signal.CounterSignals = counters
	return signal, nil
}

func (s *SignalService) List(ctx context.Context, params repository.ListSignalsParams) ([]core.Signal, *string, error) {
	signals, err := s.repo.List(ctx, params)
	if err != nil {
		return nil, nil, fmt.Errorf("signal_service.List: %w", err)
	}
	if len(signals) == 0 {
		return signals, nil, nil
	}
	last := signals[len(signals)-1]
	cursorPayload, _ := json.Marshal(core.SignalListCursor{CreatedAt: last.CreatedAt, ID: last.ID})
	cursor := base64.StdEncoding.EncodeToString(cursorPayload)
	return signals, &cursor, nil
}

func DecodeCursor(cursor string) (*core.SignalListCursor, error) {
	if cursor == "" {
		return nil, nil
	}
	raw, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return nil, fmt.Errorf("signal_service.DecodeCursor decode: %w", err)
	}
	var parsed core.SignalListCursor
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("signal_service.DecodeCursor unmarshal: %w", err)
	}
	return &parsed, nil
}
