package metrics

import (
	"fmt"
	"time"

	"github.com/termite-mail/termite/internal/db"
)

// DailySummary represents aggregated metrics for a single day.
type DailySummary struct {
	Date       string
	Cleared    int
	Sent       int
	InboxZeros int
	TimeInApp  int // seconds
}

// Totals represents all-time aggregated metrics.
type Totals struct {
	TotalCleared  int
	TotalSent     int
	TotalZeros    int
	LongestStreak int
	CurrentStreak int
}

// MetricsTracker records and queries productivity metrics.
type MetricsTracker struct {
	db           *db.DB
	sessionStart time.Time
	today        string // ISO date "2006-01-02", refreshed at midnight
}

// NewTracker creates a new MetricsTracker.
func NewTracker(database *db.DB) *MetricsTracker {
	return &MetricsTracker{
		db:           database,
		sessionStart: time.Now(),
		today:        time.Now().Format("2006-01-02"),
	}
}

// refreshToday updates the today field if the date has rolled over.
func (t *MetricsTracker) refreshToday() {
	t.today = time.Now().Format("2006-01-02")
}

// ensureRow ensures a daily_metrics row exists for the given date and account.
func (t *MetricsTracker) ensureRow(accountID string) error {
	t.refreshToday()
	_, err := t.db.Exec(`
		INSERT OR IGNORE INTO daily_metrics (date, account_id)
		VALUES (?, ?)
	`, t.today, accountID)
	return err
}

// RecordCleared increments the emails_cleared counter for today.
// Called by the engine after archiving or deleting threads.
func (t *MetricsTracker) RecordCleared(accountID string, count int) error {
	if err := t.ensureRow(accountID); err != nil {
		return fmt.Errorf("metrics: ensure row: %w", err)
	}
	_, err := t.db.Exec(`
		UPDATE daily_metrics
		SET emails_cleared = emails_cleared + ?
		WHERE date = ? AND account_id = ?
	`, count, t.today, accountID)
	return err
}

// RecordSent increments the emails_sent counter for today.
// Called by SMTP after a send succeeds.
func (t *MetricsTracker) RecordSent(accountID string) error {
	if err := t.ensureRow(accountID); err != nil {
		return fmt.Errorf("metrics: ensure row: %w", err)
	}
	_, err := t.db.Exec(`
		UPDATE daily_metrics
		SET emails_sent = emails_sent + 1
		WHERE date = ? AND account_id = ?
	`, t.today, accountID)
	return err
}

// RecordInboxZero increments the inbox_zeros counter for today.
// Called by thread_list when unread count reaches 0.
func (t *MetricsTracker) RecordInboxZero(accountID string) error {
	if err := t.ensureRow(accountID); err != nil {
		return fmt.Errorf("metrics: ensure row: %w", err)
	}
	_, err := t.db.Exec(`
		UPDATE daily_metrics
		SET inbox_zeros = inbox_zeros + 1
		WHERE date = ? AND account_id = ?
	`, t.today, accountID)
	return err
}

// FlushSession persists the current session duration into today's time_in_app_s.
// Called on clean shutdown.
func (t *MetricsTracker) FlushSession() error {
	t.refreshToday()
	elapsed := int(time.Since(t.sessionStart).Seconds())
	if elapsed <= 0 {
		return nil
	}

	// Ensure at least one row exists (use empty account_id for session-level tracking).
	_, err := t.db.Exec(`
		INSERT OR IGNORE INTO daily_metrics (date, account_id)
		VALUES (?, '')
	`, t.today)
	if err != nil {
		return fmt.Errorf("metrics: ensure session row: %w", err)
	}

	_, err = t.db.Exec(`
		UPDATE daily_metrics
		SET time_in_app_s = time_in_app_s + ?
		WHERE date = ? AND account_id = ''
	`, elapsed, t.today)
	if err != nil {
		return fmt.Errorf("metrics: flush session: %w", err)
	}

	// Reset session start for the next flush.
	t.sessionStart = time.Now()
	return nil
}

// TodaySummary returns today's aggregated summary across all accounts.
func (t *MetricsTracker) TodaySummary() (DailySummary, error) {
	t.refreshToday()
	var summary DailySummary
	summary.Date = t.today

	err := t.db.QueryRow(`
		SELECT
			COALESCE(SUM(emails_cleared), 0),
			COALESCE(SUM(emails_sent), 0),
			COALESCE(SUM(inbox_zeros), 0),
			COALESCE(SUM(time_in_app_s), 0)
		FROM daily_metrics
		WHERE date = ?
	`, t.today).Scan(&summary.Cleared, &summary.Sent, &summary.InboxZeros, &summary.TimeInApp)
	if err != nil {
		return summary, fmt.Errorf("metrics: today summary: %w", err)
	}

	return summary, nil
}

