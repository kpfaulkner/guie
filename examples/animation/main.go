// Command animation demonstrates the per-frame hook and the tween system.
//
//   - OnFrame: a label shows elapsed time, accumulated from the per-frame dt.
//   - Tween:   buttons animate a ProgressBar to a target value with different
//     easings; "Bounce" chains two tweens via OnDone; "Stop" cancels whatever
//     is running. A second tween animates a colored bar's width via a custom
//     widget so you can see motion, not just a number.
//
// Run with: go run ./examples/animation
package main

import (
	"fmt"
	"image/color"
	"log"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
	"github.com/kpfaulkner/guie/ui"
)

// bar is a tiny custom widget that fills a fraction of its width — handy for
// seeing an animated value as motion. The fraction is plain state; a tween just
// calls SetFraction each frame and the every-frame redraw shows it.
type bar struct {
	ui.BaseWidget
	frac float64
	fill color.Color
}

func newBar(fill color.Color) *bar {
	return &bar{BaseWidget: ui.NewBase(), fill: fill}
}

func (b *bar) SetFraction(f float64) { b.frac = f }

func (b *bar) MinSize() geom.Size { return geom.Size{W: 80, H: 28} }

func (b *bar) Draw(c render.Canvas) {
	r := b.Bounds()
	c.FillRoundRect(r, 6, b.ColorOf(ui.RoleSurface))
	if b.frac > 0 {
		fr := r
		fr.W = r.W * b.frac
		c.FillRoundRect(fr, 6, b.fill)
	}
	c.StrokeRoundRect(r, 6, b.ColorOf(ui.RoleBorder), 1)
}

func main() {
	app := ui.NewApp(
		ui.WithTitle("guie — animation"),
		ui.WithSize(560, 360),
	)
	pal := app.Theme().Palette

	prog := ui.NewProgressBar(0)
	motion := newBar(pal.Accent)

	// A running animation handle so "Stop" can cancel the current one.
	var running *ui.Animation
	start := func(an *ui.Animation) {
		if running != nil {
			running.Stop()
		}
		running = an
	}

	// Animate progress + the motion bar together to a target, with an easing.
	animateTo := func(to float64, ease ui.Easing) {
		from := prog.Value()
		start(app.Tween(0.8, from, to, ease, func(v float64) {
			prog.SetValue(v)
			motion.SetFraction(v)
		}))
	}

	easeBtn := ui.NewButton("Fill (ease-in-out)")
	easeBtn.OnClick(func() { animateTo(1, ui.EaseInOut) })

	linBtn := ui.NewButton("Empty (linear)")
	linBtn.OnClick(func() { animateTo(0, ui.Linear) })

	// Bounce: tween up fast, then back down on completion (chained via OnDone).
	bounceBtn := ui.NewButton("Bounce")
	bounceBtn.OnClick(func() {
		start(app.Tween(0.35, prog.Value(), 1, ui.EaseOut, func(v float64) {
			prog.SetValue(v)
			motion.SetFraction(v)
		}).OnDone(func() {
			start(app.Tween(0.55, 1, 0, ui.EaseIn, func(v float64) {
				prog.SetValue(v)
				motion.SetFraction(v)
			}))
		}))
	})

	stopBtn := ui.NewButton("Stop", ui.ButtonFlat())
	stopBtn.OnClick(func() {
		if running != nil {
			running.Stop()
		}
	})

	// OnFrame: accumulate elapsed time and show it (demonstrates the frame hook).
	clock := ui.NewLabel("elapsed: 0.0s", ui.LabelColor(pal.TextMuted))
	var elapsed float64
	app.OnFrame(func(dt float64) {
		elapsed += dt
		clock.SetText(fmt.Sprintf("elapsed: %.1fs", elapsed))
	})

	buttons := ui.NewContainer()
	buttons.SetLayout(ui.HBox(8))
	buttons.Add(easeBtn)
	buttons.Add(linBtn)
	buttons.Add(bounceBtn)
	buttons.Add(stopBtn)

	root := ui.NewContainer()
	root.SetLayout(ui.VBox(14))
	root.SetPadding(geom.UniformInsets(18))
	root.Add(ui.NewLabel("Tween a value; OnFrame ticks the clock."))
	root.Add(buttons)
	root.Add(prog)
	root.Add(motion)
	root.Add(clock, ui.Weight(1))

	app.SetContent(root)
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
