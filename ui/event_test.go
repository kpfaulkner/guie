package ui

import (
	"testing"

	"github.com/kpfaulkner/uiframework/geom"
	"github.com/kpfaulkner/uiframework/render"
)

// press simulates a left-button down at down, then up at up, against w.
func press(w Widget, down, up geom.Point) {
	d := Event{Type: EventPointerDown, Pos: down, Button: render.MouseLeft}
	w.HandleEvent(&d)
	u := Event{Type: EventPointerUp, Pos: up, Button: render.MouseLeft}
	w.HandleEvent(&u)
}

func TestButtonClickFires(t *testing.T) {
	clicks := 0
	b := NewButton("ok", OnClick(func() { clicks++ }))
	b.SetBounds(geom.Rect{X: 0, Y: 0, W: 100, H: 30})

	press(b, geom.Point{X: 10, Y: 10}, geom.Point{X: 20, Y: 15})
	if clicks != 1 {
		t.Fatalf("expected 1 click, got %d", clicks)
	}
}

func TestButtonReleaseOutsideNoClick(t *testing.T) {
	clicks := 0
	b := NewButton("ok", OnClick(func() { clicks++ }))
	b.SetBounds(geom.Rect{X: 0, Y: 0, W: 100, H: 30})

	// Pressed inside, released outside the bounds → not a click.
	press(b, geom.Point{X: 10, Y: 10}, geom.Point{X: 200, Y: 200})
	if clicks != 0 {
		t.Fatalf("expected 0 clicks on release outside, got %d", clicks)
	}
}

func TestButtonDisabledNoClick(t *testing.T) {
	clicks := 0
	b := NewButton("ok", OnClick(func() { clicks++ }))
	b.SetBounds(geom.Rect{X: 0, Y: 0, W: 100, H: 30})
	b.SetEnabled(false)

	press(b, geom.Point{X: 10, Y: 10}, geom.Point{X: 20, Y: 15})
	if clicks != 0 {
		t.Fatalf("expected 0 clicks when disabled, got %d", clicks)
	}
}

func TestButtonHoverState(t *testing.T) {
	b := NewButton("ok")
	enter := Event{Type: EventPointerEnter}
	if !b.HandleEvent(&enter) || !b.hover {
		t.Fatal("expected hover true after enter")
	}
	leave := Event{Type: EventPointerLeave}
	if !b.HandleEvent(&leave) || b.hover {
		t.Fatal("expected hover false after leave")
	}
}

func TestHitTestFindsDeepest(t *testing.T) {
	root := NewContainer()
	root.SetBounds(geom.Rect{X: 0, Y: 0, W: 200, H: 200})

	b := NewButton("ok")
	b.SetBounds(geom.Rect{X: 50, Y: 50, W: 40, H: 20})
	root.Add(b)

	if got := hitTest(root, geom.Point{X: 60, Y: 60}); got != Widget(b) {
		t.Errorf("expected to hit the button, got %T", got)
	}
	if got := hitTest(root, geom.Point{X: 10, Y: 10}); got != Widget(root) {
		t.Errorf("expected to hit the root container, got %T", got)
	}
	if got := hitTest(root, geom.Point{X: 500, Y: 500}); got != nil {
		t.Errorf("expected nil outside the tree, got %T", got)
	}
}
