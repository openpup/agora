package service

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openpup/agora/internal/core"
	pkgerrors "github.com/openpup/agora/internal/pkg/errors"
)

type AuthService struct {
	pool *pgxpool.Pool
}

func NewAuthService(pool *pgxpool.Pool) *AuthService {
	return &AuthService{pool: pool}
}

func (s *AuthService) Authenticate(ctx context.Context, apiKey string) (*core.Agent, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, api_key_hash, capabilities, data_sources, trust_score, claim_trust, counter_trust, resolver_trust, challenge_trust, metadata, status, created_at, updated_at
		FROM agents WHERE status='active'
	`)
	if err != nil {
		return nil, fmt.Errorf("auth_service.Authenticate query: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var agent core.Agent
		var capabilities, dataSources, metadata []byte
		if err := rows.Scan(&agent.ID, &agent.Name, &agent.APIKeyHash, &capabilities, &dataSources, &agent.TrustScore, &agent.TrustProfile.ClaimTrust, &agent.TrustProfile.CounterTrust, &agent.TrustProfile.ResolverTrust, &agent.TrustProfile.ChallengeTrust, &metadata, &agent.Status, &agent.CreatedAt, &agent.UpdatedAt); err != nil {
			return nil, fmt.Errorf("auth_service.Authenticate scan: %w", err)
		}
		if err := CompareAPIKey(agent.APIKeyHash, apiKey); err == nil {
			return &agent, nil
		}
	}
	return nil, pkgerrors.ErrUnauthorized
}
