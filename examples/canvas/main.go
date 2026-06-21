// Command canvas demonstrates building a custom widget by embedding
// ui.BaseWidget and drawing directly with the render.Canvas primitives:
// FillRect, StrokeRect, DrawLine, DrawText and MeasureText. It also shows that
// application code talks only to the framework's own abstractions — there is no
// EBiten import here.
//
// Run with: go run ./examples/canvas
package main

import (
	"image/color"
	"log"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
	"github.com/kpfaulkner/guie/theme"
	"github.com/kpfaulkner/guie/ui"
)

// sketch is a custom leaf widget. It receives its font and colours at
// construction (the theme is read from the App), then paints itself each frame
// through the Canvas.
type sketch struct {
	ui.BaseWidget
	font render.FontFace
	pal  theme.Palette
}

func newSketch(font render.FontFace, pal theme.Palette) *sketch {
	return &sketch{BaseWidget: ui.NewBase(), font: font, pal: pal}
}

// MinSize gives the layout a sensible minimum so the widget gets real space.
func (s *sketch) MinSize() geom.Size { return geom.Size{W: 240, H: 160} }

func (s *sketch) Draw(c render.Canvas) {
	b := s.Bounds()

	// Panel background and border.
	c.FillRect(b, s.pal.Surface)
	c.StrokeRect(b, s.pal.Border, 1)

	// A filled accent rectangle inset from the panel.
	inner := b.Inset(geom.UniformInsets(16))
	c.FillRect(inner, s.pal.Primary)

	// An X across the inner rectangle.
	c.DrawLine(inner.Min(), inner.Max(), s.pal.Accent, 3)
	c.DrawLine(
		geom.Point{X: inner.X, Y: inner.Y + inner.H},
		geom.Point{X: inner.X + inner.W, Y: inner.Y},
		s.pal.Accent, 3,
	)

	// A label centered horizontally near the top, using MeasureText.
	const caption = "drawn with Canvas primitives"
	size := c.MeasureText(caption, s.font)
	c.DrawText(
		caption,
		geom.Point{X: b.X + (b.W-size.W)/2, Y: b.Y + 6},
		s.font,
		s.pal.Text,
	)
}

func main() {
	app := ui.NewApp(
		ui.WithTitle("guie — canvas"),
		ui.WithSize(520, 360),
		ui.WithBackground(color.RGBA{R: 0x14, G: 0x14, B: 0x1c, A: 0xff}),
	)
	th := app.Theme()

	root := ui.NewContainer()
	root.SetLayout(ui.NewStack())
	root.SetPadding(geom.UniformInsets(24))
	root.Add(newSketch(th.Font, th.Palette), ui.Align(geom.AlignStretch))

	app.SetContent(root)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
