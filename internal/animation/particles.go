package animation

import (
	"math"
	"math/rand"
	"time"
)

// ParticleKind distinguishes between different particle types.
type ParticleKind int

const (
	// ParticleLeaf represents a drifting leaf falling from the tree canopy.
	ParticleLeaf ParticleKind = iota
	// ParticleFirefly represents a spring-driven glowing dot that wanders the scene.
	ParticleFirefly
	// ParticleStar represents a fixed twinkling star in the sky.
	ParticleStar
)

// Particle represents a single animated element in the scene.
type Particle struct {
	Kind    ParticleKind
	X, Y    float64 // sub-cell position for smooth drift
	VX, VY  float64 // velocity in cells/tick (used by leaf)
	Char    string
	Color   string
	Life    int     // ticks remaining (-1 = infinite)
	BlinkOn bool    // legacy compat
	Bright  float64 // 0.0-1.0 brightness for rendering
}

// fireflyState holds spring-driven position and brightness for a firefly.
type fireflyState struct {
	springX    PaneSpring
	springY    PaneSpring
	springGlow PaneSpring
	char       string
	color      string
	retargetIn int // ticks until next random target
}

// starState holds spring-driven brightness for a twinkling star.
type starState struct {
	x, y         int
	springGlow   PaneSpring
	char         string
	color        string
	retweenIn    int // ticks until next twinkle toggle
	targetBright float64
}

// ParticleSystem manages a collection of particles within a bounded area.
// It supports three particle types with different physics models.
type ParticleSystem struct {
	leaves    []Particle
	fireflies []fireflyState
	stars     []starState
	width     int
	height    int
	tickCount int
}

// NewParticleSystem creates a particle system bounded to the given dimensions.
func NewParticleSystem(width, height int) *ParticleSystem {
	return &ParticleSystem{
		leaves:    make([]Particle, 0, 64),
		fireflies: make([]fireflyState, 0, 8),
		stars:     make([]starState, 0, 32),
		width:     width,
		height:    height,
	}
}

// SpawnLeaf creates a new leaf particle at the given position.
func (ps *ParticleSystem) SpawnLeaf(x, y int, chars []string, colors []string) {
	if len(chars) == 0 || len(colors) == 0 {
		return
	}
	dx := (rand.Float64() - 0.5) * 0.6
	dy := 0.2 + rand.Float64()*0.3

	p := Particle{
		Kind:    ParticleLeaf,
		X:       float64(x),
		Y:       float64(y),
		VX:      dx,
		VY:      dy,
		Char:    chars[rand.Intn(len(chars))],
		Color:   colors[rand.Intn(len(colors))],
		Life:    60 + rand.Intn(40),
		BlinkOn: true,
		Bright:  1.0,
	}
	ps.leaves = append(ps.leaves, p)
}

// SpawnFirefly creates a spring-driven firefly that wanders the scene.
// The firefly uses harmonica springs for smooth, organic movement.
func (ps *ParticleSystem) SpawnFirefly(color string) {
	chars := []string{"·", "•", "✦"}
	startX := float64(ps.width/4 + rand.Intn(ps.width/2))
	startY := float64(ps.height/3 + rand.Intn(ps.height/3))

	sx := NewPaneSpring(2.0, 0.8)
	sx.SetPosition(startX)
	sx.SetTarget(startX)

	sy := NewPaneSpring(2.0, 0.8)
	sy.SetPosition(startY)
	sy.SetTarget(startY)

	sg := NewPaneSpring(3.0, 0.9)
	sg.SetPosition(0.0)
	sg.SetTarget(1.0)

	ff := fireflyState{
		springX:    sx,
		springY:    sy,
		springGlow: sg,
		char:       chars[rand.Intn(len(chars))],
		color:      color,
		retargetIn: 30 + rand.Intn(60),
	}
	ps.fireflies = append(ps.fireflies, ff)
}

// SpawnStar creates a twinkling star at the given fixed position.
func (ps *ParticleSystem) SpawnStar(x, y int, char, color string) {
	sg := NewPaneSpring(4.0, 0.9)
	initialBright := rand.Float64()
	sg.SetPosition(initialBright)
	sg.SetTarget(initialBright)

	s := starState{
		x:            x,
		y:            y,
		springGlow:   sg,
		char:         char,
		color:        color,
		retweenIn:    20 + rand.Intn(80),
		targetBright: initialBright,
	}
	ps.stars = append(ps.stars, s)
}

// dt is the per-tick time delta for spring updates.
var dt = time.Second / 60

// Tick advances all particles by one simulation step.
func (ps *ParticleSystem) Tick() {
	ps.tickCount++
	ps.tickLeaves()
	ps.tickFireflies()
	ps.tickStars()
}

