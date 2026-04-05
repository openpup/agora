package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/openpup/agora/internal/domain"
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
	Market             domain.Market
	SignalType         domain.SignalType
	Ticker             *string
	Direction          *domain.Direction
	Confidence         *float64
	TimeHorizon        *time.Duration
	Reasoning          domain.Reasoning
	DataRefs           []map[string]any
	Meta               map[string]any
	DisagreementPoints []domain.DisagreementPoint
}

type SignalService struct {
	repo      repository.SignalRepository
	publisher SignalPublisher
}

func NewSignalService(repo repository.SignalRepository, publisher SignalPublisher) *SignalService {
	return &SignalService{repo: repo, publisher: publisher}
}

func (s *SignalService) Create(ctx context.Context, input CreateSignalInput) (*domain.Signal, error) {
	if err := validateSignalInput(input); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	signal := &domain.Signal{
		ID:                 uuid.NewString(),
		AgentID:            input.AgentID,
		ParentID:           input.ParentID,
		Market:             input.Market,
		SignalType:         input.SignalType,
		Ticker:             input.Ticker,
		Direction:          input.Direction,
		Confidence:         input.Confidence,
		TimeHorizon:        input.TimeHorizon,
		Reasoning:          input.Reasoning,
		DataRefs:           input.DataRefs,
		Meta:               input.Meta,
		DisagreementPoints: input.DisagreementPoints,
		CreatedAt:          now,
	}
	if input.TimeHorizon != nil {
		expiresAt := now.Add(*input.TimeHorizon)
		signal.ExpiresAt = &expiresAt
	}
	if err := s.repo.Create(ctx, signal); err != nil {
		return nil, fmt.Errorf("signal_service.Create persist: %w", err)
	}
	payload, _ := json.Marshal(signal)
	subject := pubsub.SignalPublishedSubject(signal.Market, derefString(signal.Ticker))
	if input.ParentID != nil {
		subject = pubsub.SignalCounteredSubject(signal.Market, derefString(signal.Ticker))
	}
	if err := s.publisher.PublishSignal(ctx, subject, payload); err != nil {
		return nil, fmt.Errorf("signal_service.Create publish: %w", err)
	}
	return signal, nil
}

func validateSignalInput(input CreateSignalInput) error {
	if !input.Market.Valid() {
		return fmt.Errorf("signal_service.validateSignalInput market: %w", pkgerrors.ErrInvalidInput)
	}
	if len(input.Reasoning.Factors) == 0 || input.Reasoning.Summary == "" {
		return fmt.Errorf("signal_service.validateSignalInput reasoning: %w", pkgerrors.ErrInvalidInput)
	}
	if input.SignalType == domain.SignalTypePrediction {
		if input.Ticker == nil || input.Direction == nil || input.Confidence == nil || input.TimeHorizon == nil {
			return fmt.Errorf("signal_service.validateSignalInput prediction fields: %w", pkgerrors.ErrInvalidInput)
		}
	}
	if input.ParentID != nil && len(input.DisagreementPoints) == 0 {
		return fmt.Errorf("signal_service.validateSignalInput disagreement points: %w", pkgerrors.ErrInvalidInput)
	}
	return nil
}

func (s *SignalService) GetByID(ctx context.Context, signalID string) (*domain.Signal, error) {
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

func (s *SignalService) List(ctx context.Context, params repository.ListSignalsParams) ([]domain.Signal, *string, error) {
	signals, err := s.repo.List(ctx, params)
	if err != nil {
		return nil, nil, fmt.Errorf("signal_service.List: %w", err)
	}
	if len(signals) == 0 {
		return signals, nil, nil
	}
	last := signals[len(signals)-1]
	cursorPayload, _ := json.Marshal(domain.SignalListCursor{CreatedAt: last.CreatedAt, ID: last.ID})
	cursor := base64.StdEncoding.EncodeToString(cursorPayload)
	return signals, &cursor, nil
}

func DecodeCursor(cursor string) (*domain.SignalListCursor, error) {
	if cursor == "" {
		return nil, nil
	}
	raw, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return nil, fmt.Errorf("signal_service.DecodeCursor decode: %w", err)
	}
	var parsed domain.SignalListCursor
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("signal_service.DecodeCursor unmarshal: %w", err)
	}
	return &parsed, nil
}

func derefString(value *string) string {
	if value == nil {
		return "all"
	}
	return *value
}
