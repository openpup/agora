CREATE TABLE signals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id UUID NOT NULL REFERENCES agents(id),
    parent_id UUID REFERENCES signals(id),
    market VARCHAR(20) NOT NULL,
    signal_type VARCHAR(20) NOT NULL,
    ticker VARCHAR(20),
    direction VARCHAR(10),
    confidence DECIMAL(3,2),
    time_horizon INTERVAL,
    expires_at TIMESTAMPTZ,
    reasoning JSONB NOT NULL DEFAULT '{}',
    data_refs JSONB NOT NULL DEFAULT '[]',
    meta JSONB NOT NULL DEFAULT '{}',
    verified BOOLEAN DEFAULT NULL,
    verified_at TIMESTAMPTZ,
    verification_detail JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_signals_market_created ON signals (market, created_at DESC);
CREATE INDEX idx_signals_ticker ON signals (ticker, created_at DESC) WHERE ticker IS NOT NULL;
CREATE INDEX idx_signals_agent ON signals (agent_id, created_at DESC);
CREATE INDEX idx_signals_parent ON signals (parent_id) WHERE parent_id IS NOT NULL;
CREATE INDEX idx_signals_pending_verify ON signals (expires_at) WHERE verified IS NULL AND expires_at IS NOT NULL;
