// Package guitest is a headless backend and harness for testing guie apps
// without a window or GPU. It implements the render seam (Driver, Canvas,
// FontFace, Image) so an App can be driven frame by frame from a Go test:
// synthesize input, step the loop, and assert against the recorded drawing
// operations or the widget tree.
//
// Typical use:
//
//	h := guitest.New(200, 100)
//	btn := ui.NewButton("Save")
//	clicked := false
//	btn.OnClick(func() { clicked = true })
//	h.SetContent(btn)
//	h.Click(20, 20)
//	if !clicked { t.Fatal("button did not fire") }
//
// The font has simple, deterministic metrics (independent of the bundled TTF)
// so text measurements and layout are predictable in assertions. Nothing here
// allocates GPU resources; note that ui.NewRenderTarget still uses the real
// backend, so tests should avoid it (as they already do).
package guitest

import (
	"errors"
	"image/color"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
)

// --- Font -------------------------------------------------------------------

// font is a deterministic, GPU-free render.FontFace. Advance width is a fixed
// fraction of the point size per rune, so a string's measured width is just
// len(runes)*advance — easy to reason about in tests.
type font struct {
	size    float64
	advance float64
	metrics render.FontMetrics
}

// NewFont returns a headless font face at the given point size. Its metrics are
// synthetic but stable: ascent 0.8·size, descent 0.2·size, line height 1.25·size,
// and a per-rune advance of 0.6·size.
func NewFont(size float64) render.FontFace {
	if size <= 0 {
		size = 16
	}
	return &font{
		size:    size,
		advance: size * 0.6,
		metrics: render.FontMetrics{
			Ascent:     size * 0.8,
			Descent:    size * 0.2,
			LineHeight: size * 1.25,
		},
	}
}

func (f *font) Metrics() render.FontMetrics { return f.metrics }

// Measure returns the size of s: the widest newline-separated line by the total
// block height (line height per line).
func (f *font) Measure(s string) geom.Size {
	var widest, lineRunes float64
	lines := 1
	for _, r := range s {
		if r == '\n' {
			lines++
			if lineRunes > widest {
				widest = lineRunes
			}
			lineRunes = 0
			continue
		}
		lineRunes++
	}
	if lineRunes > widest {
		widest = lineRunes
	}
	return geom.Size{W: widest * f.advance, H: f.metrics.LineHeight * float64(lines)}
}

// --- Canvas -----------------------------------------------------------------

// canvas is a headless render.Canvas that records every drawing call into a
// Recording instead of rasterizing. Clipping is recorded but not enforced (all
// coordinates are absolute), and SubCanvas shares the same Recording.
type canvas struct {
	rec  *Recording
	size geom.Size
}

func (c *canvas) add(op Op)            { c.rec.Ops = append(c.rec.Ops, op) }
func (c *canvas) Size() geom.Size      { return c.size }
func (c *canvas) PushClip(r geom.Rect) { c.add(Op{Kind: OpPushClip, Rect: r}) }
func (c *canvas) PopClip()             { c.add(Op{Kind: OpPopClip}) }
func (c *canvas) Fill(col color.Color) { c.add(Op{Kind: OpFill, Color: col}) }

func (c *canvas) FillRect(r geom.Rect, col color.Color) {
	c.add(Op{Kind: OpFillRect, Rect: r, Color: col})
}
func (c *canvas) StrokeRect(r geom.Rect, col color.Color, width float64) {
	c.add(Op{Kind: OpStrokeRect, Rect: r, Color: col, Width: width})
}
func (c *canvas) FillRoundRect(r geom.Rect, radius float64, col color.Color) {
	c.add(Op{Kind: OpFillRoundRect, Rect: r, Radius: radius, Color: col})
}
func (c *canvas) StrokeRoundRect(r geom.Rect, radius float64, col color.Color, width float64) {
	c.add(Op{Kind: OpStrokeRoundRect, Rect: r, Radius: radius, Color: col, Width: width})
}
func (c *canvas) DrawLine(a, b geom.Point, col color.Color, width float64) {
	c.add(Op{Kind: OpDrawLine, A: a, B: b, Color: col, Width: width})
}
func (c *canvas) FillCircle(center geom.Point, radius float64, col color.Color) {
	c.add(Op{Kind: OpFillCircle, A: center, Radius: radius, Color: col})
}
func (c *canvas) StrokeCircle(center geom.Point, radius float64, col color.Color, width float64) {
	c.add(Op{Kind: OpStrokeCircle, A: center, Radius: radius, Color: col, Width: width})
}
func (c *canvas) DrawText(s string, pos geom.Point, face render.FontFace, col color.Color) {
	c.add(Op{Kind: OpDrawText, Text: s, A: pos, Color: col})
}
func (c *canvas) MeasureText(s string, face render.FontFace) geom.Size {
	if face == nil {
		return geom.Size{}
	}
	return face.Measure(s)
}
func (c *canvas) DrawImage(img render.Image, dst geom.Rect) {
	c.add(Op{Kind: OpDrawImage, Rect: dst})
}

// SubCanvas records into the same Recording (clipping is not enforced headlessly).
func (c *canvas) SubCanvas(r geom.Rect) render.Canvas {
	return &canvas{rec: c.rec, size: geom.Size{W: r.W, H: r.H}}
}

// --- Driver -----------------------------------------------------------------

// driver is a headless render.Driver. Run does not block: it captures the
// framework's hooks, fires the initial Resize, and returns. The Harness then
// invokes the hooks on demand via step.
type driver struct {
	hooks  render.Hooks
	width  int
	height int

	// IME state recorded via render.IMEController, for test assertions.
	imeEnabled bool
	imeRect    geom.Rect
}

func newDriver() *driver { return &driver{} }

// SetIMEEnabled and SetIMERect satisfy render.IMEController so the harness
// exercises the App's IME wiring (focus enables IME; the caret rect is reported
// each frame). The values are recorded for assertions.
func (d *driver) SetIMEEnabled(on bool)  { d.imeEnabled = on }
func (d *driver) SetIMERect(r geom.Rect) { d.imeRect = r }

func (d *driver) Run(cfg render.Config, hooks render.Hooks) error {
	d.hooks = hooks
	d.width, d.height = cfg.Width, cfg.Height
	if hooks.Resize != nil {
		hooks.Resize(d.width, d.height)
	}
	return nil
}

// resize reports a new logical surface size to the framework.
func (d *driver) resize(w, h int) {
	d.width, d.height = w, h
	if d.hooks.Resize != nil {
		d.hooks.Resize(w, h)
	}
}

// step runs one frame: Update with in, then Draw onto a fresh Recording, which
// it returns. A render.ErrTerminated from Update is treated as a clean stop.
func (d *driver) step(in render.InputState) (*Recording, error) {
	rec := &Recording{Size: geom.Size{W: float64(d.width), H: float64(d.height)}}
	if d.hooks.Update != nil {
		if err := d.hooks.Update(in); err != nil {
			if errors.Is(err, render.ErrTerminated) {
				return rec, nil
			}
			return rec, err
		}
	}
	if d.hooks.Draw != nil {
		d.hooks.Draw(&canvas{rec: rec, size: rec.Size})
	}
	return rec, nil
}
