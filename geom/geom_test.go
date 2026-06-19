package geom

import "testing"

func TestPointAddSub(t *testing.T) {
	a := Point{X: 3, Y: 4}
	b := Point{X: 1, Y: 2}
	if got := a.Add(b); got != (Point{X: 4, Y: 6}) {
		t.Errorf("Add: got %+v", got)
	}
	if got := a.Sub(b); got != (Point{X: 2, Y: 2}) {
		t.Errorf("Sub: got %+v", got)
	}
}

func TestSizeMaxAdd(t *testing.T) {
	if got := (Size{W: 10, H: 5}).Max(Size{W: 4, H: 8}); got != (Size{W: 10, H: 8}) {
		t.Errorf("Max: got %+v", got)
	}
	if got := (Size{W: 10, H: 5}).Add(2, 3); got != (Size{W: 12, H: 8}) {
		t.Errorf("Add: got %+v", got)
	}
}

func TestRectAccessors(t *testing.T) {
	r := Rect{X: 10, Y: 20, W: 30, H: 40}
	if r.Min() != (Point{X: 10, Y: 20}) {
		t.Errorf("Min: got %+v", r.Min())
	}
	if r.Max() != (Point{X: 40, Y: 60}) {
		t.Errorf("Max: got %+v", r.Max())
	}
	if r.Size() != (Size{W: 30, H: 40}) {
		t.Errorf("Size: got %+v", r.Size())
	}
	if r.Center() != (Point{X: 25, Y: 40}) {
		t.Errorf("Center: got %+v", r.Center())
	}
	if got := r.Add(Point{X: 5, Y: 5}); got != (Rect{X: 15, Y: 25, W: 30, H: 40}) {
		t.Errorf("Add: got %+v", got)
	}
}

func TestFromMinMax(t *testing.T) {
	if got := FromMinMax(Point{X: 1, Y: 2}, Point{X: 5, Y: 9}); got != (Rect{X: 1, Y: 2, W: 4, H: 7}) {
		t.Errorf("FromMinMax: got %+v", got)
	}
}

func TestRectContains(t *testing.T) {
	r := Rect{X: 0, Y: 0, W: 10, H: 10}
	cases := []struct {
		p    Point
		want bool
	}{
		{Point{X: 0, Y: 0}, true},   // inclusive top-left
		{Point{X: 5, Y: 5}, true},   // inside
		{Point{X: 10, Y: 5}, false}, // exclusive right edge
		{Point{X: 5, Y: 10}, false}, // exclusive bottom edge
		{Point{X: -1, Y: 5}, false},
	}
	for _, c := range cases {
		if got := r.Contains(c.p); got != c.want {
			t.Errorf("Contains(%+v): got %v want %v", c.p, got, c.want)
		}
	}
}

func TestRectInset(t *testing.T) {
	r := Rect{X: 0, Y: 0, W: 100, H: 100}
	got := r.Inset(Insets{Top: 5, Right: 10, Bottom: 15, Left: 20})
	want := Rect{X: 20, Y: 5, W: 70, H: 80}
	if got != want {
		t.Errorf("Inset: got %+v want %+v", got, want)
	}
	if u := UniformInsets(8); u != (Insets{8, 8, 8, 8}) {
		t.Errorf("UniformInsets: got %+v", u)
	}
}

func TestRectIntersect(t *testing.T) {
	a := Rect{X: 0, Y: 0, W: 10, H: 10}
	b := Rect{X: 5, Y: 5, W: 10, H: 10}
	if got := a.Intersect(b); got != (Rect{X: 5, Y: 5, W: 5, H: 5}) {
		t.Errorf("overlap: got %+v", got)
	}
	// Non-overlapping returns the zero rect.
	if got := a.Intersect(Rect{X: 50, Y: 50, W: 5, H: 5}); got != (Rect{}) {
		t.Errorf("disjoint should be zero, got %+v", got)
	}
}

func TestRectEmpty(t *testing.T) {
	if !(Rect{W: 0, H: 10}).Empty() {
		t.Error("zero width should be empty")
	}
	if !(Rect{W: 10, H: -1}).Empty() {
		t.Error("negative height should be empty")
	}
	if (Rect{W: 10, H: 10}).Empty() {
		t.Error("positive area should not be empty")
	}
}