// AllTimeTotals returns all-time aggregated totals.
func (t *MetricsTracker) AllTimeTotals() (Totals, error) {
	var totals Totals

	err := t.db.QueryRow(`
		SELECT
			COALESCE(SUM(emails_cleared), 0),
			COALESCE(SUM(emails_sent), 0),
			COALESCE(SUM(inbox_zeros), 0)
		FROM daily_metrics
	`).Scan(&totals.TotalCleared, &totals.TotalSent, &totals.TotalZeros)
	if err != nil {
		return totals, fmt.Errorf("metrics: all-time totals: %w", err)
	}

	// Compute streaks.
	current, longest, err := t.computeStreaks()
	if err != nil {
		return totals, err
	}
	totals.CurrentStreak = current
	totals.LongestStreak = longest

	return totals, nil
}

// CurrentStreak returns the current consecutive-day inbox zero streak.
func (t *MetricsTracker) CurrentStreak() (int, error) {
	current, _, err := t.computeStreaks()
	return current, err
}

// computeStreaks calculates current and longest streak of consecutive days
// with at least one inbox zero.
func (t *MetricsTracker) computeStreaks() (current, longest int, err error) {
	// Get all dates with at least one inbox zero, ordered descending.
	rows, err := t.db.Query(`
		SELECT date FROM daily_metrics
		WHERE inbox_zeros > 0
		GROUP BY date
		ORDER BY date DESC
	`)
	if err != nil {
		return 0, 0, fmt.Errorf("metrics: compute streaks: %w", err)
	}
	defer rows.Close()

	var dates []time.Time
	for rows.Next() {
		var dateStr string
		if err := rows.Scan(&dateStr); err != nil {
			return 0, 0, fmt.Errorf("metrics: scan date: %w", err)
		}
		d, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}
		dates = append(dates, d)
	}
	if err := rows.Err(); err != nil {
		return 0, 0, err
	}

	if len(dates) == 0 {
		return 0, 0, nil
	}

	// Check if the most recent date is today or yesterday (streak is still active).
	today := time.Now().Truncate(24 * time.Hour)
	yesterday := today.Add(-24 * time.Hour)
	mostRecent := dates[0].Truncate(24 * time.Hour)

	streakActive := mostRecent.Equal(today) || mostRecent.Equal(yesterday)

	// Walk through dates to find streaks.
	currentStreak := 1
	longestStreak := 1
	streak := 1

	for i := 1; i < len(dates); i++ {
		prev := dates[i-1].Truncate(24 * time.Hour)
		curr := dates[i].Truncate(24 * time.Hour)
		diff := prev.Sub(curr)

		if diff == 24*time.Hour {
			streak++
		} else {
			if streak > longestStreak {
				longestStreak = streak
			}
			streak = 1
		}

		// The current streak is the first contiguous run from today/yesterday.
		if i == 1 && streakActive {
			currentStreak = streak
		} else if streakActive && streak > currentStreak {
			// Still in the first contiguous run.
			currentStreak = streak
		}
	}

	if streak > longestStreak {
		longestStreak = streak
	}
	if streakActive && streak >= currentStreak {
		currentStreak = streak
	}
	if !streakActive {
		currentStreak = 0
	}

	return currentStreak, longestStreak, nil
}

// CheckMilestones checks for newly-unlocked milestones and returns them
// for toast display. It marks new milestones as unlocked in the database.
func (t *MetricsTracker) CheckMilestones() ([]Milestone, error) {
	totals, err := t.AllTimeTotals()
	if err != nil {
		return nil, fmt.Errorf("metrics: check milestones: %w", err)
	}

	var newlyUnlocked []Milestone
	now := time.Now()

	for _, def := range MilestoneDefinitions {
		// Determine the relevant total for this milestone's category.
		var value int
		switch def.Category {
		case "cleared":
			value = totals.TotalCleared
		case "sent":
			value = totals.TotalSent
		case "zero":
			value = totals.TotalZeros
		case "streak":
			value = totals.LongestStreak
		default:
			continue
		}

		// Check if this milestone's threshold has been reached.
		if value < def.Threshold {
			continue
		}

		// Check if already unlocked in the database.
		var unlockedAt *int64
		err := t.db.QueryRow(`
			SELECT unlocked_at FROM milestones WHERE id = ?
		`, def.ID).Scan(&unlockedAt)

		if err == nil && unlockedAt != nil {
			// Already unlocked, skip.
			continue
		}

		// Unlock the milestone.
		_, err = t.db.Exec(`
			INSERT INTO milestones (id, unlocked_at, shown)
			VALUES (?, ?, 0)
			ON CONFLICT(id) DO UPDATE SET unlocked_at = ?, shown = 0
		`, def.ID, now.Unix(), now.Unix())
		if err != nil {
			return nil, fmt.Errorf("metrics: unlock milestone %s: %w", def.ID, err)
		}

		newlyUnlocked = append(newlyUnlocked, Milestone{
			MilestoneDef: def,
			UnlockedAt:   now,
		})
	}

	return newlyUnlocked, nil
}

// MarkMilestoneShown marks a milestone's toast as having been displayed.
func (t *MetricsTracker) MarkMilestoneShown(milestoneID string) error {
	_, err := t.db.Exec(`
		UPDATE milestones SET shown = 1 WHERE id = ?
	`, milestoneID)
	return err
}
