package db

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"

	"github.com/termite-mail/termite/internal/config"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// DB wraps the sqlx database connection.
type DB struct {
	*sqlx.DB
}

// Open creates or opens the SQLite database at ~/.termite/cache.db
// and runs all pending migrations.
func Open() (*DB, error) {
	dataDir, err := config.DataDir()
	if err != nil {
		return nil, err
	}

	dbPath := filepath.Join(dataDir, "cache.db")
	conn, err := sqlx.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool for SQLite (single writer)
	conn.SetMaxOpenConns(1)

	db := &DB{conn}
	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	return db, nil
}

// migrate runs all SQL migration files in order.
func (db *DB) migrate() error {
	// Create migration tracking table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS _migrations (
			filename TEXT PRIMARY KEY,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Read all migration files
	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("failed to read migrations: %w", err)
	}

	// Sort by filename to ensure order
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		// Check if already applied
		var count int
		err := db.Get(&count, "SELECT COUNT(*) FROM _migrations WHERE filename = ?", entry.Name())
		if err != nil {
			return fmt.Errorf("failed to check migration status: %w", err)
		}
		if count > 0 {
			continue
		}

		// Read and execute migration
		content, err := migrationsFS.ReadFile(filepath.Join("migrations", entry.Name()))
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", entry.Name(), err)
		}

		slog.Info("applying migration", "file", entry.Name())

		tx, err := db.Beginx()
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}

		// Split SQL into statements, respecting BEGIN...END blocks (triggers).
		statements := splitSQL(string(content))
		for _, stmt := range statements {
			if _, err := tx.Exec(stmt); err != nil {
				tx.Rollback()
				return fmt.Errorf("migration %s failed: %w\nstatement: %s", entry.Name(), err, stmt)
			}
		}

		// Record migration
		if _, err := tx.Exec("INSERT INTO _migrations (filename) VALUES (?)", entry.Name()); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration: %w", err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration: %w", err)
		}
	}

	return nil
}

// splitSQL splits a SQL script into individual statements, correctly handling
// BEGIN...END blocks (used by triggers) that contain semicolons.
func splitSQL(script string) []string {
	var statements []string
	var current strings.Builder
	inBlock := false

	for _, line := range strings.Split(script, "\n") {
		trimmed := strings.TrimSpace(line)

		// Skip empty lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			current.WriteString(line)
			current.WriteString("\n")
			continue
		}

		upper := strings.ToUpper(trimmed)

		// Detect BEGIN (enters block mode where ; doesn't terminate)
		if strings.HasSuffix(upper, "BEGIN") {
			inBlock = true
		}

		current.WriteString(line)
		current.WriteString("\n")

		// Detect END; (exits block mode and terminates the statement)
		if inBlock && (strings.HasSuffix(upper, "END;") || strings.HasSuffix(upper, "END ;")) {
			inBlock = false
			stmt := strings.TrimSpace(current.String())
			// Strip the trailing semicolon for consistency, then re-add
			stmt = strings.TrimRight(stmt, "; \n\r\t")
			if stmt != "" {
				statements = append(statements, stmt)
			}
			current.Reset()
			continue
		}

		// Outside a block, a semicolon terminates the statement
		if !inBlock && strings.HasSuffix(trimmed, ";") {
			stmt := strings.TrimSpace(current.String())
			stmt = strings.TrimRight(stmt, "; \n\r\t")
			if stmt != "" {
				statements = append(statements, stmt)
			}
			current.Reset()
		}
	}

	// Catch any trailing statement without a semicolon
	if remaining := strings.TrimSpace(current.String()); remaining != "" {
		remaining = strings.TrimRight(remaining, "; \n\r\t")
		if remaining != "" {
			statements = append(statements, remaining)
		}
	}

	return statements
}

// Close closes the database connection.
func (db *DB) Close() error {
	return db.DB.Close()
}

// InsertAccount creates or updates an account record.
func (db *DB) InsertAccount(id, email, provider, displayName string) error {
	_, err := db.Exec(`
		INSERT INTO accounts (id, email, provider, display_name)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET email=excluded.email, provider=excluded.provider, display_name=excluded.display_name
	`, id, email, provider, displayName)
	return err
}

// UpdateSyncState records UIDVALIDITY and sync time for an account.
func (db *DB) UpdateSyncState(accountID string, uidValidity uint32) error {
	_, err := db.Exec(`
		UPDATE accounts SET uidvalidity = ?, last_synced_at = strftime('%s', 'now')
		WHERE id = ?
	`, uidValidity, accountID)
	return err
}

