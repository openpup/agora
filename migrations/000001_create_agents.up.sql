CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE agents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(128) NOT NULL,
    api_key_hash VARCHAR(256) NOT NULL UNIQUE,
    capabilities JSONB NOT NULL DEFAULT '[]',
    data_sources JSONB NOT NULL DEFAULT '[]',
    trust_score DECIMAL(5,4) NOT NULL DEFAULT 0.5000,
    metadata JSONB NOT NULL DEFAULT '{}',
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_agents_capabilities ON agents USING GIN (capabilities);
CREATE INDEX idx_agents_status ON agents (status) WHERE status = 'active';
