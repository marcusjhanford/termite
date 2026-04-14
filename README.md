<p align="center">
  <img src="termite_logo.png" alt="Termite Logo" width="300"/>
</p>

# Termite

> A Superhuman-inspired, keyboard-first TUI email client. Open source. Runs locally. Your data stays yours.

Termite is a lightning-fast, terminal-native email client built on Python and Textual. It brings modern email workflows—like split inboxes, instant full-text search, and Vim-style navigation—directly to your command line.

## Features

- **Keyboard-First Navigation**: Vim-inspired bindings (`j`/`k` to navigate, `e` to archive, `c` to compose).
- **Split Inboxes**: Define rules to automatically route newsletters, notifications, and primary mail into dedicated views.
- **Local-First & Offline**: Powered by an embedded SQLite database. Your emails are cached locally for instant access, even on an airplane.
- **Instant Search**: Full-Text Search (FTS5) across your entire local archive. Results appear as you type.
- **Push Updates (IMAP IDLE)**: Emails arrive instantly via long-lived IDLE connections, no polling required.
- **Background Sync Daemon**: Runs headlessly via macOS `launchd` or Linux `systemd` to keep your inbox synced before you even open the app.
- **Beautiful Theming**: 10+ built-in color schemes (Tokyo Night, Catppuccin, Dracula) using hot-swappable `.tcss` stylesheets.
- **Multi-Account Support**: Connect Gmail, Outlook, and Fastmail simultaneously.
- **Inbox Zero Gamification**: A beautiful procedural 16-bit sunset animation rewards you for clearing your inbox.

## Installation

Termite requires Python 3.12 or newer. We recommend installing it using `uv`:

```bash
uv tool install termite-mail
```

Or via standard `pip`:

```bash
pip install termite-mail
```

## Quickstart

1. Launch Termite:
   ```bash
   termite
   ```
2. Press `:` to open the Command Bar.
3. Type `/connect your.email@gmail.com` to launch the interactive setup wizard.
4. Follow the OAuth prompts in your browser.
5. Termite will begin its initial sync. Once complete, your inbox will appear!

## Keybindings

| Action | Key | Description |
| :--- | :---: | :--- |
| **Command Bar** | `:` | Open the command palette to run commands (e.g. `/theme list`). |
| **Search** | `/` | Open the command palette pre-filled with `/search `. |
| **Navigate Down** | `j` | Move to the next thread. |
| **Navigate Up** | `k` | Move to the previous thread. |
| **Compose** | `c` | Open the full-screen compose overlay. |
| **Reply** | `r` | Reply to the currently selected thread. |
| **Archive** | `e` | Archive the selected thread. |
| **Delete** | `#` | Move the selected thread to trash. |
| **Snooze** | `h` | Snooze the thread until later (Coming Soon). |
| **Quit** | `q` | Exit Termite. |

## Commands

Access the Command Bar by pressing `:`.

- `/connect <email> [gmail|outlook]`: Authenticate and add a new inbox.
- `/theme [name|list|edit]`: Hot-swap the current UI theme.
- `/sessions [list|save|restore] <name>`: Manage named UI sessions to instantly restore your layout and search context.
- `/search <query>`: Instantly search your local cache.
- `/daemon [install|status|stop]`: Manage the background sync process.

## Configuration

Termite is highly customizable. Configuration lives at `~/.termite/config.toml`. 

Example configuration:

```toml
[general]
theme = "tokyo-night"
editor = "vim"

[[accounts]]
id = "work"
name = "Work"
email = "you@company.com"
provider = "gmail"

[[split_inboxes]]
id = "newsletters"
label = "Newsletters"
accounts = ["work"]
rules = [
  { field = "list_unsubscribe", exists = true }
]
```

## Architecture

Termite is built on a modern, async Python stack:
- **TUI Framework**: `textual`
- **Database**: `aiosqlite`
- **IMAP/SMTP**: `imapclient` and `aiosmtplib`
- **Auth**: `google-auth-oauthlib`, `msal`, and `keyring`

## License

MIT License
