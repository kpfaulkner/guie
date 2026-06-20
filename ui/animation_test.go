package ui

import (
	"math"
	"testing"
)

// frames advances the app by n fixed frames.
func frames(a *App, n int) {
	for i := 0; i < n; i++ {
		a.advanceFrame(nominalFrameDelta)
	}
}

func TestTweenInterpolatesAndCompletes(t *testing.T) {
	app := NewApp()
	var v float64
	done := false
	an := app.Tween(1.0, 0, 10, Linear, func(x float64) { v = x }).OnDone(func() { done = true })

	frames(app, 30) // 0.5s
	if math.Abs(v-5) > 0.2 {
		t.Fatalf("at half duration (linear) expected ~5, got %v", v)
	}
	if an.Done() {
		t.Fatal("animation should not be done at half duration")
	}

	frames(app, 40) // past the end
	if v != 10 {
		t.Fatalf("final value should be exactly 10, got %v", v)
	}
	if !an.Done() || !done {
		t.Fatalf("animation should be done and OnDone fired (done=%v)", done)
	}
}

func TestAnimationStopHaltsAndSkipsOnDone(t *testing.T) {
	app := NewApp()
	var v float64
	fired := false
	an := app.Tween(1.0, 0, 10, Linear, func(x float64) { v = x }).OnDone(func() { fired = true })

	frames(app, 1)
	an.Stop()
	at := v
	frames(app, 120)
	if v != at {
		t.Fatalf("stopped animation kept updating: %v -> %v", at, v)
	}
	if !an.Done() {
		t.Fatal("stopped animation should report Done")
	}
	if fired {
		t.Fatal("OnDone must not fire for a stopped animation")
	}
}

func TestOnFrameReceivesFixedDelta(t *testing.T) {
	app := NewApp()
	var total float64
	calls := 0
	app.OnFrame(func(dt float64) {
		total += dt
		calls++
	})
	frames(app, 60) // ~1s
	if calls != 60 {
		t.Fatalf("expected 60 frame callbacks, got %d", calls)
	}
	if math.Abs(total-1.0) > 1e-9 {
		t.Fatalf("expected ~1s accumulated, got %v", total)
	}
}

func TestAnimateStartedFromOnDoneRunsNextFrame(t *testing.T) {
	app := NewApp()
	second := 0.0
	app.Tween(nominalFrameDelta, 0, 1, Linear, func(float64) {}).
		OnDone(func() {
			app.Tween(1.0, 0, 100, Linear, func(v float64) { second = v })
		})

	frames(app, 1) // first tween completes, schedules the second
	if second != 0 {
		t.Fatalf("second animation should not have advanced on its starting frame, got %v", second)
	}
	frames(app, 30) // ~0.5s of the second
	if math.Abs(second-50) > 2 {
		t.Fatalf("second animation should be ~halfway (~50), got %v", second)
	}
}

func TestEasings(t *testing.T) {
	for _, e := range []Easing{Linear, EaseIn, EaseOut, EaseInOut} {
		if math.Abs(e(0)) > 1e-9 {
			t.Errorf("easing(0) should be 0, got %v", e(0))
		}
		if math.Abs(e(1)-1) > 1e-9 {
			t.Errorf("easing(1) should be 1, got %v", e(1))
		}
	}
}
