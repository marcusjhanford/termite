package config

import (
	"fmt"
	"strings"
)

// ValidationError holds a single validation failure.
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// Validate checks a Config for correctness and returns an error
// describing all validation failures, if any.
func Validate(cfg *Config) error {
	var errs []ValidationError

	// General
	if cfg.General.Theme == "" {
		errs = append(errs, ValidationError{"general.theme", "must not be empty"})
	}
	if cfg.General.CheckIntervalSeconds < 10 {
		errs = append(errs, ValidationError{"general.check_interval_seconds", "must be at least 10"})
	}

	// Notifications
	validNotifyOn := map[string]bool{"unread": true, "all": true, "none": true}
	if !validNotifyOn[cfg.Notifications.NotifyOn] {
		errs = append(errs, ValidationError{"notifications.notify_on", "must be one of: unread, all, none"})
	}

	// Accounts
	accountIDs := make(map[string]bool)
	for i, acc := range cfg.Accounts {
		prefix := fmt.Sprintf("accounts[%d]", i)
		if acc.ID == "" {
			errs = append(errs, ValidationError{prefix + ".id", "must not be empty"})
		}
		if accountIDs[acc.ID] {
			errs = append(errs, ValidationError{prefix + ".id", "duplicate account ID: " + acc.ID})
		}
		accountIDs[acc.ID] = true

		if acc.Email == "" {
			errs = append(errs, ValidationError{prefix + ".email", "must not be empty"})
		}

		validProviders := map[string]bool{"gmail": true, "outlook": true, "fastmail": true, "generic": true}
		if !validProviders[acc.Provider] {
			errs = append(errs, ValidationError{prefix + ".provider", "must be one of: gmail, outlook, fastmail, generic"})
		}
	}

	// Split inboxes
	for i, inbox := range cfg.SplitInboxes {
		prefix := fmt.Sprintf("split_inboxes[%d]", i)
		if inbox.ID == "" {
			errs = append(errs, ValidationError{prefix + ".id", "must not be empty"})
		}
		if inbox.Label == "" {
			errs = append(errs, ValidationError{prefix + ".label", "must not be empty"})
		}
	}

	if len(errs) > 0 {
		msgs := make([]string, len(errs))
		for i, e := range errs {
			msgs[i] = e.Error()
		}
		return fmt.Errorf("config validation failed:\n  %s", strings.Join(msgs, "\n  "))
	}

	return nil
}
