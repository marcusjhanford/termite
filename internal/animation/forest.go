package animation

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"
	"strings"
	"time"

	lipgloss "charm.land/lipgloss/v2"
	"github.com/termite-mail/termite/internal/themes"
)

// ScenePhase represents the current phase of the forest scene lifecycle.
type ScenePhase int

const (
	PhaseGrowing  ScenePhase = iota // elements fade in layer by layer
	PhaseSettling                   // fireflies spawn, quote typewriter begins
	PhaseAlive                      // scene breathes
	PhaseFading                     // everything fades out
)

// ---------------------------------------------------------------------------
// Cell – one position in the pre-computed scene grid
// ---------------------------------------------------------------------------

type cell struct {
	ch rune
	fg color.Color
	bg color.Color
}

// ---------------------------------------------------------------------------
// ForestScene – warm sunset / golden-hour pixel-art forest
// ---------------------------------------------------------------------------

type ForestScene struct {
	width, height int
	theme         *themes.Theme
	particles     *ParticleSystem

	phase      ScenePhase
	phaseTick  int
	totalTicks int

	// Pre-computed scene grid (populated once in NewForestScene).
	grid [][]cell

	// Spring-driven layer opacities (used during grow/fade animations).
	skyOpacity     PaneSpring
	sunOpacity     PaneSpring
	treeOpacity    PaneSpring
	groundOpacity  PaneSpring
	titleOpacity   PaneSpring
	particleOpacty PaneSpring
	fadeOpacity    PaneSpring

	quote       string
	quoteReveal int

	firefliesSpawned bool
}

// NewForestScene creates a new warm-sunset forest scene and pre-renders the
// static elements into a 2D grid.
func NewForestScene(width, height int, theme *themes.Theme) *ForestScene {
	s := &ForestScene{
		width:     width,
		height:    height,
		theme:     theme,
		particles: NewParticleSystem(width, height),
		phase:     PhaseGrowing,
		quote:     SelectQuote("forest", time.Now()),
	}

	// Initialize opacity springs – all start at 0, animate to 1.
	s.skyOpacity = NewPaneSpring(3.0, 1.0)
	s.sunOpacity = NewPaneSpring(3.5, 0.9)
	s.treeOpacity = NewPaneSpring(5.0, 1.0)
	s.groundOpacity = NewPaneSpring(6.0, 1.0)
	s.titleOpacity = NewPaneSpring(4.0, 0.9)
	s.particleOpacty = NewPaneSpring(3.0, 1.0)
	s.fadeOpacity = NewPaneSpring(3.0, 1.0)
	s.fadeOpacity.SetPosition(1.0)
	s.fadeOpacity.SetTarget(1.0)

	s.buildGrid()
	return s
}

func (s *ForestScene) Phase() ScenePhase { return s.phase }

func (s *ForestScene) Tick() {
	s.totalTicks++
	s.phaseTick++
	dt := 16 * time.Millisecond

	// Tick all springs.
	s.skyOpacity.Tick(dt)
	s.sunOpacity.Tick(dt)
	s.treeOpacity.Tick(dt)
	s.groundOpacity.Tick(dt)
	s.titleOpacity.Tick(dt)
	s.particleOpacty.Tick(dt)
	s.fadeOpacity.Tick(dt)

	switch s.phase {
	case PhaseGrowing:
		// Stagger layer fade-in: sky -> sun -> trees/ground -> title -> particles
		if s.phaseTick == 1 {
			s.skyOpacity.SetTarget(1.0)
		}
		if s.phaseTick == 12 {
			s.sunOpacity.SetTarget(1.0)
		}
		if s.phaseTick == 25 {
			s.treeOpacity.SetTarget(1.0)
			s.groundOpacity.SetTarget(1.0)
		}
		if s.phaseTick == 45 {
			s.titleOpacity.SetTarget(1.0)
		}
		if s.phaseTick == 55 {
			s.particleOpacty.SetTarget(1.0)
		}
		if s.phaseTick >= 90 {
			s.phase = PhaseSettling
			s.phaseTick = 0
		}

	case PhaseSettling:
		s.particles.Tick()
		// Spawn fireflies in waves throughout the meadow area.
		if !s.firefliesSpawned {
			spawnTicks := []int{3, 10, 18, 26, 34, 42, 50, 58, 66, 74}
			for _, t := range spawnTicks {
				if s.phaseTick == t {
					s.particles.SpawnFirefly(s.fireflyColor())
				}
			}
			if s.phaseTick >= 74 {
				s.firefliesSpawned = true
			}
		}
		// Typewriter quote.
		if s.quoteReveal < len([]rune(s.quote)) {
			s.quoteReveal += 2
		}
		if s.phaseTick >= 80 {
			s.phase = PhaseAlive
			s.phaseTick = 0
		}

	case PhaseAlive:
		s.particles.Tick()
		// Occasionally spawn extra fireflies for a lively scene, up to a cap.
		if s.phaseTick%120 == 60 && s.particles.FireflyCount() < 14 {
			s.particles.SpawnFirefly(s.fireflyColor())
		}

	case PhaseFading:
		s.particles.Tick()
		s.fadeOpacity.Tick(dt)
	}
}

