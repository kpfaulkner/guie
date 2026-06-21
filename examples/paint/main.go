// Command paint is a tiny freehand drawing program built with guie. It
// shows how to read mouse position from pointer events: a custom widget records
// the cursor on pointer-down/move/up (pointer capture delivers the whole drag),
// and renders strokes via the Canvas. Left-drag draws with the selected colour;
// right-drag erases (paints with the canvas colour). Pick colours from the
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

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
	"github.com/kpfaulkner/guie/ui"
)

var canvasColour = color.RGBA{R: 0xf6, G: 0xf6, B: 0xf2, A: 0xff}

// stroke is one freehand line: a list of points with a colour and width.
type stroke struct {
	pts    []geom.Point
	colour color.Color
	width  float64
}

// sketchpad is a custom widget: it captures the pointer on press and records the
// drag path. Finished strokes are baked once into an offscreen render target
// (the "layer") and the whole layer is blitted each frame; only the in-progress
// stroke is re-drawn live. This keeps per-frame cost roughly constant no matter
// how much has been drawn — instead of re-issuing every stroke's draw calls
// every frame, which grew without bound.
type sketchpad struct {
	ui.BaseWidget
	strokes []*stroke
	cur     *stroke
	colour  color.Color
	width   float64

	layer   render.RenderTarget // baked, completed strokes (nil until first Draw)
	layerSz geom.Size           // logical size the layer was built for
}

func newSketchpad() *sketchpad {
	return &sketchpad{BaseWidget: ui.NewBase(), colour: color.RGBA{A: 0xff}, width: 3}
}

func (s *sketchpad) Clear() {
	s.strokes = nil
	s.cur = nil
	if s.layer != nil {
		s.layer.Clear(canvasColour)
	}
}

// ensureLayer (re)builds the offscreen layer when missing or when the widget's
// size changed, re-baking every stroke into it. During normal drawing the layer
// is updated incrementally (one segment per new point), so this full rebake only
// runs on first use and on resize.
func (s *sketchpad) ensureLayer() {
	b := s.Bounds()
	if b.W <= 0 || b.H <= 0 {
		return
	}
	if s.layer != nil && s.layerSz == b.Size() {
		return
	}
	if s.layer != nil {
		s.layer.Dispose()
	}
	s.layer = ui.NewRenderTarget(int(b.W), int(b.H))
	s.layerSz = b.Size()
	if s.layer == nil {
		return
	}
	s.layer.Clear(canvasColour)
	lc := s.layer.Canvas()
	for _, st := range s.strokes {
		drawStroke(lc, st, b.Min())
	}
}

func (s *sketchpad) HandleEvent(ev *ui.Event) bool {
	switch ev.Type {
	case ui.EventPointerDown:
		col, w := s.colour, s.width
		if ev.Button == render.MouseRight { // right-drag erases
			col, w = canvasColour, s.width*3
		}
		s.cur = &stroke{pts: []geom.Point{ev.Pos}, colour: col, width: w}
		s.strokes = append(s.strokes, s.cur)
		// Bake the starting dot straight into the layer.
		s.paint(func(c render.Canvas, off geom.Point) {
			c.FillCircle(sub(ev.Pos, off), w/2, col)
		})
		return true
	case ui.EventPointerMove:
		if s.cur != nil {
			// Skip points within ~1px of the last (slow drag / sub-pixel jitter).
			last := s.cur.pts[len(s.cur.pts)-1]
			dx, dy := ev.Pos.X-last.X, ev.Pos.Y-last.Y
			if dx*dx+dy*dy >= 1 {
				s.cur.pts = append(s.cur.pts, ev.Pos)
				// Bake just the new segment, once. The layer therefore always
				// holds the full drawing, so Draw is a single blit and per-frame
				// cost (and allocation) stays flat no matter how long the stroke.
				cur := s.cur
				s.paint(func(c render.Canvas, off geom.Point) {
					c.DrawLine(sub(last, off), sub(ev.Pos, off), cur.colour, cur.width)
					c.FillCircle(sub(ev.Pos, off), cur.width/2, cur.colour)
				})
			}
		}
		return true
	case ui.EventPointerUp:
		s.cur = nil // nothing to flush: the stroke is already fully baked
		return true
	}
	return false
}

func (s *sketchpad) Draw(c render.Canvas) {
	b := s.Bounds()
	s.ensureLayer()
	c.PushClip(b)
	if s.layer != nil {
		c.DrawImage(s.layer, b) // the whole drawing, in one blit
	} else {
		c.FillRect(b, canvasColour)
	}
	c.PopClip()
	c.StrokeRect(b, s.ColourOf(ui.RoleBorder), 1)
}

// paint runs draw against the layer's canvas in widget-local coordinates (off is
// the widget's top-left), if the layer exists.
func (s *sketchpad) paint(draw func(c render.Canvas, off geom.Point)) {
	if s.layer == nil {
		return
	}
	draw(s.layer.Canvas(), s.Bounds().Min())
}

// sub translates p into a coordinate space whose origin is off.
func sub(p, off geom.Point) geom.Point { return geom.Point{X: p.X - off.X, Y: p.Y - off.Y} }

// drawStroke renders a whole stroke onto c (used to re-bake on resize), with
// points translated by -off.
func drawStroke(c render.Canvas, st *stroke, off geom.Point) {
	if len(st.pts) == 1 {
		c.FillCircle(sub(st.pts[0], off), st.width/2, st.colour)
		return
	}
	for i := 1; i < len(st.pts); i++ {
		c.DrawLine(sub(st.pts[i-1], off), sub(st.pts[i], off), st.colour, st.width)
		// round the joins so the stroke looks smooth
		c.FillCircle(sub(st.pts[i], off), st.width/2, st.colour)
	}
}

// solidImage builds an 18x18 swatch of a solid colour.
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
		ui.WithTitle("guie — paint"),
		ui.WithSize(720, 560),
	)

	pad := newSketchpad()

	// Toolbar: colour swatches, a brush-size slider, and Clear.
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
		sw.OnClick(func() { pad.colour = c })
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
