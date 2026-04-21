package themes

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"
)

//go:embed builtin/*.toml
var builtinFS embed.FS

// StylesUpdatedMsg is sent via Bubble Tea when the active theme changes.
// Consumers should listen for this message and refresh their views.
type StylesUpdatedMsg struct {
	Styles Styles
}

// ThemeInfo describes a discovered theme for listing purposes.
type ThemeInfo struct {
	Name    string // human-readable name from the TOML
	ID      string // file stem used to reference the theme (e.g. "dark")
	Path    string // file path ("" for built-in themes)
	Builtin bool   // true when the theme is embedded in the binary
}

// ThemeManager discovers, loads, validates and applies themes.
type ThemeManager struct {
	mu      sync.RWMutex
	current *Theme
	styles  Styles
}

// NewThemeManager returns an initialised ThemeManager with the default
// dark theme applied.
func NewThemeManager() (*ThemeManager, error) {
	tm := &ThemeManager{}
	if err := tm.Apply("dark"); err != nil {
		return nil, fmt.Errorf("failed to apply default theme: %w", err)
	}
	return tm, nil
}

// Current returns the currently active theme. It is safe for concurrent use.
func (tm *ThemeManager) Current() *Theme {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	t := *tm.current // shallow copy
	return &t
}

// CurrentStyles returns the pre-built lipgloss styles for the active theme.
func (tm *ThemeManager) CurrentStyles() Styles {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.styles
}

// Discover returns all available themes (built-in + user-defined).
// User themes in ~/.termite/themes/ override built-in themes with the same ID.
func (tm *ThemeManager) Discover() ([]ThemeInfo, error) {
	seen := make(map[string]ThemeInfo)

	// 1. Built-in themes
	entries, err := builtinFS.ReadDir("builtin")
	if err != nil {
		return nil, fmt.Errorf("reading built-in themes: %w", err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".toml") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".toml")
		data, err := builtinFS.ReadFile("builtin/" + e.Name())
		if err != nil {
			continue
		}
		var t Theme
		if err := toml.Unmarshal(data, &t); err != nil {
			continue
		}
		name := t.Name
		if name == "" {
			name = id
		}
		seen[id] = ThemeInfo{
			Name:    name,
			ID:      id,
			Builtin: true,
		}
	}

	// 2. User themes override built-ins
	userDir, err := userThemesDir()
	if err == nil {
		dirEntries, err := os.ReadDir(userDir)
		if err == nil {
			for _, e := range dirEntries {
				if e.IsDir() || !strings.HasSuffix(e.Name(), ".toml") {
					continue
				}
				id := strings.TrimSuffix(e.Name(), ".toml")
				fp := filepath.Join(userDir, e.Name())
				data, err := os.ReadFile(fp)
				if err != nil {
					continue
				}
				var t Theme
				if err := toml.Unmarshal(data, &t); err != nil {
					continue
				}
				name := t.Name
				if name == "" {
					name = id
				}
				seen[id] = ThemeInfo{
					Name:    name,
					ID:      id,
					Path:    fp,
					Builtin: false,
				}
			}
		}
	}

	out := make([]ThemeInfo, 0, len(seen))
	for _, info := range seen {
		out = append(out, info)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out, nil
}

// Load reads and parses a theme by ID. It checks user themes first,
// then falls back to built-in themes.
func (tm *ThemeManager) Load(id string) (*Theme, error) {
	// Try user themes first.
	if t, err := loadUserTheme(id); err == nil {
		return t, nil
	}
	// Fall back to built-in.
	return loadBuiltinTheme(id)
}

// Validate checks that a theme has all required colour fields populated
// and that every colour value is a valid hex string.
func (tm *ThemeManager) Validate(t *Theme) error {
	if t == nil {
		return fmt.Errorf("theme is nil")
	}

	required := map[string]string{
		"background":       t.Background,
		"surface":          t.Surface,
		"primary":          t.Primary,
		"text":             t.Text,
		"text_muted":       t.TextMuted,
		"unread_indicator": t.UnreadIndicator,
		"unread_subject":   t.UnreadSubject,
		"read_subject":     t.ReadSubject,
		"border":           t.Border,
		"border_focus":     t.BorderFocus,
		"selection":        t.Selection,
		"selection_text":   t.SelectionText,
		"status_bar_bg":    t.StatusBarBg,
		"status_bar_text":  t.StatusBarText,
		"success":          t.Success,
		"warning":          t.Warning,
		"danger":           t.Danger,
		"info":             t.Info,
	}

	hexPattern := regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

	for field, value := range required {
		if value == "" {
			return fmt.Errorf("theme %q: missing required field %q", t.Name, field)
		}
		if !hexPattern.MatchString(value) {
			return fmt.Errorf("theme %q: field %q has invalid hex color %q", t.Name, field, value)
		}
	}

	return nil
}

// Apply loads, validates and activates the theme with the given ID.
// It returns a StylesUpdatedMsg suitable for sending through a Bubble Tea
// program so that all views can refresh.
func (tm *ThemeManager) Apply(id string) error {
	t, err := tm.Load(id)
	if err != nil {
		return fmt.Errorf("loading theme %q: %w", id, err)
	}
	if err := tm.Validate(t); err != nil {
		return fmt.Errorf("validating theme %q: %w", id, err)
	}

	styles := BuildStyles(t)

	tm.mu.Lock()
	tm.current = t
	tm.styles = styles
	tm.mu.Unlock()

	return nil
}

// ApplyMsg is like Apply but returns the StylesUpdatedMsg for use as a
// tea.Cmd return value.
func (tm *ThemeManager) ApplyMsg(id string) (StylesUpdatedMsg, error) {
	if err := tm.Apply(id); err != nil {
		return StylesUpdatedMsg{}, err
	}
	tm.mu.RLock()
	s := tm.styles
	tm.mu.RUnlock()
	return StylesUpdatedMsg{Styles: s}, nil
}

// ── helpers ──────────────────────────────────────────────────────────────────

// userThemesDir returns ~/.termite/themes, creating it if necessary.
func userThemesDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".termite", "themes")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func loadUserTheme(id string) (*Theme, error) {
	dir, err := userThemesDir()
	if err != nil {
		return nil, err
	}
	fp := filepath.Join(dir, id+".toml")
	data, err := os.ReadFile(fp)
	if err != nil {
		return nil, err
	}
	var t Theme
	if err := toml.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("parsing user theme %q: %w", id, err)
	}
	if t.Name == "" {
		t.Name = id
	}
	return &t, nil
}

func loadBuiltinTheme(id string) (*Theme, error) {
	data, err := builtinFS.ReadFile("builtin/" + id + ".toml")
	if err != nil {
		return nil, fmt.Errorf("built-in theme %q not found", id)
	}
	var t Theme
	if err := toml.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("parsing built-in theme %q: %w", id, err)
	}
	if t.Name == "" {
		t.Name = id
	}
	return &t, nil
}
