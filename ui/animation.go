package ui

// This file implements the per-frame hook and the animation/tween system.
//
// EBiten calls the loop's Update at a fixed tick rate (default 60 TPS), catching
// up with extra Update calls under load, so a fixed per-frame time step keeps
// animations wall-clock accurate and keeps tests deterministic — consistent with
// how tooltips time their delay by counting ticks.

// nominalTPS is the assumed fixed update rate; nominalFrameDelta is the seconds
// advanced per frame. (HiDPI/perf work aside, the loop ticks at this rate.)
const (
	nominalTPS        = 60.0
	nominalFrameDelta = 1.0 / nominalTPS
)

// Easing maps linear progress t in [0,1] to eased progress. The built-ins stay
// within [0,1]; a custom easing may overshoot if you want spring-like motion.
type Easing func(t float64) float64

// Linear is the identity easing (constant speed).
func Linear(t float64) float64 { return t }

// EaseIn accelerates from zero (quadratic).
func EaseIn(t float64) float64 { return t * t }

// EaseOut decelerates to zero (quadratic).
func EaseOut(t float64) float64 { return t * (2 - t) }

// EaseInOut accelerates then decelerates (quadratic).
func EaseInOut(t float64) float64 {
	if t < 0.5 {
		return 2 * t * t
	}
	return -1 + (4-2*t)*t
}

// Animation is a handle to a running animation. It is created by App.Animate or
// App.Tween and advanced once per frame by the App.
type Animation struct {
	elapsed  float64
	duration float64
	ease     Easing
	apply    func(t float64)
	onDone   func()
	stopped  bool
	finished bool
}

// Stop cancels the animation. Its apply is not called again and its OnDone
// callback does not run. Safe to call more than once.
func (an *Animation) Stop() { an.stopped = true }

// Done reports whether the animation has finished normally or been stopped.
func (an *Animation) Done() bool { return an.finished || an.stopped }

// OnDone sets a callback to run when the animation completes normally (it does
// not run if the animation is stopped). It returns the animation for chaining.
func (an *Animation) OnDone(fn func()) *Animation {
	an.onDone = fn
	return an
}

// OnFrame registers fn to run once per frame with dt, the seconds elapsed since
// the previous frame (a fixed step; see nominalFrameDelta). Use it for live
// updates or custom animation that must run on the UI goroutine. Multiple
// callbacks may be registered; they run in registration order, before layout
// and input dispatch each frame.
func (a *App) OnFrame(fn func(dt float64)) {
	if fn != nil {
		a.frameCbs = append(a.frameCbs, fn)
	}
}

// Animate starts an animation that calls apply(t) once per frame for duration
// seconds, with t advancing from 0 to 1 through ease (nil ease means Linear). On
// normal completion apply is called a final time with t = 1 (eased). A duration
// of zero or less snaps to the final state on the next frame. It returns a
// handle; call Stop to cancel.
func (a *App) Animate(duration float64, ease Easing, apply func(t float64)) *Animation {
	if ease == nil {
		ease = Linear
	}
	an := &Animation{duration: duration, ease: ease, apply: apply}
	a.anims = append(a.anims, an)
	return an
}

// Tween animates a single value from `from` to `to` over duration seconds,
// calling set with the eased interpolated value each frame. It is a thin
// convenience over Animate.
func (a *App) Tween(duration, from, to float64, ease Easing, set func(v float64)) *Animation {
	return a.Animate(duration, ease, func(t float64) {
		if set != nil {
			set(from + (to-from)*t)
		}
	})
}

// advanceFrame runs per-frame work: user frame callbacks, then active
// animations. dt is the seconds since the previous frame.
//
// It is reentrancy-safe: animations (or callbacks) may start new animations or
// stop existing ones. New animations are collected into a fresh list and first
// advance next frame.
func (a *App) advanceFrame(dt float64) {
	for _, fn := range a.frameCbs {
		fn(dt)
	}
	if len(a.anims) == 0 {
		return
	}
	current := a.anims
	a.anims = nil // anything started during apply/onDone lands here, for next frame
	for _, an := range current {
		if an.stopped {
			continue
		}
		an.elapsed += dt
		t := 1.0
		if an.duration > 0 && an.elapsed < an.duration {
			t = an.elapsed / an.duration
		}
		if an.apply != nil {
			an.apply(an.ease(t))
		}
		if t >= 1.0 {
			an.finished = true
			if an.onDone != nil {
				an.onDone()
			}
			continue // drop completed animation
		}
		a.anims = append(a.anims, an) // keep running
	}
}
