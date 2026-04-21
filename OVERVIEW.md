# Termite — Technical Specification v2.1
> A Superhuman-inspired, keyboard-first TUI email client. Built in Go. Runs locally. Your data stays yours.

---

## Why Go, and why the full Charm ecosystem

The previous spec used Python + Textual. Textual's rendering hit practical limits fast — layout math is manual, widget composition is rigid, and the CSS engine doesn't map cleanly to multi-pane terminal UIs with live overlays.

The right stack is **pure Go using the Charm ecosystem**: Bubble Tea v2 for the TUI runtime, Lip Gloss v2 for styling and layout, and Bubbles v2 for pre-built interactive components. This is the same stack Charm uses for their own production TUI apps — Crush (18k+ stars) uses Bubble Tea v2, Lip Gloss, and SQLite in exactly the same architectural shape as Termite.

There is no split between a "rendering layer" and a "backend language." Bubble Tea is Go, and Go handles IMAP, SMTP, SQLite, OAuth2, and system notifications natively. One language, one binary, one dependency tree.

### The Charm ecosystem

| Library | Role |
|---|---|
| `charm.land/bubbletea/v2` | TUI runtime — Elm Architecture (Model/Update/View), async Cmds, key/mouse events |
| `charm.land/lipgloss/v2` | Styling and layout — CSS-like API, adaptive light/dark colors, auto color downsampling |
| `charm.land/bubbles/v2` | Pre-built components — `textinput`, `textarea`, `list`, `viewport`, `spinner`, `key` |

### Why Bubble Tea specifically

Bubble Tea is built on The Elm Architecture: every UI update is a pure function `(Model, Msg) → (Model, Cmd)`. There are no goroutines in UI code — async work (IMAP sync, OAuth flows, DB queries) runs as `tea.Cmd` values that return `tea.Msg` results back to the update loop. This makes the UI fully deterministic and testable. The renderer is cell-based and high-performance, with a separate high-performance scrollable region renderer for long lists. Mouse support, focus management, and key handling are all built in.

### Lip Gloss for themes

Lip Gloss takes a declarative, CSS-like approach to terminal styling. Styles are Go values:

```go
titleStyle := lipgloss.NewStyle().
    Bold(true).
    Foreground(lipgloss.Color("#FAFAFA")).
    Background(lipgloss.Color("#7D56F4")).
    PaddingLeft(2).PaddingRight(2)
```

Themes in Termite are Go structs of `lipgloss.Color` values, loaded from TOML files at startup and hot-swapped at runtime. Because Lip Gloss automatically downsamples colors to the best available terminal profile (truecolor → 256 → 16 → monochrome), themes work correctly on any terminal without any extra code.

---

## Full tech stack

| Concern | Library | Notes |
|---|---|---|
| TUI runtime | `charm.land/bubbletea/v2` | Elm Architecture, async Cmds, cell renderer |
| Styling / layout | `charm.land/lipgloss/v2` | Declarative CSS-like styles, adaptive colors |
| UI components | `charm.land/bubbles/v2` | textinput, textarea, list, viewport, spinner |
| Spring animations | `github.com/charmbracelet/harmonica` | Physics-based spring model for smooth motion |
| IMAP client | `github.com/emersion/go-imap/v2/imapclient` | IMAP4rev2, IDLE, async |
| SMTP | `github.com/emersion/go-smtp` | SMTP client |
| Email parsing | `github.com/emersion/go-message` | MIME, charset, headers |
| Local DB | `modernc.org/sqlite` + `github.com/jmoiron/sqlx` | CGO-free SQLite; FTS5 for search |
| DB queries | `github.com/sqlc-dev/sqlc` | Generate typed Go from SQL — no raw queries in app code |
| Credential storage | `github.com/zalando/go-keyring` | OS keyring (Keychain / libsecret / Win Credential) |
| Gmail OAuth2 | `golang.org/x/oauth2` | PKCE flow, refresh token management |
| Outlook OAuth2 | `github.com/AzureAD/microsoft-authentication-library-for-go` | MSAL device code flow |
| Config | `github.com/BurntSushi/toml` | Read/write TOML config |
| Config validation | `github.com/go-playground/validator/v10` | Struct-tag validation |
| Desktop notifs | `github.com/gen2brain/beeep` | macOS / Linux / Windows native notifications |
| Markdown / HTML | `github.com/charmbracelet/glamour` | Stylesheet-based Markdown / HTML→text rendering |
| Logging | `golang.org/x/exp/slog` or `log/slog` (stdlib 1.21+) | Structured logging to `~/.termite/termite.log` |
| CLI entry | `github.com/spf13/cobra` | `termite`, `termite daemon`, `termite install-daemon` |
| Go version | `1.22+` | Required for `slog`, range-over-func |

---

## Repository structure

