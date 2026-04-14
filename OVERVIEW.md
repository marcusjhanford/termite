# Termite — Technical Specification v1.0
> A Superhuman-inspired, keyboard-first TUI email client. Open source. Runs locally. Your data stays yours.

---

## Why Python, not TypeScript

TypeScript's TUI options (`ink`, `blessed`) hit a ceiling fast. Complex layouts, scrollable panes, mouse support, and CSS-like styling all require fighting the framework rather than using it. Textual (Python) is the most serious TUI framework that exists today — a real CSS engine, a reactive component model, built-in production-grade widgets, mouse support, async-first architecture, and an active team shipping weekly. For a three-pane email client with overlays, themeable CSS, and live hot-reload, Textual has native answers for all of it.

The one argument for TypeScript is sharing logic with a future web/Electron version. For a terminal-native open-source tool, that tradeoff isn't worth it. **Python + Textual is the right call.**

---

## Tech stack

| Concern | Library | Notes |
|---|---|---|
| TUI framework | `textual>=0.82` | Reactive, CSS-driven, async-first |
| IMAP client | `imapclient>=3.0` | Higher-level than stdlib `imaplib` |
| SMTP | `aiosmtplib>=3.0` | Async SMTP |
| Email parsing | `mail-parser` + stdlib `email` | MIME, attachments, HTML→text |
| Local DB | `aiosqlite>=0.20` | SQLite with FTS5 for full-text search |
| Credential storage | `keyring>=25` | OS keyring abstraction (Keychain / libsecret / Win Credential) |
| OAuth2 | `google-auth-oauthlib`, `msal` | Gmail and Outlook flows |
| Config | `tomllib` (stdlib 3.11+) + `tomli-w` | Read/write TOML |
| HTML rendering | `html2text>=2024` | Render HTML email bodies in terminal |
| Desktop notifications | `desktop-notifier>=3.5` | macOS / Linux / Windows native notifications |
| CLI entry | `click>=8` | Entry point + flags |
| Package manager | `uv` | Modern, fast dependency management |
| Python version | `3.12+` | Required for `tomllib`, `asyncio` improvements |

---

## Repository structure

