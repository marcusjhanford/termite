package tui

import (
	"fmt"
	"log/slog"
	"strings"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"

	"github.com/termite-mail/termite/internal/config"
	"github.com/termite-mail/termite/internal/db"
	"github.com/termite-mail/termite/internal/engine"
	"github.com/termite-mail/termite/internal/metrics"
	"github.com/termite-mail/termite/internal/themes"
	"github.com/termite-mail/termite/internal/tui/commands"
	milestonetoast "github.com/termite-mail/termite/internal/tui/components/milestone_toast"
	"github.com/termite-mail/termite/internal/tui/msgs"
	composepage "github.com/termite-mail/termite/internal/tui/pages/compose"
	inboxzeropage "github.com/termite-mail/termite/internal/tui/pages/inbox_zero"
	mainpage "github.com/termite-mail/termite/internal/tui/pages/main"
	metricsdashboard "github.com/termite-mail/termite/internal/tui/pages/metrics_dashboard"
	setuppage "github.com/termite-mail/termite/internal/tui/pages/setup"
)

// Page represents the currently active page in the application.
type Page int

const (
	PageMain Page = iota
	PageCompose
	PageSetup
	PageInboxZero
	PageMetrics
)

// appModel is the root Bubble Tea model that owns all pages and state.
type appModel struct {
	cfg     *config.Config
	db      *db.DB
	themes  *themes.ThemeManager
	metrics *metrics.MetricsTracker
	keymap  KeyMap

	width  int
	height int

	activePage    Page
	mainPage      mainpage.Model
	composePage   composepage.Model
	setupPage     setuppage.Model
	inboxZeroPage inboxzeropage.Model
	metricsPage   metricsdashboard.Model

	cmdRegistry    *commands.Registry
	milestoneToast milestonetoast.Model

	statusMsg string
	ready     bool
	showHelp  bool

	// Theme picker overlay state.
	showThemePicker bool
	themeChoices    []themes.ThemeInfo
	themeCursor     int
}

// NewAppModel creates the root application model.
func NewAppModel(
	cfg *config.Config,
	database *db.DB,
	themeManager *themes.ThemeManager,
	tracker *metrics.MetricsTracker,
) appModel {
	km := BuildKeyMap(cfg)

	startPage := PageMain
	if len(cfg.Accounts) == 0 {
		startPage = PageSetup
	}

	// Build the command registry and register all commands.
	reg := commands.NewRegistry()
	reg.Register(commands.ConnectCommand())
	reg.Register(commands.InboxCommand())
	reg.Register(commands.SearchCommand())
	reg.Register(commands.ThemeCommand())
	reg.Register(commands.ShortcutsCommand())
	reg.Register(commands.MetricsCommand())
	reg.Register(commands.DaemonCommand())

	mp := mainpage.New(cfg, database, tracker)
	if len(cfg.Accounts) > 0 && database != nil {
		mp = mp.WithBackgroundSyncExpected(len(cfg.Accounts))
	}

	// Pass registered command names to the command bar for autocomplete.
	allCmds := reg.All()
	cmdNames := make([]string, len(allCmds))
	for i, c := range allCmds {
		cmdNames[i] = c.Name
	}
	mp.SetCommandNames(cmdNames)

	return appModel{
		cfg:            cfg,
		db:             database,
		themes:         themeManager,
		metrics:        tracker,
		keymap:         km,
		activePage:     startPage,
		mainPage:       mp,
		composePage:    composepage.New(),
		setupPage:      setuppage.New(cfg),
		inboxZeroPage:  inboxzeropage.New(themeManager.Current()),
		metricsPage:    metricsdashboard.New(),
		cmdRegistry:    reg,
		milestoneToast: milestonetoast.New(),
	}
}

// Init implements tea.Model. It kicks off the initial IMAP sync for every
// configured account so the inbox is populated as soon as the TUI appears.
func (m appModel) Init() tea.Cmd {
	if len(m.cfg.Accounts) == 0 || m.db == nil {
		return nil
	}
	return tea.Batch(mainpage.SyncPulseCmd(), m.startInitialSync())
}

