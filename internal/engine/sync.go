package engine

import (
	"fmt"
	"log/slog"
	"sort"

	"github.com/emersion/go-imap/v2"

	"github.com/termite-mail/termite/internal/config"
	"github.com/termite-mail/termite/internal/db"
)

const (
	// initialSyncLimit is the number of recent messages to fetch during initial sync.
	initialSyncLimit = 200
	// defaultFolder is the default mailbox to sync.
	defaultFolder = "INBOX"
)

// SyncResult holds the outcome of a sync operation for a single account.
type SyncResult struct {
	AccountID string
	NewCount  int
	Err       error
}

// InitialSync performs a full initial sync for the given account: connects via
// IMAP, fetches the last initialSyncLimit UIDs from INBOX, fetches the
// full messages, parses them, threads them, and inserts everything into the DB.
func InitialSync(account Account, database *db.DB, splitInboxes []config.SplitInboxConfig) SyncResult {
	accountID := account.Config.ID
	slog.Info("starting initial sync", "account", accountID)

	if err := database.InsertAccount(account.Config.ID, account.Config.Email, account.Config.Provider, account.Config.Name); err != nil {
		return SyncResult{AccountID: accountID, Err: fmt.Errorf("initial sync: insert account: %w", err)}
	}

	// Get credentials from the provider.
	creds, err := account.Provider.GetCredentials(accountID)
	if err != nil {
		return SyncResult{AccountID: accountID, Err: fmt.Errorf("initial sync: get credentials: %w", err)}
	}

	// Connect to IMAP.
	conn := &IMAPConn{}
	if err := conn.Connect(
		account.Provider.IMAPHost(),
		account.Provider.IMAPPort(),
		account.Provider.IMAPTLS(),
		creds,
	); err != nil {
		return SyncResult{AccountID: accountID, Err: fmt.Errorf("initial sync: %w", err)}
	}
	defer conn.Close()

	// Fetch all UIDs from INBOX.
	allUIDs, err := conn.FetchUIDsSince(defaultFolder, 0)
	if err != nil {
		return SyncResult{AccountID: accountID, Err: fmt.Errorf("initial sync: %w", err)}
	}

	if len(allUIDs) == 0 {
		slog.Info("initial sync: no messages found", "account", accountID)
		if err := database.UpdateSyncState(accountID, 0); err != nil {
			return SyncResult{AccountID: accountID, Err: fmt.Errorf("initial sync: update sync state: %w", err)}
		}
		return SyncResult{AccountID: accountID, NewCount: 0}
	}

	// Sort UIDs ascending so we can take the latest N.
	sort.Slice(allUIDs, func(i, j int) bool { return allUIDs[i] < allUIDs[j] })

	// Take only the last initialSyncLimit UIDs.
	fetchUIDs := allUIDs
	if len(fetchUIDs) > initialSyncLimit {
		fetchUIDs = fetchUIDs[len(fetchUIDs)-initialSyncLimit:]
	}

	// Convert to imap.UID.
	imapUIDs := make([]imap.UID, len(fetchUIDs))
	for i, u := range fetchUIDs {
		imapUIDs[i] = imap.UID(u)
	}

	// Fetch full messages.
	rawMessages, err := conn.FetchMessages(defaultFolder, imapUIDs)
	if err != nil {
		return SyncResult{AccountID: accountID, Err: fmt.Errorf("initial sync: %w", err)}
	}

	// Parse and insert messages.
	newCount, err := parseAndInsert(rawMessages, accountID, database, splitInboxes)
	if err != nil {
		return SyncResult{AccountID: accountID, Err: fmt.Errorf("initial sync: %w", err)}
	}

	// Update sync state with the highest UID.
	highestUID := fetchUIDs[len(fetchUIDs)-1]
	if err := database.UpdateSyncState(accountID, highestUID); err != nil {
		return SyncResult{AccountID: accountID, Err: fmt.Errorf("initial sync: update sync state: %w", err)}
	}

	slog.Info("initial sync complete", "account", accountID, "new", newCount)
	return SyncResult{AccountID: accountID, NewCount: newCount}
}

// IncrementalSync fetches new messages since the last known UID for the given
// account, parses them, threads them, and inserts them into the DB.
func IncrementalSync(account Account, database *db.DB, splitInboxes []config.SplitInboxConfig) SyncResult {
	accountID := account.Config.ID
	slog.Info("starting incremental sync", "account", accountID)

	if err := database.InsertAccount(account.Config.ID, account.Config.Email, account.Config.Provider, account.Config.Name); err != nil {
		return SyncResult{AccountID: accountID, Err: fmt.Errorf("incremental sync: insert account: %w", err)}
	}

	// Get the last synced UID from the database.
	lastUID, err := getLastSyncedUID(database, accountID)
	if err != nil {
		return SyncResult{AccountID: accountID, Err: fmt.Errorf("incremental sync: %w", err)}
	}

	// Get credentials from the provider.
	creds, err := account.Provider.GetCredentials(accountID)
	if err != nil {
		return SyncResult{AccountID: accountID, Err: fmt.Errorf("incremental sync: get credentials: %w", err)}
	}

	// Connect to IMAP.
	conn := &IMAPConn{}
	if err := conn.Connect(
		account.Provider.IMAPHost(),
		account.Provider.IMAPPort(),
		account.Provider.IMAPTLS(),
		creds,
	); err != nil {
		return SyncResult{AccountID: accountID, Err: fmt.Errorf("incremental sync: %w", err)}
	}
	defer conn.Close()

	// Fetch UIDs since last known.
	newUIDs, err := conn.FetchUIDsSince(defaultFolder, lastUID)
	if err != nil {
		return SyncResult{AccountID: accountID, Err: fmt.Errorf("incremental sync: %w", err)}
	}

	if len(newUIDs) == 0 {
		slog.Info("incremental sync: no new messages", "account", accountID)
		return SyncResult{AccountID: accountID, NewCount: 0}
	}

	// Convert to imap.UID.
	imapUIDs := make([]imap.UID, len(newUIDs))
	for i, u := range newUIDs {
		imapUIDs[i] = imap.UID(u)
	}

	// Fetch full messages.
	rawMessages, err := conn.FetchMessages(defaultFolder, imapUIDs)
	if err != nil {
		return SyncResult{AccountID: accountID, Err: fmt.Errorf("incremental sync: %w", err)}
	}

	// Parse and insert messages.
	newCount, err := parseAndInsert(rawMessages, accountID, database, splitInboxes)
	if err != nil {
		return SyncResult{AccountID: accountID, Err: fmt.Errorf("incremental sync: %w", err)}
	}

	// Update sync state with the highest UID.
	sort.Slice(newUIDs, func(i, j int) bool { return newUIDs[i] < newUIDs[j] })
	highestUID := newUIDs[len(newUIDs)-1]
	if err := database.UpdateSyncState(accountID, highestUID); err != nil {
		return SyncResult{AccountID: accountID, Err: fmt.Errorf("incremental sync: update sync state: %w", err)}
	}

	slog.Info("incremental sync complete", "account", accountID, "new", newCount)
	return SyncResult{AccountID: accountID, NewCount: newCount}
}

