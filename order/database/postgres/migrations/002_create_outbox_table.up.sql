CREATE TABLE IF NOT EXISTS outbox (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    aggregate_type TEXT NOT NULL,
    aggregate_id TEXT NOT NULL,
    event_name TEXT NOT NULL,
    event_version INT NOT NULL,
    payload JSONB NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ NULL,
    retry_count INT NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_outbox_unpublished_created_at
    ON outbox (created_at)
    WHERE published_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_outbox_aggregate
    ON outbox (aggregate_type, aggregate_id);