```
termite/
├── pyproject.toml
├── README.md
├── CONTRIBUTING.md
├── .github/
│   ├── ISSUE_TEMPLATE/
│   │   ├── bug_report.md
│   │   ├── feature_request.md
│   │   └── new_provider.md
│   └── workflows/
│       ├── ci.yml          # pytest + ruff + mypy on every PR
│       └── release.yml     # publish to PyPI on version tag
├── termite/
│   ├── __init__.py
│   ├── __main__.py         # python -m termite entry
│   ├── cli.py              # click entry point: `termite` command
│   ├── app.py              # Textual App root
│   ├── daemon.py           # headless sync loop for background service
│   ├── config/
│   │   ├── __init__.py
│   │   ├── schema.py       # Pydantic models for config validation
│   │   ├── loader.py       # load/save ~/.termite/config.toml
│   │   └── defaults.py     # default keybindings, theme, split inbox rules
│   ├── engine/
│   │   ├── __init__.py
│   │   ├── account.py      # Account dataclass + manager
│   │   ├── imap.py         # IMAPClient wrapper (async via run_in_executor)
│   │   ├── smtp.py         # aiosmtplib wrapper
│   │   ├── sync.py         # background sync loop (poll + IDLE)
│   │   ├── thread.py       # email threading algorithm (JWZ)
│   │   └── parser.py       # MIME parsing, HTML→text, attachment extraction
│   ├── providers/
│   │   ├── __init__.py
│   │   ├── base.py         # BaseProvider protocol
│   │   ├── gmail.py        # Gmail OAuth2 flow + IMAP/SMTP settings
│   │   ├── outlook.py      # Microsoft MSAL flow + IMAP/SMTP settings
│   │   ├── fastmail.py     # Fastmail app password flow
│   │   └── generic.py      # Plain IMAP/SMTP with app password
│   ├── cache/
│   │   ├── __init__.py
│   │   ├── db.py           # aiosqlite connection pool + migrations
│   │   ├── schema.sql      # DDL: messages, threads, accounts, labels, fts
│   │   └── queries.py      # typed query functions (no raw SQL in app code)
│   ├── commands/
│   │   ├── __init__.py
│   │   ├── registry.py     # CommandRegistry: registers + dispatches /commands
│   │   ├── connect.py      # /connect — add a new inbox
│   │   ├── inbox.py        # /inbox — switch active inbox / split inbox
│   │   ├── search.py       # /search — full-text search
│   │   ├── snooze.py       # /snooze — snooze a thread
│   │   ├── shortcuts.py    # /shortcuts — show keybinding cheatsheet
│   │   ├── sessions.py     # /sessions — manage named sessions
│   │   ├── theme.py        # /theme — switch, list, edit themes
│   │   └── daemon.py       # /daemon install|status|stop
│   ├── notifications/
│   │   ├── __init__.py
│   │   ├── manager.py      # NotificationManager: routes to correct backend
│   │   ├── desktop.py      # desktop-notifier wrapper
│   │   ├── tmux.py         # tmux title/flag integration
│   │   └── status.py       # ~/.termite/status.json writer
│   ├── themes/
│   │   ├── dark.tcss
│   │   ├── light.tcss
│   │   ├── dracula.tcss
│   │   ├── tokyo-night.tcss
│   │   ├── catppuccin-mocha.tcss
│   │   ├── catppuccin-latte.tcss
│   │   ├── gruvbox.tcss
│   │   ├── nord.tcss
│   │   ├── solarized-dark.tcss
│   │   ├── high-contrast.tcss
│   │   └── matrix.tcss
│   └── ui/
│       ├── __init__.py
│       ├── screens/
│       │   ├── main.py         # MainScreen: three-pane layout
│       │   ├── compose.py      # ComposeScreen: full-screen compose
│       │   ├── search.py       # SearchScreen: inline search overlay
│       │   └── setup.py        # SetupScreen: first-run wizard
│       ├── widgets/
│       │   ├── inbox_list.py       # InboxList: left-pane scrollable inbox
│       │   ├── thread_pane.py      # ThreadPane: middle-pane thread list
│       │   ├── message_view.py     # MessageView: right-pane full message
│       │   ├── command_bar.py      # CommandBar: / trigger, fuzzy autocomplete
│       │   ├── status_bar.py       # StatusBar: sync status, account, keyhints
│       │   └── compose_editor.py   # ComposeEditor: textarea + to/cc/subject
│       └── theme_manager.py        # ThemeManager: discover, validate, hot-swap
└── tests/
    ├── conftest.py
    ├── engine/
    ├── cache/
    ├── commands/
    ├── notifications/
    └── ui/
```

---

## Configuration — `~/.termite/config.toml`

```toml
[general]
theme = "tokyo-night"     # built-in name OR path to ~/.termite/themes/custom.tcss
editor = "vim"            # for compose
check_interval_seconds = 60
startup_inbox = "primary"

[notifications]
desktop = true            # OS-level notifications via desktop-notifier
terminal_bell = false     # \a BEL character on new mail
tmux_title = true         # set tmux window title with unread count
status_file = true        # write ~/.termite/status.json for external tools
notify_on = "unread"      # "unread" | "all" | "none"

[keybindings]
compose      = "c"
reply        = "r"
reply_all    = "shift+r"
forward      = "f"
archive      = "e"
delete       = "#"
mark_read    = "m"
mark_unread  = "shift+m"
snooze       = "h"
next_thread  = "j"
prev_thread  = "k"
open_thread  = "enter"
inbox_zero   = "shift+i"
search       = "/"
command      = ":"
quit         = "q"

[[accounts]]
id    = "work"
name  = "Work"
email = "you@company.com"
provider = "gmail"        # gmail | outlook | fastmail | generic

[[accounts]]
id    = "personal"
name  = "Personal"
email = "you@gmail.com"
provider = "gmail"

[[split_inboxes]]
id    = "primary"
label = "Primary"
accounts = ["work", "personal"]
rules = [
  { field = "from", not_contains = ["newsletter", "noreply", "notifications"] }
]

[[split_inboxes]]
id    = "newsletters"
label = "Newsletters"
accounts = ["personal"]
rules = [
  { field = "list_unsubscribe", exists = true }
]

[[split_inboxes]]
id    = "notifications"
label = "Notifs"
accounts = ["work", "personal"]
rules = [
  { field = "from", contains = ["noreply", "no-reply", "notifications"] }
]
```

Credentials (tokens, passwords) are **never stored in config.toml**. They are stored via `keyring` under the key `termite:{account_id}`.

---

## Database schema — `~/.termite/cache.db`

```sql
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
```

