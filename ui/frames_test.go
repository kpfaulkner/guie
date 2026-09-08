package ui

import (
	"image/color"
	"testing"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
)

// pacingDriver is a headless driver that records the frame pacing the App asks
// for. Run captures the hooks and returns, so a test drives frames itself.
type pacingDriver struct {
	hooks      render.Hooks
	continuous []bool // every SetContinuousFrames call, in order
	requests   int    // RequestFrame calls
}

func (d *pacingDriver) Run(cfg render.Config, hooks render.Hooks) error {
	d.hooks = hooks
	return nil
}

func (d *pacingDriver) SetContinuousFrames(on bool) { d.continuous = append(d.continuous, on) }
func (d *pacingDriver) RequestFrame()               { d.requests++ }

// last reports the pacing currently in effect: the last mode asked for, or
// true, since a driver starts continuous and is only switched off the default.
func (d *pacingDriver) last() bool {
	if len(d.continuous) == 0 {
		return true
	}
	return d.continuous[len(d.continuous)-1]
}

// pacedApp returns an app on a pacingDriver, already running and with its first
// frame drawn - the state a real window is in once it is on screen.
func pacedApp(t *testing.T) (*App, *pacingDriver) {
	t.Helper()
	d := &pacingDriver{}
	a := NewApp(WithDriver(d))
	if err := a.Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}
	return a, d
}

// frame runs one Update and one Draw, which is what a driver does per frame.
func frame(t *testing.T, a *App) {
	t.Helper()
	if err := a.update(render.InputState{}); err != nil {
		t.Fatalf("update: %v", err)
	}
	a.draw(nullCanvas{})
}

// nullCanvas satisfies render.Canvas and throws every draw away. These tests
// are about when frames happen, not what lands on them.
type nullCanvas struct{}

func (nullCanvas) Size() geom.Size                                           { return geom.Size{W: 800, H: 600} }
func (nullCanvas) PushClip(geom.Rect)                                        {}
func (nullCanvas) PopClip()                                                  {}
func (nullCanvas) Fill(color.Color)                                          {}
func (nullCanvas) FillRect(geom.Rect, color.Color)                           {}
func (nullCanvas) StrokeRect(geom.Rect, color.Color, float64)                {}
func (nullCanvas) FillRoundRect(geom.Rect, float64, color.Color)             {}
func (nullCanvas) StrokeRoundRect(geom.Rect, float64, color.Color, float64)  {}
func (nullCanvas) DrawLine(geom.Point, geom.Point, color.Color, float64)     {}
func (nullCanvas) FillCircle(geom.Point, float64, color.Color)               {}
func (nullCanvas) StrokeCircle(geom.Point, float64, color.Color, float64)    {}
func (nullCanvas) DrawText(string, geom.Point, render.FontFace, color.Color) {}
func (nullCanvas) MeasureText(string, render.FontFace) geom.Size             { return geom.Size{} }
func (nullCanvas) DrawImage(render.Image, geom.Rect)                         {}
func (c nullCanvas) SubCanvas(geom.Rect) render.Canvas                       { return c }

// Frames stop only once one has been drawn. A backend can need frames of its
// own to get a window up, so an Update before the first Draw must not be what
// switches presenting off.
func TestFramesStopAfterTheFirstDrawNotBefore(t *testing.T) {
	a, d := pacedApp(t)

	if err := a.update(render.InputState{}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if len(d.continuous) != 0 {
		t.Fatalf("pacing changed before the first frame was drawn: %v", d.continuous)
	}

	a.draw(nullCanvas{})
	if d.last() {
		t.Error("still presenting every refresh after the first frame; want on demand")
	}
}

// Animations, toasts and frame callbacks all run off the frame clock, so each
// one keeps frames coming and lets them stop again when it is done.
func TestWorkOnTheFrameClockKeepsFramesComing(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start func(a *App)
	}{
		{name: "animation", start: func(a *App) { a.Animate(10, nil, func(float64) {}) }},
		{name: "toast", start: func(a *App) { a.ShowToast("hello") }},
		{name: "frame callback", start: func(a *App) { a.OnFrame(func(float64) {}) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, d := pacedApp(t)
			frame(t, a)
			if d.last() {
				t.Fatalf("idle app is still presenting every refresh")
			}

			tc.start(a)
			frame(t, a)
			if !d.last() {
				t.Error("presenting on demand while work needs the frame clock")
			}
		})
	}
}

// An animation that finishes releases the app back to on-demand presenting: the
// point of the pacing is that the cost is paid only while something is running.
func TestFramesStopAgainWhenAnimationEnds(t *testing.T) {
	a, d := pacedApp(t)
	frame(t, a)

	a.Animate(3*nominalFrameDelta, nil, func(float64) {})
	frame(t, a)
	if !d.last() {
		t.Fatal("presenting on demand while an animation is running")
	}

	// Enough frames for the animation to reach its end and be dropped.
	for range 5 {
		frame(t, a)
	}
	if d.last() {
		t.Error("still presenting every refresh after the animation finished")
	}
}

