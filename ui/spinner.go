package ui

import (
	"math"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
)

const (
	spinnerDots        = 8  // number of dots in the ring
	spinnerDefaultSize = 24 // default diameter in logical pixels
)

// Spinner is an indeterminate busy/activity indicator: a ring of dots with a
// bright "head" that rotates, the trailing dots fading out. It conveys that work
// is in progress without a known duration (use ProgressBar for determinate
// progress).
//
// It animates by advancing on each Draw (the framework redraws every frame), so
// it needs no timer wiring. Start/Stop control the animation; a stopped spinner
// freezes in place — hide it with SetVisible(false) when there is nothing to
// show.
type Spinner struct {
	BaseWidget
	size     float64
	speed    float64 // revolutions per second
	phase    float64 // accumulated revolutions
	spinning bool
}

// SpinnerOption configures a Spinner.
type SpinnerOption func(*Spinner)

// SpinnerSize sets the spinner's diameter in logical pixels.
func SpinnerSize(d float64) SpinnerOption {
	return func(s *Spinner) {
		if d > 0 {
			s.size = d
		}
	}
}

// SpinnerSpeed sets the rotation speed in revolutions per second.
func SpinnerSpeed(revsPerSec float64) SpinnerOption {
	return func(s *Spinner) {
		if revsPerSec > 0 {
			s.speed = revsPerSec
		}
	}
}

// NewSpinner returns a Spinner that starts spinning immediately.
func NewSpinner(opts ...SpinnerOption) *Spinner {
	s := &Spinner{BaseWidget: NewBase(), size: spinnerDefaultSize, speed: 1, spinning: true}
	for _, o := range opts {
		o(s)
	}
	return s
}

// Start resumes the animation.
func (s *Spinner) Start() { s.spinning = true }

// Stop freezes the animation in place.
func (s *Spinner) Stop() { s.spinning = false }

// Spinning reports whether the spinner is animating.
func (s *Spinner) Spinning() bool { return s.spinning }

// MinSize returns the spinner's square footprint.
func (s *Spinner) MinSize() geom.Size { return geom.Size{W: s.size, H: s.size} }

// Draw paints the dot ring and, while spinning, advances the rotation by one
// frame.
func (s *Spinner) Draw(canvas render.Canvas) {
	if s.spinning {
		s.phase += s.speed * nominalFrameDelta
	}

	b := s.Bounds()
	d := minF(b.W, b.H)
	if d <= 0 {
		return
	}
	cx, cy := b.X+b.W/2, b.Y+b.H/2
	dotR := d / 12
	ringR := d/2 - dotR - 1
	col := s.ColourOf(RolePrimary)

	head := s.phase * spinnerDots // which dot is currently brightest
	for i := 0; i < spinnerDots; i++ {
		ang := 2*math.Pi*float64(i)/spinnerDots - math.Pi/2 // start at the top
		x := cx + ringR*math.Cos(ang)
		y := cy + ringR*math.Sin(ang)

		// age: how many dots behind the head this one is, in [0, spinnerDots).
		age := math.Mod(head-float64(i), spinnerDots)
		if age < 0 {
			age += spinnerDots
		}
		alpha := 1 - age/spinnerDots // head = 1, trailing dots fade toward 0
		canvas.FillCircle(geom.Point{X: x, Y: y}, dotR, withAlpha(col, 0.15+0.85*alpha))
	}
}
