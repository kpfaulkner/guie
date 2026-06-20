package ebitenbackend

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
)

// canvas implements render.Canvas over an *ebiten.Image.
//
// Coordinates from the framework are logical pixels; the backing surface is
// physical pixels. The canvas carries a device scale factor and multiplies
// every draw coordinate by it, so widgets stay device-independent while
// rendering happens at the display's native resolution (crisp on HiDPI). The
// clip stack is kept in logical coordinates and converted to physical pixel
// bounds only when taking sub-images.
//
// Clipping is implemented with sub-images: each PushClip intersects the
// requested rectangle with the current clip and pushes a sub-image restricted
// to it. Draw calls target the sub-image on top of the stack, so anything drawn
// outside the active clip is discarded.
type canvas struct {
	base  *ebiten.Image
	stack []clipEntry
	scale float64 // device scale factor: logical → physical pixels
}

type clipEntry struct {
	target *ebiten.Image
	rect   geom.Rect // logical coordinates
}

func newCanvas() *canvas { return &canvas{scale: 1} }

// reset rebinds the canvas to surface for a new frame at the given device scale,
// clearing the clip stack.
func (c *canvas) reset(surface *ebiten.Image, scale float64) {
	if scale <= 0 {
		scale = 1
	}
	c.base = surface
	c.scale = scale
	b := surface.Bounds()
	full := geom.Rect{
		X: float64(b.Min.X) / scale,
		Y: float64(b.Min.Y) / scale,
		W: float64(b.Dx()) / scale,
		H: float64(b.Dy()) / scale,
	}
	c.stack = append(c.stack[:0], clipEntry{target: surface, rect: full})
}

// scaleRect maps a logical rectangle to physical coordinates.
func (c *canvas) scaleRect(r geom.Rect) geom.Rect {
	return geom.Rect{X: r.X * c.scale, Y: r.Y * c.scale, W: r.W * c.scale, H: r.H * c.scale}
}

// phys converts a logical rectangle to integer physical pixel bounds.
func (c *canvas) phys(r geom.Rect) image.Rectangle {
	return toImageRect(c.scaleRect(r))
}

func (c *canvas) top() clipEntry { return c.stack[len(c.stack)-1] }

func (c *canvas) Size() geom.Size {
	b := c.base.Bounds()
	return geom.Size{W: float64(b.Dx()) / c.scale, H: float64(b.Dy()) / c.scale}
}

func (c *canvas) PushClip(r geom.Rect) {
	clip := c.top().rect.Intersect(r)
	sub := c.base.SubImage(c.phys(clip)).(*ebiten.Image)
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
	r = c.scaleRect(r)
	vector.FillRect(c.top().target, float32(r.X), float32(r.Y), float32(r.W), float32(r.H), clr, true)
}

func (c *canvas) StrokeRect(r geom.Rect, clr color.Color, width float64) {
	r = c.scaleRect(r)
	vector.StrokeRect(c.top().target, float32(r.X), float32(r.Y), float32(r.W), float32(r.H), float32(width*c.scale), clr, true)
}

// roundRectPath builds a rounded-rectangle path, clamping the radius to half the
// smaller side.
func roundRectPath(r geom.Rect, radius float64) *vector.Path {
	x, y, w, h := float32(r.X), float32(r.Y), float32(r.W), float32(r.H)
	rad := float32(radius)
	if rad > w/2 {
		rad = w / 2
	}
	if rad > h/2 {
		rad = h / 2
	}
	var p vector.Path
	p.MoveTo(x+rad, y)
	p.LineTo(x+w-rad, y)
	p.ArcTo(x+w, y, x+w, y+rad, rad)
	p.LineTo(x+w, y+h-rad)
	p.ArcTo(x+w, y+h, x+w-rad, y+h, rad)
	p.LineTo(x+rad, y+h)
	p.ArcTo(x, y+h, x, y+h-rad, rad)
	p.LineTo(x, y+rad)
	p.ArcTo(x, y, x+rad, y, rad)
	p.Close()
	return &p
}