```
termite/
├── go.mod
├── go.sum
├── main.go                     # entry point → cmd.Execute()
├── README.md
├── CONTRIBUTING.md
├── AGENTS.md                   # instructions for coding agents working on the repo
├── .github/
│   ├── ISSUE_TEMPLATE/
│   │   ├── bug_report.md
│   │   ├── feature_request.md
│   │   └── new_provider.md
│   └── workflows/
│       ├── ci.yml              # go test ./... + go vet + staticcheck on every PR
│       └── release.yml         # goreleaser on version tag
├── cmd/
│   ├── root.go                 # cobra root command, global flags
│   ├── run.go                  # `termite` — launch TUI
│   ├── daemon.go               # `termite daemon` — headless sync loop
│   └── install_daemon.go       # `termite install-daemon` — write launchd/systemd service
├── internal/
│   ├── app/
│   │   └── app.go              # App struct: wires together all services, starts TUI
│   ├── config/
│   │   ├── config.go           # Config struct + TOML loading/saving
│   │   ├── defaults.go         # default keybindings, theme name, check interval
│   │   └── validate.go         # validator tags + custom rules
│   ├── engine/
│   │   ├── account.go          # Account type + manager
│   │   ├── imap.go             # go-imap v2 wrapper: connect, fetch, IDLE, move, delete
│   │   ├── smtp.go             # go-smtp wrapper: send, STARTTLS
│   │   ├── sync.go             # SyncWorker: initial + incremental + IDLE loop
│   │   ├── thread.go           # JWZ threading algorithm
│   │   └── parser.go           # go-message MIME parsing, HTML→text, attachments
│   ├── providers/
│   │   ├── base.go             # Provider interface
│   │   ├── gmail.go            # Gmail OAuth2 (localhost redirect) + IMAP/SMTP settings
│   │   ├── outlook.go          # Outlook MSAL device code flow + IMAP/SMTP settings
│   │   ├── fastmail.go         # Fastmail app password + IMAP/SMTP settings
│   │   └── generic.go          # Plain IMAP/SMTP, user-supplied host/port
│   ├── db/
│   │   ├── migrations/         # numbered .sql migration files
│   │   │   ├── 001_initial.sql
│   │   │   ├── 002_fts.sql
│   │   │   └── 003_metrics.sql # NEW: metrics + milestones tables
│   │   ├── sql/                # raw SQL consumed by sqlc
│   │   │   ├── messages.sql
│   │   │   ├── threads.sql
│   │   │   ├── accounts.sql
│   │   │   └── metrics.sql     # NEW: metrics queries
│   │   ├── db.go               # open connection, run migrations
│   │   └── queries.go          # sqlc-generated typed query functions
│   ├── notifications/
│   │   ├── manager.go          # NotificationManager: routes to correct backend
│   │   ├── desktop.go          # beeep wrapper
│   │   ├── tmux.go             # tmux window title + pane flag
│   │   └── status.go           # ~/.termite/status.json writer
│   ├── themes/
│   │   ├── theme.go            # Theme struct (all lipgloss.Color fields)
│   │   ├── manager.go          # ThemeManager: discover, validate, apply, hot-swap
│   │   ├── builtin/            # built-in .toml theme files
│   │   │   ├── dark.toml
│   │   │   ├── light.toml
│   │   │   ├── dracula.toml
│   │   │   ├── tokyo-night.toml
│   │   │   ├── catppuccin-mocha.toml
│   │   │   ├── catppuccin-latte.toml
│   │   │   ├── gruvbox.toml
│   │   │   ├── nord.toml
│   │   │   ├── solarized-dark.toml
│   │   │   ├── high-contrast.toml
│   │   │   └── matrix.toml
│   │   └── styles.go           # build lipgloss.Style values from a Theme
│   ├── animation/              # NEW: animation subsystem
│   │   ├── spring.go           # harmonica spring wrappers for smooth motion
│   │   ├── transitions.go      # page/pane transition frames
│   │   ├── forest.go           # inbox-zero forest scene renderer
│   │   ├── particles.go        # leaf-fall and firefly particle system
│   │   ├── milestone.go        # milestone achievement toast animations
│   │   └── quotes.go           # curated calming quote pool
│   ├── metrics/                # NEW: productivity metrics subsystem
│   │   ├── tracker.go          # MetricsTracker: record events, query summaries
│   │   ├── milestones.go       # milestone definitions + unlock logic
│   │   └── export.go           # export metrics to JSON/CSV
│   ├── tui/
│   │   ├── app_model.go        # root appModel: implements tea.Model, owns all pages + dialogs
│   │   ├── keymap.go           # KeyMap struct + default bindings from config
│   │   ├── messages.go         # all custom tea.Msg types (SyncDone, NewMail, etc.)
│   │   ├── pages/
│   │   │   ├── main/
│   │   │   │   ├── model.go    # mainPage: three-pane layout
│   │   │   │   ├── update.go
│   │   │   │   └── view.go
│   │   │   ├── compose/
│   │   │   │   ├── model.go    # composePage: full-screen compose overlay
│   │   │   │   ├── update.go
│   │   │   │   └── view.go
│   │   │   ├── setup/
│   │   │   │   ├── model.go    # setupPage: first-run wizard
│   │   │   │   ├── update.go
│   │   │   │   └── view.go
│   │   │   ├── inbox_zero/     # NEW: inbox zero celebration page
│   │   │   │   ├── model.go
│   │   │   │   ├── update.go
│   │   │   │   └── view.go
│   │   │   └── metrics_dashboard/ # NEW: metrics dashboard page
│   │   │       ├── model.go
│   │   │       ├── update.go
│   │   │       └── view.go
│   │   ├── components/
│   │   │   ├── inbox_list/     # left pane: split inbox tabs + unread counts
│   │   │   ├── thread_list/    # middle pane: scrollable thread list (bubbles/list)
│   │   │   ├── message_view/   # right pane: rendered message body (bubbles/viewport)
│   │   │   ├── command_bar/    # bottom command input with fuzzy autocomplete
│   │   │   ├── status_bar/     # sync status, account label, unread count, key hints
│   │   │   ├── compose_editor/ # to/cc/subject inputs + body textarea
│   │   │   └── milestone_toast/ # NEW: achievement toast overlay
│   │   └── commands/
│   │       ├── registry.go     # CommandRegistry: register + dispatch /commands
│   │       ├── connect.go      # /connect wizard
│   │       ├── inbox.go        # /inbox switch
│   │       ├── search.go       # /search
│   │       ├── snooze.go       # /snooze quick-pick
│   │       ├── sessions.go     # /sessions
│   │       ├── theme.go        # /theme list|switch|edit|validate
│   │       ├── shortcuts.go    # /shortcuts cheatsheet overlay
│   │       ├── metrics.go      # NEW: /metrics dashboard command
│   │       └── daemon_cmd.go   # /daemon install|status|stop
│   └── daemon/
│       └── daemon.go           # headless sync loop for launchd/systemd
└── tests/
    ├── engine/
    ├── db/
    ├── providers/
    ├── animation/              # NEW: snapshot tests for animation frames
    ├── metrics/                # NEW: milestone unlock + tracker tests
    ├── tui/                    # teatest snapshot + pilot tests
    └── testdata/
        └── dovecot/            # Docker Dovecot config for integration tests
```

---

## The Elm Architecture in Termite

Every component in Termite is a `tea.Model`:

```go
type Model interface {
    Init() tea.Cmd
    Update(tea.Msg) (tea.Model, tea.Cmd)
    View() string
}
```

The root `appModel` owns the entire application state and routes messages down to the active page and any open dialogs. Pages own their component state. Components (inbox list, thread list, message view) manage their own local state and bubble events up via custom `tea.Msg` types.

**Critical pattern — no goroutines in UI code.** All async work (IMAP sync, DB queries, OAuth flows) runs as `tea.Cmd` functions that return a `tea.Msg` to the update loop:

```go
// CORRECT: wrap async work in a Cmd
func syncAccountCmd(account Account, db *DB) tea.Cmd {
    return func() tea.Msg {
        msgs, err := engine.IncrementalSync(account)
        if err != nil {
            return SyncErrorMsg{Err: err}
        }
        db.InsertMessages(msgs)
        return SyncDoneMsg{Account: account.ID, NewCount: len(msgs)}
    }
}

// WRONG: spawning a goroutine inside Update
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    go func() { /* never do this */ }()
}
```

---

## ✦ Animation & Motion Design System

### Philosophy

Termite targets the same kinetic quality as modern coding agents (Claude Code, Gemini CLI, Warp) — where the terminal feels *alive* rather than static. Every state change, navigation event, and async operation has a visual response. The guiding principle is **purposeful motion**: animations communicate meaning (loading, completing, erranding), never decorate for its own sake.

Three layers of motion:

| Layer | Mechanism | Examples |
|---|---|---|
| **Spring physics** | `github.com/charmbracelet/harmonica` | Pane resize, list scroll easing, focus ring movement |
| **Frame animation** | `tea.Tick` loop at 30 fps | Spinners, forest scene, particle effects, typing indicators |
| **Instant transitions** | Lipgloss color interpolation | Highlight on selection, unread→read fade, status bar flashes |

### `internal/animation/spring.go`

Harmonica provides a critically-damped spring model. Termite uses it for:

```go
// Spring drives smooth pane width animation when resizing
type PaneSpring struct {
    spring    harmonica.Spring
    pos       float64   // current rendered width (fractional)
    target    float64   // target width
    velocity  float64
}

func (s *PaneSpring) Tick(delta time.Duration) {
    s.pos, s.velocity = s.spring.Update(s.pos, s.velocity, s.target)
}

func (s *PaneSpring) Width() int { return int(math.Round(s.pos)) }
```

Spring parameters:
- **Focus ring** (snappy): frequency = 8.0, damping = 1.0
- **Pane resize** (elastic): frequency = 5.0, damping = 0.7
- **Scroll position** (fluid): frequency = 4.0, damping = 0.9
- **Progress bars** (bubbles/progress built-in spring): default blend settings

### `internal/animation/transitions.go`

Page transitions use a shared `TransitionModel` that cross-fades between the outgoing and incoming page view strings, character by character, over 8 frames at 30fps (≈ 267ms total):

```go
type TransitionState int
const (
    TransitionNone TransitionState = iota
    TransitionOut    // dimming current page
    TransitionIn     // revealing new page
)

type TransitionModel struct {
    state       TransitionState
    frame       int       // 0–7
    outView     string    // snapshot of page being left
    inModel     tea.Model // incoming page
}

// View renders the cross-fade by blending fg colors toward Background
// using lipgloss.Darken / lipgloss.Lighten per frame step.
func (t *TransitionModel) View(theme *themes.Theme) string { ... }
```

### UI motion catalogue

| Interaction | Animation |
|---|---|
| `j` / `k` navigation | Cursor slides to new row with spring easing; row highlight colour fades in (3 frames) |
| Archive / delete | Row dims to `TextDim` colour, collapses height over 5 frames, neighbouring rows spring-fill the gap |
| Compose overlay open | Panel slides in from bottom edge (8-frame spring, `TransitionIn`) |
| Sync in progress | Animated `bubbles/spinner` (Dots variant) in status bar; rotates at 10 fps |
| New mail arrives | Status bar unread count increments with a 2-frame flash of `Primary` colour |
| Search results populate | Items fade in with 1-frame stagger per row (max 8 rows staggered) |
| Theme switch | Full-screen cross-fade: old palette dissolves to new over 12 frames |
| First-run wizard steps | Each step slides in from the right (spring transition) |
| Milestone toast | Slides up from bottom-right, holds 3 s, slides back down |
| Inbox zero reached | Full-screen forest scene replaces main view (see §Forest Scene below) |

