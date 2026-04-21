package animation

import (
	"strings"
)

// TransitionState represents the current phase of a page transition.
type TransitionState int

const (
	// TransitionNone means no transition is active.
	TransitionNone TransitionState = iota
	// TransitionOut means the outgoing page is dimming/fading.
	TransitionOut
	// TransitionIn means the incoming page is being revealed.
	TransitionIn
)

const (
	// transitionFrames is the total number of frames for the cross-fade (8 frames at 30fps ≈ 267ms).
	transitionFrames = 8
)

// TransitionModel handles cross-fade transitions between two page views.
// It blends the outgoing and incoming view strings character by character
// over a fixed number of frames.
type TransitionModel struct {
	state   TransitionState
	frame   int
	outView string // snapshot of the page being left
	inView  string // snapshot of the page being entered
}

// NewTransition creates a new transition with the given outgoing view snapshot.
// The transition starts in TransitionOut state.
func NewTransition(outView string) *TransitionModel {
	return &TransitionModel{
		state:   TransitionOut,
		frame:   0,
		outView: outView,
	}
}

// SetIncoming sets the incoming view and switches to TransitionIn state.
// This should be called once the incoming page has rendered its first frame.
func (t *TransitionModel) SetIncoming(view string) {
	t.inView = view
	t.state = TransitionIn
	t.frame = 0
}

// Tick advances the transition by one frame. Returns true when the transition
// is complete and should be removed.
func (t *TransitionModel) Tick() bool {
	t.frame++
	switch t.state {
	case TransitionOut:
		if t.frame >= transitionFrames {
			// If no incoming view has been set, just finish.
			if t.inView == "" {
				t.state = TransitionNone
				return true
			}
			// Auto-switch to TransitionIn if incoming view is available.
			t.state = TransitionIn
			t.frame = 0
		}
	case TransitionIn:
		if t.frame >= transitionFrames {
			t.state = TransitionNone
			return true
		}
	case TransitionNone:
		return true
	}
	return false
}

// View renders the current transition frame by blending outView and inView.
// During TransitionOut, the outgoing view progressively fades (characters replaced with spaces).
// During TransitionIn, the incoming view progressively reveals (spaces replaced with content).
func (t *TransitionModel) View() string {
	switch t.state {
	case TransitionOut:
		return t.fadeOut()
	case TransitionIn:
		return t.fadeIn()
	case TransitionNone:
		if t.inView != "" {
			return t.inView
		}
		return t.outView
	}
	return ""
}

// Active reports whether a transition is currently running.
func (t *TransitionModel) Active() bool {
	return t.state != TransitionNone
}

// State returns the current transition state.
func (t *TransitionModel) State() TransitionState {
	return t.state
}

// Frame returns the current frame number within the active phase.
func (t *TransitionModel) Frame() int {
	return t.frame
}

// fadeOut progressively replaces characters in the outgoing view with spaces.
// At frame 0, the full view is shown. At frame transitionFrames, it's all spaces.
func (t *TransitionModel) fadeOut() string {
	if t.outView == "" {
		return ""
	}
	progress := float64(t.frame) / float64(transitionFrames)
	return blendToBlank(t.outView, progress)
}

// fadeIn progressively reveals characters of the incoming view from spaces.
// At frame 0, it's all spaces. At frame transitionFrames, the full view is shown.
func (t *TransitionModel) fadeIn() string {
	if t.inView == "" {
		return ""
	}
	progress := float64(t.frame) / float64(transitionFrames)
	return blendFromBlank(t.inView, progress)
}

// blendToBlank replaces a fraction of visible (non-newline) characters with spaces.
// progress ranges from 0.0 (all visible) to 1.0 (all blank).
func blendToBlank(view string, progress float64) string {
	lines := strings.Split(view, "\n")
	result := make([]string, len(lines))
	for i, line := range lines {
		runes := []rune(line)
		total := len(runes)
		blanked := int(float64(total) * progress)
		out := make([]rune, total)
		for j, r := range runes {
			// Blank characters from the end of each line moving left.
			if j >= total-blanked {
				out[j] = ' '
			} else {
				out[j] = r
			}
		}
		result[i] = string(out)
	}
	return strings.Join(result, "\n")
}

// blendFromBlank reveals characters of a view from spaces.
// progress ranges from 0.0 (all blank) to 1.0 (all visible).
func blendFromBlank(view string, progress float64) string {
	lines := strings.Split(view, "\n")
	result := make([]string, len(lines))
	for i, line := range lines {
		runes := []rune(line)
		total := len(runes)
		revealed := int(float64(total) * progress)
		out := make([]rune, total)
		for j, r := range runes {
			// Reveal characters from the start of each line moving right.
			if j < revealed {
				out[j] = r
			} else {
				out[j] = ' '
			}
		}
		result[i] = string(out)
	}
	return strings.Join(result, "\n")
}