// InsertMessage inserts a message into the database.
func (db *DB) InsertMessage(msg *Message) error {
	_, err := db.NamedExec(`
		INSERT OR IGNORE INTO messages
			(id, thread_id, account_id, uid, folder, from_addr, to_addrs, cc_addrs,
			 subject, date, body_text, body_html, raw_headers, is_read, is_starred,
			 has_attachment, in_reply_to, "references")
		VALUES
			(:id, :thread_id, :account_id, :uid, :folder, :from_addr, :to_addrs, :cc_addrs,
			 :subject, :date, :body_text, :body_html, :raw_headers, :is_read, :is_starred,
			 :has_attachment, :in_reply_to, :references)
	`, msg)
	return err
}

// InsertThread inserts or updates a thread.
func (db *DB) InsertThread(thread *Thread) error {
	_, err := db.NamedExec(`
		INSERT INTO threads
			(id, account_id, subject, snippet, participants, message_count, unread_count,
			 has_attachment, labels, last_message_at, split_inbox_id)
		VALUES
			(:id, :account_id, :subject, :snippet, :participants, :message_count, :unread_count,
			 :has_attachment, :labels, :last_message_at, :split_inbox_id)
		ON CONFLICT(id) DO UPDATE SET
			subject=excluded.subject, snippet=excluded.snippet,
			participants=excluded.participants, message_count=excluded.message_count,
			unread_count=excluded.unread_count, has_attachment=excluded.has_attachment,
			labels=excluded.labels, last_message_at=excluded.last_message_at,
			split_inbox_id=excluded.split_inbox_id
	`, thread)
	return err
}

// GetThreads returns threads for a split inbox, ordered by last message time.
// When unreadOnly is true, only threads with unread_count > 0 are returned.
func (db *DB) GetThreads(splitInboxID string, limit int, unreadOnly bool) ([]Thread, error) {
	var threads []Thread
	q := `
		SELECT * FROM threads
		WHERE split_inbox_id = ? AND is_archived = 0 AND is_deleted = 0`
	if unreadOnly {
		q += ` AND unread_count > 0`
	}
	q += `
		ORDER BY last_message_at DESC
		LIMIT ?`
	err := db.Select(&threads, q, splitInboxID, limit)
	return threads, err
}

// GetThreadMessages returns all messages in a thread, ordered by date.
func (db *DB) GetThreadMessages(threadID string) ([]Message, error) {
	var msgs []Message
	err := db.Select(&msgs, `
		SELECT * FROM messages
		WHERE thread_id = ?
		ORDER BY date ASC
	`, threadID)
	return msgs, err
}

// FormatFTS5MatchQuery turns free-text user input into an FTS5 MATCH expression.
// Tokens are whitespace-separated, stripped of leading FTS operators (-, ^),
// double-quoted (with internal quotes escaped), and combined with AND so
// punctuation and reserved words do not break the query.
func FormatFTS5MatchQuery(userQuery string) string {
	userQuery = strings.TrimSpace(userQuery)
	if userQuery == "" {
		return ""
	}
	var parts []string
	for _, raw := range strings.Fields(userQuery) {
		t := raw
		for len(t) > 0 && (t[0] == '-' || t[0] == '^') {
			t = t[1:]
		}
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		t = strings.ReplaceAll(t, `"`, `""`)
		parts = append(parts, `"`+t+`"`)
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " AND ")
}

// SearchMessages performs full-text search using FTS5.
func (db *DB) SearchMessages(query string, limit int) ([]Message, error) {
	match := FormatFTS5MatchQuery(query)
	if match == "" {
		return nil, nil
	}
	var msgs []Message
	err := db.Select(&msgs, `
		SELECT m.* FROM messages m
		JOIN messages_fts fts ON m.rowid = fts.rowid
		WHERE messages_fts MATCH ?
		ORDER BY rank
		LIMIT ?
	`, match, limit)
	return msgs, err
}

// ArchiveThread marks a thread as archived.
func (db *DB) ArchiveThread(threadID string) error {
	_, err := db.Exec("UPDATE threads SET is_archived = 1 WHERE id = ?", threadID)
	return err
}

// DeleteThread marks a thread as deleted.
func (db *DB) DeleteThread(threadID string) error {
	_, err := db.Exec("UPDATE threads SET is_deleted = 1 WHERE id = ?", threadID)
	return err
}