func (s *ForestScene) GrowStep() bool {
	s.Tick()
	return s.phase != PhaseGrowing
}

func (s *ForestScene) BeginFade() {
	if s.phase != PhaseFading {
		s.phase = PhaseFading
		s.phaseTick = 0
		s.fadeOpacity.SetTarget(0.0)
	}
}

func (s *ForestScene) FadeDone() bool {
	return s.phase == PhaseFading && s.fadeOpacity.Value() < 0.05
}

// ---------------------------------------------------------------------------
// Rendering
// ---------------------------------------------------------------------------

func (s *ForestScene) View() string {
	w, h := s.width, s.height
	if w <= 0 || h <= 0 {
		return ""
	}

	fade := clampF(s.fadeOpacity.Value(), 0, 1)
	skyA := clampF(s.skyOpacity.Value(), 0, 1)
	sunA := clampF(s.sunOpacity.Value(), 0, 1)
	treeA := clampF(s.treeOpacity.Value(), 0, 1)
	groundA := clampF(s.groundOpacity.Value(), 0, 1)
	titleA := clampF(s.titleOpacity.Value(), 0, 1)
	partA := clampF(s.particleOpacty.Value(), 0, 1)

	// Build a quick lookup for particles by (x,y).
	type pKey struct{ x, y int }
	particleMap := make(map[pKey]Particle, len(s.particles.Particles()))
	for _, p := range s.particles.Particles() {
		px, py := int(p.X), int(p.Y)
		if px >= 0 && px < w && py >= 0 && py < h {
			particleMap[pKey{px, py}] = p
		}
	}

	// Title and subtitle positions.
	title := "I n b o x   Z e r o"
	titleY := h / 5
	titleX := (w - len(title)) / 2
	subtitle := "~ all caught up ~"
	subY := titleY + 2
	subX := (w - len(subtitle)) / 2

	// Quote.
	var revealed string
	var quoteX int
	if s.quoteReveal > 0 {
		revealed = RevealQuote(s.quote, s.quoteReveal)
		quoteX = (w - len([]rune(revealed))) / 2
		if quoteX < 0 {
			quoteX = 0
		}
	}
	quoteY := h - 1

	// Determine layer boundaries for opacity mapping.
	groundStart := h - 5

	// Use direct ANSI escapes for performance -- lipgloss per-cell is too slow.
	lines := make([]string, h)
	for y := 0; y < h; y++ {
		var sb strings.Builder
		sb.Grow(w * 20) // pre-allocate

		for x := 0; x < w; x++ {
			c := s.grid[y][x]
			ch := c.ch
			fg := c.fg
			bg := c.bg

			// ---- Determine layer alpha for this cell ----
			// The sky gradient covers the full screen as a base layer,
			// so skyA is the base opacity for all cells.
			layerAlpha := skyA
			if isSunChar(ch) {
				layerAlpha = skyA * sunA
			}
			if y >= groundStart {
				layerAlpha = groundA
			} else {
				layerAlpha = treeA
			}

			// ---- Check for particle overlay ----
			if p, ok := particleMap[pKey{x, y}]; ok && partA > 0.1 {
				if pch, pfg, draw := particleCell(p, partA); draw {
					ch = pch
					fg = pfg
				}
			}

			// ---- Check for title overlay ----
			if titleA > 0.1 {
				if y == titleY && x >= titleX && x < titleX+len(title) {
					tch := rune(title[x-titleX])
					if tch != ' ' {
						ch = tch
						fg = scaleColor(hexColor("#f5d563"), titleA)
					}
				}
				if y == subY && x >= subX && x < subX+len(subtitle) {
					sch := rune(subtitle[x-subX])
					if sch != ' ' {
						ch = sch
						fg = scaleColor(hexColor("#e8c870"), titleA*0.7)
					}
				}
			}

			// ---- Check for quote overlay ----
			if s.quoteReveal > 0 && y == quoteY {
				idx := x - quoteX
				rr := []rune(revealed)
				if idx >= 0 && idx < len(rr) {
					qch := rr[idx]
					if qch != ' ' {
						ch = qch
						fg = hexColor("#f5e6c8")
					}
				}
			}

			// ---- Apply layer + fade alpha ----
			alpha := layerAlpha * fade
			if alpha < 0.02 {
				sb.WriteRune(' ')
				continue
			}

			// Render using direct ANSI escape sequences for performance.
			writeAnsiCell(&sb, ch, fg, bg, alpha)
		}
		// Reset at end of each line.
		sb.WriteString("\033[0m")
		lines[y] = sb.String()
	}

	return strings.Join(lines, "\n")
}