// startInitialSync returns a tea.Batch that launches one sync command per
// configured account. Each command runs in its own goroutine and reports
// back with a SyncDoneMsg or SyncErrorMsg.
func (m appModel) startInitialSync() tea.Cmd {
	var cmds []tea.Cmd
	for _, acctCfg := range m.cfg.Accounts {
		acctCfg := acctCfg
		database := m.db
		splitInboxes := m.cfg.SplitInboxes

		cmds = append(cmds, func() tea.Msg {
			acct, err := engine.NewAccount(acctCfg)
			if err != nil {
				slog.Warn("init sync: failed to create account", "id", acctCfg.ID, "err", err)
				return msgs.SyncErrorMsg{AccountID: acctCfg.ID, Err: err}
			}
			result := engine.InitialSync(acct, database, splitInboxes)
			if result.Err != nil {
				return msgs.SyncErrorMsg{AccountID: result.AccountID, Err: result.Err}
			}
			return msgs.SyncDoneMsg{AccountID: result.AccountID, NewCount: result.NewCount}
		})
	}
	return tea.Batch(cmds...)
}

// Update implements tea.Model. It routes messages to the active page
// and handles global key events and system messages.
func (m appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.milestoneToast.SetSize(msg.Width, msg.Height)

		// Propagate resize to the active page only. Inactive pages
		// will receive the correct size when they become active.
		var cmd tea.Cmd
		switch m.activePage {
		case PageMain:
			m.mainPage, cmd = m.mainPage.Update(msg)
		case PageCompose:
			m.composePage, cmd = m.composePage.Update(msg)
		case PageSetup:
			m.setupPage, cmd = m.setupPage.Update(msg)
		case PageInboxZero:
			m.inboxZeroPage, cmd = m.inboxZeroPage.Update(msg)
		case PageMetrics:
			m.metricsPage, cmd = m.metricsPage.Update(msg)
		}
		cmds = append(cmds, cmd)

		return m, tea.Batch(cmds...)

	case themes.StylesUpdatedMsg:
		// Theme changed — propagate to active page only.
		var cmd tea.Cmd
		switch m.activePage {
		case PageMain:
			m.mainPage, cmd = m.mainPage.Update(msg)
		case PageCompose:
			m.composePage, cmd = m.composePage.Update(msg)
		case PageSetup:
			m.setupPage, cmd = m.setupPage.Update(msg)
		case PageInboxZero:
			m.inboxZeroPage, cmd = m.inboxZeroPage.Update(msg)
		case PageMetrics:
			m.metricsPage, cmd = m.metricsPage.Update(msg)
		}
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case msgs.SyncPulseMsg:
		var cmd tea.Cmd
		m.mainPage, cmd = m.mainPage.Update(msg)
		return m, cmd

	case msgs.SyncDoneMsg:
		m.statusMsg = "Sync complete"
		var cmd tea.Cmd
		m.mainPage, cmd = m.mainPage.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case msgs.SyncErrorMsg:
		m.statusMsg = "Sync error: " + msg.Err.Error()
		var cmd tea.Cmd
		m.mainPage, cmd = m.mainPage.Update(msg)
		return m, cmd

	case msgs.NewMailMsg:
		m.statusMsg = "New mail"
		var cmd tea.Cmd
		m.mainPage, cmd = m.mainPage.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case msgs.InboxZeroMsg:
		// Create a fresh inbox zero page with the current theme and switch to it.
		m.inboxZeroPage = inboxzeropage.New(m.themes.Current())
		m.activePage = PageInboxZero
		initCmd := m.inboxZeroPage.Init()
		// Also propagate the current window size so the scene can be created.
		var sizeCmd tea.Cmd
		m.inboxZeroPage, sizeCmd = m.inboxZeroPage.Update(tea.WindowSizeMsg{
			Width:  m.width,
			Height: m.height,
		})
		return m, tea.Batch(initCmd, sizeCmd)

	case msgs.ComposeMsg:
		m.initCompose(msg.Mode, msg.ThreadID)
		return m, nil

	case msgs.NavigateMsg:
		switch msg.Page {
		case "main":
			m.activePage = PageMain
		case "compose":
			m.activePage = PageCompose
		case "setup":
			m.activePage = PageSetup
		case "inbox_zero":
			m.activePage = PageInboxZero
		case "metrics":
			m.activePage = PageMetrics
		}
		// Send the newly active page the current window size.
		m.sendSizeToActivePage()

		if msg.Page == "main" && msg.KickInitialSync && len(m.cfg.Accounts) > 0 && m.db != nil {
			m.mainPage = m.mainPage.WithBackgroundSyncExpected(len(m.cfg.Accounts))
			return m, tea.Batch(mainpage.SyncPulseCmd(), m.startInitialSync())
		}
		return m, nil

	case msgs.StatusMsg:
		m.statusMsg = msg.Text
		return m, nil

	case msgs.MilestoneUnlockedMsg:
		m.statusMsg = "Milestone: " + msg.Title
		m.milestoneToast.Queue(milestonetoast.MilestoneDisplay{
			Icon:  "",
			Label: msg.Title,
			Desc:  msg.Description,
		})
		cmd := m.milestoneToast.StartIfIdle()
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	case msgs.CommandMsg:
		// Dispatch through the command registry.
		cmd := m.cmdRegistry.Dispatch(msg.Command)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	// ── Command result messages ──────────────────────────────────────

	case commands.SwitchInboxMsg:
		// Forward to the main page — it handles inbox switching internally.
		var cmd tea.Cmd
		m.mainPage, cmd = m.mainPage.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case commands.SearchMsg:
		// Forward to the main page — it handles search internally.
		var cmd tea.Cmd
		m.mainPage, cmd = m.mainPage.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case commands.ThemeChangeMsg:
		stylesMsg, err := m.themes.ApplyMsg(msg.ThemeID)
		if err != nil {
			m.statusMsg = "Theme error: " + err.Error()
			return m, nil
		}
		m.statusMsg = "Theme changed to " + msg.ThemeID
		// Re-enter Update with the StylesUpdatedMsg so all pages refresh.
		return m, func() tea.Msg { return stylesMsg }

	case commands.ThemeListMsg:
		available, err := m.themes.Discover()
		if err != nil {
			m.statusMsg = "Could not list themes: " + err.Error()
		} else {
			m.themeChoices = available
			m.themeCursor = 0
			cur := m.themes.Current()
			for i, t := range available {
				if t.ID == cur.Name || t.Name == cur.Name {
					m.themeCursor = i
					break
				}
			}
			m.showThemePicker = true
		}
		return m, nil

	case commands.CommandErrorMsg:
		m.statusMsg = msg.Err
		return m, nil

	case commands.ShortcutsMsg:
		m.statusMsg = "Shortcuts: c=compose r=reply R=replyall f=forward e=archive d=delete /=search :=command q=quit(main) ctrl+c=quit(any)"
		return m, nil

	case commands.DaemonStatusMsg:
		m.statusMsg = "Daemon: checking status..."
		return m, nil

	case commands.DaemonStartMsg:
		m.statusMsg = "Daemon: starting..."
		return m, nil

	case commands.DaemonStopMsg:
		m.statusMsg = "Daemon: stopping..."
		return m, nil

	case commands.MetricsExportMsg:
		m.statusMsg = "Exporting metrics as " + msg.Format + "..."
		return m, nil

	case tea.KeyPressMsg:
		keyStr := msg.String()

		// Global quit must be checked before any overlay intercepts.
		if keyStr == "ctrl+c" {
			return m, tea.Quit
		}

		// When the main page's command bar is active, let the main page
		// handle ALL key events — don't intercept with app-level bindings.
		if m.activePage == PageMain && m.mainPage.CommandBarActive() {
			var cmd tea.Cmd
			m.mainPage, cmd = m.mainPage.Update(msg)
			cmds = append(cmds, cmd)
			return m, tea.Batch(cmds...)
		}

		// Toggle help modal with ? key.
		if keyStr == "?" {
			m.showHelp = !m.showHelp
			return m, nil
		}

		// Dismiss help on any key if it's showing.
		if m.showHelp {
			m.showHelp = false
			return m, nil
		}

		// Theme picker overlay navigation.
		if m.showThemePicker {
			switch keyStr {
			case "j", "down":
				if m.themeCursor < len(m.themeChoices)-1 {
					m.themeCursor++
				}
			case "k", "up":
				if m.themeCursor > 0 {
					m.themeCursor--
				}
			case "enter":
				if m.themeCursor < len(m.themeChoices) {
					chosen := m.themeChoices[m.themeCursor]
					m.showThemePicker = false
					stylesMsg, err := m.themes.ApplyMsg(chosen.ID)
					if err != nil {
						m.statusMsg = "Theme error: " + err.Error()
						return m, nil
					}
					m.statusMsg = "Theme changed to " + chosen.ID
					return m, func() tea.Msg { return stylesMsg }
				}
			case "esc", "q":
				m.showThemePicker = false
			}
			return m, nil
		}

		// Quit with q only on the main inbox view. Other pages collect text
		// (email, compose body, etc.); ctrl+c still quits from anywhere.
		if keyStr == "q" && m.activePage == PageMain {
			return m, tea.Quit
		}

		// Escape from overlay pages returns to main.
		if keyStr == "esc" {
			switch m.activePage {
			case PageCompose, PageInboxZero, PageMetrics:
				m.activePage = PageMain
				return m, nil
			case PageSetup:
				if m.setupPage.CanExit() {
					m.activePage = PageMain
					return m, nil
				}
			}
		}

		// Debug: Ctrl+Z triggers inbox zero forest scene directly.
		if m.activePage == PageMain && keyStr == "ctrl+z" {
			m.inboxZeroPage = inboxzeropage.New(m.themes.Current())
			m.activePage = PageInboxZero
			initCmd := m.inboxZeroPage.Init()
			var sizeCmd tea.Cmd
			m.inboxZeroPage, sizeCmd = m.inboxZeroPage.Update(tea.WindowSizeMsg{
				Width:  m.width,
				Height: m.height,
			})
			return m, tea.Batch(initCmd, sizeCmd)
		}

		// Main-page-only keybindings — use string matching to avoid
		// any issues with key.Matches and binding state.
		if m.activePage == PageMain {
			switch keyStr {
			case "c":
				m.initCompose("new", "")
				return m, nil
			case "r":
				m.initCompose("reply", "")
				return m, nil
			case "R":
				m.initCompose("replyall", "")
				return m, nil
			case "f":
				m.initCompose("forward", "")
				return m, nil
			case "e":
				return m, m.archiveSelected()
			case "#":
				return m, m.deleteSelected()
			case "m":
				return m, m.markSelectedRead()
			}
			// All other keys (tab, j, k, /, :, etc.) fall through
			// to the active page router below.
		}
	}

	// Route remaining messages to the active page.
	var cmd tea.Cmd
	switch m.activePage {
	case PageMain:
		m.mainPage, cmd = m.mainPage.Update(msg)
	case PageCompose:
		m.composePage, cmd = m.composePage.Update(msg)
	case PageSetup:
		m.setupPage, cmd = m.setupPage.Update(msg)
		// Check if the setup page emitted a NavigateMsg.
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	case PageInboxZero:
		m.inboxZeroPage, cmd = m.inboxZeroPage.Update(msg)
	case PageMetrics:
		m.metricsPage, cmd = m.metricsPage.Update(msg)
	}
	cmds = append(cmds, cmd)

	// Also let the milestone toast process (for its internal ticks).
	var toastCmd tea.Cmd
	m.milestoneToast, toastCmd = m.milestoneToast.Update(msg)
	if toastCmd != nil {
		cmds = append(cmds, toastCmd)
	}

	return m, tea.Batch(cmds...)
}

