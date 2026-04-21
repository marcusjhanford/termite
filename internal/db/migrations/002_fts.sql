CREATE VIRTUAL TABLE IF NOT EXISTS messages_fts USING fts5(
  subject, body_text, from_addr, to_addrs,
  content='messages', content_rowid='rowid'
);

CREATE TRIGGER IF NOT EXISTS messages_ai AFTER INSERT ON messages BEGIN
  INSERT INTO messages_fts(rowid, subject, body_text, from_addr, to_addrs)
  VALUES (new.rowid, new.subject, new.body_text, new.from_addr, new.to_addrs);
END;

CREATE TRIGGER IF NOT EXISTS messages_ad AFTER DELETE ON messages BEGIN
  INSERT INTO messages_fts(messages_fts, rowid, subject, body_text, from_addr, to_addrs)
  VALUES ('delete', old.rowid, old.subject, old.body_text, old.from_addr, old.to_addrs);
END;

CREATE TRIGGER IF NOT EXISTS messages_au AFTER UPDATE ON messages BEGIN
  INSERT INTO messages_fts(messages_fts, rowid, subject, body_text, from_addr, to_addrs)
  VALUES ('delete', old.rowid, old.subject, old.body_text, old.from_addr, old.to_addrs);
  INSERT INTO messages_fts(rowid, subject, body_text, from_addr, to_addrs)
  VALUES (new.rowid, new.subject, new.body_text, new.from_addr, new.to_addrs);
END;
