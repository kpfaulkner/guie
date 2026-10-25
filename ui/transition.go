package ui

import "image/color"

// motionSeconds is how long a hover/press/focus state change takes to fade, from
// the theme (Theme.Motion.Duration, 0.12 s by default): short enough to feel
// immediate, long enough to read as a transition rather than a flicker.
func (b *BaseWidget) motionSeconds() float64 {
	if b.ctx != nil && b.ctx.theme != nil {
		if d := b.ctx.theme.Motion.Duration; d > 0 {
			return d
		}
	}
	return 0.12
}

// reducedMotion reports whether the theme asks for no movement: state changes then
// jump to their end state in the frame they happen.
func (b *BaseWidget) reducedMotion() bool {
	return b.ctx != nil && b.ctx.theme != nil && b.ctx.theme.Motion.Reduced
}

// fade is a 0..1 level that eases toward a target, used to cross-fade a widget
// between visual states (idle <-> hover, up <-> pressed). The zero value is
// level 0, idle. A widget embeds one per state it animates and reads level in
// Draw.
type fade struct {
	level  float64
	target float64
	anim   *Animation
}

// fadeTo eases f toward target (0 or 1). It does nothing if f is already
// heading there. Before the widget is mounted (no app to run animations) the
// level snaps, so state set up front is correct on the first frame.
// With reduced motion on, the level snaps the same way and nothing is left running.
//
// The animation lives on the App's animation list, so the app keeps presenting
// frames only while a fade is running and goes idle again as soon as it lands.
func (b *BaseWidget) fadeTo(f *fade, target float64) {
	if f.target == target {
		return
	}
	f.target = target
	if f.anim != nil {
		f.anim.Stop()
		f.anim = nil
	}
	if b.ctx == nil || b.ctx.animate == nil || b.reducedMotion() {
		f.level = target
		return
	}
	from := f.level
	if from == target {
		return
	}
	f.anim = b.ctx.animate(b.motionSeconds(), EaseOut, func(t float64) {
		f.level = from + (target-from)*t
	})
}

// mixColour linearly blends a toward b by t (0 gives a, 1 gives b). Channels are
// blended premultiplied, so translucent colours blend correctly too.
func mixColour(a, b color.Color, t float64) color.Color {
	if t <= 0 {
		return a
	}
	if t >= 1 {
		return b
	}
	ar, ag, ab, aa := a.RGBA()
	br, bg, bb, ba := b.RGBA()
	mix := func(x, y uint32) uint8 {
		return uint8((float64(x) + (float64(y)-float64(x))*t) / 257)
	}
	return color.RGBA{R: mix(ar, br), G: mix(ag, bg), B: mix(ab, bb), A: mix(aa, ba)}
}
