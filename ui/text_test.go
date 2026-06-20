package ui

import (
	"image/color"
	"testing"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
)

// recordingCanvas is a no-op render.Canvas that records DrawText calls, for
// asserting text layout without a GPU.
type recordingCanvas struct {
	texts []drawnText
}

type drawnText struct {
	s   string
	pos geom.Point
}

func (c *recordingCanvas) Size() geom.Size                                          { return geom.Size{W: 1000, H: 1000} }
func (c *recordingCanvas) PushClip(geom.Rect)                                       {}
func (c *recordingCanvas) PopClip()                                                 {}
func (c *recordingCanvas) Fill(color.Color)                                         {}
func (c *recordingCanvas) FillRect(geom.Rect, color.Color)                          {}
func (c *recordingCanvas) StrokeRect(geom.Rect, color.Color, float64)               {}
func (c *recordingCanvas) FillRoundRect(geom.Rect, float64, color.Color)            {}
func (c *recordingCanvas) StrokeRoundRect(geom.Rect, float64, color.Color, float64) {}
func (c *recordingCanvas) DrawLine(geom.Point, geom.Point, color.Color, float64)    {}
func (c *recordingCanvas) FillCircle(geom.Point, float64, color.Color)              {}
func (c *recordingCanvas) StrokeCircle(geom.Point, float64, color.Color, float64)   {}
func (c *recordingCanvas) DrawImage(render.Image, geom.Rect)                        {}
func (c *recordingCanvas) MeasureText(s string, f render.FontFace) geom.Size {
	return f.Measure(s)
}
func (c *recordingCanvas) SubCanvas(geom.Rect) render.Canvas { return c }
func (c *recordingCanvas) DrawText(s string, pos geom.Point, _ render.FontFace, _ color.Color) {
	c.texts = append(c.texts, drawnText{s: s, pos: pos})
}

// TestDrawTextStacksLines verifies multi-line text is drawn as separate,
// non-overlapping lines (one DrawText per line, increasing Y), rather than a
// single string with an embedded newline.
func TestDrawTextStacksLines(t *testing.T) {
	f := DefaultFont(14)
	rect := geom.Rect{X: 0, Y: 0, W: 300, H: 100}
	c := &recordingCanvas{}

	drawText(c, "first line\nsecond line", rect, geom.AlignStart, f, color.White)

	if len(c.texts) != 2 {
		t.Fatalf("expected 2 draw calls (one per line), got %d", len(c.texts))
	}
	if c.texts[0].s != "first line" || c.texts[1].s != "second line" {
		t.Fatalf("lines not split correctly: %q, %q", c.texts[0].s, c.texts[1].s)
	}
	lineH := f.Metrics().LineHeight
	if got := c.texts[1].pos.Y - c.texts[0].pos.Y; got < lineH-0.01 || got > lineH+0.01 {
		t.Fatalf("second line should be one line height below the first: got dy=%.3f want %.3f", got, lineH)
	}
}

// TestDrawTextSingleLineCentered verifies the single-line path still centers
// vertically exactly as vCenterY did (backward compatibility).
func TestDrawTextSingleLineCentered(t *testing.T) {
	f := DefaultFont(14)
	rect := geom.Rect{X: 10, Y: 20, W: 200, H: 50}
	c := &recordingCanvas{}

	drawText(c, "solo", rect, geom.AlignStart, f, color.White)

	if len(c.texts) != 1 {
		t.Fatalf("expected 1 draw call, got %d", len(c.texts))
	}
	want := vCenterY(f, rect.Y, rect.H)
	if got := c.texts[0].pos.Y; got < want-0.01 || got > want+0.01 {
		t.Fatalf("single-line Y should match vCenterY: got %.3f want %.3f", got, want)
	}
}
