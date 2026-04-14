CREATE TABLE accounts (
  id            TEXT PRIMARY KEY,
  email         TEXT NOT NULL,
  provider      TEXT NOT NULL,
  display_name  TEXT,
  uidvalidity   INTEGER,
  last_synced_at INTEGER
);

CREATE TABLE threads (
  id              TEXT PRIMARY KEY,   -- JWZ message-id based hash
  account_id      TEXT REFERENCES accounts(id),
  subject         TEXT,
  snippet         TEXT,
  participants    TEXT,               -- JSON array of addresses
  message_count   INTEGER DEFAULT 1,
  unread_count    INTEGER DEFAULT 0,
  has_attachment  INTEGER DEFAULT 0,
  labels          TEXT,               -- JSON array
  last_message_at INTEGER,            -- unix timestamp
  snoozed_until   INTEGER,
  is_archived     INTEGER DEFAULT 0,
  is_deleted      INTEGER DEFAULT 0,
  split_inbox_id  TEXT
);

CREATE TABLE messages (
  id              TEXT PRIMARY KEY,   -- message-id header
  thread_id       TEXT REFERENCES threads(id),
  account_id      TEXT REFERENCES accounts(id),
  uid             INTEGER,            -- IMAP UID
  folder          TEXT,
  from_addr       TEXT,
  to_addrs        TEXT,               -- JSON
  cc_addrs        TEXT,               -- JSON
  subject         TEXT,
  date            INTEGER,            -- unix timestamp
  body_text       TEXT,
  body_html       TEXT,
  raw_headers     TEXT,
  is_read         INTEGER DEFAULT 0,
  is_starred      INTEGER DEFAULT 0,
  has_attachment  INTEGER DEFAULT 0,
  in_reply_to     TEXT,
  references      TEXT                -- space-separated message IDs for threading
);

CREATE TABLE attachments (
  id            TEXT PRIMARY KEY,
  message_id    TEXT REFERENCES messages(id),
  filename      TEXT,
  content_type  TEXT,
  size_bytes    INTEGER,
  local_path    TEXT                  -- cached to ~/.termite/attachments/
);

-- FTS5 virtual table for instant search
CREATE VIRTUAL TABLE messages_fts USING fts5(
  subject, body_text, from_addr, to_addrs,
  content='messages', content_rowid='rowid'
);

CREATE TRIGGER messages_ai AFTER INSERT ON messages BEGIN
  INSERT INTO messages_fts(rowid, subject, body_text, from_addr, to_addrs)
  VALUES (new.rowid, new.subject, new.body_text, new.from_addr, new.to_addrs);
END;