### Spinner catalogue

All spinners use `bubbles/spinner` with custom `Spinner` structs rather than the built-in defaults, to stay on-brand:

```go
var (
    // Used during initial sync
    SpinnerSync = spinner.Spinner{
        Frames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
        FPS:    time.Second / 12,
    }
    // Used during OAuth flow
    SpinnerAuth = spinner.Spinner{
        Frames: []string{"◐", "◓", "◑", "◒"},
        FPS:    time.Second / 8,
    }
    // Used for background IMAP operations
    SpinnerBg = spinner.Spinner{
        Frames: []string{"·", "·", "•", "●", "•", "·"},
        FPS:    time.Second / 10,
    }
)
```

---

## ✦ Inbox Zero Forest Scene

### Design rationale

Reaching inbox zero is a meaningful moment. Rather than a blank list or a terse "All done!", Termite fills the terminal with a living forest — trees growing, leaves drifting, a firefly blinking in the undergrowth, and a single calming quote fading in. The aesthetic reference is the **cbonsai** family of terminal tree generators, adapted into Bubble Tea's tick-driven rendering loop and layered with Lip Gloss color themes.

The technical reference implementation for the tree growth algorithm is `gobonsai` (`github.com/nothub/gobonsai`) — a pure Go port of cbonsai. Termite does **not** take a runtime dependency on gobonsai; instead, `internal/animation/forest.go` implements the same branching algorithm inline so it integrates cleanly with Bubble Tea's `tea.Tick` loop and renders into Lip Gloss–styled strings rather than ncurses cells.

### `internal/animation/forest.go`

```go
// ForestScene renders a growing tree with leaf particles and a quote.
// It is driven by tea.Tick at 30ms (≈33fps) during growth,
// slowing to 80ms once the tree is fully grown.
type ForestScene struct {
    width, height int
    tree          treeState      // current branch grid (sparse cell map)
    particles     []Particle     // falling leaves + firefly
    growTick      int            // branch growth step counter
    maxLife       int            // total branch steps (scales with terminal width)
    phase         ScenePhase     // Growing | Settling | Alive | Fading
    fadeAlpha     float64        // 0.0–1.0 for fade-out when leaving scene
    quote         string         // chosen at scene init, never changes
    quoteReveal   int            // characters revealed so far (typewriter effect)
}

type ScenePhase int
const (
    PhaseGrowing  ScenePhase = iota // tree branches extend step-by-step
    PhaseSettling                   // final leaves cascade in
    PhaseAlive                      // scene breathes: particles loop, firefly blinks
    PhaseFading                     // fade out when user presses any key
)
```

#### Tree growth algorithm

The algorithm is a direct adaptation of cbonsai's branching logic, ported to Go and made tick-safe:

```go
type cellType int
const (
    CellEmpty  cellType = iota
    CellTrunk           // trunk character: │ ║
    CellBranch          // branch chars: ─ ╱ ╲ ┐ └
    CellLeaf            // leaf chars: & * @ # chosen from theme.leafChars
)

type treeCell struct {
    kind  cellType
    color lipgloss.Color
}

type branchState struct {
    x, y     int
    life     int     // remaining steps before this branch dies
    kind     string  // "trunk" | "shootLeft" | "shootRight" | "dying"
    dx, dy   int
}

// GrowStep advances the tree by one tick.
// Returns true if growth is complete.
func (s *ForestScene) GrowStep() bool {
    next := make([]branchState, 0, len(s.branches))
    for _, b := range s.branches {
        s.drawBranch(b)           // paint cell into s.tree
        children := s.split(b)   // 0, 1, or 2 children
        next = append(next, children...)
    }
    s.branches = next
    return len(s.branches) == 0
}
```

Branch splitting rules (ported from cbonsai):

```go
func (s *ForestScene) split(b branchState) []branchState {
    switch b.kind {
    case "trunk":
        // Trunk continues straight up; occasionally sprouts left/right shoots
        if b.life < s.maxLife/4 {
            return s.spawnDying(b)
        }
        if rand.Float64() < 0.15 {
            return []branchState{s.straight(b), s.shootLeft(b), s.shootRight(b)}
        }
        return []branchState{s.straight(b)}
    case "shootLeft", "shootRight":
        // Lateral branches arc outward, thinning over time
        if b.life <= 0 { return nil }
        if rand.Float64() < 0.25 { return []branchState{s.sprout(b)} }
        return []branchState{s.continueShoot(b)}
    case "dying":
        // Dying branches place leaf cells and terminate
        s.placeLeaf(b.x, b.y)
        return nil
    }
    return nil
}
```

#### Character and color selection

Characters and colors are drawn from the active theme's forest palette. Each theme TOML gains a `[forest]` section:

```toml
# Example: tokyo-night.toml forest section
[forest]
trunk_chars   = ["│", "║", "╷"]
branch_chars  = ["─", "╱", "╲", "┐", "└", "┘", "┌"]
leaf_chars    = ["&", "*", "@", "✦", "✿"]
ground_chars  = ["▁", "▂", "▃", "█"]

trunk_color   = "#6a4f3b"
branch_color  = "#5a4030"
leaf_color_1  = "#9ece6a"   # bright green
leaf_color_2  = "#73daca"   # teal accent
leaf_color_3  = "#e0af68"   # warm gold (autumn leaf)
ground_color  = "#3b4261"
firefly_color = "#e0af68"
sky_color     = "#1a1b26"   # equals theme Background
```

Light themes use warmer trunk browns and softer leaf greens. The `matrix.toml` theme renders the forest entirely in shades of green-on-black with `@` and `#` as leaf chars, consistent with its palette.

#### Particle system

```go
type ParticleKind int
const (
    ParticleLeaf    ParticleKind = iota
    ParticleFirefly
)

type Particle struct {
    kind      ParticleKind
    x, y      float64   // sub-cell position for smooth drift
    vx, vy    float64   // velocity in cells/tick
    char      string
    color     lipgloss.Color
    life      int       // ticks remaining
    blinkOn   bool      // for firefly: toggles every ~15 ticks
}
```

Leaf particles spawn from the tree canopy once `PhaseSettling` begins: 1–3 new leaves per tick, drifting diagonally downward-left or downward-right with slight random deviation. They disappear when they reach the ground row.

One firefly particle spawns at `PhaseAlive`. It wanders via a random-walk with soft bounds, blinking every 15 ticks by toggling between `firefly_color` and transparent.

#### Quote typewriter

```go
// internal/animation/quotes.go
var forestQuotes = []string{
    "The quieter you become, the more you can hear.",
    "Simplicity is the ultimate sophistication.",
    "You have arrived.",
    "The present moment always will have been.",
    "Rest is not idleness.",
    "Inhale the future. Exhale the past.",
    "Nothing is lost. Everything is transformed.",
    "The inbox is empty. The mind can breathe.",
    "Done is a place you can visit.",
    "Every ending is a clearing.",
}
```

One quote is selected at scene init using a hash of `(accountID + date)` so the same account sees the same quote per day rather than a random one per visit. The quote is revealed character-by-character at 2 chars/tick (60ms each) starting when `PhaseSettling` begins, centered below the tree.

#### Full scene render pipeline

