package daemon

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/termite-mail/termite/internal/config"
	"github.com/termite-mail/termite/internal/db"
	"github.com/termite-mail/termite/internal/engine"
	"github.com/termite-mail/termite/internal/notifications"
)

// Event is emitted by the Daemon when a sync completes and new mail was found.
type Event struct {
	AccountID string
	NewCount  int
}

// Daemon runs the headless sync loop either in standalone or embedded mode.
type Daemon struct {
	cfg             *config.Config
	database        *db.DB
	accounts        []engine.Account
	notifier        *notifications.Manager
	interval        time.Duration
	events          chan Event
	ctx             context.Context
	cancel          context.CancelFunc
	mu              sync.Mutex
	running         bool
	ownsDB          bool
	suppressNotifs  bool
}

// New creates a Daemon from the given configuration. If database is nil, the
// Daemon will open its own DB connection and own it (standalone mode). If a
// database is provided, the caller retains ownership and must close it.
// suppressNotifs disables desktop notifications — used when the daemon runs
// inside the TUI where the user is already present.
func New(cfg *config.Config, database *db.DB, suppressNotifs bool) (*Daemon, error) {
	var ownsDB bool
	dbConn := database
	if dbConn == nil {
		var err error
		dbConn, err = db.Open()
		if err != nil {
			return nil, fmt.Errorf("daemon: open database: %w", err)
		}
		ownsDB = true
	}

	accounts := make([]engine.Account, 0, len(cfg.Accounts))
	for _, acctCfg := range cfg.Accounts {
		acct, err := engine.NewAccount(acctCfg)
		if err != nil {
			slog.Error("daemon: failed to create account", "id", acctCfg.ID, "error", err)
			continue
		}

		if err := dbConn.InsertAccount(acctCfg.ID, acctCfg.Email, acctCfg.Provider, acctCfg.Name); err != nil {
			slog.Error("daemon: failed to insert account", "id", acctCfg.ID, "error", err)
			continue
		}

		accounts = append(accounts, acct)
	}

	if len(accounts) == 0 {
		if ownsDB {
			dbConn.Close()
		}
		return nil, fmt.Errorf("daemon: no accounts configured")
	}

	interval := time.Duration(cfg.General.CheckIntervalSeconds) * time.Second
	if interval < 10*time.Second {
		interval = 60 * time.Second
	}

	var notifier *notifications.Manager
	if !suppressNotifs {
		notifier = notifications.NewManager(cfg.Notifications)
	}

	return &Daemon{
		cfg:            cfg,
		database:       dbConn,
		accounts:       accounts,
		notifier:       notifier,
		interval:       interval,
		events:         make(chan Event, 16),
		ownsDB:         ownsDB,
		suppressNotifs: suppressNotifs,
	}, nil
}

// Start launches the background sync loop. It is safe to call multiple times;
// subsequent calls are no-ops if already running.
func (d *Daemon) Start() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.running {
		return nil
	}

	d.ctx, d.cancel = context.WithCancel(context.Background())
	d.running = true

	// Initial sync pass.
	for _, acct := range d.accounts {
		d.runSync(acct)
	}

	// Update status file after initial pass.
	if d.notifier != nil {
		if err := d.notifier.WriteStatus(d.database); err != nil {
			slog.Warn("daemon: failed to write status", "error", err)
		}
	}

	go d.runLoop()

	slog.Info("daemon: started sync loop",
		"accounts", len(d.accounts),
		"interval", d.interval,
	)

	return nil
}

// Stop cancels the sync loop and cleans up resources owned by the Daemon.
// It is safe to call multiple times.
func (d *Daemon) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.running {
		return
	}

	d.running = false
	if d.cancel != nil {
		d.cancel()
	}
	close(d.events)

	if d.ownsDB && d.database != nil {
		_ = d.database.Close()
	}

	slog.Info("daemon: stopped")
}

// Running reports whether the sync loop is currently active.
func (d *Daemon) Running() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.running
}