// View implements tea.Model. It renders the active page.
func (m appModel) View() tea.View {
	if !m.ready {
		v := tea.NewView("Loading...")
		v.AltScreen = true
		return v
	}

	var s string
	switch m.activePage {
	case PageCompose:
		s = m.composePage.View()
	case PageSetup:
		s = m.setupPage.View()
	case PageInboxZero:
		s = m.inboxZeroPage.View()
	case PageMetrics:
		s = m.metricsPage.View()
	default:
		s = m.mainPage.View()
	}

	// Overlay theme picker if showing.
	if m.showThemePicker {
		s = m.renderThemePicker()
	}

	// Overlay help modal if showing.
	if m.showHelp {
		s = m.renderHelp()
	}

	// Overlay the milestone toast if visible.
	if m.milestoneToast.Visible() {
		toast := m.milestoneToast.View()
		if toast != "" {
			s += "\n" + toast
		}
	}

	v := tea.NewView(s)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

// renderHelp renders the help modal overlay.
func (m appModel) renderHelp() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4")).
		MarginBottom(1)

	sectionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#58a6ff")).
		MarginTop(1)

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#e2e4e8")).
		Bold(true).
		Width(14)

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#8b949e"))

	dimStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#555555")).
		Italic(true)

	row := func(k, d string) string {
		return keyStyle.Render(k) + descStyle.Render(d)
	}

	var lines []string
	lines = append(lines, titleStyle.Render("Termite — Keyboard Shortcuts"))

	lines = append(lines, sectionStyle.Render("Navigation"))
	lines = append(lines, row("j / k", "Move down / up"))
	lines = append(lines, row("Tab", "Next pane"))
	lines = append(lines, row("Shift+Tab", "Previous pane"))
	lines = append(lines, row("Enter", "Open / select"))
	lines = append(lines, row("Esc", "Back / cancel"))

	lines = append(lines, sectionStyle.Render("Actions"))
	lines = append(lines, row("c", "Compose new message"))
	lines = append(lines, row("r", "Reply"))
	lines = append(lines, row("R", "Reply all"))
	lines = append(lines, row("f", "Forward"))
	lines = append(lines, row("e", "Archive thread"))
	lines = append(lines, row("d", "Delete thread"))
	lines = append(lines, row("m", "Mark as read"))

	lines = append(lines, sectionStyle.Render("Command & Search"))
	lines = append(lines, row("/", "Search messages"))
	lines = append(lines, row(":", "Open command bar"))

	// List registered commands.
	allCmds := m.cmdRegistry.All()
	if len(allCmds) > 0 {
		cmdNames := make([]string, len(allCmds))
		for i, c := range allCmds {
			cmdNames[i] = c.Name
		}
		lines = append(lines, descStyle.Render(fmt.Sprintf("  Commands: %s", strings.Join(cmdNames, ", "))))
	}

	lines = append(lines, sectionStyle.Render("Other"))
	lines = append(lines, row("?", "Toggle this help"))
		lines = append(lines, row("q", "Quit (main inbox only)"))
		lines = append(lines, row("Ctrl+C", "Quit (anywhere)"))

	lines = append(lines, "")
	lines = append(lines, dimStyle.Render("Press any key to dismiss"))

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)

	boxWidth := 52
	if m.width > 0 && boxWidth > m.width-4 {
		boxWidth = m.width - 4
	}
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7D56F4")).
		Width(boxWidth).
		Padding(1, 2)

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		boxStyle.Render(content),
	)
}

