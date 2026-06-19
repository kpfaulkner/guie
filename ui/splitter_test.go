package ui

import (
	"testing"

	"github.com/kpfaulkner/uiframework/geom"
)

func TestSplitHorizontalLayoutHalf(t *testing.T) {
	a, b := newStub(10, 10), newStub(10, 10)
	sp := HSplit(a, b)
	sp.SetBounds(geom.Rect{X: 0, Y: 0, W: 100, H: 50})
	sp.Layout()

	// avail = 100 - 6 = 94; half each = 47.
	wantRect(t, "first", a.Bounds(), geom.Rect{X: 0, Y: 0, W: 47, H: 50})
	wantRect(t, "second", b.Bounds(), geom.Rect{X: 53, Y: 0, W: 47, H: 50})
}

func TestSplitVerticalLayoutHalf(t *testing.T) {
	a, b := newStub(10, 10), newStub(10, 10)
	sp := VSplit(a, b)
	sp.SetBounds(geom.Rect{X: 0, Y: 0, W: 50, H: 100})
	sp.Layout()

	wantRect(t, "first", a.Bounds(), geom.Rect{X: 0, Y: 0, W: 50, H: 47})
	wantRect(t, "second", b.Bounds(), geom.Rect{X: 0, Y: 53, W: 50, H: 47})
}

func TestSplitDragMovesDivider(t *testing.T) {
	a, b := newStub(1, 1), newStub(1, 1)
	sp := HSplit(a, b, SplitMinSizes(10, 10))
	sp.SetBounds(geom.Rect{X: 0, Y: 0, W: 100, H: 50})

	// Grab the divider (centered around x=50) and drag it to x=20.
	sp.HandleEvent(&Event{Type: EventPointerDown, Pos: geom.Point{X: 50, Y: 25}})
	sp.HandleEvent(&Event{Type: EventPointerMove, Pos: geom.Point{X: 20, Y: 25}})
	sp.HandleEvent(&Event{Type: EventPointerUp, Pos: geom.Point{X: 20, Y: 25}})
	sp.Layout()

	// fs = 20 - 3 (half thickness) = 17, within [10, 84].
	if !approx(a.Bounds().W, 17) {
		t.Fatalf("first pane width after drag should be 17, got %v", a.Bounds().W)
	}
}

func TestSplitDragClampsToMin(t *testing.T) {
	a, b := newStub(1, 1), newStub(1, 1)
	sp := HSplit(a, b, SplitMinSizes(10, 10))
	sp.SetBounds(geom.Rect{X: 0, Y: 0, W: 100, H: 50})

	sp.HandleEvent(&Event{Type: EventPointerDown, Pos: geom.Point{X: 50, Y: 25}})
	sp.HandleEvent(&Event{Type: EventPointerMove, Pos: geom.Point{X: 0, Y: 25}}) // far left
	sp.Layout()
	if !approx(a.Bounds().W, 10) {
		t.Fatalf("first pane should clamp to its minimum (10), got %v", a.Bounds().W)
	}

	sp.HandleEvent(&Event{Type: EventPointerMove, Pos: geom.Point{X: 999, Y: 25}}) // far right
	sp.Layout()
	// avail 94, second min 10 → first max 84.
	if !approx(a.Bounds().W, 84) {
		t.Fatalf("first pane should clamp so second keeps its minimum (84), got %v", a.Bounds().W)
	}
}

func TestSplitChildren(t *testing.T) {
	a, b := newStub(1, 1), newStub(1, 1)
	sp := HSplit(a, b)
	kids := sp.Children()
	if len(kids) != 2 || kids[0] != Widget(a) || kids[1] != Widget(b) {
		t.Fatalf("Children should return both panes in order")
	}
}

func TestSplitMinSize(t *testing.T) {
	sp := HSplit(newStub(40, 30), newStub(20, 50), SplitMinSizes(10, 10))
	got := sp.MinSize()
	// width = max(10,40) + max(10,20) + 6 = 66; height = max(30,50) = 50.
	if !approx(got.W, 66) || !approx(got.H, 50) {
		t.Fatalf("MinSize got %+v, want {66 50}", got)
	}
}
