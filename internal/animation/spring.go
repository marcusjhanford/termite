package animation

import (
	"math"
	"time"

	"github.com/charmbracelet/harmonica"
)

// Spring presets for different animation contexts.
var (
	// FocusSpring is snappy, used for focus ring movement.
	FocusSpring = SpringPreset{Frequency: 8.0, Damping: 1.0}
	// ResizeSpring is elastic, used for pane width animation.
	ResizeSpring = SpringPreset{Frequency: 5.0, Damping: 0.7}
	// ScrollSpring is fluid, used for scroll position easing.
	ScrollSpring = SpringPreset{Frequency: 4.0, Damping: 0.9}
)

// SpringPreset holds parameters for constructing a PaneSpring.
type SpringPreset struct {
	Frequency float64
	Damping   float64
}

// PaneSpring drives smooth value animation using a critically-damped spring.
type PaneSpring struct {
	spring   harmonica.Spring
	pos      float64 // current rendered value (fractional)
	target   float64 // target value
	velocity float64
}

// NewPaneSpring creates a new PaneSpring with the given frequency and damping ratio.
// frequency controls the speed of oscillation, damping controls how quickly it settles
// (1.0 = critically damped, <1.0 = underdamped/bouncy, >1.0 = overdamped/sluggish).
func NewPaneSpring(frequency, damping float64) PaneSpring {
	return PaneSpring{
		spring: harmonica.NewSpring(harmonica.FPS(60), frequency, damping),
	}
}

// NewPaneSpringFromPreset creates a PaneSpring from a preset configuration.
func NewPaneSpringFromPreset(preset SpringPreset) PaneSpring {
	return NewPaneSpring(preset.Frequency, preset.Damping)
}

// SetTarget sets the value the spring is animating toward.
func (s *PaneSpring) SetTarget(target float64) {
	s.target = target
}

// SetPosition instantly moves the spring to a position without animation.
func (s *PaneSpring) SetPosition(pos float64) {
	s.pos = pos
	s.velocity = 0
}

// Tick advances the spring simulation by the given time delta.
// The harmonica spring update uses its own internal timestep, so delta
// is used to determine how many substeps to run.
func (s *PaneSpring) Tick(delta time.Duration) {
	// harmonica.Spring.Update takes current position, velocity, and target.
	// It returns the new position and velocity.
	// We call it once per tick; harmonica handles the timestep internally.
	_ = delta // harmonica uses its internal FPS-based timestep
	s.pos, s.velocity = s.spring.Update(s.pos, s.velocity, s.target)
}

// Value returns the current spring position as a float64.
func (s *PaneSpring) Value() float64 {
	return s.pos
}

// Width returns the current spring position rounded to the nearest integer,
// useful for column/width calculations.
func (s *PaneSpring) Width() int {
	return int(math.Round(s.pos))
}

// Done reports whether the spring has settled at its target (within a small
// epsilon) and has negligible velocity.
func (s *PaneSpring) Done() bool {
	const epsilon = 0.5
	const velEpsilon = 0.01
	return math.Abs(s.pos-s.target) < epsilon && math.Abs(s.velocity) < velEpsilon
}

// Target returns the current target value.
func (s *PaneSpring) Target() float64 {
	return s.target
}

// Velocity returns the current velocity.
func (s *PaneSpring) Velocity() float64 {
	return s.velocity
}