func (c *canvas) FillRoundRect(r geom.Rect, radius float64, clr color.Color) {
	if radius <= 0 {
		c.FillRect(r, clr)
		return
	}
	op := &vector.DrawPathOptions{}
	op.AntiAlias = true
	op.ColorScale.ScaleWithColor(clr)
	vector.FillPath(c.top().target, roundRectPath(c.scaleRect(r), radius*c.scale), nil, op)
}

func (c *canvas) StrokeRoundRect(r geom.Rect, radius float64, clr color.Color, width float64) {
	if radius <= 0 {
		c.StrokeRect(r, clr, width)
		return
	}
	sop := &vector.StrokeOptions{Width: float32(width * c.scale), MiterLimit: 10}
	op := &vector.DrawPathOptions{}
	op.AntiAlias = true
	op.ColorScale.ScaleWithColor(clr)
	vector.StrokePath(c.top().target, roundRectPath(c.scaleRect(r), radius*c.scale), sop, op)
}

func (c *canvas) DrawLine(a, b geom.Point, clr color.Color, width float64) {
	s := c.scale
	vector.StrokeLine(c.top().target, float32(a.X*s), float32(a.Y*s), float32(b.X*s), float32(b.Y*s), float32(width*s), clr, true)
}

func (c *canvas) FillCircle(center geom.Point, radius float64, clr color.Color) {
	s := c.scale
	vector.FillCircle(c.top().target, float32(center.X*s), float32(center.Y*s), float32(radius*s), clr, true)
}

func (c *canvas) StrokeCircle(center geom.Point, radius float64, clr color.Color, width float64) {
	s := c.scale
	vector.StrokeCircle(c.top().target, float32(center.X*s), float32(center.Y*s), float32(radius*s), float32(width*s), clr, true)
}

func (c *canvas) DrawText(s string, pos geom.Point, face render.FontFace, clr color.Color) {
	ff, ok := face.(*fontFace)
	if !ok || ff == nil {
		return
	}
	drawn := ff.face
	if c.scale != 1 {
		// Rasterize glyphs at physical size so text stays crisp on HiDPI
		// displays. The cached face keeps its logical size for measurement; this
		// is a shallow copy that shares the underlying font source.
		scaled := *ff.face
		scaled.Size = ff.face.Size * c.scale
		drawn = &scaled
	}
	op := &text.DrawOptions{}
	op.GeoM.Translate(pos.X*c.scale, pos.Y*c.scale)
	op.ColorScale.ScaleWithColor(clr)
	text.Draw(c.top().target, s, drawn, op)
}

func (c *canvas) MeasureText(s string, face render.FontFace) geom.Size {
	if face == nil {
		return geom.Size{}
	}
	return face.Measure(s)
}

// ebitenImager is implemented by backend image types that wrap an *ebiten.Image
// (both imageHandle and renderTarget), so DrawImage can blit either.
type ebitenImager interface{ ebitenImage() *ebiten.Image }

func (c *canvas) DrawImage(img render.Image, dst geom.Rect) {
	ei, ok := img.(ebitenImager)
	if !ok {
		return
	}
	src := ei.ebitenImage()
	if src == nil {
		return
	}
	b := src.Bounds()
	op := &ebiten.DrawImageOptions{}
	if b.Dx() > 0 && b.Dy() > 0 {
		op.GeoM.Scale(dst.W*c.scale/float64(b.Dx()), dst.H*c.scale/float64(b.Dy()))
	}
	op.GeoM.Translate(dst.X*c.scale, dst.Y*c.scale)
	c.top().target.DrawImage(src, op)
}

func (c *canvas) SubCanvas(r geom.Rect) render.Canvas {
	clip := c.top().rect.Intersect(r)
	sub := c.base.SubImage(c.phys(clip)).(*ebiten.Image)
	return &canvas{
		base:  c.base,
		stack: []clipEntry{{target: sub, rect: clip}},
		scale: c.scale,
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