// MarkThreadRead marks all messages in a thread as read.
func (db *DB) MarkThreadRead(threadID string) error {
	_, err := db.Exec("UPDATE messages SET is_read = 1 WHERE thread_id = ?", threadID)
	if err != nil {
		return err
	}
	_, err = db.Exec("UPDATE threads SET unread_count = 0 WHERE id = ?", threadID)
	return err
}

// GetUnreadCount returns the total unread count for a split inbox.
func (db *DB) GetUnreadCount(splitInboxID string) (int, error) {
	var count int
	err := db.Get(&count, `
		SELECT COALESCE(SUM(unread_count), 0) FROM threads
		WHERE split_inbox_id = ? AND is_archived = 0 AND is_deleted = 0
	`, splitInboxID)
	return count, err
}

// GetAllUnreadCount returns the total unread count across all inboxes.
func (db *DB) GetAllUnreadCount() (int, error) {
	var count int
	err := db.Get(&count, `
		SELECT COALESCE(SUM(unread_count), 0) FROM threads
		WHERE is_archived = 0 AND is_deleted = 0
	`)
	return count, err
}

// GetKnownEmails returns distinct email addresses seen in from/to/cc fields.
func (db *DB) GetKnownEmails(limit int) ([]string, error) {
	var emails []string
	err := db.Select(&emails, `
		SELECT DISTINCT from_addr FROM messages
		WHERE from_addr != ''
		UNION
		SELECT DISTINCT to_addrs FROM messages
		WHERE to_addrs != ''
		LIMIT ?
	`, limit)
	return emails, err
}

// Message represents a cached email message.
type Message struct {
	ID            string         `db:"id"`
	ThreadID      string         `db:"thread_id"`
	AccountID     string         `db:"account_id"`
	UID           uint32         `db:"uid"`
	Folder        string         `db:"folder"`
	FromAddr      string         `db:"from_addr"`
	ToAddrs       string         `db:"to_addrs"`
	CcAddrs       string         `db:"cc_addrs"`
	Subject       string         `db:"subject"`
	Date          int64          `db:"date"`
	BodyText      string         `db:"body_text"`
	BodyHTML      string         `db:"body_html"`
	RawHeaders    string         `db:"raw_headers"`
	IsRead        bool           `db:"is_read"`
	IsStarred     bool           `db:"is_starred"`
	HasAttachment bool           `db:"has_attachment"`
	InReplyTo     sql.NullString `db:"in_reply_to"`
	References    sql.NullString `db:"references"`
}

// Thread represents a conversation thread.
type Thread struct {
	ID            string        `db:"id"`
	AccountID     string        `db:"account_id"`
	Subject       string        `db:"subject"`
	Snippet       string        `db:"snippet"`
	Participants  string        `db:"participants"`
	MessageCount  int           `db:"message_count"`
	UnreadCount   int           `db:"unread_count"`
	HasAttachment bool          `db:"has_attachment"`
	Labels        string        `db:"labels"`
	LastMessageAt int64         `db:"last_message_at"`
	SnoozedUntil  sql.NullInt64 `db:"snoozed_until"`
	IsArchived    bool          `db:"is_archived"`
	IsDeleted     bool          `db:"is_deleted"`
	SplitInboxID  string        `db:"split_inbox_id"`
}

// SplitInbox represents a user-created split inbox.
type SplitInbox struct {
	ID         string `db:"id"`
	Label      string `db:"label"`
	SortOrder  int    `db:"sort_order"`
	CreatedAt  int64  `db:"created_at"`
	UnreadCount int   // computed, not stored
}

// SenderRoute maps a sender pattern to a split inbox.
type SenderRoute struct {
	ID           string `db:"id"`
	Pattern      string `db:"pattern"`
	MatchType    string `db:"match_type"`
	SplitInboxID string `db:"split_inbox_id"`
	CreatedAt    int64  `db:"created_at"`
}

// --- Split Inbox Management ---

// CreateSplitInbox inserts a new split inbox into the database.
func (db *DB) CreateSplitInbox(id, label string, sortOrder int) error {
	_, err := db.Exec(`
		INSERT INTO split_inboxes (id, label, sort_order, created_at)
		VALUES (?, ?, ?, strftime('%s','now'))
		ON CONFLICT(id) DO UPDATE SET label=excluded.label, sort_order=excluded.sort_order
	`, id, label, sortOrder)
	return err
}

