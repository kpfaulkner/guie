package main

import (
	"image/color"
	"log"

	"github.com/kpfaulkner/uiframework/geom"
	"github.com/kpfaulkner/uiframework/render"
	"github.com/kpfaulkner/uiframework/ui"
)

// textWidget is a minimal leaf widget used by the step-2 demo to exercise the
// retained tree and draw traversal. It shows how a custom widget is built:
// embed ui.BaseWidget and override Draw. (Real Label/Button widgets arrive in
// step 4.)
type textWidget struct {
	ui.BaseWidget
	str   string
	font  render.FontFace
	color color.Color
}

func newText(s string, font render.FontFace, c color.Color) *textWidget {
	return &textWidget{BaseWidget: ui.NewBase(), str: s, font: font, color: c}
}

func (t *textWidget) Draw(c render.Canvas) {
	b := t.Bounds()
	c.DrawText(t.str, geom.Point{X: b.X, Y: b.Y}, t.font, t.color)
}

// Step-2 demo: a retained widget tree drawn top-down. A transparent root holds
// a surface panel; the panel holds three text widgets, one of which overflows
// the panel's content area to demonstrate clipping. Still no EBiten import.
func main() {
	app := ui.NewApp(
		ui.WithTitle("uiframework — step 2"),
		ui.WithSize(640, 400),
	)
	th := app.Theme()
	pal := th.Palette

	root := ui.NewContainer()

	panel := ui.NewContainer()
	panel.SetBackground(pal.Surface)
	panel.SetPadding(geom.UniformInsets(12))
	panel.SetBounds(geom.Rect{X: 40, Y: 40, W: 360, H: 280})

	title := newText("Step 2: retained widget tree", th.Font, pal.Text)
	title.SetBounds(geom.Rect{X: 56, Y: 60, W: 320, H: 24})

	body := newText("Containers draw children and clip them.", th.Font, pal.TextMuted)
	body.SetBounds(geom.Rect{X: 56, Y: 96, W: 320, H: 24})

	// Bounds run past the panel's right edge to show the content-area clip.
	overflow := newText("This long line is clipped to the panel >>>>>>>>>>>>>>>>>>>>", th.Font, pal.Accent)
	overflow.SetBounds(geom.Rect{X: 56, Y: 140, W: 600, H: 24})

	panel.Add(title)
	panel.Add(body)
	panel.Add(overflow)
	root.Add(panel)

	app.SetContent(root)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
