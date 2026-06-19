// Package geom provides backend-neutral 2D geometry types used throughout the
// framework. Coordinates are float64 logical pixels; the rendering backend is
// responsible for mapping them to physical device pixels.
package geom

import "math"

// Point is a 2D coordinate in logical pixels.
type Point struct {
	X, Y float64
}

// Add returns the component-wise sum of p and q.
func (p Point) Add(q Point) Point { return Point{p.X + q.X, p.Y + q.Y} }

// Sub returns the component-wise difference p - q.
func (p Point) Sub(q Point) Point { return Point{p.X - q.X, p.Y - q.Y} }

// Size is a width/height pair in logical pixels.
type Size struct {
	W, H float64
}

// Max returns a Size whose width and height are the larger of the two inputs'.
func (s Size) Max(o Size) Size { return Size{math.Max(s.W, o.W), math.Max(s.H, o.H)} }

// Add returns a Size grown by w and h.
func (s Size) Add(w, h float64) Size { return Size{s.W + w, s.H + h} }

// Rect is an axis-aligned rectangle defined by a top-left origin and a size.
type Rect struct {
	X, Y, W, H float64
}

// FromMinMax builds the Rect spanning from min to max.
func FromMinMax(min, max Point) Rect {
	return Rect{X: min.X, Y: min.Y, W: max.X - min.X, H: max.Y - min.Y}
}

// Min returns the top-left corner of r.
func (r Rect) Min() Point { return Point{r.X, r.Y} }

// Max returns the bottom-right corner of r.
func (r Rect) Max() Point { return Point{r.X + r.W, r.Y + r.H} }

// Size returns the dimensions of r.
func (r Rect) Size() Size { return Size{r.W, r.H} }

// Center returns the midpoint of r.
func (r Rect) Center() Point { return Point{r.X + r.W/2, r.Y + r.H/2} }

// Add returns r translated by p.
func (r Rect) Add(p Point) Rect { return Rect{r.X + p.X, r.Y + p.Y, r.W, r.H} }

// Empty reports whether r has zero or negative area.
func (r Rect) Empty() bool { return r.W <= 0 || r.H <= 0 }

// Contains reports whether p lies within r (inclusive of the top-left edge,
// exclusive of the bottom-right edge).
func (r Rect) Contains(p Point) bool {
	return p.X >= r.X && p.X < r.X+r.W && p.Y >= r.Y && p.Y < r.Y+r.H
}

// Inset returns r shrunk on each side by the corresponding inset.
func (r Rect) Inset(in Insets) Rect {
	return Rect{
		X: r.X + in.Left,
		Y: r.Y + in.Top,
		W: r.W - in.Left - in.Right,
		H: r.H - in.Top - in.Bottom,
	}
}

// Intersect returns the overlap of r and s, or the zero Rect if they do not
// overlap.
func (r Rect) Intersect(s Rect) Rect {
	min := Point{math.Max(r.X, s.X), math.Max(r.Y, s.Y)}
	max := Point{math.Min(r.X+r.W, s.X+s.W), math.Min(r.Y+r.H, s.Y+s.H)}
	if max.X <= min.X || max.Y <= min.Y {
		return Rect{}
	}
	return FromMinMax(min, max)
}

// Insets describes per-side padding or margins in logical pixels.
type Insets struct {
	Top, Right, Bottom, Left float64
}

// UniformInsets returns Insets with the same value on every side.
func UniformInsets(v float64) Insets { return Insets{v, v, v, v} }

// Alignment controls how a widget is positioned within the space a container
// gives it along one axis.
type Alignment int

const (
	// AlignStart pins the widget to the leading edge (left/top).
	AlignStart Alignment = iota
	// AlignCenter centers the widget within the available space.
	AlignCenter
	// AlignEnd pins the widget to the trailing edge (right/bottom).
	AlignEnd
	// AlignStretch expands the widget to fill the available space.
	AlignStretch
)

// Direction is the primary axis along which a layout arranges its children.
type Direction int

const (
	// Horizontal arranges children left to right.
	Horizontal Direction = iota
	// Vertical arranges children top to bottom.
	Vertical
)