```go
func (s *ForestScene) View() string {
    // 1. Allocate a width×height cell grid, pre-filled with sky_color spaces
    grid := newCellGrid(s.width, s.height)

    // 2. Paint ground row at height-2
    s.renderGround(grid)

    // 3. Paint all tree cells from s.tree sparse map
    s.renderTree(grid)

    // 4. Paint active particles
    for _, p := range s.particles {
        grid.set(int(p.x), int(p.y), p.char, p.color)
    }

    // 5. Build pot / base ASCII art at center-bottom
    s.renderPot(grid)

    // 6. Render grid rows → Lip Gloss styled strings, joined with \n
    rows := grid.toLines()

    // 7. Overlay centered quote with fade-in opacity simulation
    //    (approximated by choosing between text_muted and text colors
    //     based on quoteReveal progress)
    rows = s.overlayQuote(rows)

    // 8. Apply fadeAlpha: darken all colors toward Background if PhaseFading
    if s.phase == PhaseFading {
        rows = s.applyFade(rows)
    }

    return strings.Join(rows, "\n")
}
```

#### Integration with `appModel`

```go
// In tui/app_model.go

type appModel struct {
    // ... existing fields ...
    inboxZero    *inboxzero.Model   // non-nil when showing forest scene
}

func (m appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case InboxZeroMsg:
        // Fired by thread_list when unread count hits 0
        forest := animation.NewForestScene(m.width, m.height, m.theme)
        m.inboxZero = inboxzero.New(forest)
        return m, m.inboxZero.Init()

    case tea.KeyPressMsg:
        if m.inboxZero != nil {
            // Any key fades the scene out, returns to main view
            m.inboxZero.BeginFade()
            return m, m.inboxZero.FadeCmd()
        }
    }
    // ...
}

func (m appModel) View() string {
    if m.inboxZero != nil {
        return m.inboxZero.View()  // full-screen forest replaces all panes
    }
    return m.mainPage.View()
}
```

#### Scene timing summary

| Phase | Duration | tick interval |
|---|---|---|
| `PhaseGrowing` | Variable — ~3–6 s depending on terminal width | 30ms |
| `PhaseSettling` | 1.5 s — leaves fall, quote types in | 60ms |
| `PhaseAlive` | Until any key is pressed | 80ms |
| `PhaseFading` | 500ms (12 frames) | 40ms |

The scene is non-blocking: the user can press any key at any phase to begin the fade-out immediately.

#### Accessibility opt-out

Users who find animations distracting can set `reduce_motion = true` in `[general]`. With this flag:
- The forest scene is replaced by a static centered text block showing just the quote, styled with the theme's `Primary` color and a simple tree ligature `🌲` (or `*` on terminals without emoji support).
- All spring animations snap to their target value instantly.
- Spinner frame rates halve.

```toml
[general]
reduce_motion = false   # default
```

---

## ✦ Productivity Metrics & Milestones

### Philosophy

Termite tracks how much work you are genuinely doing in your inbox — not to gamify email addiction, but to make the invisible labor of email management visible and to reward moments of real clarity. Metrics are always opt-in to display; the tracking itself is always-on (local only, never transmitted) so historical data is available whenever the user wants it.

### Database schema additions — `003_metrics.sql`

```sql
-- 003_metrics.sql

CREATE TABLE daily_metrics (
    date           TEXT PRIMARY KEY,      -- ISO date: "2025-10-14"
    account_id     TEXT,
    emails_cleared INTEGER DEFAULT 0,     -- archived + deleted
    emails_sent    INTEGER DEFAULT 0,
    inbox_zeros    INTEGER DEFAULT 0,     -- times unread count hit 0
    time_in_app_s  INTEGER DEFAULT 0,     -- seconds with termite focused
    streak_days    INTEGER DEFAULT 0      -- computed at read time
);

CREATE TABLE milestones (
    id             TEXT PRIMARY KEY,      -- e.g. "sent_100"
    unlocked_at    INTEGER,               -- unix timestamp, NULL if locked
    shown          INTEGER DEFAULT 0      -- 1 once the toast has been displayed
);

CREATE TABLE milestone_definitions (
    id             TEXT PRIMARY KEY,
    category       TEXT,                  -- "cleared" | "sent" | "streak" | "zero"
    threshold      INTEGER,
    label          TEXT,
    description    TEXT,
    icon           TEXT                   -- single unicode char for toast display
);
```

### `internal/metrics/tracker.go`

```go
type MetricsTracker struct {
    db        *db.DB
    sessionStart time.Time
    today     string  // ISO date, refreshed at midnight
}

// Called by engine after archiving or deleting a thread
func (t *MetricsTracker) RecordCleared(accountID string, count int) error

// Called by smtp after a send succeeds
func (t *MetricsTracker) RecordSent(accountID string) error

// Called by thread_list when unread count reaches 0
func (t *MetricsTracker) RecordInboxZero(accountID string) error

// Called on clean shutdown — persists session duration
func (t *MetricsTracker) FlushSession() error

// Returns today's aggregated summary across all accounts
func (t *MetricsTracker) TodaySummary() (DailySummary, error)

// Returns all-time totals
func (t *MetricsTracker) AllTimeTotals() (Totals, error)

// Returns current streak (consecutive days with at least 1 inbox zero)
func (t *MetricsTracker) CurrentStreak() (int, error)

// Checks for newly-unlocked milestones; returns them for toast display
func (t *MetricsTracker) CheckMilestones() ([]Milestone, error)
```

### Milestone definitions

```go
// internal/metrics/milestones.go

var MilestoneDefinitions = []MilestoneDef{
    // --- Emails cleared ---
    {ID: "cleared_1",     Category: "cleared", Threshold: 1,     Icon: "✦", Label: "First Clear",       Desc: "Archived or deleted your first email"},
    {ID: "cleared_10",    Category: "cleared", Threshold: 10,    Icon: "◆", Label: "Getting Started",    Desc: "10 emails cleared"},
    {ID: "cleared_50",    Category: "cleared", Threshold: 50,    Icon: "▲", Label: "Momentum",           Desc: "50 emails cleared"},
    {ID: "cleared_100",   Category: "cleared", Threshold: 100,   Icon: "●", Label: "Century",            Desc: "100 emails cleared"},
    {ID: "cleared_500",   Category: "cleared", Threshold: 500,   Icon: "★", Label: "Five Hundred",       Desc: "500 emails cleared"},
    {ID: "cleared_1000",  Category: "cleared", Threshold: 1000,  Icon: "✸", Label: "The Archivist",      Desc: "1,000 emails cleared. Your inbox fears you."},
    {ID: "cleared_5000",  Category: "cleared", Threshold: 5000,  Icon: "⬟", Label: "Email Monk",         Desc: "5,000 emails cleared. Total inner peace."},
    {ID: "cleared_10000", Category: "cleared", Threshold: 10000, Icon: "⬡", Label: "Ascended",           Desc: "10,000 emails cleared. You are the inbox."},

    // --- Emails sent ---
    {ID: "sent_1",        Category: "sent",    Threshold: 1,     Icon: "↗", Label: "First Send",         Desc: "Sent your first email from Termite"},
    {ID: "sent_10",       Category: "sent",    Threshold: 10,    Icon: "↗", Label: "In Conversation",     Desc: "10 emails sent"},
    {ID: "sent_100",      Category: "sent",    Threshold: 100,   Icon: "✉", Label: "The Correspondent",   Desc: "100 emails sent"},
    {ID: "sent_500",      Category: "sent",    Threshold: 500,   Icon: "✉", Label: "Prolific",            Desc: "500 emails sent"},
    {ID: "sent_1000",     Category: "sent",    Threshold: 1000,  Icon: "✦", Label: "The Networker",       Desc: "1,000 emails sent"},

    // --- Inbox zeros ---
    {ID: "zero_1",        Category: "zero",    Threshold: 1,     Icon: "○", Label: "First Zero",          Desc: "Reached inbox zero for the first time"},
    {ID: "zero_7",        Category: "zero",    Threshold: 7,     Icon: "◎", Label: "Weekly Zero",         Desc: "Inbox zero 7 times"},
    {ID: "zero_30",       Category: "zero",    Threshold: 30,    Icon: "◉", Label: "Monthly Zero",        Desc: "Inbox zero 30 times"},
    {ID: "zero_100",      Category: "zero",    Threshold: 100,   Icon: "✦", Label: "The Minimalist",      Desc: "Inbox zero 100 times. A way of life."},

    // --- Streaks ---
    {ID: "streak_3",      Category: "streak",  Threshold: 3,     Icon: "~", Label: "3-Day Streak",        Desc: "3 consecutive days reaching inbox zero"},
    {ID: "streak_7",      Category: "streak",  Threshold: 7,     Icon: "≈", Label: "Week Streak",         Desc: "7-day inbox zero streak"},
    {ID: "streak_14",     Category: "streak",  Threshold: 14,    Icon: "≋", Label: "Fortnight",           Desc: "14-day streak. This is a practice now."},
    {ID: "streak_30",     Category: "streak",  Threshold: 30,    Icon: "∿", Label: "The Ritual",          Desc: "30-day streak. You've made peace with email."},
    {ID: "streak_100",    Category: "streak",  Threshold: 100,   Icon: "∞", Label: "Infinite Zero",       Desc: "100-day streak. You may never be bothered again."},
}
```

