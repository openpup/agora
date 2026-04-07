CREATE TABLE resolution_attestations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    claim_id UUID NOT NULL REFERENCES signals(id) ON DELETE CASCADE,
    agent_id UUID NOT NULL REFERENCES agents(id),
    kind VARCHAR(20) NOT NULL,
    verdict BOOLEAN,
    confidence DECIMAL(3,2) NOT NULL DEFAULT 0.50,
    reasoning JSONB NOT NULL DEFAULT '{}',
    evidence JSONB NOT NULL DEFAULT '[]',
    meta JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_resolution_attestations_claim_created
    ON resolution_attestations (claim_id, created_at DESC);

CREATE INDEX idx_resolution_attestations_agent_created
    ON resolution_attestations (agent_id, created_at DESC);