// writeAnsiCell writes a single styled character using raw ANSI escapes.
// This avoids the overhead of lipgloss.NewStyle() per cell.
func writeAnsiCell(sb *strings.Builder, ch rune, fg, bg color.Color, alpha float64) {
	hasFG := fg != nil
	hasBG := bg != nil

	if !hasFG && !hasBG {
		sb.WriteRune(ch)
		return
	}

	if hasFG {
		r, g, b := colorRGB(fg, alpha)
		fmt.Fprintf(sb, "\033[38;2;%d;%d;%dm", r, g, b)
	}
	if hasBG {
		r, g, b := colorRGB(bg, alpha)
		fmt.Fprintf(sb, "\033[48;2;%d;%d;%dm", r, g, b)
	}
	sb.WriteRune(ch)
	sb.WriteString("\033[0m")
}

// colorRGB extracts scaled RGB from a color.Color.
func colorRGB(c color.Color, alpha float64) (uint8, uint8, uint8) {
	r, g, b, _ := c.RGBA()
	return uint8(float64(r>>8) * alpha),
		uint8(float64(g>>8) * alpha),
		uint8(float64(b>>8) * alpha)
}

func isSunChar(ch rune) bool {
	return ch == '█' || ch == '░' || ch == '▒' || ch == '▓'
}

func particleCell(p Particle, alpha float64) (rune, color.Color, bool) {
	switch p.Kind {
	case ParticleFirefly:
		if p.Bright < 0.15 {
			return 0, nil, false
		}
		r := []rune(p.Char)
		if len(r) > 0 {
			return r[0], scaleColor(hexColor(p.Color), p.Bright*alpha), true
		}
	case ParticleLeaf:
		r := []rune(p.Char)
		if len(r) > 0 {
			return r[0], scaleColor(hexColor(p.Color), alpha), true
		}
	case ParticleStar:
		if p.Bright < 0.2 {
			return 0, nil, false
		}
		r := []rune(p.Char)
		if len(r) > 0 {
			return r[0], scaleColor(hexColor(p.Color), p.Bright*alpha), true
		}
	}
	return 0, nil, false
}

// ---------------------------------------------------------------------------
// Grid construction – pre-computes the entire scene
// ---------------------------------------------------------------------------

func (s *ForestScene) buildGrid() {
	w, h := s.width, s.height
	s.grid = make([][]cell, h)
	for y := 0; y < h; y++ {
		s.grid[y] = make([]cell, w)
		for x := 0; x < w; x++ {
			s.grid[y][x] = cell{ch: ' '}
		}
	}

	// Render layers bottom-up so later layers overwrite.
	s.renderSky()
	s.renderClouds()
	s.renderSun()
	s.renderGround()
	s.renderPath()
	s.renderDistantPines()
	s.renderBushes()
	s.renderLeftTree()
	s.renderRightTree()
	s.renderWildflowers()
}

// ---------------------------------------------------------------------------
// Sky – warm sunset gradient using background colors
// ---------------------------------------------------------------------------

// skyGradient defines the vertical color stops for the sunset sky.
var skyGradient = []struct {
	t float64
	c color.RGBA
}{
	{0.00, color.RGBA{R: 45, G: 27, B: 78, A: 255}},    // deep purple
	{0.15, color.RGBA{R: 92, G: 45, B: 130, A: 255}},   // dark magenta
	{0.30, color.RGBA{R: 156, G: 62, B: 110, A: 255}},  // warm rose mid
	{0.50, color.RGBA{R: 194, G: 84, B: 125, A: 255}},  // warm rose
	{0.65, color.RGBA{R: 232, G: 132, B: 74, A: 255}},  // burnt orange
	{0.80, color.RGBA{R: 240, G: 184, B: 56, A: 255}},  // golden
	{0.92, color.RGBA{R: 245, G: 213, B: 99, A: 255}},  // bright warm yellow
	{1.00, color.RGBA{R: 248, G: 225, B: 140, A: 255}}, // horizon glow
}

func (s *ForestScene) renderSky() {
	w, h := s.width, s.height

	// Sky gradient covers the entire screen as a background layer.
	// Trees, ground, and other elements paint over it.
	for y := 0; y < h; y++ {
		t := float64(y) / float64(h)
		baseColor := sampleGradient(skyGradient, t)

		for x := 0; x < w; x++ {
			// Add subtle horizontal variation for atmosphere.
			xf := float64(x) / float64(w)
			variation := math.Sin(xf*math.Pi) * 8
			c := color.RGBA{
				R: clampU8(float64(baseColor.R) + variation),
				G: clampU8(float64(baseColor.G) + variation*0.5),
				B: clampU8(float64(baseColor.B) - variation*0.3),
				A: 255,
			}
			s.grid[y][x] = cell{ch: ' ', bg: c}
		}
	}
}

