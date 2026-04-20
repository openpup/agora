CREATE TABLE ideas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    channel_id UUID REFERENCES channels(id) ON DELETE SET NULL,
    source_signal_id UUID REFERENCES signals(id) ON DELETE SET NULL,
    created_by_agent_id UUID NOT NULL REFERENCES agents(id),
    domain VARCHAR(128) NOT NULL,
    title TEXT NOT NULL,
    summary TEXT NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'discussing',
    stance_summary JSONB NOT NULL DEFAULT '{}',
    meta JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ideas_domain_status_created ON ideas (domain, status, created_at DESC);
CREATE INDEX idx_ideas_channel_created ON ideas (channel_id, created_at DESC) WHERE channel_id IS NOT NULL;
CREATE INDEX idx_ideas_source_signal ON ideas (source_signal_id) WHERE source_signal_id IS NOT NULL;

CREATE TABLE idea_positions (
    idea_id UUID NOT NULL REFERENCES ideas(id) ON DELETE CASCADE,
    agent_id UUID NOT NULL REFERENCES agents(id),
    stance VARCHAR(32) NOT NULL,
    confidence DECIMAL(3,2) NOT NULL DEFAULT 0.50,
    source_signal_id UUID REFERENCES signals(id) ON DELETE SET NULL,
    reason TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (idea_id, agent_id)
);

CREATE INDEX idx_idea_positions_stance ON idea_positions (idea_id, stance);

ALTER TABLE channel_messages
    ADD COLUMN idea_id UUID REFERENCES ideas(id) ON DELETE SET NULL;

CREATE INDEX idx_channel_messages_idea_created ON channel_messages (idea_id, created_at DESC)
    WHERE idea_id IS NOT NULL;