// renderThemePicker renders the interactive theme selector overlay.
func (m appModel) renderThemePicker() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4")).
		MarginBottom(1)

	activeStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#0d1117")).
		Background(lipgloss.Color("#58a6ff")).
		Padding(0, 1)

	normalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#e2e4e8")).
		Padding(0, 1)

	checkStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#3fb950"))

	dimStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#555555")).
		Italic(true)

	var lines []string
	lines = append(lines, titleStyle.Render("Select Theme"))

	cur := m.themes.Current()
	for i, t := range m.themeChoices {
		label := t.Name
		if t.Builtin {
			label += " (built-in)"
		}
		isCurrent := t.ID == cur.Name || t.Name == cur.Name
		prefix := "  "
		if isCurrent {
			prefix = checkStyle.Render("✓ ")
		}
		if i == m.themeCursor {
			lines = append(lines, prefix+activeStyle.Render(label))
		} else {
			lines = append(lines, prefix+normalStyle.Render(label))
		}
	}

	lines = append(lines, "")
	lines = append(lines, dimStyle.Render("j/k to navigate • Enter to apply • Esc to cancel"))

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)

	boxWidth := 40
	if m.width > 0 && boxWidth > m.width-4 {
		boxWidth = m.width - 4
	}
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7D56F4")).
		Width(boxWidth).
		Padding(1, 2)

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		boxStyle.Render(content),
	)
}