// Events returns a read-only channel that emits sync events. The channel is
// closed when the Daemon is stopped.
func (d *Daemon) Events() <-chan Event {
	return d.events
}

func (d *Daemon) runLoop() {
	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()

	for {
		select {
		case <-d.ctx.Done():
			return
		case <-ticker.C:
			for _, acct := range d.accounts {
				d.runSync(acct)
			}
			if d.notifier != nil {
				if err := d.notifier.WriteStatus(d.database); err != nil {
					slog.Warn("daemon: failed to write status", "error", err)
				}
			}
		}
	}
}

func (d *Daemon) runSync(acct engine.Account) {
	result := engine.IncrementalSync(acct, d.database, d.cfg.SplitInboxes)

	if result.Err != nil {
		slog.Error("daemon: sync failed",
			"account", result.AccountID,
			"error", result.Err,
		)
		return
	}

	slog.Info("daemon: sync complete",
		"account", result.AccountID,
		"new", result.NewCount,
	)

	if result.NewCount > 0 {
		// Emit event for TUI listeners.
		select {
		case d.events <- Event{AccountID: result.AccountID, NewCount: result.NewCount}:
		default:
		}

		// Notify on new mail (skipped when suppressed).
		if d.notifier != nil {
			msgs, err := d.database.GetThreadMessages(result.AccountID)
			if err != nil {
				slog.Warn("daemon: failed to fetch messages for notification",
					"account", result.AccountID,
					"error", err,
				)
				return
			}

			if len(msgs) > result.NewCount {
				msgs = msgs[len(msgs)-result.NewCount:]
			}

			if err := d.notifier.Notify(msgs); err != nil {
				slog.Warn("daemon: notification failed",
					"account", result.AccountID,
					"error", err,
				)
			}
		}
	}
}

// Run starts the daemon in blocking standalone mode. It acquires a PID lock
// so only one daemon process runs at a time, blocks until an unrecoverable
// error or signal, then cleans up.
func Run(cfg *config.Config) error {
	lockPath, err := pidLockPath()
	if err != nil {
		return fmt.Errorf("daemon: get lock path: %w", err)
	}

	if err := acquirePIDLock(lockPath); err != nil {
		return err
	}
	defer releasePIDLock(lockPath)

	d, err := New(cfg, nil, false)
	if err != nil {
		return err
	}

	slog.Info("starting termite daemon")
	if err := d.Start(); err != nil {
		return err
	}

	// Block until the context is cancelled (signal handling is left to the OS / caller).
	<-d.ctx.Done()
	d.Stop()
	return nil
}

// pidLockPath returns the path to the PID lock file.
func pidLockPath() (string, error) {
	dir, err := config.DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "daemon.pid"), nil
}

// acquirePIDLock writes the current PID to the lock file. If the file already
// exists and refers to a running process, it returns an error.
func acquirePIDLock(path string) error {
	if _, err := os.Stat(path); err == nil {
		data, err := os.ReadFile(path)
		if err == nil {
			pid, _ := strconv.Atoi(string(data))
			if pid > 0 && processExists(pid) {
				return fmt.Errorf("daemon: another daemon is already running (pid %d)", pid)
			}
		}
		// Stale lock — overwrite.
	}

	pid := os.Getpid()
	if err := os.WriteFile(path, []byte(strconv.Itoa(pid)+"\n"), 0o644); err != nil {
		return fmt.Errorf("daemon: write pid lock: %w", err)
	}
	return nil
}

// releasePIDLock removes the PID lock file.
func releasePIDLock(path string) {
	_ = os.Remove(path)
}

// processExists checks whether a process with the given PID exists.
// On Unix it sends signal 0; on Windows it opens the process.
func processExists(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Signal 0 is a no-op check on Unix; on Windows this always returns nil
	// for the current user's processes, which is acceptable for our purposes.
	err = proc.Signal(os.Signal(nil))
	return err == nil
}
