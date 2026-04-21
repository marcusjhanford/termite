CREATE TABLE IF NOT EXISTS accounts (
  id             TEXT PRIMARY KEY,
  email          TEXT NOT NULL,
  provider       TEXT NOT NULL,
  display_name   TEXT,
  uidvalidity    INTEGER,
  last_synced_at INTEGER
);

CREATE TABLE IF NOT EXISTS threads (
  id              TEXT PRIMARY KEY,
  account_id      TEXT REFERENCES accounts(id),
  subject         TEXT,
  snippet         TEXT,
  participants    TEXT,
  message_count   INTEGER DEFAULT 1,
  unread_count    INTEGER DEFAULT 0,
  has_attachment  INTEGER DEFAULT 0,
  labels          TEXT,
  last_message_at INTEGER,
  snoozed_until   INTEGER,
  is_archived     INTEGER DEFAULT 0,
  is_deleted      INTEGER DEFAULT 0,
  split_inbox_id  TEXT
);

CREATE TABLE IF NOT EXISTS messages (
  id              TEXT PRIMARY KEY,
  thread_id       TEXT REFERENCES threads(id),
  account_id      TEXT REFERENCES accounts(id),
  uid             INTEGER,
  folder          TEXT,
  from_addr       TEXT,
  to_addrs        TEXT,
  cc_addrs        TEXT,
  subject         TEXT,
  date            INTEGER,
  body_text       TEXT,
  body_html       TEXT,
  raw_headers     TEXT,
  is_read         INTEGER DEFAULT 0,
  is_starred      INTEGER DEFAULT 0,
  has_attachment  INTEGER DEFAULT 0,
  in_reply_to     TEXT,
  "references"    TEXT
);

CREATE TABLE IF NOT EXISTS attachments (
  id            TEXT PRIMARY KEY,
  message_id    TEXT REFERENCES messages(id),
  filename      TEXT,
  content_type  TEXT,
  size_bytes    INTEGER,
  local_path    TEXT
);

CREATE INDEX IF NOT EXISTS idx_threads_inbox ON threads(split_inbox_id, is_archived, is_deleted);
CREATE INDEX IF NOT EXISTS idx_threads_account ON threads(account_id);
CREATE INDEX IF NOT EXISTS idx_messages_thread ON messages(thread_id);
CREATE INDEX IF NOT EXISTS idx_messages_account ON messages(account_id);
CREATE INDEX IF NOT EXISTS idx_messages_uid ON messages(account_id, uid);
