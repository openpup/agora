package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/openpup/agora/internal/domain"
	pkgerrors "github.com/openpup/agora/internal/pkg/errors"
	"github.com/openpup/agora/internal/repository"
)

type AgentService struct {
	repo         repository.AgentRepository
	apiKeyPrefix string
}

func NewAgentService(repo repository.AgentRepository, apiKeyPrefix string) *AgentService {
	return &AgentService{repo: repo, apiKeyPrefix: apiKeyPrefix}
}

func (s *AgentService) Register(ctx context.Context, name string, capabilities, dataSources []string, metadata map[string]any) (*domain.Agent, string, error) {
	apiKey, err := s.generateAPIKey()
	if err != nil {
		return nil, "", fmt.Errorf("agent_service.Register generate api key: %w", err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(apiKey), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", fmt.Errorf("agent_service.Register hash key: %w", err)
	}
	now := time.Now().UTC()
	agent := &domain.Agent{
		ID:           uuid.NewString(),
		Name:         name,
		APIKeyHash:   string(hash),
		Capabilities: capabilities,
		DataSources:  dataSources,
		TrustScore:   0.5,
		Metadata:     metadata,
		Status:       domain.AgentStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.repo.Create(ctx, agent); err != nil {
		return nil, "", fmt.Errorf("agent_service.Register create agent: %w", err)
	}
	return agent, apiKey, nil
}

func (s *AgentService) GetMe(ctx context.Context, agentID string) (*domain.Agent, error) {
	agent, err := s.repo.GetByID(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("agent_service.GetMe: %w", err)
	}
	return agent, nil
}

func (s *AgentService) Update(ctx context.Context, agentID, name string, capabilities, dataSources []string, metadata map[string]any) (*domain.Agent, error) {
	agent, err := s.repo.GetByID(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("agent_service.Update get agent: %w", err)
	}
	if name != "" {
		agent.Name = name
	}
	if capabilities != nil {
		agent.Capabilities = capabilities
	}
	if dataSources != nil {
		agent.DataSources = dataSources
	}
	if metadata != nil {
		agent.Metadata = metadata
	}
	agent.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(ctx, agent); err != nil {
		return nil, fmt.Errorf("agent_service.Update update agent: %w", err)
	}
	return agent, nil
}

func (s *AgentService) GetTrackRecord(ctx context.Context, agentID string) ([]domain.AgentTrackRecord, error) {
	records, err := s.repo.ListTrackRecords(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("agent_service.GetTrackRecord: %w", err)
	}
	return records, nil
}

func (s *AgentService) Authenticate(ctx context.Context, apiKey string) (*domain.Agent, error) {
	// BCrypt hashes are salted, so the repo lookup is not useful; scan active agents is too expensive.
	// Use SHA256 fingerprinting in Redis/headers later if needed. For Phase 1, load by iterating over candidate hashes is avoided.
	_ = sha256.Sum256([]byte(apiKey))
	return nil, fmt.Errorf("agent_service.Authenticate: %w", pkgerrors.ErrUnauthorized)
}

func CompareAPIKey(hash string, apiKey string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(apiKey))
}

func (s *AgentService) generateAPIKey() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return s.apiKeyPrefix + hex.EncodeToString(buf), nil
}
