-- Migration: drop links table
-- Run: psql -h localhost -U linkmind_user -d linkmind_db -f migrations/001_create_links.down.sql

-- Indices are dropped automatically with the table
DROP TABLE IF EXISTS links CASCADE;
