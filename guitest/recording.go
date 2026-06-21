package guitest

import (
	"image/color"
	"strings"

	"github.com/kpfaulkner/guie/geom"
)

// OpKind identifies a recorded drawing operation.
type OpKind int

const (
	OpFill            OpKind = iota // Fill(colour) over the current clip
	OpFillRect                      // FillRect(rect, colour)
	OpStrokeRect                    // StrokeRect(rect, colour, width)
	OpFillRoundRect                 // FillRoundRect(rect, radius, colour)
	OpStrokeRoundRect               // StrokeRoundRect(rect, radius, colour, width)
	OpDrawLine                      // DrawLine(a, b, colour, width)
	OpFillCircle                    // FillCircle(center=A, radius, colour)
	OpStrokeCircle                  // StrokeCircle(center=A, radius, colour, width)
	OpDrawText                      // DrawText(text, pos=A, colour)
	OpDrawImage                     // DrawImage(image into rect)
	OpPushClip                      // PushClip(rect)
	OpPopClip                       // PopClip()
)

// Op is a single recorded drawing call. Which fields are meaningful depends on
// Kind (see the OpKind constants).
type Op struct {
	Kind   OpKind
	Rect   geom.Rect
	A, B   geom.Point
	Colour color.Color
	Text   string
	Width  float64
	Radius float64
}

// Recording is the list of drawing operations produced during one frame, in the
// order they were issued (back-to-front). A headless Canvas appends to it; tests
// query it to assert what was painted without a real surface.
type Recording struct {
	Ops  []Op
	Size geom.Size
}

// Texts returns the strings of every DrawText op, in draw order.
func (r *Recording) Texts() []string {
	var out []string
	for _, op := range r.Ops {
		if op.Kind == OpDrawText {
			out = append(out, op.Text)
		}
	}
	return out
}

// HasText reports whether any DrawText op drew exactly s.
func (r *Recording) HasText(s string) bool {
	for _, op := range r.Ops {
		if op.Kind == OpDrawText && op.Text == s {
			return true
		}
	}
	return false
}

// TextContaining reports whether any DrawText op's text contains substr.
func (r *Recording) TextContaining(substr string) bool {
	for _, op := range r.Ops {
		if op.Kind == OpDrawText && strings.Contains(op.Text, substr) {
			return true
		}
	}
	return false
}

// Count returns how many ops of the given kind were recorded.
func (r *Recording) Count(kind OpKind) int {
	n := 0
	for _, op := range r.Ops {
		if op.Kind == kind {
			n++
		}
	}
	return n
}

// OpsOfKind returns every op of the given kind, in draw order.
func (r *Recording) OpsOfKind(kind OpKind) []Op {
	var out []Op
	for _, op := range r.Ops {
		if op.Kind == kind {
			out = append(out, op)
		}
	}
	return out
}

// FillsOfColour returns the rectangles of every FillRect / FillRoundRect op whose
// colour matches c (compared by RGBA). Useful for asserting selection/hover
// highlights or themed backgrounds were painted.
func (r *Recording) FillsOfColour(c color.Color) []geom.Rect {
	var out []geom.Rect
	want := rgba(c)
	for _, op := range r.Ops {
		if op.Kind != OpFillRect && op.Kind != OpFillRoundRect {
			continue
		}
		if op.Colour != nil && rgba(op.Colour) == want {
			out = append(out, op.Rect)
		}
	}
	return out
}

// TextAt returns the text of the first DrawText op whose origin is within tol
// pixels of (x,y), or "" if none. Handy for asserting a label landed where
// layout placed it.
func (r *Recording) TextAt(x, y, tol float64) string {
	for _, op := range r.Ops {
		if op.Kind != OpDrawText {
			continue
		}
		if absf(op.A.X-x) <= tol && absf(op.A.Y-y) <= tol {
			return op.Text
		}
	}
	return ""
}

type rgbaKey struct{ r, g, b, a uint32 }

func rgba(c color.Color) rgbaKey {
	r, g, b, a := c.RGBA()
	return rgbaKey{r, g, b, a}
}

func absf(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
