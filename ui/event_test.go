package ui

import (
	"testing"

	"github.com/kpfaulkner/uiframework/geom"
	"github.com/kpfaulkner/uiframework/render"
)

// down/up build InputState snapshots for a left button press/release at pos. A
// press reports the button as both "just pressed" and "currently held".
func downAt(pos geom.Point) render.InputState {
	return render.InputState{
		MousePos:     pos,
		MousePressed: ButtonSetOf(render.MouseLeft),
		MouseDown:    ButtonSetOf(render.MouseLeft),
	}
}

func upAt(pos geom.Point) render.InputState {
	return render.InputState{MousePos: pos, MouseReleased: ButtonSetOf(render.MouseLeft)}
}

// ButtonSetOf is a tiny test helper mirroring render.ButtonSet.Set.
func ButtonSetOf(b render.MouseButton) render.ButtonSet {
	return render.ButtonSet(0).Set(b)
}

// appWithButton returns an App whose root contains a single button with fixed
// bounds, ready for direct dispatchPointer calls (no layout pass needed).
func appWithButton(onClick func()) (*App, *Button) {
	app := NewApp()
	root := NewContainer()
	root.SetBounds(geom.Rect{X: 0, Y: 0, W: 200, H: 200})
	b := NewButton("ok")
	b.OnClick(onClick)
	b.SetBounds(geom.Rect{X: 10, Y: 10, W: 100, H: 30})
	root.Add(b)
	app.SetContent(root)
	// SetContent leaves the manually assigned bounds intact (surface size is 0).
	return app, b
}

func TestButtonClickEventFires(t *testing.T) {
	clicks := 0
	b := NewButton("ok")
	b.OnClick(func() { clicks++ })
	ev := Event{Type: EventClick, Button: render.MouseLeft}
	if !b.HandleEvent(&ev) {
		t.Fatal("button should consume EventClick")
	}
	if clicks != 1 {
		t.Fatalf("expected 1 click, got %d", clicks)
	}
}

func TestButtonDisabledIgnoresClick(t *testing.T) {
	clicks := 0
	b := NewButton("ok")
	b.OnClick(func() { clicks++ })
	b.SetEnabled(false)
	ev := Event{Type: EventClick, Button: render.MouseLeft}
	if b.HandleEvent(&ev) {
		t.Fatal("disabled button should not consume events")
	}
	if clicks != 0 {
		t.Fatalf("expected 0 clicks when disabled, got %d", clicks)
	}
}

func TestPointerPressReleaseInsideClicks(t *testing.T) {
	clicks := 0
	app, _ := appWithButton(func() { clicks++ })

	inside := geom.Point{X: 20, Y: 20}
	app.dispatchPointer(downAt(inside))
	app.dispatchPointer(upAt(inside))

	if clicks != 1 {
		t.Fatalf("press+release inside should click once, got %d", clicks)
	}
}

func TestPointerReleaseOutsideNoClick(t *testing.T) {
	clicks := 0
	app, _ := appWithButton(func() { clicks++ })

	app.dispatchPointer(downAt(geom.Point{X: 20, Y: 20}))
	app.dispatchPointer(upAt(geom.Point{X: 180, Y: 180})) // released off the button

	if clicks != 0 {
		t.Fatalf("release outside should not click, got %d", clicks)
	}
}

func TestPointerClickFocusesButton(t *testing.T) {
	app, b := appWithButton(func() {})
	app.dispatchPointer(downAt(geom.Point{X: 20, Y: 20}))
	if app.focused != Widget(b) {
		t.Fatal("pressing a focusable button should focus it")
	}
}

// recPad records the button of pointer-down and click events it receives.
type recPad struct {
	BaseWidget
	downs  []render.MouseButton
	clicks []render.MouseButton
}

func (r *recPad) HandleEvent(ev *Event) bool {
	switch ev.Type {
	case EventPointerDown:
		r.downs = append(r.downs, ev.Button)
		return true
	case EventClick:
		r.clicks = append(r.clicks, ev.Button)
		return true
	}
	return false
}

func TestPointerDispatchesAllButtons(t *testing.T) {
	app := NewApp()
	rec := &recPad{BaseWidget: NewBase()}
	app.SetContent(rec)
	rec.SetBounds(geom.Rect{X: 0, Y: 0, W: 100, H: 100})
	pos := geom.Point{X: 10, Y: 10}

	for _, btn := range []render.MouseButton{render.MouseRight, render.MouseMiddle} {
		app.dispatchPointer(render.InputState{MousePos: pos, MousePressed: ButtonSetOf(btn), MouseDown: ButtonSetOf(btn)})
		app.dispatchPointer(render.InputState{MousePos: pos, MouseReleased: ButtonSetOf(btn)})
	}

	want := []render.MouseButton{render.MouseRight, render.MouseMiddle}
	if len(rec.downs) != 2 || rec.downs[0] != want[0] || rec.downs[1] != want[1] {
		t.Fatalf("downs = %v, want %v", rec.downs, want)
	}
	if len(rec.clicks) != 2 || rec.clicks[0] != want[0] || rec.clicks[1] != want[1] {
		t.Fatalf("clicks = %v, want %v", rec.clicks, want)
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