### Milestone toast — `components/milestone_toast/`

When `CheckMilestones()` returns newly-unlocked milestones, the `appModel` queues them for display as a non-blocking overlay toast.

```
╭──────────────────────────────╮
│  ★  Century                  │
│     100 emails cleared       │
╰──────────────────────────────╯
```

The toast:
- Renders at bottom-right of the terminal using absolute Lip Gloss positioning
- Slides in from off-screen-right (8-frame spring animation)
- Holds for 3 seconds, then slides back out
- Multiple unlocked milestones queue and display sequentially (1.5 s gap between)
- Does not block keyboard input or interfere with any other UI layer
- Uses `Primary` color for the icon and label, `Text` for the description
- Dismissed immediately on any keypress if the user prefers not to wait

```go
// tui/components/milestone_toast/model.go
type Model struct {
    queue    []metrics.Milestone
    current  *metrics.Milestone
    spring   animation.PaneSpring    // x-offset spring: 0 = on-screen, width = hidden
    holdTick int                     // countdown ticks at 100ms while holding
    phase    toastPhase              // SlideIn | Holding | SlideOut | Hidden
}
```

### Metrics dashboard — `/metrics` command

```
/metrics
```

Opens a full-screen dashboard page (`pages/metrics_dashboard/`) with three sections, navigable by `Tab`:

#### Section 1 — Today

```
  Today · Monday 14 Oct                      streak: 7 days  ≈

  Cleared   ████████████████████░░░░  42     ↑ 12 from yesterday
  Sent      ████░░░░░░░░░░░░░░░░░░░░   8
  Zeros     ●●●○○                            3 of 5 possible

  Time in app   1h 23m
```

#### Section 2 — All time

```
  All Time

  Emails cleared    1,247   ✸  The Archivist (unlocked 3 days ago)
  Emails sent         312   ✉  The Correspondent
  Inbox zeros          89   ◉  Monthly Zero (11 away from The Minimalist)
  Longest streak       14   ≋  Fortnight
  Current streak        7   ≈  Week Streak

  Best day   63 emails cleared · 2 Oct
```

#### Section 3 — Milestones

A scrollable grid of all milestone definitions. Unlocked milestones show their icon in `Primary` colour with the unlock date. Locked milestones are dimmed in `TextDim` with their threshold shown.

```
  ✦ First Clear          ✦ cleared ·  1    · 14 Sep
  ◆ Getting Started      ◆ cleared · 10    · 14 Sep
  ▲ Momentum             ▲ cleared · 50    · 28 Sep
  ● Century              ● cleared · 100   · 3 Oct
  ░ Five Hundred         ░ cleared · 500   · ---
  ░ The Archivist        ░ cleared · 1000  · ---
```

#### Progress bars

The `bubbles/progress` component renders all bars with its built-in spring animation — they smoothly fill to the correct percentage when the dashboard opens.

### Status bar integration

The status bar gains a small persistent indicator showing today's cleared count and streak (if > 2):

```
[work] Primary  ·  12 unread  ·  Synced 30s ago      cleared: 42  streak: 7≈      [/ search]
```

This is rendered in `TextMuted` colour and never takes primary visual attention. It is hidden entirely when `show_metrics_in_statusbar = false` (default: `true`).

### Metrics export

```go
// internal/metrics/export.go
func ExportJSON(db *db.DB, path string) error  // writes ~/.termite/metrics.json
func ExportCSV(db *db.DB, path string) error   // writes ~/.termite/metrics.csv
```

Available via:
```
/metrics export json
/metrics export csv
```

The export includes `daily_metrics` and `milestones` tables. Format is stable across Termite versions (additive only). This allows users to build their own dashboards in external tools (Datasette, Observable, Excel, etc.).

---

## Configuration — `~/.termite/config.toml`

```toml
[general]
theme = "tokyo-night"         # built-in name, or path to ~/.termite/themes/custom.toml
editor = "vim"
check_interval_seconds = 60
startup_inbox = "primary"
reduce_motion = false         # NEW: skip all spring/frame animations

[notifications]
desktop = true
terminal_bell = false
tmux_title = true
status_file = true
notify_on = "unread"          # "unread" | "all" | "none"

[metrics]                     # NEW section
enabled = true
show_in_statusbar = true
toast_milestones = true

[keybindings]
compose     = "c"
reply       = "r"
reply_all   = "R"
forward     = "f"
archive     = "e"
delete      = "#"
mark_read   = "m"
mark_unread = "M"
snooze      = "h"
next        = "j"
prev        = "k"
open        = "enter"
zero        = "I"
search      = "/"
command     = ":"
quit        = "q"

[[accounts]]
id       = "work"
name     = "Work"
email    = "you@company.com"
provider = "gmail"            # gmail | outlook | fastmail | generic

[[accounts]]
id       = "personal"
name     = "Personal"
email    = "you@gmail.com"
provider = "gmail"

[[split_inboxes]]
id       = "primary"
label    = "Primary"
accounts = ["work", "personal"]
rules    = [
  { field = "from", not_contains = ["newsletter", "noreply", "notifications"] }
]

[[split_inboxes]]
id       = "newsletters"
label    = "Newsletters"
accounts = ["personal"]
rules    = [{ field = "list_unsubscribe", exists = true }]

[[split_inboxes]]
id       = "notifications"
label    = "Notifs"
accounts = ["work", "personal"]
rules    = [{ field = "from", contains = ["noreply", "no-reply"] }]
```

Credentials are **never stored in config.toml**. They live in the OS keyring under `termite:{account_id}`.

---

## Database schema — `~/.termite/cache.db`

```sql
-- 001_initial.sql
CREATE TABLE accounts (
  id             TEXT PRIMARY KEY,
  email          TEXT NOT NULL,
  provider       TEXT NOT NULL,
  display_name   TEXT,
  uidvalidity    INTEGER,
  last_synced_at INTEGER
);

CREATE TABLE threads (
  id              TEXT PRIMARY KEY,
  account_id      TEXT REFERENCES accounts(id),
  subject         TEXT,
  snippet         TEXT,
  participants    TEXT,       -- JSON array
  message_count   INTEGER DEFAULT 1,
  unread_count    INTEGER DEFAULT 0,
  has_attachment  INTEGER DEFAULT 0,
  labels          TEXT,       -- JSON array
  last_message_at INTEGER,
  snoozed_until   INTEGER,
  is_archived     INTEGER DEFAULT 0,
  is_deleted      INTEGER DEFAULT 0,
  split_inbox_id  TEXT
);

CREATE TABLE messages (
  id              TEXT PRIMARY KEY,
  thread_id       TEXT REFERENCES threads(id),
  account_id      TEXT REFERENCES accounts(id),
  uid             INTEGER,
  folder          TEXT,
  from_addr       TEXT,
  to_addrs        TEXT,       -- JSON
  cc_addrs        TEXT,       -- JSON
  subject         TEXT,
  date            INTEGER,
  body_text       TEXT,
  body_html       TEXT,
  raw_headers     TEXT,
  is_read         INTEGER DEFAULT 0,
  is_starred      INTEGER DEFAULT 0,
  has_attachment  INTEGER DEFAULT 0,
  in_reply_to     TEXT,
  references      TEXT
);

CREATE TABLE attachments (
  id            TEXT PRIMARY KEY,
  message_id    TEXT REFERENCES messages(id),
  filename      TEXT,
  content_type  TEXT,
  size_bytes    INTEGER,
  local_path    TEXT
);

-- 002_fts.sql
CREATE VIRTUAL TABLE messages_fts USING fts5(
  subject, body_text, from_addr, to_addrs,
  content='messages', content_rowid='rowid'
);

CREATE TRIGGER messages_ai AFTER INSERT ON messages BEGIN
  INSERT INTO messages_fts(rowid, subject, body_text, from_addr, to_addrs)
  VALUES (new.rowid, new.subject, new.body_text, new.from_addr, new.to_addrs);
END;
```

