-- +goose Up
ALTER TABLE diff ADD COLUMN delta_latency_ms INTEGER NOT NULL DEFAULT 0;
ALTER TABLE diff ADD COLUMN delta_bytes INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE diff DROP COLUMN delta_latency_ms;
ALTER TABLE diff DROP COLUMN delta_bytes;