func (ps *ParticleSystem) tickLeaves() {
	alive := ps.leaves[:0]
	for i := range ps.leaves {
		p := &ps.leaves[i]
		p.X += p.VX
		p.Y += p.VY
		// Spring-like wobble: sinusoidal lateral drift.
		p.VX += (rand.Float64() - 0.5) * 0.05
		p.Life--
		if p.Life <= 0 || p.Y >= float64(ps.height-1) || p.X < 0 || p.X >= float64(ps.width) {
			continue
		}
		alive = append(alive, *p)
	}
	ps.leaves = alive
}

func (ps *ParticleSystem) tickFireflies() {
	for i := range ps.fireflies {
		ff := &ps.fireflies[i]

		// Update springs.
		ff.springX.Tick(dt)
		ff.springY.Tick(dt)
		ff.springGlow.Tick(dt)

		// Retarget periodically for wandering behavior.
		ff.retargetIn--
		if ff.retargetIn <= 0 {
			// Pick a new random position within bounds.
			newX := float64(ps.width/6) + rand.Float64()*float64(ps.width*2/3)
			newY := float64(ps.height/4) + rand.Float64()*float64(ps.height/2)
			ff.springX.SetTarget(newX)
			ff.springY.SetTarget(newY)

			// Toggle glow target for pulsing effect.
			if ff.springGlow.Target() > 0.5 {
				ff.springGlow.SetTarget(0.1 + rand.Float64()*0.3)
			} else {
				ff.springGlow.SetTarget(0.7 + rand.Float64()*0.3)
			}

			ff.retargetIn = 40 + rand.Intn(50)
		}
	}
}

func (ps *ParticleSystem) tickStars() {
	for i := range ps.stars {
		s := &ps.stars[i]
		s.springGlow.Tick(dt)

		s.retweenIn--
		if s.retweenIn <= 0 {
			// Toggle between bright and dim.
			if s.targetBright > 0.5 {
				s.targetBright = rand.Float64() * 0.3
			} else {
				s.targetBright = 0.6 + rand.Float64()*0.4
			}
			s.springGlow.SetTarget(s.targetBright)
			s.retweenIn = 30 + rand.Intn(100)
		}
	}
}

// Particles returns a unified slice of all active particles for rendering.
// This maintains API compatibility with the existing ForestScene.
func (ps *ParticleSystem) Particles() []Particle {
	total := len(ps.leaves) + len(ps.fireflies) + len(ps.stars)
	result := make([]Particle, 0, total)

	result = append(result, ps.leaves...)

	for _, ff := range ps.fireflies {
		glow := ff.springGlow.Value()
		if glow < 0 {
			glow = 0
		}
		if glow > 1 {
			glow = 1
		}
		result = append(result, Particle{
			Kind:    ParticleFirefly,
			X:       ff.springX.Value(),
			Y:       ff.springY.Value(),
			Char:    ff.char,
			Color:   ff.color,
			Life:    -1,
			BlinkOn: glow > 0.2,
			Bright:  glow,
		})
	}

	for _, s := range ps.stars {
		glow := s.springGlow.Value()
		if glow < 0 {
			glow = 0
		}
		if glow > 1 {
			glow = 1
		}
		result = append(result, Particle{
			Kind:    ParticleStar,
			X:       float64(s.x),
			Y:       float64(s.y),
			Char:    s.char,
			Color:   s.color,
			Life:    -1,
			BlinkOn: glow > 0.15,
			Bright:  glow,
		})
	}

	return result
}

// FireflyCount returns the number of active fireflies.
func (ps *ParticleSystem) FireflyCount() int {
	return len(ps.fireflies)
}

// StarCount returns the number of active stars.
func (ps *ParticleSystem) StarCount() int {
	return len(ps.stars)
}

// Count returns the total number of active particles.
func (ps *ParticleSystem) Count() int {
	return len(ps.leaves) + len(ps.fireflies) + len(ps.stars)
}

// Clear removes all particles.
func (ps *ParticleSystem) Clear() {
	ps.leaves = ps.leaves[:0]
	ps.fireflies = ps.fireflies[:0]
	ps.stars = ps.stars[:0]
}

// clampBright clamps a brightness value to [0, 1].
func clampBright(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// BrightnessToAlpha maps a 0-1 brightness to a rough character density ramp.
// Returns an index 0-7 into a density ramp string like " .:-=+*#".
func BrightnessToAlpha(b float64) int {
	b = clampBright(b)
	idx := int(math.Round(b * 7))
	if idx > 7 {
		idx = 7
	}
	return idx
}
