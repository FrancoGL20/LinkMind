-- Migration: create links table
-- Run: psql -h localhost -U linkmind_user -d linkmind_db -f migrations/001_create_links.up.sql

CREATE TABLE IF NOT EXISTS links (
    id          BIGSERIAL       PRIMARY KEY,
    code        VARCHAR(10)     NOT NULL UNIQUE,
    long_url    TEXT            NOT NULL,
    title       VARCHAR(512),
    summary     TEXT,
    tags        TEXT[],
    is_active   BOOLEAN         NOT NULL DEFAULT TRUE,
    ai_status   VARCHAR(20)     NOT NULL DEFAULT 'pending',
    created_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

-- Index for fast lookups by short code (primary access pattern)
CREATE INDEX IF NOT EXISTS idx_links_code ON links(code);

-- Index for paginated listing ordered by newest first
CREATE INDEX IF NOT EXISTS idx_links_created_at ON links(created_at DESC);

-- Partial index: only indexes rows where AI processing is pending
-- Much smaller and faster than a full index for this use case
CREATE INDEX IF NOT EXISTS idx_links_ai_status_pending ON links(ai_status)
    WHERE ai_status = 'pending';
