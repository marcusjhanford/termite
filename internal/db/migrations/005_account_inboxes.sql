-- Migration: associate split inboxes with accounts and update queries.

-- Add account_id to split inboxes (nullable for backward compat).
ALTER TABLE split_inboxes ADD COLUMN account_id TEXT;

-- Update existing split_inboxes to belong to the first account if one exists.
-- This is a one-time fix; on fresh installs the seed below handles it.

-- Drop and recreate the seed to include account_id.
-- We seed per-account Primary/Spam in application code instead.
