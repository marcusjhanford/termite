CREATE TABLE IF NOT EXISTS split_inboxes (
    id          TEXT PRIMARY KEY,
    label       TEXT NOT NULL,
    sort_order  INTEGER DEFAULT 0,
    created_at  INTEGER
);

CREATE TABLE IF NOT EXISTS sender_routes (
    id              TEXT PRIMARY KEY,
    pattern         TEXT NOT NULL,
    match_type      TEXT NOT NULL CHECK (match_type IN ('exact', 'domain')),
    split_inbox_id  TEXT NOT NULL REFERENCES split_inboxes(id) ON DELETE CASCADE,
    created_at      INTEGER
);

CREATE INDEX IF NOT EXISTS idx_sender_routes_pattern ON sender_routes(pattern);
CREATE INDEX IF NOT EXISTS idx_sender_routes_inbox ON sender_routes(split_inbox_id);

-- Seed the default 'primary' and 'spam' inboxes if they don't exist.
INSERT OR IGNORE INTO split_inboxes (id, label, sort_order, created_at)
VALUES ('primary', 'Primary', 0, strftime('%s','now'));

INSERT OR IGNORE INTO split_inboxes (id, label, sort_order, created_at)
VALUES ('spam', 'Spam', 99, strftime('%s','now'));
