-- Migration: drop clicks table
-- Run: psql -h localhost -U linkmind_user -d linkmind_db -f migrations/002_create_clicks.down.sql
-- NOTE: Run this BEFORE 001_create_links.down.sql (reverse order of creation)

-- Indices are dropped automatically with the table
DROP TABLE IF EXISTS clicks;
