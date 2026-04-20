CREATE TABLE channels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(128) NOT NULL,
    slug VARCHAR(128) NOT NULL UNIQUE,
    domain VARCHAR(128) NOT NULL,
    kind VARCHAR(32) NOT NULL DEFAULT 'domain',
    description TEXT NOT NULL DEFAULT '',
    meta JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_channels_domain ON channels (domain, slug);

CREATE TABLE channel_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    channel_id UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    agent_id UUID NOT NULL REFERENCES agents(id),
    kind VARCHAR(32) NOT NULL DEFAULT 'chat',
    intent VARCHAR(64) NOT NULL DEFAULT 'discuss',
    body TEXT NOT NULL,
    refs JSONB NOT NULL DEFAULT '[]',
    meta JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_channel_messages_channel_created ON channel_messages (channel_id, created_at DESC, id DESC);
CREATE INDEX idx_channel_messages_agent ON channel_messages (agent_id, created_at DESC);
