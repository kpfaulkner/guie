package ui

import (
	"testing"

	"github.com/kpfaulkner/guie/theme"
)

// hoverButtonUnder mounts a button in an app with theme th and hovers it.
func hoverButtonUnder(t *testing.T, th theme.Theme) (*App, *Button) {
	t.Helper()
	app := NewApp(WithTheme(th))
	b := NewButton("ok")
	app.SetContent(b)
	app.resize(200, 100)
	frame(t, app)
	b.HandleEvent(&Event{Type: EventPointerEnter})
	return app, b
}

// framesToSettle counts frames until the button's hover fade lands on 1.
func framesToSettle(t *testing.T, app *App, b *Button) int {
	t.Helper()
	for n := 1; n <= 500; n++ {
		frame(t, app)
		if b.hoverFade.level == 1 {
			return n
		}
	}
	t.Fatal("hover fade never settled")
	return 0
}

func TestFadeLengthFollowsTheTheme(t *testing.T) {
	slow, fast := theme.Default(), theme.Default()
	slow.Motion.Duration, fast.Motion.Duration = 0.6, 0.03
	def := theme.Default()

	settle := func(th theme.Theme) int {
		app, b := hoverButtonUnder(t, th)
		return framesToSettle(t, app, b)
	}
	ns, nd, nf := settle(slow), settle(def), settle(fast)
	if !(nf < nd && nd < ns) {
		t.Errorf("frames to settle: 0.03s=%d, default 0.12s=%d, 0.6s=%d; want them in that order", nf, nd, ns)
	}
	if nd < 4 || nd > 14 {
		t.Errorf("the default fade took %d frames; 0.12s at 60 fps is about 7", nd)
	}
}

func TestReducedMotionSnapsInTheSameFrame(t *testing.T) {
	th := theme.Default().WithReducedMotion()
	app, b := hoverButtonUnder(t, th)
	if b.hoverFade.level != 1 {
		t.Errorf("with reduced motion the hover level is %v the moment the pointer enters, want 1", b.hoverFade.level)
	}
	b.HandleEvent(&Event{Type: EventPointerLeave})
	if b.hoverFade.level != 0 {
		t.Errorf("leaving: level %v, want 0 at once", b.hoverFade.level)
	}
	// Nothing is left animating, so the app can go idle.
	frame(t, app)
	if app.needsFrames() {
		t.Error("a reduced-motion hover left the app asking for frames")
	}

	// Pressed snaps too.
	b.HandleEvent(&Event{Type: EventPointerEnter})
	b.HandleEvent(&Event{Type: EventPointerDown, Button: 0})
	if b.pressFade.level != 1 {
		t.Errorf("press level %v, want 1 at once", b.pressFade.level)
	}
}

func TestOrdinaryHoverDoesAskForFrames(t *testing.T) {
	// The contrast case for the test above, so "needsFrames is false" means something.
	app, _ := hoverButtonUnder(t, theme.Default())
	if !app.needsFrames() {
		t.Error("a normal fade should keep frames coming until it lands")
	}
}

func TestMotionFollowsSetTheme(t *testing.T) {
	app, b := hoverButtonUnder(t, theme.Default())
	framesToSettle(t, app, b)
	b.HandleEvent(&Event{Type: EventPointerLeave})
	frame(t, app)
	if b.hoverFade.level == 0 {
		t.Fatal("a normal fade should still be in flight a frame after leaving")
	}
	for range 40 {
		frame(t, app)
	}

	app.SetTheme(theme.Default().WithReducedMotion())
	b.HandleEvent(&Event{Type: EventPointerEnter})
	if b.hoverFade.level != 1 {
		t.Errorf("after SetTheme to reduced motion the next hover should snap, level %v", b.hoverFade.level)
	}
	app.SetTheme(theme.Default())
	b.HandleEvent(&Event{Type: EventPointerLeave})
	if b.hoverFade.level == 0 {
		t.Error("after switching back, fades should animate again")
	}
}

func TestTabIndicatorSnapsUnderReducedMotion(t *testing.T) {
	app := NewApp(WithTheme(theme.Default().WithReducedMotion()))
	tc := NewTabContainer()
	tc.AddTab("One", newStub(10, 10))
	tc.AddTab("Two", newStub(10, 10))
	app.SetContent(tc)
	app.resize(400, 300)
	frame(t, app)

	tc.Select(1)
	if tc.indAnim != nil {
		t.Error("the indicator started sliding despite reduced motion")
	}
	if got, want := tc.indicatorSpan(), tc.span(1); got != want {
		t.Errorf("indicator at %+v, want it already on the new tab %+v", got, want)
	}
	if app.needsFrames() {
		t.Error("the app is still asking for frames")
	}
}

func TestTabIndicatorSlideFollowsDuration(t *testing.T) {
	slides := func(d float64) int {
		th := theme.Default()
		th.Motion.Duration = d
		app := NewApp(WithTheme(th))
		tc := NewTabContainer()
		tc.AddTab("One", newStub(10, 10))
		tc.AddTab("Two", newStub(10, 10))
		app.SetContent(tc)
		app.resize(400, 300)
		frame(t, app)
		tc.Select(1)
		for n := 1; n <= 500; n++ {
			frame(t, app)
			if tc.indAnim == nil {
				return n
			}
		}
		t.Fatal("slide never ended")
		return 0
	}
	if fast, slow := slides(0.04), slides(0.4); !(fast < slow) {
		t.Errorf("slide frames: 0.04s=%d, 0.4s=%d; the longer duration should take longer", fast, slow)
	}
}

// Reduced motion covers the widgets' own state fades. Motion the application starts
// itself is its own business and keeps running.
func TestReducedMotionLeavesAppAnimationsAlone(t *testing.T) {
	app := NewApp(WithTheme(theme.Default().WithReducedMotion()))
	app.SetContent(NewLabel("x"))
	frame(t, app)
	var seen []float64
	app.Animate(0.2, Linear, func(v float64) { seen = append(seen, v) })
	for range 30 {
		frame(t, app)
	}
	if len(seen) < 5 {
		t.Errorf("App.Animate ran %d steps under reduced motion, want a normal run", len(seen))
	}
	if last := seen[len(seen)-1]; last != 1 {
		t.Errorf("animation ended at %v, want 1", last)
	}
}

func TestMotionDurationFallsBackToTheDefault(t *testing.T) {
	var b BaseWidget = NewBase()
	if got := b.motionSeconds(); got != 0.12 {
		t.Errorf("unmounted widget fade = %v, want the default 0.12", got)
	}
	if b.reducedMotion() {
		t.Error("an unmounted widget is not in reduced motion")
	}
}