func sampleGradient(stops []struct {
	t float64
	c color.RGBA
}, t float64) color.RGBA {
	if t <= stops[0].t {
		return stops[0].c
	}
	if t >= stops[len(stops)-1].t {
		return stops[len(stops)-1].c
	}
	for i := 1; i < len(stops); i++ {
		if t <= stops[i].t {
			f := (t - stops[i-1].t) / (stops[i].t - stops[i-1].t)
			a := stops[i-1].c
			b := stops[i].c
			return color.RGBA{
				R: uint8(float64(a.R)*(1-f) + float64(b.R)*f),
				G: uint8(float64(a.G)*(1-f) + float64(b.G)*f),
				B: uint8(float64(a.B)*(1-f) + float64(b.B)*f),
				A: 255,
			}
		}
	}
	return stops[len(stops)-1].c
}

// ---------------------------------------------------------------------------
// Sun – bright glowing orb near top-center
// ---------------------------------------------------------------------------

func (s *ForestScene) renderSun() {
	w, h := s.width, s.height
	// Sun center at roughly 25% from top, centered horizontally.
	sunCX := w / 2
	sunCY := int(float64(h) * 0.15)
	if sunCY < 2 {
		sunCY = 2
	}

	// Glow ring (outer) – ░▒▓ in warm colors, radius ~5-7 cells.
	for dy := -5; dy <= 5; dy++ {
		for dx := -8; dx <= 8; dx++ {
			y := sunCY + dy
			x := sunCX + dx
			if y < 0 || y >= h || x < 0 || x >= w {
				continue
			}
			dist := math.Sqrt(float64(dx*dx)/4.0 + float64(dy*dy))
			if dist > 5.0 {
				continue
			}
			if dist > 3.5 {
				// Outer glow – dim warm orange.
				s.setFG(x, y, '░', color.RGBA{R: 245, G: 200, B: 80, A: 255})
			} else if dist > 2.5 {
				// Mid glow.
				s.setFG(x, y, '▒', color.RGBA{R: 250, G: 220, B: 100, A: 255})
			} else if dist > 1.5 {
				// Inner glow.
				s.setFG(x, y, '▓', color.RGBA{R: 255, G: 235, B: 140, A: 255})
			} else {
				// Sun core – bright white/yellow.
				s.setFG(x, y, '█', color.RGBA{R: 255, G: 250, B: 220, A: 255})
			}
		}
	}
}

// setFG sets the foreground character and color on an existing cell,
// preserving the background.
func (s *ForestScene) setFG(x, y int, ch rune, fg color.Color) {
	if y >= 0 && y < s.height && x >= 0 && x < s.width {
		s.grid[y][x].ch = ch
		s.grid[y][x].fg = fg
	}
}

// ---------------------------------------------------------------------------
// Clouds – wispy warm-toned clouds
// ---------------------------------------------------------------------------

