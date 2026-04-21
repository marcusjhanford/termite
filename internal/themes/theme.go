package themes

// Theme holds all color values for a Termite theme.
type Theme struct {
	Name string `toml:"name"`

	// Backgrounds
	Background       string `toml:"background"`
	Surface          string `toml:"surface"`
	SurfaceAlt       string `toml:"surface_alt"`
	SurfaceHighlight string `toml:"surface_highlight"`

	// Accents
	Primary   string `toml:"primary"`
	Secondary string `toml:"secondary"`
	Accent    string `toml:"accent"`

	// Text
	Text      string `toml:"text"`
	TextMuted string `toml:"text_muted"`
	TextDim   string `toml:"text_dim"`

	// Unread state
	UnreadIndicator string `toml:"unread_indicator"`
	UnreadSubject   string `toml:"unread_subject"`
	ReadSubject     string `toml:"read_subject"`
	ReadPreview     string `toml:"read_preview"`

	// Chrome
	Border        string `toml:"border"`
	BorderFocus   string `toml:"border_focus"`
	Selection     string `toml:"selection"`
	SelectionText string `toml:"selection_text"`
	StatusBarBg   string `toml:"status_bar_bg"`
	StatusBarText string `toml:"status_bar_text"`
	CommandBarBg  string `toml:"command_bar_bg"`
	CommandBorder string `toml:"command_border"`

	// Semantic
	Success string `toml:"success"`
	Warning string `toml:"warning"`
	Danger  string `toml:"danger"`
	Info    string `toml:"info"`

	// Forest scene
	Forest ForestPalette `toml:"forest"`
}

// ForestPalette defines the characters and colors used to render
// the animated forest scene in the Termite TUI.
type ForestPalette struct {
	TrunkChars   []string `toml:"trunk_chars"`
	BranchChars  []string `toml:"branch_chars"`
	LeafChars    []string `toml:"leaf_chars"`
	GroundChars  []string `toml:"ground_chars"`
	TrunkColor   string   `toml:"trunk_color"`
	BranchColor  string   `toml:"branch_color"`
	LeafColor1   string   `toml:"leaf_color_1"`
	LeafColor2   string   `toml:"leaf_color_2"`
	LeafColor3   string   `toml:"leaf_color_3"`
	GroundColor  string   `toml:"ground_color"`
	FireflyColor string   `toml:"firefly_color"`
}