// A tooltip's hover delay counts Update ticks after the pointer has stopped
// moving, so no input is coming to produce those frames. The pointer moving
// onto the widget is what has to leave frames running: every one of those
// frames resets the delay, so the last of them - the one that decides whether
// frames continue - is always at zero ticks.
func TestPendingTooltipKeepsFramesComing(t *testing.T) {
	a, d, _ := tooltipApp(t)

	moveTo(t, a, geom.Point{X: 20, Y: 20})
	moveTo(t, a, geom.Point{X: 21, Y: 21})
	if !d.last() {
		t.Fatal("presenting on demand with a tooltip still to appear")
	}

	// Resting on the widget counts the delay out and shows the tooltip.
	for range tooltipDelayTicks + 2 {
		moveTo(t, a, geom.Point{X: 21, Y: 21})
	}
	if a.tooltipText != "tip" {
		t.Fatalf("tooltip did not appear, got %q", a.tooltipText)
	}
	// Shown, so there is nothing left to count and frames may stop again.
	if d.last() {
		t.Error("still presenting every refresh once the tooltip is up")
	}
}

// A pointer resting where there is no tooltip to show must not hold frames.
func TestHoverWithoutATooltipDoesNotKeepFramesComing(t *testing.T) {
	a, d, b := tooltipApp(t)
	b.SetTooltip("")

	moveTo(t, a, geom.Point{X: 20, Y: 20})
	if d.last() {
		t.Error("presenting every refresh while hovering a widget with no tooltip")
	}
}

// tooltipApp returns a paced app whose whole surface is a tooltipped button,
// with its first frame already drawn.
func tooltipApp(t *testing.T) (*App, *pacingDriver, *Button) {
	t.Helper()
	d := &pacingDriver{}
	a := NewApp(WithDriver(d))
	b := NewButton("ok")
	b.SetTooltip("tip")
	a.SetContent(b)
	if err := a.Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}
	a.resize(800, 600)
	frame(t, a)
	return a, d, b
}

// moveTo runs one frame with the pointer at p, as a driver does per input.
func moveTo(t *testing.T, a *App, p geom.Point) {
	t.Helper()
	if err := a.update(render.InputState{MousePos: p}); err != nil {
		t.Fatalf("update: %v", err)
	}
	a.draw(nullCanvas{})
}

// Do and Quit are the two calls a background goroutine makes, and both are
// useless if nothing wakes the window: Do would not be seen until the next
// input, and Quit would leave the window open.
func TestDoAndQuitAskForAFrame(t *testing.T) {
	a, d := pacedApp(t)

	a.Do(func() {})
	if d.requests != 1 {
		t.Errorf("Do asked for %d frames, want 1", d.requests)
	}
	a.Quit()
	if d.requests != 2 {
		t.Errorf("Quit asked for %d frames, want 1 more", d.requests)
	}
}

// Invalidate is the escape hatch for a change the framework cannot see, and has
// to be safe on an app whose driver does not gate frames at all.
func TestInvalidateWithoutAFrameSchedulingDriver(t *testing.T) {
	a := NewApp(WithDriver(&plainDriver{}))
	a.Invalidate() // must not panic
	a.Quit()
}

// SetContinuousRedraw is the opt-out, for an app that changes what it draws
// without telling the framework.
func TestSetContinuousRedrawOptsOut(t *testing.T) {
	a, d := pacedApp(t)
	frame(t, a)
	if d.last() {
		t.Fatal("idle app is still presenting every refresh")
	}

	a.SetContinuousRedraw(true)
	if !d.last() {
		t.Error("SetContinuousRedraw(true) did not restore continuous presenting")
	}
	// And it holds across frames, with nothing on the frame clock to hold it.
	frame(t, a)
	if !d.last() {
		t.Error("continuous presenting was switched off again on the next frame")
	}

	a.SetContinuousRedraw(false)
	if d.last() {
		t.Error("SetContinuousRedraw(false) did not return to on-demand presenting")
	}
}

// The option is the same opt-out as the method, in place before the first
// frame - so an app that needs continuous presenting never has one that is not.
func TestWithContinuousRedrawHoldsFromTheStart(t *testing.T) {
	d := &pacingDriver{}
	a := NewApp(WithDriver(d), WithContinuousRedraw(true))
	if err := a.Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}
	frame(t, a)
	if !d.last() {
		t.Error("WithContinuousRedraw(true) still dropped to on-demand presenting")
	}
}

// plainDriver is a driver with no optional capabilities at all.
type plainDriver struct{}

func (d *plainDriver) Run(cfg render.Config, hooks render.Hooks) error { return nil }