func (s *ForestScene) renderClouds() {
	w, h := s.width, s.height
	if w < 30 || h < 10 {
		return
	}

	type cloudDef struct {
		cx, cy int
		shape  []string
	}

	clouds := []cloudDef{
		{
			cx: w / 5,
			cy: int(float64(h) * 0.08),
			shape: []string{
				"  ░░▒▒░  ",
				" ░▒▒▒▒▒░ ",
				"  ░░▒░░  ",
			},
		},
		{
			cx: w * 3 / 4,
			cy: int(float64(h) * 0.12),
			shape: []string{
				" ░▒▒░░ ",
				"░▒▒▒▒▒░",
				" ░░▒░  ",
			},
		},
	}

	if w > 80 {
		clouds = append(clouds, cloudDef{
			cx: w * 2 / 5,
			cy: int(float64(h) * 0.06),
			shape: []string{
				"  ░░▒░ ",
				" ░▒▒▒░ ",
				"  ░░   ",
			},
		})
	}

	cloudColors := []color.RGBA{
		{R: 220, G: 140, B: 160, A: 255}, // warm pink
		{R: 235, G: 170, B: 130, A: 255}, // warm peach
		{R: 200, G: 120, B: 150, A: 255}, // muted rose
	}

	for ci, cd := range clouds {
		cc := cloudColors[ci%len(cloudColors)]
		for dy, row := range cd.shape {
			runes := []rune(row)
			for dx, ch := range runes {
				if ch == ' ' {
					continue
				}
				x := cd.cx + dx - len(runes)/2
				y := cd.cy + dy
				if x >= 0 && x < w && y >= 0 && y < h {
					// Slightly vary color per character.
					brightness := 1.0
					if ch == '░' {
						brightness = 0.7
					} else if ch == '▒' {
						brightness = 0.85
					}
					fc := color.RGBA{
						R: clampU8(float64(cc.R) * brightness),
						G: clampU8(float64(cc.G) * brightness),
						B: clampU8(float64(cc.B) * brightness),
						A: 255,
					}
					s.setFG(x, y, ch, fc)
				}
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Left tree – large tree framing the left side
// ---------------------------------------------------------------------------

func (s *ForestScene) renderLeftTree() {
	w, h := s.width, s.height
	if w < 30 || h < 15 {
		return
	}

	// Pine tree: triangular silhouette with layered boughs.
	canopy := []string{
		"          ░          ",
		"         ░▓░         ",
		"        ░▓█▓░        ",
		"       ░▓███▓░       ",
		"      ░▓█████▓░      ",
		"     ░▓███████▓░     ",
		"       ░▓███▓░       ",
		"      ░▓█████▓░      ",
		"     ░▓███████▓░     ",
		"    ░▓█████████▓░    ",
		"   ░▓███████████▓░   ",
		"      ░▓█████▓░      ",
		"     ░▓███████▓░     ",
		"    ░▓█████████▓░    ",
		"   ░▓███████████▓░   ",
		"  ░▓█████████████▓░  ",
		" ░▓███████████████▓░ ",
	}
	canopyW := 21

	treeX := 2
	canopyBottom := int(float64(h) * 0.58)
	canopyTop := canopyBottom - len(canopy)
	if canopyTop < 0 {
		canopyTop = 0
	}

	darkGreen := color.RGBA{R: 15, G: 55, B: 35, A: 255}
	midGreen := color.RGBA{R: 30, G: 85, B: 50, A: 255}
	lightGreen := color.RGBA{R: 50, G: 120, B: 65, A: 255}
	edgeGreen := color.RGBA{R: 70, G: 150, B: 85, A: 255}

	if s.theme != nil && s.theme.Forest.LeafColor1 != "" {
		tc := colorToRGBA(hexColor(s.theme.Forest.LeafColor1))
		midGreen = tc
		darkGreen = darken(tc, 0.5)
		lightGreen = lighten(tc, 0.3)
		edgeGreen = lighten(tc, 0.5)
	}

	s.renderCanopy(canopy, canopyW, treeX, canopyTop, darkGreen, midGreen, lightGreen, edgeGreen)

	trunkColor := color.RGBA{R: 74, G: 55, B: 40, A: 255}
	trunkHighlight := color.RGBA{R: 107, G: 82, B: 64, A: 255}
	if s.theme != nil && s.theme.Forest.TrunkColor != "" {
		trunkColor = colorToRGBA(hexColor(s.theme.Forest.TrunkColor))
		trunkHighlight = lighten(trunkColor, 0.3)
	}
	trunkX := treeX + canopyW/2
	trunkTop := canopyBottom
	trunkBottom := h - 4
	for y := trunkTop; y <= trunkBottom && y < h; y++ {
		if trunkX >= 0 && trunkX < w {
			s.grid[y][trunkX] = cell{ch: '║', fg: trunkColor, bg: s.grid[y][trunkX].bg}
		}
		if trunkX+1 >= 0 && trunkX+1 < w {
			s.grid[y][trunkX+1] = cell{ch: '║', fg: trunkHighlight, bg: s.grid[y][trunkX+1].bg}
		}
		if y > trunkBottom-2 {
			if trunkX-1 >= 0 && trunkX-1 < w {
				s.grid[y][trunkX-1] = cell{ch: '▓', fg: trunkColor, bg: s.grid[y][trunkX-1].bg}
			}
			if trunkX+2 >= 0 && trunkX+2 < w {
				s.grid[y][trunkX+2] = cell{ch: '▓', fg: trunkHighlight, bg: s.grid[y][trunkX+2].bg}
			}
		}
	}
}

// renderCanopy paints a canopy template onto the grid using density-based coloring.
func (s *ForestScene) renderCanopy(canopy []string, canopyW, treeX, canopyTop int, dark, mid, light, edge color.RGBA) {
	w, h := s.width, s.height
	for dy, row := range canopy {
		y := canopyTop + dy
		if y < 0 || y >= h {
			continue
		}
		runes := []rune(row)
		for dx := 0; dx < len(runes) && dx < canopyW; dx++ {
			ch := runes[dx]
			if ch == ' ' {
				continue
			}
			x := treeX + dx
			if x < 0 || x >= w {
				continue
			}
			var fc color.Color
			switch ch {
			case '█':
				fc = dark
			case '▓':
				fc = mid
			case '▒':
				fc = light
			case '░':
				fc = edge
			default:
				fc = mid
			}
			s.grid[y][x] = cell{ch: ch, fg: fc, bg: s.grid[y][x].bg}
		}
	}
}

// ---------------------------------------------------------------------------
// Right tree – large tree framing the right side
// ---------------------------------------------------------------------------

func (s *ForestScene) renderRightTree() {
	w, h := s.width, s.height
	if w < 30 || h < 15 {
		return
	}

	// Taller pine tree for the right side with slightly different proportions.
	canopy := []string{
		"          ░          ",
		"         ░▓░         ",
		"        ░▓█▓░        ",
		"       ░▓███▓░       ",
		"        ░▓█▓░        ",
		"       ░▓███▓░       ",
		"      ░▓█████▓░      ",
		"     ░▓███████▓░     ",
		"       ░▓███▓░       ",
		"      ░▓█████▓░      ",
		"     ░▓███████▓░     ",
		"    ░▓█████████▓░    ",
		"   ░▓███████████▓░   ",
		"     ░▓███████▓░     ",
		"    ░▓█████████▓░    ",
		"   ░▓███████████▓░   ",
		"  ░▓█████████████▓░  ",
		" ░▓███████████████▓░ ",
		"░▓█████████████████▓░",
	}
	canopyW := 21

	treeX := w - canopyW - 2
	if treeX < 0 {
		treeX = 0
	}
	canopyBottom := int(float64(h) * 0.58)
	canopyTop := canopyBottom - len(canopy)
	if canopyTop < 0 {
		canopyTop = 0
	}

	darkGreen := color.RGBA{R: 12, G: 48, B: 30, A: 255}
	midGreen := color.RGBA{R: 25, G: 78, B: 45, A: 255}
	lightGreen := color.RGBA{R: 45, G: 110, B: 60, A: 255}
	edgeGreen := color.RGBA{R: 65, G: 140, B: 78, A: 255}

	if s.theme != nil && s.theme.Forest.LeafColor1 != "" {
		tc := colorToRGBA(hexColor(s.theme.Forest.LeafColor1))
		midGreen = darken(tc, 0.15)
		darkGreen = darken(tc, 0.55)
		lightGreen = lighten(tc, 0.2)
		edgeGreen = lighten(tc, 0.45)
	}

	s.renderCanopy(canopy, canopyW, treeX, canopyTop, darkGreen, midGreen, lightGreen, edgeGreen)

	trunkColor := color.RGBA{R: 68, G: 50, B: 36, A: 255}
	trunkHighlight := color.RGBA{R: 100, G: 76, B: 58, A: 255}
	if s.theme != nil && s.theme.Forest.TrunkColor != "" {
		trunkColor = colorToRGBA(hexColor(s.theme.Forest.TrunkColor))
		trunkHighlight = lighten(trunkColor, 0.3)
	}
	trunkX := treeX + canopyW/2
	trunkTop := canopyBottom
	trunkBottom := h - 4
	for y := trunkTop; y <= trunkBottom && y < h; y++ {
		if trunkX >= 0 && trunkX < w {
			s.grid[y][trunkX] = cell{ch: '║', fg: trunkColor, bg: s.grid[y][trunkX].bg}
		}
		if trunkX+1 >= 0 && trunkX+1 < w {
			s.grid[y][trunkX+1] = cell{ch: '║', fg: trunkHighlight, bg: s.grid[y][trunkX+1].bg}
		}
		if y > trunkBottom-2 {
			if trunkX-1 >= 0 && trunkX-1 < w {
				s.grid[y][trunkX-1] = cell{ch: '▓', fg: trunkColor, bg: s.grid[y][trunkX-1].bg}
			}
			if trunkX+2 >= 0 && trunkX+2 < w {
				s.grid[y][trunkX+2] = cell{ch: '▓', fg: trunkHighlight, bg: s.grid[y][trunkX+2].bg}
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Distant pines – small silhouette pine trees in the background midfield
// ---------------------------------------------------------------------------

func (s *ForestScene) renderDistantPines() {
	w, h := s.width, s.height
	if w < 50 || h < 15 {
		return
	}

	smallPine := []string{
		"  ░  ",
		" ░▓░ ",
		"░▓█▓░",
		" ░▓░ ",
		"░▓█▓░",
		"▓███▓",
		"  ║  ",
		"  ║  ",
	}
	pineW := 5

	// Muted dark green for distant trees (further away = darker / less saturated).
	dc := color.RGBA{R: 18, G: 45, B: 28, A: 255}
	mc := color.RGBA{R: 28, G: 65, B: 40, A: 255}
	lc := color.RGBA{R: 40, G: 85, B: 52, A: 255}
	ec := color.RGBA{R: 50, G: 100, B: 60, A: 255}

	groundLine := h - 5
	positions := []int{w/3 - 3, w/3 + 5, w*2/3 - 4, w*2/3 + 4}
	if w > 100 {
		positions = append(positions, w/2-8, w/2+8)
	}

	for _, px := range positions {
		top := groundLine - len(smallPine) + 2
		if top < 0 {
			top = 0
		}
		s.renderCanopy(smallPine, pineW, px, top, dc, mc, lc, ec)
	}
}

// ---------------------------------------------------------------------------
// Bushes / undergrowth – smaller vegetation along the bottom
// ---------------------------------------------------------------------------

func (s *ForestScene) renderBushes() {
	w, h := s.width, s.height
	if w < 20 || h < 10 {
		return
	}

	bush := []string{
		" ░▓▓░ ",
		"▒████▒",
		"▓████▓",
		" ▒▓▓▒ ",
	}

	bushColors := []color.RGBA{
		{R: 30, G: 85, B: 45, A: 255},
		{R: 42, G: 110, B: 58, A: 255},
		{R: 25, G: 75, B: 40, A: 255},
	}

	// Place bushes at strategic positions along the ground.
	groundY := h - 5
	positions := []int{
		w/4 - 3,
		w/2 - 4,
		w*3/4 + 1,
	}
	if w > 80 {
		positions = append(positions, w/6, w*5/6-3)
	}

	for bi, bx := range positions {
		bc := bushColors[bi%len(bushColors)]
		for dy, row := range bush {
			runes := []rune(row)
			y := groundY - len(bush) + dy + 1
			if y < 0 || y >= h {
				continue
			}
			for dx, ch := range runes {
				if ch == ' ' {
					continue
				}
				x := bx + dx
				if x < 0 || x >= w {
					continue
				}
				brightness := 0.8
				switch ch {
				case '█':
					brightness = 0.7
				case '▓':
					brightness = 0.8
				case '▒':
					brightness = 0.9
				case '░':
					brightness = 1.0
				}
				fc := color.RGBA{
					R: clampU8(float64(bc.R) * brightness),
					G: clampU8(float64(bc.G) * brightness),
					B: clampU8(float64(bc.B) * brightness),
					A: 255,
				}
				s.grid[y][x] = cell{ch: ch, fg: fc, bg: s.grid[y][x].bg}
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Ground / meadow – dense grass with earth-tone gradient
// ---------------------------------------------------------------------------

func (s *ForestScene) renderGround() {
	w, h := s.width, s.height
	groundRows := 5
	groundStart := h - groundRows
	if groundStart < 0 {
		groundStart = 0
	}

	// Ground color gradient: top rows are lighter/greener, bottom is darker.
	groundColors := []color.RGBA{
		{R: 55, G: 120, B: 50, A: 255}, // lush green
		{R: 45, G: 105, B: 42, A: 255}, // mid green
		{R: 38, G: 90, B: 36, A: 255},  // darker green
		{R: 30, G: 75, B: 30, A: 255},  // deep green
		{R: 25, G: 60, B: 25, A: 255},  // darkest
	}

	grassChars := []rune(".:;·,'.;:·.,':;")
	denseChars := []rune("▓▒░.:·")

	for dy := 0; dy < groundRows; dy++ {
		y := groundStart + dy
		if y >= h {
			break
		}
		gc := groundColors[dy%len(groundColors)]
		for x := 0; x < w; x++ {
			var ch rune
			if dy == 0 {
				// Top of ground: grass blades.
				ch = grassChars[(x*7+dy*3)%len(grassChars)]
			} else {
				ch = denseChars[(x*5+dy*7)%len(denseChars)]
			}
			// Add variation.
			variation := math.Sin(float64(x)*0.3+float64(dy)*1.2) * 12
			fc := color.RGBA{
				R: clampU8(float64(gc.R) + variation*0.3),
				G: clampU8(float64(gc.G) + variation),
				B: clampU8(float64(gc.B) + variation*0.2),
				A: 255,
			}
			// Earthy background for ground.
			bgShade := 0.3 + float64(dy)*0.05
			bgc := color.RGBA{
				R: clampU8(float64(gc.R) * bgShade),
				G: clampU8(float64(gc.G) * bgShade),
				B: clampU8(float64(gc.B) * bgShade),
				A: 255,
			}
			s.grid[y][x] = cell{ch: ch, fg: fc, bg: bgc}
		}
	}
}

// ---------------------------------------------------------------------------
// Path – winding trail through the meadow
// ---------------------------------------------------------------------------

func (s *ForestScene) renderPath() {
	w, h := s.width, s.height
	groundStart := h - 5
	if groundStart < 0 {
		return
	}

	pathChars := []rune("·:▪·.:·")
	pathColor := color.RGBA{R: 196, G: 166, B: 106, A: 255} // sandy
	pathColorD := color.RGBA{R: 184, G: 155, B: 94, A: 255} // darker sand

	centerX := w / 2
	for dy := 0; dy < 5; dy++ {
		y := groundStart + dy
		if y >= h {
			break
		}
		// Path winds slightly.
		offset := int(math.Sin(float64(dy)*0.8) * 2)
		pathW := 4 + dy // widens toward bottom
		for px := -pathW / 2; px <= pathW/2; px++ {
			x := centerX + offset + px
			if x < 0 || x >= w {
				continue
			}
			ch := pathChars[(x+dy*3)%len(pathChars)]
			fc := pathColor
			if abs(px) == pathW/2 {
				fc = pathColorD // edges are darker
			}
			s.grid[y][x] = cell{ch: ch, fg: fc, bg: darken(fc, 0.7)}
		}
	}
}

// ---------------------------------------------------------------------------
// Wildflowers – scattered colorful dots in the meadow
// ---------------------------------------------------------------------------

func (s *ForestScene) renderWildflowers() {
	w, h := s.width, s.height
	groundStart := h - 5
	if groundStart < 1 || w < 10 {
		return
	}

	flowerChars := []rune("✿❀*·")
	flowerColors := []color.RGBA{
		{R: 240, G: 140, B: 180, A: 255}, // pink
		{R: 255, G: 220, B: 80, A: 255},  // yellow
		{R: 190, G: 130, B: 220, A: 255}, // purple
		{R: 255, G: 180, B: 100, A: 255}, // orange
		{R: 200, G: 100, B: 150, A: 255}, // magenta
	}

	// Use deterministic placement so it's consistent.
	rng := rand.New(rand.NewSource(42))
	numFlowers := w / 4
	if numFlowers > 30 {
		numFlowers = 30
	}

	for i := 0; i < numFlowers; i++ {
		x := rng.Intn(w)
		dy := rng.Intn(3)
		y := groundStart + dy
		if y >= h || y < 0 {
			continue
		}
		// Don't place on path center.
		if abs(x-w/2) < 4 {
			continue
		}
		ch := flowerChars[rng.Intn(len(flowerChars))]
		fc := flowerColors[rng.Intn(len(flowerColors))]
		s.grid[y][x] = cell{ch: ch, fg: fc, bg: s.grid[y][x].bg}
	}
}

// ---------------------------------------------------------------------------
// Particles – flower petals drifting
// ---------------------------------------------------------------------------

func (s *ForestScene) spawnPetal() {
	petalChars := []string{"✿", "❀", "*", "·", "°"}
	if s.theme != nil && len(s.theme.Forest.LeafChars) > 0 {
		petalChars = s.theme.Forest.LeafChars
	}
	petalColors := []string{"#f08cb4", "#ffdc50", "#be82dc", "#ffb464"}
	if s.theme != nil {
		if s.theme.Forest.LeafColor1 != "" {
			petalColors = []string{s.theme.Forest.LeafColor1}
		}
		if s.theme.Forest.LeafColor2 != "" {
			petalColors = append(petalColors, s.theme.Forest.LeafColor2)
		}
	}
	x := s.width/6 + rand.Intn(2*s.width/3)
	y := s.height/4 + rand.Intn(s.height/4)
	s.particles.SpawnLeaf(x, y, petalChars, petalColors)
}

func (s *ForestScene) fireflyColor() string {
	if s.theme != nil && s.theme.Forest.FireflyColor != "" {
		return s.theme.Forest.FireflyColor
	}
	return "#f0e68c"
}

// ---------------------------------------------------------------------------
// Color utilities
// ---------------------------------------------------------------------------

func hexColor(hex string) color.Color {
	if len(hex) == 0 {
		return color.RGBA{A: 255}
	}
	return lipgloss.Color(hex)
}

func scaleColor(c color.Color, alpha float64) color.Color {
	if alpha >= 1.0 {
		return c
	}
	if alpha <= 0.0 {
		return color.RGBA{A: 255}
	}
	r, g, b, _ := c.RGBA()
	return color.RGBA{
		R: uint8(float64(r>>8) * alpha),
		G: uint8(float64(g>>8) * alpha),
		B: uint8(float64(b>>8) * alpha),
		A: 255,
	}
}

func lerpColor(a, b color.Color, t float64) color.Color {
	ar, ag, ab, _ := a.RGBA()
	br, bg, bb, _ := b.RGBA()
	return color.RGBA{
		R: uint8(float64(ar>>8)*(1-t) + float64(br>>8)*t),
		G: uint8(float64(ag>>8)*(1-t) + float64(bg>>8)*t),
		B: uint8(float64(ab>>8)*(1-t) + float64(bb>>8)*t),
		A: 255,
	}
}

func clampF(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func clampU8(v float64) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}

func colorToRGBA(c color.Color) color.RGBA {
	r, g, b, a := c.RGBA()
	return color.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
}

func darken(c color.RGBA, amount float64) color.RGBA {
	factor := 1.0 - amount
	return color.RGBA{
		R: clampU8(float64(c.R) * factor),
		G: clampU8(float64(c.G) * factor),
		B: clampU8(float64(c.B) * factor),
		A: 255,
	}
}

func lighten(c color.RGBA, amount float64) color.RGBA {
	return color.RGBA{
		R: clampU8(float64(c.R) + (255-float64(c.R))*amount),
		G: clampU8(float64(c.G) + (255-float64(c.G))*amount),
		B: clampU8(float64(c.B) + (255-float64(c.B))*amount),
		A: 255,
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