All queries are generated by `sqlc` from `.sql` files in `internal/db/sql/`. No raw query strings live in application code.

---

## IMAP engine — `internal/engine/`

### `engine/imap.go`

Uses `github.com/emersion/go-imap/v2/imapclient`. Unlike `imapclient` in Python, go-imap v2 is natively async and non-blocking:

```go
type IMAPConn struct {
    client *imapclient.Client
}

func (c *IMAPConn) Connect(host string, port int, tlsConfig *tls.Config, creds Credentials) error
func (c *IMAPConn) ListFolders() ([]string, error)
func (c *IMAPConn) FetchUIDsSince(folder string, sinceUID uint32) ([]uint32, error)
func (c *IMAPConn) FetchMessages(uids []uint32) ([]RawMessage, error)
func (c *IMAPConn) FetchHeadersOnly(uids []uint32) ([]RawMessage, error)
func (c *IMAPConn) MarkRead(uids []uint32) error
func (c *IMAPConn) MarkUnread(uids []uint32) error
func (c *IMAPConn) Move(uids []uint32, dest string) error
func (c *IMAPConn) Delete(uids []uint32) error
func (c *IMAPConn) StartIdle() (*imapclient.IdleCommand, error)
func (c *IMAPConn) Close() error
```

go-imap v2 has a built-in `UnilateralDataHandler` for push notifications from IMAP IDLE — no manual polling logic needed:

```go
options := &imapclient.Options{
    UnilateralDataHandler: &imapclient.UnilateralDataHandler{
        Mailbox: func(data *imapclient.UnilateralDataMailbox) {
            if data.NumMessages != nil {
                // new message arrived — send a tea.Msg to the UI
                p.Send(NewMailArrivedMsg{})
            }
        },
    },
}
```

### `engine/sync.go`

`SyncWorker` runs as a `tea.Cmd`. On startup:
1. Initial sync: fetch UIDs + last 200 messages per account, parse, cache
2. Send `SyncDoneMsg` to the Bubble Tea program
3. Start IMAP IDLE loop per connection for push-like updates
4. Fall back to polling every `check_interval_seconds` if IDLE is unsupported

After each sync, `SyncWorker` calls `MetricsTracker.CheckMilestones()` and fires `MilestoneUnlockedMsg` for any newly unlocked milestones.

### `engine/thread.go`

JWZ threading algorithm:
- Hash map of `message-id` → message
- Link via `In-Reply-To` and `References` headers
- Walk tree to assign thread IDs
- Subject-based fallback (`Re:` / `Fwd:` stripping) when headers are missing

---

## Provider adapters — `internal/providers/`

```go
type Provider interface {
    IMAPHost() string
    IMAPPort() int
    IMAPTLS() bool
    SMTPHost() string
    SMTPPort() int
    SMTPTLS() bool
    GetCredentials(accountID string) (Credentials, error)
    RunAuthFlow(accountID string) (Credentials, error)
    RefreshToken(accountID string) (Credentials, error)
}
```

### Gmail (`providers/gmail.go`)
- `golang.org/x/oauth2` with PKCE
- Scope: `https://mail.google.com/`
- Auth: open `http://localhost:8765` in the system browser, catch redirect with `net/http` listener
- Refresh token stored in OS keyring
- IMAP: `imap.gmail.com:993 TLS`
- SMTP: `smtp.gmail.com:587 STARTTLS`

### Outlook (`providers/outlook.go`)
- MSAL `PublicClientApplication` device code flow (safer for CLI — no browser redirect required)
- Scopes: `https://outlook.office.com/IMAP.AccessAsUser.All`, `SMTP.Send`
- IMAP: `outlook.office365.com:993 TLS`
- SMTP: `smtp.office365.com:587 STARTTLS`

### Generic (`providers/generic.go`)
- Plain username + app password
- User provides host, port, TLS bool
- Password stored in OS keyring

---

## TUI layout — `internal/tui/`

### Three-pane main layout

```
┌──────────┬────────────────────────┬──────────────────────────────┐
│ Inboxes  │  Thread list           │  Message view                │
│          │                        │                              │
│ ● Primary│  ● Alice Re: Q3 deck   │  From: alice@company.com     │
│   Notifs │    2h ago · 3 msgs     │  To: you@company.com         │
│   Newsltr│  ○ Bob Proposal        │  Subject: Re: Q3 deck        │
│          │    Mon · 1 msg         │                              │
│          │  ● Carol Hey           │  Hey! Just following up...   │
│          │    Tue · 2 msgs        │                              │
├──────────┴────────────────────────┴──────────────────────────────┤
│ [work] Primary  ·  12 unread  ·  Synced 30s ago  cleared: 42 7≈ │
└──────────────────────────────────────────────────────────────────┘
```

Layout is composed using `lipgloss.JoinHorizontal` and `lipgloss.JoinVertical` in the `View()` method — no layout engine needed beyond Lip Gloss:

```go
func (m mainModel) View() string {
    left  := m.inboxList.View()
    mid   := m.threadList.View()
    right := m.messageView.View()

    content := lipgloss.JoinHorizontal(lipgloss.Top, left, mid, right)
    return lipgloss.JoinVertical(lipgloss.Left,
        content,
        m.statusBar.View(),
        m.commandBar.View(),
    )
}
```

### `components/thread_list/`

Uses `bubbles/list` as its base — a production-ready scrollable list with:
- Custom item renderer (unread indicator, sender, subject, preview, timestamp)
- `j/k` navigation built in with spring-eased cursor motion
- Filterable (powers `/search`)
- Fires `InboxZeroMsg` when the visible unread count reaches 0

### `components/message_view/`

Uses `bubbles/viewport` — a scrollable region with configurable height, mouse wheel support, and PgUp/PgDn. Message body is rendered through `glamour` for HTML and Markdown content.

### `components/command_bar/`

Triggered by `:` or `/`. Uses `bubbles/textinput` for input, with fuzzy autocomplete rendered below as a small list of matching commands.

---

## Command system — `internal/tui/commands/`

```go
type Command struct {
    Name        string
    Description string
    Handler     func(args string, app *app.App) tea.Cmd
}

type Registry struct {
    commands map[string]Command
}

func (r *Registry) Register(name, description string, handler func(string, *app.App) tea.Cmd)
func (r *Registry) Dispatch(input string, app *app.App) tea.Cmd
func (r *Registry) Completions(prefix string) []Command
```

### Full command reference

| Command | Description |
|---|---|
| `/connect` | Add a new inbox: provider → auth flow → IMAP/SMTP test → config write → initial sync |
| `/inbox [name]` | Switch the active split inbox |
| `/search [query]` | Full-text search across cached messages (FTS5), remote IMAP fallback |
| `/snooze` | Quick-pick: later today / tomorrow morning / weekend / next week / custom |
| `/shortcuts` | Display keybinding cheatsheet overlay |
| `/sessions` | List, restore, or delete named UI sessions |
| `/theme [name]` | Switch theme live; `list` to browse; `edit` to open in `$EDITOR`; `validate` to lint |
| `/metrics` | Open metrics dashboard; `export json\|csv` to write files |
| `/daemon install` | Install headless sync daemon as launchd (macOS) or systemd user service (Linux) |
| `/daemon status` | Show whether the daemon is running and last sync time |
| `/daemon stop` | Stop the background daemon |

---

## Split inbox system

Rules are evaluated at insert time in `engine/parser.go` when messages arrive from sync. Each message's thread is stamped with a `split_inbox_id`.