// initCompose creates a compose page with the given mode, sends it the current
// window size, and populates email autocomplete from the database.
func (m *appModel) initCompose(mode, threadID string) {
	m.composePage = composepage.NewWithMode(mode, threadID)
	m.composePage, _ = m.composePage.Update(tea.WindowSizeMsg{
		Width:  m.width,
		Height: m.height,
	})
	// Populate email autocomplete from known addresses.
	if m.db != nil {
		if emails, err := m.db.GetKnownEmails(500); err == nil {
			m.composePage.SetKnownEmails(emails)
		}
	}
	m.activePage = PageCompose
}

// sendSizeToActivePage sends the current window dimensions to whichever
// page just became active. This is needed because we only propagate
// WindowSizeMsg to the active page, so pages created or switched to later
// would never get their dimensions set.
func (m *appModel) sendSizeToActivePage() {
	if m.width == 0 && m.height == 0 {
		return
	}
	sizeMsg := tea.WindowSizeMsg{Width: m.width, Height: m.height}
	switch m.activePage {
	case PageMain:
		m.mainPage, _ = m.mainPage.Update(sizeMsg)
	case PageCompose:
		m.composePage, _ = m.composePage.Update(sizeMsg)
	case PageSetup:
		m.setupPage, _ = m.setupPage.Update(sizeMsg)
	case PageInboxZero:
		m.inboxZeroPage, _ = m.inboxZeroPage.Update(sizeMsg)
	case PageMetrics:
		m.metricsPage, _ = m.metricsPage.Update(sizeMsg)
	}
}