// DeleteSplitInbox removes a split inbox. Threads in that inbox move to 'primary'.
func (db *DB) DeleteSplitInbox(id string) error {
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("UPDATE threads SET split_inbox_id = 'primary' WHERE split_inbox_id = ?", id)
	if err != nil {
		return err
	}

	_, err = tx.Exec("DELETE FROM sender_routes WHERE split_inbox_id = ?", id)
	if err != nil {
		return err
	}

	_, err = tx.Exec("DELETE FROM split_inboxes WHERE id = ?", id)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// ListSplitInboxes returns all split inboxes ordered by sort_order.
func (db *DB) ListSplitInboxes() ([]SplitInbox, error) {
	var inboxes []SplitInbox
	err := db.Select(&inboxes, `
		SELECT id, label, sort_order, created_at
		FROM split_inboxes
		ORDER BY sort_order ASC, created_at ASC
	`)
	return inboxes, err
}

// GetUnreadCountByInbox returns a map of inbox ID -> unread count.
func (db *DB) GetUnreadCountByInbox() (map[string]int, error) {
	rows, err := db.Query(`
		SELECT split_inbox_id, COALESCE(SUM(unread_count), 0)
		FROM threads
		WHERE is_archived = 0 AND is_deleted = 0
		GROUP BY split_inbox_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var id string
		var c int
		if err := rows.Scan(&id, &c); err != nil {
			return nil, err
		}
		counts[id] = c
	}
	return counts, rows.Err()
}

// MoveThreadToInbox updates a thread's split_inbox_id.
func (db *DB) MoveThreadToInbox(threadID, inboxID string) error {
	_, err := db.Exec(`
		UPDATE threads SET split_inbox_id = ? WHERE id = ?
	`, inboxID, threadID)
	return err
}

// --- Sender Routes ---

// CreateSenderRoute adds a new sender route.
func (db *DB) CreateSenderRoute(pattern, matchType, inboxID string) error {
	id := fmt.Sprintf("%s|%s|%s", pattern, matchType, inboxID)
	_, err := db.Exec(`
		INSERT INTO sender_routes (id, pattern, match_type, split_inbox_id, created_at)
		VALUES (?, ?, ?, ?, strftime('%s','now'))
		ON CONFLICT(id) DO UPDATE SET split_inbox_id=excluded.split_inbox_id
	`, id, pattern, matchType, inboxID)
	return err
}

// DeleteSenderRoute removes a sender route by its composite ID.
func (db *DB) DeleteSenderRoute(id string) error {
	_, err := db.Exec("DELETE FROM sender_routes WHERE id = ?", id)
	return err
}

// GetSenderRoutes returns all sender routes.
func (db *DB) GetSenderRoutes() ([]SenderRoute, error) {
	var routes []SenderRoute
	err := db.Select(&routes, `
		SELECT id, pattern, match_type, split_inbox_id, created_at
		FROM sender_routes
		ORDER BY created_at DESC
	`)
	return routes, err
}

// GetInboxForSender looks up the split inbox for a given from address.
// Exact match takes precedence over domain match.
func (db *DB) GetInboxForSender(fromAddr string) (string, error) {
	var inboxID string

	// Exact match.
	err := db.Get(&inboxID, `
		SELECT split_inbox_id FROM sender_routes
		WHERE pattern = ? AND match_type = 'exact'
		LIMIT 1
	`, fromAddr)
	if err == nil {
		return inboxID, nil
	}

	// Domain match.
	parts := strings.Split(fromAddr, "@")
	if len(parts) == 2 {
		domain := parts[1]
		err = db.Get(&inboxID, `
			SELECT split_inbox_id FROM sender_routes
			WHERE pattern = ? AND match_type = 'domain'
			LIMIT 1
		`, domain)
		if err == nil {
			return inboxID, nil
		}
	}

	return "", nil
}

// ApplyRouteRetroactively moves all existing matching threads to the target inbox.
func (db *DB) ApplyRouteRetroactively(pattern, matchType, inboxID string) error {
	var query string
	var arg string
	if matchType == "exact" {
		query = `
			UPDATE threads
			SET split_inbox_id = ?
			WHERE id IN (
				SELECT DISTINCT thread_id FROM messages
				WHERE from_addr = ?
			)
		`
		arg = pattern
	} else {
		query = `
			UPDATE threads
			SET split_inbox_id = ?
			WHERE id IN (
				SELECT DISTINCT thread_id FROM messages
				WHERE from_addr LIKE ?
			)
		`
		arg = "%@" + pattern
	}
	_, err := db.Exec(query, inboxID, arg)
	return err
}