```go
type Rule struct {
    Field       string  // "from" | "to" | "subject" | "list_unsubscribe" | "header"
    Contains    string
    NotContains string
    Exists      *bool
    HeaderName  string  // used when Field = "header"
}

func EvaluateSplitInboxRules(msg *Message, inboxes []SplitInbox) string {
    for _, inbox := range inboxes {
        if allRulesMatch(msg, inbox.Rules) {
            return inbox.ID
        }
    }
    return "primary"
}
```

---

## Theme system — `internal/themes/`

### Theme struct

```go
// internal/themes/theme.go
type Theme struct {
    Name string

    // Backgrounds
    Background       string
    Surface          string
    SurfaceAlt       string
    SurfaceHighlight string

    // Accents
    Primary   string
    Secondary string
    Accent    string

    // Text
    Text      string
    TextMuted string
    TextDim   string

    // Unread state
    UnreadIndicator string
    UnreadSubject   string
    ReadSubject     string
    ReadPreview     string

    // Chrome
    Border        string
    BorderFocus   string
    Selection     string
    SelectionText string
    StatusBarBg   string
    StatusBarText string
    CommandBarBg  string
    CommandBorder string

    // Semantic
    Success string
    Warning string
    Danger  string
    Info    string

    // Forest scene (NEW)
    Forest ForestPalette
}

// ForestPalette holds the colors and characters for the inbox zero forest scene.
type ForestPalette struct {
    TrunkChars   []string
    BranchChars  []string
    LeafChars    []string
    GroundChars  []string
    TrunkColor   string
    BranchColor  string
    LeafColor1   string
    LeafColor2   string
    LeafColor3   string
    GroundColor  string
    FireflyColor string
}
```

### Theme TOML format

```toml
# internal/themes/builtin/tokyo-night.toml
name = "tokyo-night"

background        = "#1a1b26"
surface           = "#16161e"
surface_alt       = "#1f2335"
surface_highlight = "#292e42"

primary   = "#7aa2f7"
secondary = "#bb9af7"
accent    = "#73daca"

text       = "#c0caf5"
text_muted = "#565f89"
text_dim   = "#3b4261"

unread_indicator = "#7aa2f7"
unread_subject   = "#c0caf5"
read_subject     = "#565f89"
read_preview     = "#3b4261"

border        = "#292e42"
border_focus  = "#7aa2f7"
selection     = "#283457"
selection_text = "#c0caf5"
status_bar_bg   = "#16161e"
status_bar_text = "#565f89"
command_bar_bg  = "#16161e"
command_border  = "#7aa2f7"

success = "#9ece6a"
warning = "#e0af68"
danger  = "#f7768e"
info    = "#7dcfff"

[forest]
trunk_chars   = ["│", "║", "╷"]
branch_chars  = ["─", "╱", "╲", "┐", "└", "┘", "┌"]
leaf_chars    = ["&", "*", "@", "✦", "✿"]
ground_chars  = ["▁", "▂", "▃"]
trunk_color   = "#6a4f3b"
branch_color  = "#5a4030"
leaf_color_1  = "#9ece6a"
leaf_color_2  = "#73daca"
leaf_color_3  = "#e0af68"
ground_color  = "#3b4261"
firefly_color = "#e0af68"
```

### `themes/styles.go`

Converts a `Theme` into a `Styles` struct of ready-to-use `lipgloss.Style` values:

```go
type Styles struct {
    ThreadUnread     lipgloss.Style
    ThreadRead       lipgloss.Style
    ThreadSelected   lipgloss.Style
    MessageHeader    lipgloss.Style
    MessageBody      lipgloss.Style
    InboxLabel       lipgloss.Style
    InboxBadge       lipgloss.Style
    CommandBarInput  lipgloss.Style
    CommandBarMatch  lipgloss.Style
    StatusBar        lipgloss.Style
    Border           lipgloss.Style
    BorderFocused    lipgloss.Style
    MilestoneToast   lipgloss.Style  // NEW
    MetricsBar       lipgloss.Style  // NEW
}

func BuildStyles(t *Theme) Styles {
    return Styles{
        ThreadUnread: lipgloss.NewStyle().
            Foreground(lipgloss.Color(t.UnreadSubject)).
            Bold(true),
        ThreadRead: lipgloss.NewStyle().
            Foreground(lipgloss.Color(t.ReadSubject)),
        // ... all other styles
    }
}
```

### `themes/manager.go`

```go
type Manager struct {
    current  *Theme
    styles   Styles
    builtins map[string]*Theme  // loaded from internal/themes/builtin/*.toml
    user     map[string]*Theme  // loaded from ~/.termite/themes/*.toml
}

func (m *Manager) Discover() []ThemeInfo
func (m *Manager) Load(nameOrPath string) (*Theme, error)
func (m *Manager) Validate(t *Theme) []ValidationError  // check all required fields are non-empty; warn if [forest] section missing
func (m *Manager) Apply(nameOrPath string) tea.Cmd      // hot-swap: reload Styles, send StylesUpdatedMsg
func (m *Manager) Current() *Theme
```

Hot-swap sends a `StylesUpdatedMsg` to the Bubble Tea program, which propagates new `Styles` to all components on the next update cycle. No restart required.

### Built-in themes

| Name | Description |
|---|---|
| `dark` | Default — dark navy, blue accents |
| `light` | Clean white, blue accents |
| `dracula` | Purple/pink Dracula palette |
| `tokyo-night` | Muted blues and purples |
| `catppuccin-mocha` | Warm dark, pastel accents |
| `catppuccin-latte` | Catppuccin light variant |
| `gruvbox` | Warm retro browns and greens |
| `nord` | Arctic blue-grey |
| `solarized-dark` | Original solarized dark |
| `high-contrast` | Accessibility-first, WCAG AA |
| `matrix` | Green on black — forest renders in all-green characters |

User themes live in `~/.termite/themes/*.toml`. Community themes are shared via a `termite-themes` GitHub repo — drop any `.toml` file in and Termite discovers it automatically. Theme contributors are encouraged to add a `[forest]` section; themes without one fall back to the `dark` theme's forest palette.

---

## Notifications — `internal/notifications/`

Four layers, all user-configurable via `config.toml`:

### Desktop (`notifications/desktop.go`)
Uses `gen2brain/beeep` which wraps:
- macOS: `osascript` notification
- Linux: `libnotify` / `notify-send`
- Windows: toast via PowerShell

Fired by the background daemon when new mail arrives. Shows sender + subject.

### Terminal bell (`notifications/manager.go`)
Writes `\a` (BEL) to stdout on new mail. Zero dependencies. Opt-in only.

### tmux (`notifications/tmux.go`)
Detects `$TMUX` environment variable. Sets the window title via escape sequence:
```go
fmt.Printf("\033]2;termite [%d new]\033\\", unreadCount)
```

### Status file (`notifications/status.go`)
Writes `~/.termite/status.json` on every sync:
```json
{
  "unread": 12,
  "last_sync": "2025-10-14T09:32:00Z",
  "accounts": { "work": 8, "personal": 4 },
  "metrics": {
    "cleared_today": 42,
    "sent_today": 8,
    "streak_days": 7
  }
}
```

### Background daemon (`internal/daemon/daemon.go`)

```go
func Run(cfg *config.Config) error {
    db, err := db.Open()
    notifier := notifications.NewManager(cfg.Notifications)

    for {
        for _, account := range cfg.Accounts {
            newMsgs, err := engine.IncrementalSync(account, db)
            if err != nil {
                slog.Error("sync failed", "account", account.ID, "err", err)
                continue
            }
            if len(newMsgs) > 0 {
                notifier.Notify(newMsgs)
                notifications.WriteStatusFile(db)
            }
        }
        time.Sleep(time.Duration(cfg.General.CheckIntervalSeconds) * time.Second)
    }
}
```

`termite install-daemon` writes:
- macOS: `~/Library/LaunchAgents/land.charm.termite.plist`
- Linux: `~/.config/systemd/user/termite.service`

No manual plist or unit editing required.