// parseAndInsert parses raw IMAP messages, evaluates split inbox rules,
// threads the messages, and inserts them into the database. Returns the
// count of new messages inserted.
func parseAndInsert(rawMessages []RawMessage, accountID string, database *db.DB, splitInboxes []config.SplitInboxConfig) (int, error) {
	if len(rawMessages) == 0 {
		return 0, nil
	}

	// Parse all raw messages.
	var parsedMsgs []db.Message
	for _, raw := range rawMessages {
		msg, err := ParseRawMessage(raw, accountID)
		if err != nil {
			slog.Warn("sync: failed to parse message", "uid", raw.UID, "err", err)
			continue
		}

		// Set folder on the parsed message.
		msg.Folder = defaultFolder
		parsedMsgs = append(parsedMsgs, *msg)
	}

	if len(parsedMsgs) == 0 {
		if len(rawMessages) > 0 {
			slog.Warn("sync: fetched messages but none parsed successfully", "account", accountID, "raw", len(rawMessages))
		}
		return 0, nil
	}

	// Build threads.
	threads := BuildThreads(parsedMsgs)

	// Insert threads and messages into the database.
	for threadID, threadMsgs := range threads {
		if len(threadMsgs) == 0 {
			continue
		}

		// Determine split inbox for the thread based on the first message.
		splitInboxID := EvaluateSplitInboxRules(&threadMsgs[0], splitInboxes)

		// Count unread messages in the thread.
		unreadCount := 0
		hasAttachment := false
		var lastMessageAt int64
		for _, m := range threadMsgs {
			if !m.IsRead {
				unreadCount++
			}
			if m.HasAttachment {
				hasAttachment = true
			}
			if m.Date > lastMessageAt {
				lastMessageAt = m.Date
			}
		}

		// Build the thread's snippet from the last message.
		lastMsg := threadMsgs[len(threadMsgs)-1]
		snippet := lastMsg.BodyText
		if len(snippet) > 200 {
			snippet = snippet[:200]
		}

		// Collect unique participants.
		participants := collectParticipants(threadMsgs)

		thread := &db.Thread{
			ID:            threadID,
			AccountID:     accountID,
			Subject:       lastMsg.Subject,
			Snippet:       snippet,
			Participants:  participants,
			MessageCount:  len(threadMsgs),
			UnreadCount:   unreadCount,
			HasAttachment: hasAttachment,
			Labels:        "[]",
			LastMessageAt: lastMessageAt,
			SplitInboxID:  splitInboxID,
		}

		if err := database.InsertThread(thread); err != nil {
			return 0, fmt.Errorf("sync: insert thread: %w", err)
		}

		for i := range threadMsgs {
			threadMsgs[i].ThreadID = threadID
			if err := database.InsertMessage(&threadMsgs[i]); err != nil {
				slog.Warn("sync: failed to insert message", "id", threadMsgs[i].ID, "err", err)
			}
		}
	}

	return len(parsedMsgs), nil
}

// getLastSyncedUID retrieves the last synced UID validity (used as last UID marker)
// for the given account from the database.
func getLastSyncedUID(database *db.DB, accountID string) (uint32, error) {
	var uidValidity uint32
	err := database.Get(&uidValidity, `
		SELECT COALESCE(uidvalidity, 0) FROM accounts WHERE id = ?
	`, accountID)
	if err != nil {
		// Account might not exist yet; that's fine for initial sync.
		return 0, nil
	}
	return uidValidity, nil
}

// collectParticipants builds a JSON array string of unique email addresses
// from the thread's messages.
func collectParticipants(msgs []db.Message) string {
	seen := make(map[string]bool)
	var addrs []string
	for _, m := range msgs {
		if m.FromAddr != "" && !seen[m.FromAddr] {
			seen[m.FromAddr] = true
			addrs = append(addrs, m.FromAddr)
		}
	}
	if len(addrs) == 0 {
		return "[]"
	}

	// Build a simple JSON array.
	result := "["
	for i, a := range addrs {
		if i > 0 {
			result += ","
		}
		result += `"` + a + `"`
	}
	result += "]"
	return result
}
