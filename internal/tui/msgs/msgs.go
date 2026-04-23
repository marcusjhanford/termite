// Package msgs defines shared message types used across TUI packages
// to avoid import cycles between tui and tui/pages/*.
package msgs

// SyncDoneMsg is sent when IMAP sync completes for an account.
type SyncDoneMsg struct {
	AccountID string
	NewCount  int
}

// SyncErrorMsg is sent when an IMAP sync fails.
type SyncErrorMsg struct {
	AccountID string
	Err       error
}

// SyncPulseMsg drives a lightweight indeterminate progress animation while
// background IMAP sync runs (see mainpage.SyncPulseCmd).
type SyncPulseMsg struct{}

// NewMailMsg is sent when new mail arrives via IDLE.
type NewMailMsg struct {
	AccountID string
	Count     int
}

// InboxZeroMsg is fired when the unread count for an account reaches 0.
type InboxZeroMsg struct {
	AccountID string
}

// ReloadInboxMsg triggers a thread list refresh on the main page (e.g. after archive).
type ReloadInboxMsg struct{}

// MailSendResultMsg reports the outcome of an asynchronous SMTP send from compose.
type MailSendResultMsg struct {
	Err       error
	AccountID string
}

// MilestoneUnlockedMsg is sent when a new achievement milestone is unlocked.
type MilestoneUnlockedMsg struct {
	MilestoneID string
	Title       string
	Description string
}

// ComposeMsg opens the compose view.
// Mode is one of "new", "reply", "replyall", "forward".
type ComposeMsg struct {
	Mode     string
	ThreadID string
}

// NavigateMsg requests page navigation.
type NavigateMsg struct {
	Page string
	// KickInitialSync, when true with Page "main", triggers the same background
	// IMAP sync as a cold start (e.g. after first-run setup added accounts).
	KickInitialSync bool
}

// StatusMsg updates the status bar text.
type StatusMsg struct {
	Text string
}

// SearchResultsMsg carries results from an FTS5 search.
type SearchResultsMsg struct {
	Query     string
	ThreadIDs []string
	Total     int
}

// CommandMsg is relayed from mainpage when a command bar command
// needs to be handled at the app level.
type CommandMsg struct {
	Command string
}

// InboxesChangedMsg is sent when split inboxes are created, deleted, or
// updated so the TUI can refresh the inbox list and counts.
type InboxesChangedMsg struct{}

// MoveThreadMsg requests moving a thread to a different split inbox.
type MoveThreadMsg struct {
	ThreadID    string
	TargetInbox string
	CreateRoute bool
	MatchDomain bool
	ApplyPast   bool
}

// SpamThreadMsg requests marking a thread as spam.
type SpamThreadMsg struct {
	ThreadID    string
	ApplyPast   bool
}