---

## Compose flow

`composePage` is a full-screen overlay pushed onto the page stack:
- `to`, `cc`, `bcc`, `subject` as `bubbles/textinput` widgets with tab-switching
- Body as `bubbles/textarea`
- **Undo send**: on send, enqueue the email with a 5-second countdown. Show a dismissable status bar notice "Sent — press `u` to undo (5s)". Cancel the send `tea.Cmd` if `u` is pressed.
- **Snippets**: typing `/snippet {name}` in the body expands a text block from config
- On successful send, `MetricsTracker.RecordSent()` is called and milestones are checked

---

## Search

Two-tier:

1. **Local FTS5**: `SELECT ... FROM messages_fts WHERE messages_fts MATCH ?` with BM25 ranking. Zero network round-trips. Results populate as the user types.
2. **Remote IMAP**: Fallback to `IMAP SEARCH` when the user explicitly requests "search all mail."

Search overlay slides up from the bottom — the thread list and message view stay visible for context.

---

## First-run experience

On launch with no `~/.termite/config.toml`:
1. Push `setupPage` — a friendly step-by-step wizard with spring transitions between steps
2. Step 1: select provider
3. Step 2: run auth flow (spinner: `SpinnerAuth` during OAuth)
4. Step 3: initial sync with `bubbles/spinner` (`SpinnerSync`) progress indicator
5. Drop into `mainPage` with inbox populated
6. If the newly-synced inbox is already empty (rare but possible), immediately trigger the forest scene

---

## Keybinding system

```go
type KeyMap struct {
    Compose   key.Binding
    Reply     key.Binding
    ReplyAll  key.Binding
    Forward   key.Binding
    Archive   key.Binding
    Delete    key.Binding
    MarkRead  key.Binding
    MarkUnread key.Binding
    Snooze    key.Binding
    Next      key.Binding
    Prev      key.Binding
    Open      key.Binding
    Zero      key.Binding
    Search    key.Binding
    Command   key.Binding
    Quit      key.Binding
}

func BuildKeyMap(cfg *config.Config) KeyMap {
    return KeyMap{
        Compose: key.NewBinding(
            key.WithKeys(cfg.Keybindings.Compose),
            key.WithHelp(cfg.Keybindings.Compose, "compose"),
        ),
        // ...
    }
}
```

All keybindings are user-remappable via `config.toml`. Defaults are Superhuman-inspired and vim-flavored.

---

## Packaging and distribution

```go
// go.mod (key dependencies)
require (
    charm.land/bubbletea/v2        latest
    charm.land/lipgloss/v2         latest
    charm.land/bubbles/v2          latest
    github.com/charmbracelet/harmonica latest  // NEW: spring physics
    github.com/emersion/go-imap/v2 latest
    github.com/emersion/go-smtp    latest
    github.com/emersion/go-message latest
    modernc.org/sqlite             latest
    github.com/jmoiron/sqlx        latest
    github.com/sqlc-dev/sqlc       latest  // dev tool, not runtime dep
    github.com/zalando/go-keyring  latest
    golang.org/x/oauth2            latest
    github.com/AzureAD/microsoft-authentication-library-for-go latest
    github.com/BurntSushi/toml     latest
    github.com/gen2brain/beeep     latest
    github.com/charmbracelet/glamour latest
    github.com/spf13/cobra         latest
    github.com/go-playground/validator/v10 latest
)
```

`modernc.org/sqlite` is used instead of `mattn/go-sqlite3` because it's **CGO-free** — this means `go build` works without a C compiler and cross-compilation just works.

### Distribution via goreleaser

```yaml
# .goreleaser.yml
builds:
  - id: termite
    binary: termite
    env: [CGO_ENABLED=0]
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]

archives:
  - format: tar.gz
    name_template: "termite_{{ .Version }}_{{ .Os }}_{{ .Arch }}"

brews:
  - tap:
      owner: termite-mail
      name: homebrew-tap
    homepage: https://github.com/termite-mail/termite
    description: "A Superhuman-inspired TUI email client"
```

Distribution targets:
- **Homebrew tap** — `brew install termite-mail/tap/termite`
- **GitHub releases** — pre-built binaries for macOS (arm64/amd64), Linux (arm64/amd64), Windows
- **go install** — `go install github.com/termite-mail/termite@latest`
- **Scoop** (Windows) — `scoop install termite`

---

## Testing strategy

- Unit tests: engine, DB queries, threading algorithm, split inbox rules, metrics tracker, milestone unlock logic
- Integration tests: IMAP sync against Dovecot in Docker (CI)
- TUI tests: `github.com/charmbracelet/x/teatest` — snapshot tests for rendered views, `Pilot` for simulating keystrokes
- Animation tests: snapshot each forest scene phase at a fixed seed; assert character grid matches golden file
- Theme tests: `ThemeManager.Validate()` run against every bundled `.toml` file; warn on missing `[forest]` section

```go
// Example TUI test using teatest
func TestThreadListRender(t *testing.T) {
    m := threadlist.New(sampleThreads, defaultStyles)
    tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 40))
    tm.Send(tea.KeyMsg{Type: tea.KeyDown})
    teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
        return bytes.Contains(bts, []byte("Alice"))
    }, teatest.WithDuration(time.Second))
}

// Example forest scene snapshot test
func TestForestSceneGrowth(t *testing.T) {
    rng := rand.New(rand.NewSource(42))   // deterministic seed
    scene := animation.NewForestSceneWithRNG(80, 24, themes.Dark(), rng)
    for i := 0; i < 120; i++ { scene.GrowStep() }  // advance 120 ticks
    got := scene.View()
    golden.Assert(t, got, "testdata/forest_grown.golden")
}
```

---

## Phase 1 MVP scope

Build in this order:

1. `internal/config/` — TOML loading, defaults, validation
2. `internal/db/` — SQLite setup, FTS5 migrations, sqlc queries, `003_metrics.sql`
3. `internal/providers/gmail.go` — OAuth2 + IMAP/SMTP settings
4. `internal/engine/imap.go` + `sync.go` — initial sync (no IDLE yet)
5. `internal/engine/thread.go` — JWZ threading
6. `internal/themes/` — Theme struct (including `ForestPalette`), TOML loading, `dark.toml` + `light.toml` with `[forest]` sections, `BuildStyles()`
7. `internal/tui/` — `appModel` root, `mainPage` three-pane layout
8. `components/inbox_list/`, `thread_list/` (with `InboxZeroMsg`), `message_view/`
9. Core keybindings: `j/k`, `e` archive, `r` reply, `c` compose, `/` search, `:` command
10. `commands/connect.go` Gmail wizard
11. Local FTS5 search
12. `internal/notifications/status.json` writer (with metrics fields)
13. `internal/metrics/` — `tracker.go`, `milestones.go` (basic tracking, no dashboard yet)
14. `internal/animation/spring.go` + `transitions.go` — spring physics and basic transitions
15. `internal/animation/forest.go` + `particles.go` + `quotes.go` — forest scene
16. `tui/pages/inbox_zero/` — forest scene page integration
17. `components/milestone_toast/` — achievement toast
18. `cmd/` entry point with `cobra`

### Deferred to Phase 2
IMAP IDLE, Outlook + Fastmail providers, `/snooze`, `/sessions`, undo send, snippets, attachments, full desktop/tmux notification suite, daemon install command, remaining 9 themes, `/metrics` dashboard page, metrics export, community themes repo, Scoop package, Windows testing.

---

## Open source structure

- **License**: MIT
- **Module path**: `github.com/termite-mail/termite`
- **PyPI equivalent**: `go install github.com/termite-mail/termite@latest`
- **Community themes repo**: `termite-mail/termite-themes` — one `.toml` file per theme (must include `[forest]` section to be accepted), PR to contribute
- **Provider contributions**: `CONTRIBUTING.md` includes a step-by-step guide to implementing `Provider` interface
- **Versioning**: CalVer (`YYYY.MM.patch`) to make release recency obvious to users
- **`AGENTS.md`**: instructions for coding agents — read this before touching the TUI (mirrors the Crush pattern)
