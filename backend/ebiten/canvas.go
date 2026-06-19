package ebitenbackend

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/kpfaulkner/uiframework/geom"
	"github.com/kpfaulkner/uiframework/render"
)

// canvas implements render.Canvas over an *ebiten.Image.
//
// Clipping is implemented with sub-images: each PushClip intersects the
// requested rectangle with the current clip and pushes a sub-image restricted
// to it. Draw calls target the sub-image on top of the stack, so anything drawn
// outside the active clip is discarded.
type canvas struct {
	base  *ebiten.Image
	stack []clipEntry
}

type clipEntry struct {
	target *ebiten.Image
	rect   geom.Rect
}

func newCanvas() *canvas { return &canvas{} }

// reset rebinds the canvas to surface for a new frame, clearing the clip stack.
func (c *canvas) reset(surface *ebiten.Image) {
	c.base = surface
	b := surface.Bounds()
	full := geom.Rect{X: float64(b.Min.X), Y: float64(b.Min.Y), W: float64(b.Dx()), H: float64(b.Dy())}
	c.stack = append(c.stack[:0], clipEntry{target: surface, rect: full})
}

func (c *canvas) top() clipEntry { return c.stack[len(c.stack)-1] }

func (c *canvas) Size() geom.Size {
	b := c.base.Bounds()
	return geom.Size{W: float64(b.Dx()), H: float64(b.Dy())}
}

func (c *canvas) PushClip(r geom.Rect) {
	clip := c.top().rect.Intersect(r)
	sub := c.base.SubImage(toImageRect(clip)).(*ebiten.Image)
	c.stack = append(c.stack, clipEntry{target: sub, rect: clip})
}

func (c *canvas) PopClip() {
	if len(c.stack) > 1 {
		c.stack = c.stack[:len(c.stack)-1]
	}
}

func (c *canvas) Fill(clr color.Color) {
	c.top().target.Fill(clr)
}

func (c *canvas) FillRect(r geom.Rect, clr color.Color) {
	vector.FillRect(c.top().target, float32(r.X), float32(r.Y), float32(r.W), float32(r.H), clr, true)
}

func (c *canvas) StrokeRect(r geom.Rect, clr color.Color, width float64) {
	vector.StrokeRect(c.top().target, float32(r.X), float32(r.Y), float32(r.W), float32(r.H), float32(width), clr, true)
}

func (c *canvas) DrawLine(a, b geom.Point, clr color.Color, width float64) {
	vector.StrokeLine(c.top().target, float32(a.X), float32(a.Y), float32(b.X), float32(b.Y), float32(width), clr, true)
}

func (c *canvas) FillCircle(center geom.Point, radius float64, clr color.Color) {
	vector.FillCircle(c.top().target, float32(center.X), float32(center.Y), float32(radius), clr, true)
}

func (c *canvas) StrokeCircle(center geom.Point, radius float64, clr color.Color, width float64) {
	vector.StrokeCircle(c.top().target, float32(center.X), float32(center.Y), float32(radius), float32(width), clr, true)
}

func (c *canvas) DrawText(s string, pos geom.Point, face render.FontFace, clr color.Color) {
	ff, ok := face.(*fontFace)
	if !ok || ff == nil {
		return
	}
	op := &text.DrawOptions{}
	op.GeoM.Translate(pos.X, pos.Y)
	op.ColorScale.ScaleWithColor(clr)
	text.Draw(c.top().target, s, ff.face, op)
}

func (c *canvas) MeasureText(s string, face render.FontFace) geom.Size {
	if face == nil {
		return geom.Size{}
	}
	return face.Measure(s)
}

func (c *canvas) DrawImage(img render.Image, dst geom.Rect) {
	ih, ok := img.(*imageHandle)
	if !ok || ih == nil {
		return
	}
	src := ih.img.Bounds()
	op := &ebiten.DrawImageOptions{}
	if src.Dx() > 0 && src.Dy() > 0 {
		op.GeoM.Scale(dst.W/float64(src.Dx()), dst.H/float64(src.Dy()))
	}
	op.GeoM.Translate(dst.X, dst.Y)
	c.top().target.DrawImage(ih.img, op)
}

func (c *canvas) SubCanvas(r geom.Rect) render.Canvas {
	clip := c.top().rect.Intersect(r)
	sub := c.base.SubImage(toImageRect(clip)).(*ebiten.Image)
	return &canvas{
		base:  c.base,
		stack: []clipEntry{{target: sub, rect: clip}},
	}
}

// toImageRect converts a logical rectangle to integer pixel bounds, rounding
// outward so nothing within r is clipped away.
func toImageRect(r geom.Rect) image.Rectangle {
	return image.Rect(
		int(math.Floor(r.X)),
		int(math.Floor(r.Y)),
		int(math.Ceil(r.X+r.W)),
		int(math.Ceil(r.Y+r.H)),
	)
}
