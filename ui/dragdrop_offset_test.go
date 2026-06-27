package ui

import (
	"image/color"
	"testing"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
)

// recCanvas is a render.Canvas that records the (already-translated) coordinates
// each call receives, so offsetCanvas's translation can be asserted.
type recCanvas struct {
	rect    geom.Rect
	a, b    geom.Point
	subRect geom.Rect
}

func (r *recCanvas) Size() geom.Size                                        { return geom.Size{} }
func (r *recCanvas) PushClip(c geom.Rect)                                   { r.rect = c }
func (r *recCanvas) PopClip()                                               {}
func (r *recCanvas) Fill(c color.Color)                                     {}
func (r *recCanvas) FillRect(rc geom.Rect, c color.Color)                   { r.rect = rc }
func (r *recCanvas) StrokeRect(rc geom.Rect, c color.Color, w float64)      { r.rect = rc }
func (r *recCanvas) FillRoundRect(rc geom.Rect, rad float64, c color.Color) { r.rect = rc }
func (r *recCanvas) StrokeRoundRect(rc geom.Rect, rad float64, c color.Color, w float64) {
	r.rect = rc
}
func (r *recCanvas) DrawLine(a, b geom.Point, c color.Color, w float64) { r.a, r.b = a, b }
func (r *recCanvas) FillCircle(center geom.Point, rad float64, c color.Color) {
	r.a = center
}
func (r *recCanvas) StrokeCircle(center geom.Point, rad float64, c color.Color, w float64) {
	r.a = center
}
func (r *recCanvas) DrawText(s string, pos geom.Point, f render.FontFace, c color.Color) {
	r.a = pos
}
func (r *recCanvas) MeasureText(s string, f render.FontFace) geom.Size { return geom.Size{} }
func (r *recCanvas) DrawImage(img render.Image, dst geom.Rect)         { r.rect = dst }
func (r *recCanvas) SubCanvas(rc geom.Rect) render.Canvas {
	r.subRect = rc
	return r
}

func TestOffsetCanvasTranslatesRects(t *testing.T) {
	rc := &recCanvas{}
	o := &offsetCanvas{Canvas: rc, dx: 100, dy: 200}
	in := geom.Rect{X: 1, Y: 2, W: 3, H: 4}
	want := geom.Rect{X: 101, Y: 202, W: 3, H: 4}

	checks := []struct {
		name string
		call func()
	}{
		{"PushClip", func() { o.PushClip(in) }},
		{"FillRect", func() { o.FillRect(in, color.Black) }},
		{"StrokeRect", func() { o.StrokeRect(in, color.Black, 1) }},
		{"FillRoundRect", func() { o.FillRoundRect(in, 2, color.Black) }},
		{"StrokeRoundRect", func() { o.StrokeRoundRect(in, 2, color.Black, 1) }},
		{"DrawImage", func() { o.DrawImage(nil, in) }},
	}
	for _, c := range checks {
		rc.rect = geom.Rect{}
		c.call()
		if rc.rect != want {
			t.Errorf("%s: got %+v, want %+v", c.name, rc.rect, want)
		}
	}
}

func TestOffsetCanvasTranslatesPoints(t *testing.T) {
	rc := &recCanvas{}
	o := &offsetCanvas{Canvas: rc, dx: 10, dy: 20}

	o.DrawLine(geom.Point{X: 1, Y: 2}, geom.Point{X: 3, Y: 4}, color.Black, 1)
	if rc.a != (geom.Point{X: 11, Y: 22}) || rc.b != (geom.Point{X: 13, Y: 24}) {
		t.Errorf("DrawLine: got a=%+v b=%+v", rc.a, rc.b)
	}

	o.FillCircle(geom.Point{X: 5, Y: 6}, 3, color.Black)
	if rc.a != (geom.Point{X: 15, Y: 26}) {
		t.Errorf("FillCircle center: got %+v", rc.a)
	}

	o.StrokeCircle(geom.Point{X: 7, Y: 8}, 3, color.Black, 1)
	if rc.a != (geom.Point{X: 17, Y: 28}) {
		t.Errorf("StrokeCircle center: got %+v", rc.a)
	}

	o.DrawText("x", geom.Point{X: 1, Y: 1}, nil, color.Black)
	if rc.a != (geom.Point{X: 11, Y: 21}) {
		t.Errorf("DrawText pos: got %+v", rc.a)
	}
}

func TestOffsetCanvasSubCanvasKeepsOffset(t *testing.T) {
	rc := &recCanvas{}
	o := &offsetCanvas{Canvas: rc, dx: 100, dy: 200}

	sub := o.SubCanvas(geom.Rect{X: 1, Y: 2, W: 3, H: 4})
	if rc.subRect != (geom.Rect{X: 101, Y: 202, W: 3, H: 4}) {
		t.Errorf("SubCanvas rect: got %+v", rc.subRect)
	}
	so, ok := sub.(*offsetCanvas)
	if !ok || so.dx != 100 || so.dy != 200 {
		t.Errorf("SubCanvas should return an offsetCanvas with the same offset, got %#v", sub)
	}
}

func TestSnapshotGhostNilTargetIsSafe(t *testing.T) {
	g := &snapshotGhost{} // nil target
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("nil-target ghost panicked: %v", r)
		}
	}()
	g.DrawGhost(&recCanvas{}, geom.Point{X: 1, Y: 1}) // no-op
	g.dispose()                                       // no-op
}

func TestNewSnapshotGhostZeroBoundsReturnsNil(t *testing.T) {
	// A source with no drawable area yields no ghost (the w<=0 / h<=0 guard).
	if g := newSnapshotGhost(newStub(0, 0)); g != nil {
		t.Errorf("zero-bounds source should produce no ghost, got %#v", g)
	}
}
