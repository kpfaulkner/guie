// Command paint is a tiny freehand drawing program built with uiframework. It
// shows how to read mouse position from pointer events: a custom widget records
// the cursor on pointer-down/move/up (pointer capture delivers the whole drag),
// and renders strokes via the Canvas. Left-drag draws with the selected color;
// right-drag erases (paints with the canvas color). Pick colors from the
// swatches, set the brush size with the slider, and Clear to reset.
//
// Run with: go run ./examples/paint
package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"log"

	"github.com/kpfaulkner/uiframework/geom"
	"github.com/kpfaulkner/uiframework/render"
	"github.com/kpfaulkner/uiframework/ui"
)

var canvasColor = color.RGBA{R: 0xf6, G: 0xf6, B: 0xf2, A: 0xff}

// stroke is one freehand line: a list of points with a color and width.
type stroke struct {
	pts   []geom.Point
	color color.Color
	width float64
}

// sketchpad is a custom widget: it captures the pointer on press and records the
// drag path, then draws the accumulated strokes.
type sketchpad struct {
	ui.BaseWidget
	strokes []*stroke
	cur     *stroke
	color   color.Color
	width   float64
}

func newSketchpad() *sketchpad {
	return &sketchpad{BaseWidget: ui.NewBase(), color: color.RGBA{A: 0xff}, width: 3}
}

func (s *sketchpad) Clear() {
	s.strokes = nil
	s.cur = nil
}

func (s *sketchpad) HandleEvent(ev *ui.Event) bool {
	switch ev.Type {
	case ui.EventPointerDown:
		col, w := s.color, s.width
		if ev.Button == render.MouseRight { // right-drag erases
			col, w = canvasColor, s.width*3
		}
		s.cur = &stroke{pts: []geom.Point{ev.Pos}, color: col, width: w}
		s.strokes = append(s.strokes, s.cur)
		return true
	case ui.EventPointerMove:
		if s.cur != nil {
			s.cur.pts = append(s.cur.pts, ev.Pos)
		}
		return true
	case ui.EventPointerUp:
		s.cur = nil
		return true
	}
	return false
}

func (s *sketchpad) Draw(c render.Canvas) {
	b := s.Bounds()
	c.FillRect(b, canvasColor)
	c.PushClip(b)
	for _, st := range s.strokes {
		if len(st.pts) == 1 {
			c.FillCircle(st.pts[0], st.width/2, st.color)
			continue
		}
		for i := 1; i < len(st.pts); i++ {
			c.DrawLine(st.pts[i-1], st.pts[i], st.color, st.width)
			// round the joins so the stroke looks smooth
			c.FillCircle(st.pts[i], st.width/2, st.color)
		}
	}
	c.PopClip()
	c.StrokeRect(b, s.ColorOf(ui.RoleBorder), 1)
}

// solidImage builds an 18x18 swatch of a solid color.
func solidImage(c color.RGBA) render.Image {
	const sz = 18
	img := image.NewRGBA(image.Rect(0, 0, sz, sz))
	for y := 0; y < sz; y++ {
		for x := 0; x < sz; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		log.Fatal(err)
	}
	im, err := ui.LoadImageBytes(buf.Bytes())
	if err != nil {
		log.Fatal(err)
	}
	return im
}

func main() {
	app := ui.NewApp(
		ui.WithTitle("uiframework — paint"),
		ui.WithSize(720, 560),
	)

	pad := newSketchpad()

	// Toolbar: color swatches, a brush-size slider, and Clear.
	toolbar := ui.NewContainer()
	toolbar.SetLayout(ui.HBox(8))
	toolbar.SetPadding(geom.UniformInsets(8))
	toolbar.SetBackground(app.Theme().Palette.Surface)

	palette := []color.RGBA{
		{R: 0x20, G: 0x20, B: 0x20, A: 0xff}, // black
		{R: 0xd0, G: 0x3a, B: 0x3a, A: 0xff}, // red
		{R: 0x2e, G: 0x9e, B: 0x44, A: 0xff}, // green
		{R: 0x2f, G: 0x6f, B: 0xd0, A: 0xff}, // blue
		{R: 0xf0, G: 0xb4, B: 0x29, A: 0xff}, // yellow
	}
	for _, col := range palette {
		c := col
		sw := ui.NewButton("", ui.ButtonImage(solidImage(c)), ui.ButtonFlat())
		sw.OnClick(func() { pad.color = c })
		toolbar.Add(sw)
	}

	toolbar.Add(ui.NewLabel("  Brush:"))
	brush := ui.NewSlider(ui.SliderValue(0.15))
	brush.OnChange(func(v float64) { pad.width = 1 + v*23 })
	toolbar.Add(brush, ui.Weight(1))

	clear := ui.NewButton("Clear")
	clear.OnClick(pad.Clear)
	toolbar.Add(clear)

	hint := ui.NewLabel("Left-drag to draw · right-drag to erase")

	canvasWrap := ui.NewContainer()
	canvasWrap.SetLayout(ui.NewStack())
	canvasWrap.SetPadding(geom.UniformInsets(8))
	canvasWrap.Add(pad)

	root := ui.NewContainer()
	root.SetLayout(ui.VBox(0))
	root.Add(toolbar)
	root.Add(canvasWrap, ui.Weight(1))
	statusBar := ui.NewContainer()
	statusBar.SetLayout(ui.VBox(0))
	statusBar.SetPadding(geom.UniformInsets(6))
	statusBar.Add(hint)
	root.Add(statusBar)

	app.SetContent(root)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