// archiveSelected archives the currently selected thread and refreshes.
func (m *appModel) archiveSelected() tea.Cmd {
	if m.db == nil {
		return nil
	}
	// The main page's thread list manages the selected thread internally.
	// We trigger the archive through the DB using the thread list's selection,
	// but since the threadList is unexported, we send a status message and
	// let the main page refresh via a SyncDoneMsg-style reload.
	// For now, emit a status and forward a refresh.
	return func() tea.Msg {
		return msgs.StatusMsg{Text: "Archive: select a thread first (use main page actions)"}
	}
}

// deleteSelected deletes the currently selected thread and refreshes.
func (m *appModel) deleteSelected() tea.Cmd {
	if m.db == nil {
		return nil
	}
	return func() tea.Msg {
		return msgs.StatusMsg{Text: "Delete: select a thread first (use main page actions)"}
	}
}

// markSelectedRead marks the currently selected thread as read.
func (m *appModel) markSelectedRead() tea.Cmd {
	if m.db == nil {
		return nil
	}
	return func() tea.Msg {
		return msgs.StatusMsg{Text: "Mark read: select a thread first (use main page actions)"}
	}
}

// archiveThread archives a thread by ID and returns a refresh command.
func (m *appModel) archiveThread(threadID string) tea.Cmd {
	if m.db == nil || threadID == "" {
		return nil
	}
	if err := m.db.ArchiveThread(threadID); err != nil {
		slog.Warn("failed to archive thread", "thread", threadID, "err", err)
		return func() tea.Msg {
			return msgs.StatusMsg{Text: "Archive failed: " + err.Error()}
		}
	}
	return func() tea.Msg {
		return msgs.StatusMsg{Text: "Thread archived"}
	}
}

// deleteThread deletes a thread by ID and returns a refresh command.
func (m *appModel) deleteThread(threadID string) tea.Cmd {
	if m.db == nil || threadID == "" {
		return nil
	}
	if err := m.db.DeleteThread(threadID); err != nil {
		slog.Warn("failed to delete thread", "thread", threadID, "err", err)
		return func() tea.Msg {
			return msgs.StatusMsg{Text: "Delete failed: " + err.Error()}
		}
	}
	return func() tea.Msg {
		return msgs.StatusMsg{Text: "Thread deleted"}
	}
}

// markThreadRead marks a thread as read by ID.
func (m *appModel) markThreadRead(threadID string) tea.Cmd {
	if m.db == nil || threadID == "" {
		return nil
	}
	if err := m.db.MarkThreadRead(threadID); err != nil {
		slog.Warn("failed to mark thread read", "thread", threadID, "err", err)
		return func() tea.Msg {
			return msgs.StatusMsg{Text: "Mark read failed: " + err.Error()}
		}
	}
	return func() tea.Msg {
		return msgs.StatusMsg{Text: "Thread marked as read"}
	}
}