All queries live in `cache/queries.py` as typed async functions — no raw SQL scattered through app code.

---

## Core engine — `engine/`

### `engine/imap.py`

`imapclient` is synchronous. Wrap all IMAP calls in `asyncio.get_event_loop().run_in_executor(None, ...)` to avoid blocking the Textual event loop.

Key methods:

```python
class IMAPConnection:
    async def connect(self, host, port, ssl, credentials) -> None
    async def list_folders(self) -> list[str]
    async def fetch_uids_since(self, folder, since_uid) -> list[int]
    async def fetch_messages(self, uids: list[int]) -> list[RawMessage]
    async def fetch_headers_only(self, uids: list[int]) -> list[RawMessage]
    async def mark_read(self, uids: list[int]) -> None
    async def mark_unread(self, uids: list[int]) -> None
    async def move(self, uids: list[int], destination: str) -> None
    async def delete(self, uids: list[int]) -> None
    async def idle_start(self) -> None   # IMAP IDLE for push-like updates
    async def idle_done(self) -> None
    async def close(self) -> None
```

### `engine/sync.py`

Background sync loop runs as a Textual `Worker`. On startup:
1. Fetch all unread UIDs + last 200 messages per account
2. Parse and cache to SQLite
3. Post a `SyncComplete` message to the Textual app

On interval (default 60s, configurable):
1. Fetch UIDs greater than last known UID
2. Delta-sync only new messages
3. If server supports IMAP IDLE, use it instead of polling

```python
class SyncWorker:
    async def initial_sync(self, account: Account) -> None
    async def incremental_sync(self, account: Account) -> None
    async def idle_loop(self, account: Account) -> None
```

### `engine/thread.py`

