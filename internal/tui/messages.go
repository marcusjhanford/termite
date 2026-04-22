package tui

import "github.com/termite-mail/termite/internal/tui/msgs"

// Re-export shared message types so existing consumers of the tui package
// continue to work without changes.
type (
	SyncDoneMsg          = msgs.SyncDoneMsg
	SyncErrorMsg         = msgs.SyncErrorMsg
	NewMailMsg           = msgs.NewMailMsg
	InboxZeroMsg         = msgs.InboxZeroMsg
	MilestoneUnlockedMsg = msgs.MilestoneUnlockedMsg
	ComposeMsg           = msgs.ComposeMsg
	NavigateMsg          = msgs.NavigateMsg
	StatusMsg            = msgs.StatusMsg
	SearchResultsMsg     = msgs.SearchResultsMsg
	CommandMsg           = msgs.CommandMsg
)
