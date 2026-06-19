package ui

import (
	"testing"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
)

// twoButtonApp builds an App whose root holds two enabled buttons and one
// disabled button, and returns the app and the two enabled buttons.
func twoButtonApp(t *testing.T) (*App, *Button, *Button) {
	t.Helper()
	app := NewApp()
	root := NewContainer()
	root.SetLayout(VBox(0))
	a := NewButton("a")
	b := NewButton("b")
	disabled := NewButton("disabled")
	disabled.SetEnabled(false)
	root.Add(a)
	root.Add(disabled)
	root.Add(b)
	app.SetContent(root)
	return app, a, b
}

func TestTabMovesFocusSkippingDisabled(t *testing.T) {
	app, a, b := twoButtonApp(t)

	app.moveFocus(1)
	if app.focused != Widget(a) {
		t.Fatalf("first Tab should focus button a")
	}
	app.moveFocus(1)
	if app.focused != Widget(b) {
		t.Fatalf("second Tab should focus button b (skipping disabled)")
	}
	app.moveFocus(1)
	if app.focused != Widget(a) {
		t.Fatalf("third Tab should wrap back to button a")
	}
	app.moveFocus(-1)
	if app.focused != Widget(b) {
		t.Fatalf("Shift+Tab should wrap back to button b")
	}
}

func TestFocusGainedLostEvents(t *testing.T) {
	app, a, b := twoButtonApp(t)
	app.setFocus(a)
	if !a.focused {
		t.Fatal("button a should report focused")
	}
	app.setFocus(b)
	if a.focused {
		t.Fatal("button a should have lost focus")
	}
	if !b.focused {
		t.Fatal("button b should report focused")
	}
}

func TestClickEmptyClearsFocus(t *testing.T) {
	app, a, _ := twoButtonApp(t)
	app.setFocus(a)
	// A label is not focusable; focusing from it clears focus.
	app.focusFromPointer(NewLabel("x"))
	if app.focused != nil {
		t.Fatal("clicking a non-focusable target should clear focus")
	}
}

func TestKeyboardActivation(t *testing.T) {
	clicks := 0
	b := NewButton("ok")
	b.OnClick(func() { clicks++ })
	down := Event{Type: EventKeyDown, Key: render.KeySpace}
	b.HandleEvent(&down)
	enter := Event{Type: EventKeyDown, Key: render.KeyEnter}
	b.HandleEvent(&enter)
	if clicks != 2 {
		t.Fatalf("expected Space and Enter to activate, got %d", clicks)
	}
}

func TestEventBusReceivesClick(t *testing.T) {
	app, a, _ := twoButtonApp(t)
	a.SetBounds(geom.Rect{X: 0, Y: 0, W: 50, H: 20})

	got := 0
	app.Events().Subscribe(EventClick, func(Event) { got++ })

	app.dispatch(a, Event{Type: EventClick, Pos: geom.Point{X: 5, Y: 5}, Button: render.MouseLeft})
	if got != 1 {
		t.Fatalf("expected bus subscriber to observe 1 click, got %d", got)
	}
}

func TestBubblingReachesAncestor(t *testing.T) {
	// A non-consuming leaf should let an event bubble up to a consuming ancestor.
	app := &App{bus: newEventBus()}
	parent := &wheelSink{BaseWidget: NewBase()}
	leaf := newStub(1, 1)
	leaf.mount(leaf, parent, nil) // wire leaf's parent directly

	if app.dispatch(leaf, Event{Type: EventWheel, Wheel: geom.Point{Y: -1}}) != true {
		t.Fatal("event should have been consumed by the ancestor")
	}
	if !parent.got {
		t.Fatal("wheel event should bubble from leaf to the consuming ancestor")
	}
}

// wheelSink is a leaf widget that consumes wheel events, used to test bubbling.
type wheelSink struct {
	BaseWidget
	got bool
}

func (w *wheelSink) HandleEvent(ev *Event) bool {
	if ev.Type == EventWheel {
		w.got = true
		return true
	}
	return false
}
