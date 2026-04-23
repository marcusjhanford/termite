package config

// Default returns a Config with sane defaults for all fields.
func Default() *Config {
	return &Config{
		General: GeneralConfig{
			Theme:                "dark",
			Editor:               "vim",
			CheckIntervalSeconds: 60,
			StartupInbox:         "primary",
			ReduceMotion:         false,
			AutoStartDaemon:      true,
		},
		Notifications: NotificationConfig{
			Desktop:      true,
			TerminalBell: false,
			TmuxTitle:    true,
			StatusFile:   true,
			NotifyOn:     "unread",
		},
		Metrics: MetricsConfig{
			Enabled:         true,
			ShowInStatusbar: true,
			ToastMilestones: true,
		},
		Keybindings:  DefaultKeybindings(),
		Accounts:     nil,
		SplitInboxes: nil,
	}
}

// DefaultKeybindings returns the default Superhuman-inspired keybindings.
func DefaultKeybindings() KeybindingsConfig {
	return KeybindingsConfig{
		Compose:    "c",
		Reply:      "r",
		ReplyAll:   "R",
		Forward:    "f",
		Archive:    "e",
		Delete:     "#",
		MarkRead:   "m",
		MarkUnread: "M",
		Snooze:     "h",
		Next:       "j",
		Prev:       "k",
		Open:       "enter",
		Zero:       "I",
		Search:     "/",
		Command:    ":",
		Quit:       "q",
	}
}
