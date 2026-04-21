package metrics

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"

	"github.com/termite-mail/termite/internal/db"
)

// dailyMetricRow represents a row from the daily_metrics table for export.
type dailyMetricRow struct {
	Date          string `json:"date" db:"date"`
	AccountID     string `json:"account_id" db:"account_id"`
	EmailsCleared int    `json:"emails_cleared" db:"emails_cleared"`
	EmailsSent    int    `json:"emails_sent" db:"emails_sent"`
	InboxZeros    int    `json:"inbox_zeros" db:"inbox_zeros"`
	TimeInAppS    int    `json:"time_in_app_s" db:"time_in_app_s"`
	StreakDays    int    `json:"streak_days" db:"streak_days"`
}

// milestoneRow represents a row from the milestones table for export.
type milestoneRow struct {
	ID         string `json:"id" db:"id"`
	UnlockedAt *int64 `json:"unlocked_at" db:"unlocked_at"`
	Shown      int    `json:"shown" db:"shown"`
}

// exportData is the top-level structure for JSON export.
type exportData struct {
	DailyMetrics []dailyMetricRow `json:"daily_metrics"`
	Milestones   []milestoneRow   `json:"milestones"`
}

// ExportJSON writes all metrics data to the specified path as JSON.
func ExportJSON(database *db.DB, path string) error {
	data, err := gatherExportData(database)
	if err != nil {
		return fmt.Errorf("export json: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("export json: create file: %w", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		return fmt.Errorf("export json: encode: %w", err)
	}

	return nil
}

// ExportCSV writes all metrics data to the specified path as CSV.
// The CSV contains daily_metrics rows with a header row.
func ExportCSV(database *db.DB, path string) error {
	data, err := gatherExportData(database)
	if err != nil {
		return fmt.Errorf("export csv: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("export csv: create file: %w", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	// Write header.
	header := []string{"date", "account_id", "emails_cleared", "emails_sent", "inbox_zeros", "time_in_app_s", "streak_days"}
	if err := w.Write(header); err != nil {
		return fmt.Errorf("export csv: write header: %w", err)
	}

	// Write daily metrics rows.
	for _, row := range data.DailyMetrics {
		record := []string{
			row.Date,
			row.AccountID,
			fmt.Sprintf("%d", row.EmailsCleared),
			fmt.Sprintf("%d", row.EmailsSent),
			fmt.Sprintf("%d", row.InboxZeros),
			fmt.Sprintf("%d", row.TimeInAppS),
			fmt.Sprintf("%d", row.StreakDays),
		}
		if err := w.Write(record); err != nil {
			return fmt.Errorf("export csv: write row: %w", err)
		}
	}

	// Write a blank separator line, then milestones.
	if err := w.Write([]string{}); err != nil {
		return fmt.Errorf("export csv: write separator: %w", err)
	}

	milestoneHeader := []string{"milestone_id", "unlocked_at", "shown"}
	if err := w.Write(milestoneHeader); err != nil {
		return fmt.Errorf("export csv: write milestone header: %w", err)
	}

	for _, m := range data.Milestones {
		unlockedStr := ""
		if m.UnlockedAt != nil {
			unlockedStr = fmt.Sprintf("%d", *m.UnlockedAt)
		}
		record := []string{
			m.ID,
			unlockedStr,
			fmt.Sprintf("%d", m.Shown),
		}
		if err := w.Write(record); err != nil {
			return fmt.Errorf("export csv: write milestone row: %w", err)
		}
	}

	return nil
}

// gatherExportData queries the database for all metrics and milestone data.
func gatherExportData(database *db.DB) (*exportData, error) {
	data := &exportData{}

	// Query daily_metrics.
	metricsRows, err := database.Query(`
		SELECT date, account_id, emails_cleared, emails_sent, inbox_zeros, time_in_app_s, streak_days
		FROM daily_metrics
		ORDER BY date ASC, account_id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("query daily_metrics: %w", err)
	}
	defer metricsRows.Close()

	for metricsRows.Next() {
		var row dailyMetricRow
		if err := metricsRows.Scan(
			&row.Date, &row.AccountID, &row.EmailsCleared,
			&row.EmailsSent, &row.InboxZeros, &row.TimeInAppS, &row.StreakDays,
		); err != nil {
			return nil, fmt.Errorf("scan daily_metrics: %w", err)
		}
		data.DailyMetrics = append(data.DailyMetrics, row)
	}
	if err := metricsRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate daily_metrics: %w", err)
	}

	// Query milestones.
	milestoneRows, err := database.Query(`
		SELECT id, unlocked_at, shown
		FROM milestones
		ORDER BY id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("query milestones: %w", err)
	}
	defer milestoneRows.Close()

	for milestoneRows.Next() {
		var row milestoneRow
		if err := milestoneRows.Scan(&row.ID, &row.UnlockedAt, &row.Shown); err != nil {
			return nil, fmt.Errorf("scan milestones: %w", err)
		}
		data.Milestones = append(data.Milestones, row)
	}
	if err := milestoneRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate milestones: %w", err)
	}

	return data, nil
}
