// Command framepacing demonstrates on-demand frame presenting: the framework
// asks the backend for a frame only when something has changed, instead of
// presenting one every display refresh. An idle window then costs almost
// nothing - on macOS this is the difference between ~14% of a core and ~0.6%
// for an empty window, and 44% vs 3.4% for one drawing 400 strings.
//
// The meter panel counts the frames actually drawn. Leave the window alone and
// watch the counter stop: that is the whole feature. Then use the controls to
// see each thing that legitimately brings frames back.
//
// Things to try, in order:
//
//   - Stop touching the mouse. "frames last second" falls to 0 and the clock
//     freezes. Move the pointer over the window and both resume: input is what
//     wakes an on-demand window.
//   - Rest the pointer on "Hover me" without moving. The tooltip still appears
//     after ~0.5s, even though no input is arriving to produce those frames -
//     the App holds frames open while a tooltip is pending.
//   - Click "Animate 1s" and "Show toast". Both run off the frame clock, so
//     frames resume for their duration and stop again when they finish.
//   - Click either background-work button. Both finish a second later with the
//     window idle; App.Do and App.Invalidate are what get the result on screen.
//   - Tick "Continuous redraw". The clock runs smoothly and the counter pins at
//     the refresh rate. That is the old behaviour, and the escape hatch for a
//     widget whose Draw reads something the framework cannot see - like this
//     clock.
//
// To measure rather than watch, -stats prints the frames actually drawn each
// second, and -continuous runs the same window the old way for comparison:
//
//	go run ./examples/framepacing -stats              # idles at 0 frames/sec
//	go run ./examples/framepacing -stats -continuous  # pinned at the refresh rate
//
// Measured on Windows with this example: idling costs 14% of a core presenting
// every refresh and 0.4% on demand. Moving the pointer over the window runs the
// other way - input-scheduled frames are not held to the refresh rate, so
// sustained motion presents over 100 frames/sec and costs a few points more
// than presenting continuously would.
//
// Run with: go run ./examples/framepacing
package main

import (
	"flag"
	"fmt"
	"image/color"
	"log"
	"sync/atomic"
	"time"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
	"github.com/kpfaulkner/guie/theme"
	"github.com/kpfaulkner/guie/ui"
)

// meter is a custom widget that reports how much presenting is going on. It
// counts its own Draw calls, which is exactly the cost the pacing removes, and
// draws the wall clock - a value the framework has no way to know has changed,
// so it visibly freezes whenever frames stop.
type meter struct {
	ui.BaseWidget
	font render.FontFace
	pal  theme.Palette

	drawn  atomic.Int64 // frames drawn since start, readable off the UI goroutine
	total  int          // frames drawn since start
	window int          // frames drawn in the current one-second window
	last   int          // frames drawn in the previous complete window
	since  time.Time    // start of the current window

	// bg is written by a background goroutine and read here, so it is atomic:
	// Invalidate gets the frame, it does not make the value safe to share.
	bg *atomic.Int64
}

func newMeter(font render.FontFace, pal theme.Palette, bg *atomic.Int64) *meter {
	return &meter{BaseWidget: ui.NewBase(), font: font, pal: pal, bg: bg, since: time.Now()}
}

func (m *meter) MinSize() geom.Size { return geom.Size{W: 360, H: 150} }

func (m *meter) Draw(c render.Canvas) {
	m.drawn.Add(1)
	m.total++
	m.window++
	if d := time.Since(m.since); d >= time.Second {
		m.last = m.window
		m.window = 0
		m.since = time.Now()
	}

	b := m.Bounds()
	c.FillRect(b, m.pal.Surface)
	c.StrokeRect(b, m.pal.Border, 1)

	inner := b.Inset(geom.UniformInsets(14))
	y := inner.Y
	lh := m.font.Metrics().LineHeight + 4
	line := func(s string, col color.Color) {
		c.DrawText(s, geom.Point{X: inner.X, Y: y}, m.font, col)
		y += lh
	}

	line(fmt.Sprintf("frames last second: %d", m.last), m.pal.Text)
	line(fmt.Sprintf("frames total:       %d", m.total), m.pal.TextMuted)
	// The clock is the honest tell: it only advances on a frame, so a stale
	// reading is a frame that was never presented.
	line("clock:              "+time.Now().Format("15:04:05.000"), m.pal.Text)

	if n := m.bg.Load(); n > 0 {
		line(fmt.Sprintf("background result:  %d (via Invalidate)", n), m.pal.Accent)
	}
}

