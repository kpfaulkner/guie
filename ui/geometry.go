package ui

// Point is an integer 2D coordinate, in pixels.
type Point struct {
	X, Y int
}

// Add returns the component-wise sum of p and q.
func (p Point) Add(q Point) Point { return Point{p.X + q.X, p.Y + q.Y} }

// Sub returns the component-wise difference p - q.
func (p Point) Sub(q Point) Point { return Point{p.X - q.X, p.Y - q.Y} }

// Rect is an axis-aligned rectangle defined by a top-left origin and a size.
type Rect struct {
	X, Y, W, H int
}

// Origin returns the top-left corner of r.
func (r Rect) Origin() Point { return Point{r.X, r.Y} }

// Size returns the width and height of r as a Point.
func (r Rect) Size() Point { return Point{r.W, r.H} }

// Add returns r translated by p.
func (r Rect) Add(p Point) Rect { return Rect{r.X + p.X, r.Y + p.Y, r.W, r.H} }

// Contains reports whether p lies within r (inclusive of the top-left edge,
// exclusive of the bottom-right edge).
func (r Rect) Contains(p Point) bool {
	return p.X >= r.X && p.X < r.X+r.W && p.Y >= r.Y && p.Y < r.Y+r.H
}