Implement the **JWZ threading algorithm** (https://www.jwz.org/doc/threading.html):
- Build a hash map of message-id → message
- Link messages by `In-Reply-To` and `References` headers
- Walk the tree to assign thread IDs
- Fall back to subject-based grouping (`Re:` stripping) when headers are missing

---

## Provider adapters — `providers/`

Each provider implements `BaseProvider`:

```python
class BaseProvider(Protocol):
    imap_host: str
    imap_port: int
    imap_ssl: bool
    smtp_host: str
    smtp_port: int
    smtp_ssl: bool

    async def get_credentials(self, account_id: str) -> Credentials
    async def run_auth_flow(self, account_id: str) -> Credentials
    async def refresh_token(self, account_id: str) -> Credentials
```

### Gmail (`providers/gmail.py`)
- OAuth2 via `google-auth-oauthlib`
- Scopes: `https://mail.google.com/`
- Auth flow: open `http://localhost:8765` redirect in the system browser, catch the callback with a tiny `http.server` listener
- Store refresh token in OS keyring
- IMAP: `imap.gmail.com:993 SSL`
- SMTP: `smtp.gmail.com:587 STARTTLS`

### Outlook (`providers/outlook.py`)
- MSAL `PublicClientApplication` device code flow
- Scopes: `https://outlook.office.com/IMAP.AccessAsUser.All`, `https://outlook.office.com/SMTP.Send`
- IMAP: `outlook.office365.com:993 SSL`
- SMTP: `smtp.office365.com:587 STARTTLS`

### Generic (`providers/generic.py`)
- Plain username + app password
- User provides host, port, SSL boolean
- Password stored in OS keyring

---

## UI layout — Textual

### `ui/screens/main.py` — three-pane layout

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
│ [work] Primary  ·  12 unread  ·  Synced 30s ago      [/ search] │
└──────────────────────────────────────────────────────────────────┘
```

```python
class MainScreen(Screen):
    def compose(self) -> ComposeResult:
        yield Header()
        with Horizontal():
            yield InboxList(id="inbox-list")
            yield ThreadPane(id="thread-pane")
            yield MessageView(id="message-view")
        yield StatusBar()
        yield CommandBar()
```

Key Textual patterns:
- Use `reactive()` fields for selected thread, active inbox, sync status
- Use `@on(ListView.Selected)` for thread selection events
- Use `self.app.push_screen()` for compose and search overlays
- Use `self.run_worker()` for all async sync operations
- Use `self.notify()` for transient toasts ("Archived", "Snoozed for 3h")

### `ui/widgets/command_bar.py`

Triggered by `:` (vim-style) or `/` for search. Renders as a bottom overlay with fuzzy-matching autocomplete of all registered commands.

```python
class CommandBar(Widget):
    def on_key(self, event: Key) -> None:
        if event.key == "colon":
            self.show()
        elif event.key == "slash":
            self.show(prefix="/search ")

    def show(self, prefix: str = "") -> None:
        self.display = True
        self.input.value = prefix
        self.input.focus()
```

---

## Command system — `commands/`

### `commands/registry.py`

```python
@dataclass
class Command:
    name: str
    description: str
    handler: Callable

class CommandRegistry:
    _commands: dict[str, Command] = {}

    def register(self, name, description):
        def decorator(fn):
            self._commands[name] = Command(name, description, fn)
            return fn
        return decorator

    async def dispatch(self, raw_input: str, app: App) -> None:
        parts = raw_input.strip().lstrip("/").split(maxsplit=1)
        name = parts[0]
        args = parts[1] if len(parts) > 1 else ""
        if name in self._commands:
            await self._commands[name].handler(args, app)
```

### Full command reference

| Command | Description |
|---|---|
| `/connect` | Interactive wizard to add a new inbox (provider → auth → test → sync) |
| `/inbox [name]` | Switch the active split inbox |
| `/search [query]` | Full-text search across all cached messages |
| `/snooze` | Snooze the selected thread (quick-pick: today / tomorrow / weekend / custom) |
| `/shortcuts` | Show full keybinding cheatsheet overlay |
| `/sessions` | List, restore, or delete named UI sessions |
| `/theme [name]` | Switch theme live; `:theme list` to browse; `:theme edit` to open in `$EDITOR` |
| `/daemon install` | Install the background sync daemon (launchd on macOS, systemd on Linux) |
| `/daemon status` | Show whether the daemon is running and last sync time |
| `/daemon stop` | Stop the background daemon |

### `/connect` flow
1. Prompt for provider type (Gmail / Outlook / Fastmail / Generic IMAP)
2. Prompt for display name and email address
3. Run the appropriate OAuth2 or app-password flow
4. Test IMAP and SMTP connectivity
5. Write the account block to `config.toml`
6. Trigger an initial sync

### `/sessions` — named sessions
Saves and restores the current UI state: active split inbox, selected thread, search query, scroll position. Stored as JSON in `~/.termite/sessions.json`. Inspired by opencode's `/sessions` pattern.

### `/snooze`
Quick-pick options: "Later today", "Tomorrow morning", "This weekend", "Next week", "Custom...". Sets `snoozed_until` on the thread. A `SnoozeChecker` async timer fires every 5 minutes to un-snooze threads that have come due.

---

## Split inbox system

Split inboxes are filter rules evaluated at cache-write time, not query time. When a message is inserted into SQLite, `evaluate_split_inbox_rules(message)` runs all configured rule sets in order and stamps the thread with a `split_inbox_id`.

Rule schema (AND within a rule, OR across multiple rules):

```python
@dataclass
class Rule:
    field: Literal["from", "to", "subject", "list_unsubscribe", "header"]
    contains: str | None = None
    not_contains: str | None = None
    exists: bool | None = None
    header_name: str | None = None
```

---

## Notifications — `notifications/`

Four notification layers, all user-configurable:

### Desktop notifications (`notifications/desktop.py`)
Uses `desktop-notifier` to fire native OS notifications (macOS Notification Center, Linux libnotify/D-Bus, Windows toast) from the background daemon when new mail arrives. Shows sender + subject. On macOS, clicking the notification focuses the terminal window via `NSUserNotificationCenter` callback.

### Terminal bell (`notifications/manager.py`)
Emits `\a` (BEL) on new mail. Zero dependencies. Respects the terminal's visual bell setting. Opt-in only.

### tmux integration (`notifications/tmux.py`)
If Termite detects it's running inside tmux, sets the window title and flags the pane:
```python
printf('\033]2;termite [3 new]\033\\')
```
Makes the tmux status bar show unread count without any plugins.

### Status file (`notifications/status.py`)
Writes `~/.termite/status.json` on every sync:
```json
{ "unread": 12, "last_sync": "2025-10-14T09:32:00Z", "accounts": { "work": 8, "personal": 4 } }
```
Any external tool — Starship prompt, waybar, i3status, sketchybar, tmux plugins — can read this file and surface unread counts wherever the user wants. This is the most composable approach for power users.

### Background daemon (`daemon.py`)
```python
async def run_daemon():
    """Headless sync loop. Runs as a background service via launchd/systemd."""
    config = load_config()
    cache = await open_cache()
    notifier = NotificationManager(config.notifications)

    while True:
        for account in config.accounts:
            new_messages = await incremental_sync(account, cache)
            if new_messages:
                await notifier.notify(new_messages)
                await write_status_file(cache)
        await asyncio.sleep(config.general.check_interval_seconds)
```

`termite install-daemon` (or `/daemon install`) writes the appropriate service file and registers it — no manual plist/unit editing required.

macOS launchd plist written to `~/Library/LaunchAgents/com.termite.daemon.plist`. Linux systemd unit written to `~/.config/systemd/user/termite.service`.

---

## Theme system — `themes/` + `ui/theme_manager.py`

Termite's theme system has three layers: a base contract, built-in themes, and user themes.

### Theme contract

Every theme is a `.tcss` file that defines these CSS variables. All widget styles reference variables — never hardcoded colors.

```css
/* Required variables — every Termite theme must define all of these */

$background: #1a1a2e;
$surface: #16213e;
$surface-alt: #0f3460;
$surface-highlight: #1f2b4a;

$primary: #e94560;
$secondary: #533483;
$accent: #0f9b8e;

$text: #eaeaea;
$text-muted: #888888;
$text-dim: #555555;

$unread-indicator: #e94560;
$unread-subject: #ffffff;
$read-subject: #888888;
$read-preview: #666666;

$border: #2a2a4a;
$border-focus: #e94560;
$selection: #533483;
$selection-text: #ffffff;

$success: #4caf50;
$warning: #ff9800;
$danger: #f44336;
$info: #2196f3;

$inbox-badge: #e94560;
$inbox-badge-text: #ffffff;
$status-bar-bg: #0d0d1a;
$status-bar-text: #888888;
$command-bar-bg: #0d0d1a;
$command-bar-border: #e94560;
```

### Built-in themes

| Name | Description |
|---|---|
| `dark` | Default — dark navy, red accents |
| `light` | Clean white, blue accents |
| `dracula` | The classic purple/pink Dracula palette |
| `tokyo-night` | Muted blues and purples, soft accents |
| `catppuccin-mocha` | Warm dark, pastel accents |
| `catppuccin-latte` | Catppuccin light variant |
| `gruvbox` | Warm retro browns and greens |
| `nord` | Arctic blue-grey palette |
| `solarized-dark` | The original solarized dark |
| `high-contrast` | Accessibility-first, WCAG AA compliant |
| `matrix` | Green on black, because terminal |

### User themes

Users drop any `.tcss` file into `~/.termite/themes/`. Termite auto-discovers it on next launch or on `/theme list`. Install community themes the same way oh-my-zsh works — the community maintains a `termite-themes` GitHub repo, users clone or copy individual files.

### `ui/theme_manager.py`

```python
class ThemeManager:
    def discover(self) -> list[ThemeInfo]:
        """Scan package themes/ dir + ~/.termite/themes/ and return all valid themes."""

    def validate(self, path: Path) -> ValidationResult:
        """Check that all required CSS variables are defined."""

    def apply(self, name_or_path: str, app: App) -> None:
        """Hot-swap the active stylesheet. No restart required."""

    def current(self) -> ThemeInfo:
        """Return the currently active theme."""
```

Hot-swapping uses `app.stylesheet.replace(new_theme_path)` — Textual supports this natively.

### Theme config

```toml
[general]
theme = "tokyo-night"
# OR a path:
theme = "~/.termite/themes/my-custom.tcss"
```

### `/theme` command

```
:theme dracula          # switch immediately, hot-reloads
:theme list             # show all available themes with preview swatches
:theme edit             # open ~/.termite/themes/current.tcss in $EDITOR
:theme validate         # check current theme file for missing variables
```

---

## Keybinding system

All keybindings are defined in `config/defaults.py` as a flat dict, overridden by `config.toml`. Passed to Textual's `BINDINGS` at the Screen level:

```python
class MainScreen(Screen):
    @classmethod
    def build_bindings(cls, config: Config) -> list[Binding]:
        kb = config.keybindings
        return [
            Binding(kb.compose,      "compose",      "Compose"),
            Binding(kb.reply,        "reply",        "Reply"),
            Binding(kb.archive,      "archive",      "Archive"),
            Binding(kb.delete,       "delete",       "Delete"),
            Binding(kb.next_thread,  "next_thread",  "Next"),
            Binding(kb.prev_thread,  "prev_thread",  "Prev"),
            Binding(kb.snooze,       "snooze",       "Snooze"),
            Binding(kb.search,       "search",       "Search"),
            Binding(kb.inbox_zero,   "inbox_zero",   "Zero"),
        ]
```

Users can remap everything in `config.toml`. Defaults are Superhuman-inspired and vim-flavored.

---

## Compose flow

`ComposeScreen` is a full-screen overlay pushed onto the Textual screen stack:
- `to`, `cc`, `bcc`, `subject` as `Input` widgets with tab-navigation between fields
- Body as Textual's built-in `TextArea` widget
- **Undo send**: on send, queue the email in a 5-second delay (configurable). Show a dismissable toast "Sent — Undo (5s)". If tapped, cancel the send coroutine.
- **Snippets**: `/snippet {name}` in the body expands to a stored text block defined in config

---

## Search

Two-tier:
1. **Local FTS5** — instant results from SQLite `messages_fts` virtual table using BM25 ranking. Zero network round-trips, results appear as the user types.
2. **Remote IMAP search** — fallback to `IMAP SEARCH` when the user explicitly requests "search all mail" (i.e. wants results beyond the local cache window).

`SearchScreen` is a slide-up panel overlay, not a full-screen replacement, so the user retains context of what they were looking at.

---

## First-run experience

On first launch with no config:
1. Detect if `~/.termite/config.toml` exists
2. If not, push `SetupScreen` — a friendly wizard that walks through `/connect`
3. After first account is connected and initial sync completes, drop into `MainScreen`

---

## Packaging and distribution

```toml
# pyproject.toml
[project]
name = "termite-mail"
version = "0.1.0"
requires-python = ">=3.12"
dependencies = [
    "textual>=0.82",
    "imapclient>=3.0",
    "aiosmtplib>=3.0",
    "aiosqlite>=0.20",
    "keyring>=25",
    "google-auth-oauthlib>=1.2",
    "msal>=1.28",
    "html2text>=2024.2",
    "mail-parser>=3.15",
    "click>=8.1",
    "tomli-w>=1.0",
    "pydantic>=2.7",
    "structlog>=24",
    "rich>=13",
    "desktop-notifier>=3.5",
]

[project.scripts]
termite = "termite.cli:main"

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"
```

### Distribution targets
- **PyPI** — `pip install termite-mail` / `uv tool install termite-mail`
- **Homebrew tap** — `brew install termite-mail` (formula wraps the PyPI package)
- **GitHub releases** — single-file binary via `pyinstaller` or `shiv` for users who don't want Python installed

---

## Testing strategy

- Mock IMAP responses using `imapclient`'s mockable `IMAPClient` interface
- Snapshot tests for Textual UI using `textual.testing.Pilot`
- All sync engine logic tested against a local Dovecot instance in CI (Docker)
- Theme validation tested by running `ThemeManager.validate()` against every bundled `.tcss` file

---

## Phase 1 MVP scope

Build these first, in order:

1. `config/` loader + Pydantic schema
2. `providers/gmail.py` OAuth2 flow + IMAP/SMTP settings
3. `cache/` DB setup + FTS5 migrations
4. `engine/imap.py` + `engine/sync.py` (initial sync only, no IDLE yet)
5. `engine/thread.py` JWZ threading
6. `ui/screens/main.py` three-pane layout (InboxList, ThreadPane, MessageView)
7. Core keybindings: `j/k` navigate, `e` archive, `r` reply, `c` compose, `/` search
8. `commands/connect.py` Gmail wizard
9. Local FTS5 search
10. `themes/dark.tcss` + `themes/light.tcss` + `ui/theme_manager.py` hot-swap
11. `notifications/status.py` status.json writer
12. `cli.py` entry point + `--daemon` flag

### Deferred to Phase 2
Outlook provider, IMAP IDLE, snooze, sessions, undo send, snippets, Fastmail, attachments, full notification suite, daemon install command, remaining built-in themes, Homebrew formula, community themes repo.

---

## Open source notes

- **License**: MIT
- **Community themes repo**: `termite-themes` on GitHub, same model as oh-my-zsh themes — one `.tcss` file per theme, PR to contribute
- **Provider contributions**: `CONTRIBUTING.md` includes a step-by-step guide for adding a new provider by implementing `BaseProvider`
- **Versioning**: CalVer (`YYYY.MM.patch`) to make release recency obvious to users
