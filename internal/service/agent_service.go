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

	"github.com/openpup/agora/internal/core"
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

func (s *AgentService) Register(ctx context.Context, name string, capabilities, dataSources []string, metadata map[string]any) (*core.Agent, string, error) {
	apiKey, err := s.generateAPIKey()
	if err != nil {
		return nil, "", fmt.Errorf("agent_service.Register generate api key: %w", err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(apiKey), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", fmt.Errorf("agent_service.Register hash key: %w", err)
	}
	now := time.Now().UTC()
	agent := &core.Agent{
		ID:           uuid.NewString(),
		Name:         name,
		APIKeyHash:   string(hash),
		Capabilities: capabilities,
		DataSources:  dataSources,
		TrustScore:   0.5,
		TrustProfile: core.AgentTrustProfile{
			ClaimTrust:     0.5,
			CounterTrust:   0.5,
			ResolverTrust:  0.5,
			ChallengeTrust: 0.5,
		},
		Metadata:  metadata,
		Status:    core.AgentStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repo.Create(ctx, agent); err != nil {
		return nil, "", fmt.Errorf("agent_service.Register create agent: %w", err)
	}
	return agent, apiKey, nil
}

func (s *AgentService) GetMe(ctx context.Context, agentID string) (*core.Agent, error) {
	agent, err := s.repo.GetByID(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("agent_service.GetMe: %w", err)
	}
	return agent, nil
}

func (s *AgentService) Update(ctx context.Context, agentID, name string, capabilities, dataSources []string, metadata map[string]any) (*core.Agent, error) {
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

func (s *AgentService) GetTrackRecord(ctx context.Context, agentID string) ([]core.AgentTrackRecord, error) {
	records, err := s.repo.ListTrackRecords(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("agent_service.GetTrackRecord: %w", err)
	}
	return records, nil
}

func (s *AgentService) ListPublic(ctx context.Context, limit int) ([]core.Agent, error) {
	agents, err := s.repo.ListPublic(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("agent_service.ListPublic: %w", err)
	}
	return agents, nil
}

func (s *AgentService) Authenticate(ctx context.Context, apiKey string) (*core.Agent, error) {
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
