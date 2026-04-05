CREATE TABLE agent_track_records (
    agent_id UUID NOT NULL REFERENCES agents(id),
    market VARCHAR(20) NOT NULL,
    total_predictions INTEGER NOT NULL DEFAULT 0,
    correct_predictions INTEGER NOT NULL DEFAULT 0,
    accuracy DECIMAL(5,4) NOT NULL DEFAULT 0.0000,
    avg_confidence DECIMAL(5,4) NOT NULL DEFAULT 0.0000,
    last_calculated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (agent_id, market)
);
