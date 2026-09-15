-- Migration: create clicks table
-- Run: psql -h localhost -U linkmind_user -d linkmind_db -f migrations/002_create_clicks.up.sql

CREATE TABLE IF NOT EXISTS clicks (
    id          BIGSERIAL       PRIMARY KEY,
    link_id     BIGINT          NOT NULL REFERENCES links(id) ON DELETE CASCADE,
    clicked_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    ip_hash     VARCHAR(64),
    user_agent  VARCHAR(512),
    referer     VARCHAR(2048)
);

-- Index for aggregating clicks per link (JOIN + COUNT pattern)
CREATE INDEX IF NOT EXISTS idx_clicks_link_id ON clicks(link_id);

-- Index for time-range queries (e.g. clicks_today)
CREATE INDEX IF NOT EXISTS idx_clicks_clicked_at ON clicks(clicked_at DESC);