func main() {
	// The same opt-out as the checkbox below, but in place from the first frame,
	// which is what makes the two runs comparable.
	continuous := flag.Bool("continuous", false, "present every display refresh instead of on demand")
	stats := flag.Bool("stats", false, "print frames drawn per second to stdout")
	flag.Parse()

	app := ui.NewApp(
		ui.WithTitle("guie - frame pacing"),
		ui.WithSize(560, 420),
		ui.WithContinuousRedraw(*continuous),
	)
	th := app.Theme()

	var bg atomic.Int64
	m := newMeter(th.Font, th.Palette, &bg)

	root := ui.NewContainer()
	root.SetLayout(ui.VBox(12))
	root.SetPadding(geom.UniformInsets(20))

	root.Add(ui.NewLabel("Frames are presented on demand. Stop touching the"))
	root.Add(ui.NewLabel("window and the counter and clock below stop with it."))
	root.Add(m, ui.Align(geom.AlignStretch))

	status := ui.NewLabel("idle")
	root.Add(status)

	row1 := ui.NewContainer()
	row1.SetLayout(ui.HBox(10))

	// An animation runs off the frame clock, so the App holds frames open for
	// its duration and lets them stop when it finishes.
	anim := ui.NewButton("Animate 1s")
	anim.OnClick(func() {
		app.Tween(1, 0, 100, ui.EaseInOut, func(v float64) {
			status.SetText(fmt.Sprintf("animating: %.0f%%", v))
		})
	})
	row1.Add(anim)

	// A toast ages by dt, so it needs frames for as long as it is on screen.
	toast := ui.NewButton("Show toast")
	toast.OnClick(func() { app.ShowToast("frames run while I am up") })
	row1.Add(toast)

	// The tooltip delay counts frames after the pointer stops, which is exactly
	// when no input is arriving to produce any.
	hover := ui.NewButton("Hover me")
	hover.SetTooltip("Shown with no input arriving at all")
	row1.Add(hover)

	root.Add(row1, ui.Align(geom.AlignStart))

	row2 := ui.NewContainer()
	row2.SetLayout(ui.HBox(10))

	// Do queues work onto the UI goroutine and asks for a frame itself, so
	// background results reach the screen without the caller thinking about it.
	viaDo := ui.NewButton("Background -> Do")
	viaDo.OnClick(func() {
		status.SetText("working (1s)...")
		go func() {
			time.Sleep(time.Second)
			app.Do(func() { status.SetText("Do: finished at " + time.Now().Format("15:04:05.000")) })
		}()
	})
	row2.Add(viaDo)

	// Invalidate is the escape hatch for state the framework cannot see: this
	// goroutine writes an atomic the meter reads in Draw, then asks for the one
	// frame that will show it.
	viaInvalidate := ui.NewButton("Background -> Invalidate")
	viaInvalidate.OnClick(func() {
		go func() {
			time.Sleep(time.Second)
			bg.Add(1)
			app.Invalidate()
		}()
	})
	row2.Add(viaInvalidate)

	root.Add(row2, ui.Align(geom.AlignStart))

	// The opt-out, for a Draw that reads something the framework does not know
	// about - this example's clock being exactly that.
	cont := ui.NewCheckbox("Continuous redraw (every refresh)", ui.Checked(*continuous))
	cont.OnChange(func(v bool) {
		app.SetContinuousRedraw(v)
		if v {
			status.SetText("continuous: presenting every refresh")
		} else {
			status.SetText("on demand: presenting only when something changed")
		}
	})
	root.Add(cont)

	app.SetContent(root)

	// Reporting from a plain goroutine rather than OnFrame is deliberate: it
	// reads an atomic and asks for nothing, so measuring does not itself keep
	// the window awake. An OnFrame that printed this would pin the app at the
	// refresh rate and report only its own cost.
	if *stats {
		go func() {
			prev := int64(0)
			for range time.Tick(time.Second) {
				n := m.drawn.Load()
				fmt.Printf("frames/sec: %d\n", n-prev)
				prev = n
			}
		}()
	}

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
