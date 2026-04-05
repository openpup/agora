CREATE TABLE subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id UUID NOT NULL REFERENCES agents(id),
    filter JSONB NOT NULL,
    delivery VARCHAR(20) NOT NULL DEFAULT 'websocket',
    webhook_url VARCHAR(512),
    nats_subject VARCHAR(256),
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(agent_id, filter)
);
