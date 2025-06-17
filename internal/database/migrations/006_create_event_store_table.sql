-- +migrate Up
CREATE TABLE IF NOT EXISTS event_store (
    id SERIAL PRIMARY KEY,
    aggregate_id VARCHAR(255) NOT NULL,
    aggregate_type VARCHAR(100) NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    event_data JSONB NOT NULL,
    version INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    metadata JSONB DEFAULT '{}'::jsonb,
    UNIQUE(aggregate_id, version)
);

CREATE INDEX idx_event_store_aggregate_id ON event_store(aggregate_id);
CREATE INDEX idx_event_store_aggregate_type ON event_store(aggregate_type);
CREATE INDEX idx_event_store_event_type ON event_store(event_type);
CREATE INDEX idx_event_store_created_at ON event_store(created_at);

-- +migrate Down
DROP INDEX IF EXISTS idx_event_store_created_at;
DROP INDEX IF EXISTS idx_event_store_event_type;
DROP INDEX IF EXISTS idx_event_store_aggregate_type;
DROP INDEX IF EXISTS idx_event_store_aggregate_id;
DROP TABLE IF EXISTS event_store; 