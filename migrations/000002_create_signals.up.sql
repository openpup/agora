CREATE TABLE domains (
    id VARCHAR(128) PRIMARY KEY,
    name VARCHAR(256) NOT NULL,
    namespace VARCHAR(128) NOT NULL,
    claim_schema JSONB NOT NULL DEFAULT '{}',
    resolution JSONB NOT NULL DEFAULT '{}',
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_domains_namespace ON domains (namespace);

CREATE TABLE signals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id UUID NOT NULL REFERENCES agents(id),
    parent_id UUID REFERENCES signals(id),
    domain VARCHAR(128) NOT NULL,
    kind VARCHAR(20) NOT NULL,

    statement TEXT NOT NULL DEFAULT '',
    structured JSONB NOT NULL DEFAULT '{}',
    confidence DECIMAL(3,2) NOT NULL DEFAULT 0.50,
    verifiable_by TIMESTAMPTZ,
    resolution JSONB,

    reasoning JSONB NOT NULL DEFAULT '{}',
    evidence JSONB NOT NULL DEFAULT '[]',
    disagreement JSONB NOT NULL DEFAULT '[]',

    refs JSONB NOT NULL DEFAULT '[]',

    verified BOOLEAN DEFAULT NULL,
    verified_at TIMESTAMPTZ,
    verification_detail JSONB,

    meta JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_signals_domain_created ON signals (domain, created_at DESC);
CREATE INDEX idx_signals_agent ON signals (agent_id, created_at DESC);
CREATE INDEX idx_signals_parent ON signals (parent_id) WHERE parent_id IS NOT NULL;
CREATE INDEX idx_signals_pending ON signals (verifiable_by)
    WHERE verified IS NULL AND verifiable_by IS NOT NULL;
CREATE INDEX idx_signals_structured ON signals USING GIN (structured);
