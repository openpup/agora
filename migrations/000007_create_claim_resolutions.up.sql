CREATE TABLE claim_resolutions (
    claim_id UUID PRIMARY KEY REFERENCES signals(id) ON DELETE CASCADE,
    domain VARCHAR(128) NOT NULL,
    strategy VARCHAR(64) NOT NULL,
    state VARCHAR(20) NOT NULL DEFAULT 'open',
    outcome BOOLEAN,
    resolution_score DOUBLE PRECISION NOT NULL DEFAULT 0,
    resolver_count INTEGER NOT NULL DEFAULT 0,
    challenge_count INTEGER NOT NULL DEFAULT 0,
    summary JSONB NOT NULL DEFAULT '{}',
    resolved_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_claim_resolutions_domain_updated
    ON claim_resolutions (domain, updated_at DESC);
