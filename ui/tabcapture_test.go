package ui

import (
	"testing"

	"github.com/kpfaulkner/guie/render"
)

// tabTaker is a focusable canvas that asks for Tab.
type tabTaker struct {
	*CanvasWidget
	capture bool
	keys    []render.Key
}

func (w *tabTaker) CapturesTab() bool { return w.capture }

func newTabTaker(capture bool) *tabTaker {
	w := &tabTaker{capture: capture}
	w.CanvasWidget = NewCanvasWidget(nil, CanvasFocusable(), CanvasOnEvent(func(ev *Event) bool {
		if ev.Type == EventKeyDown {
			w.keys = append(w.keys, ev.Key)
			return true
		}
		return false
	}))
	return w
}

func tabApp(t *testing.T, capture bool) (*App, *tabTaker, *Button) {
	t.Helper()
	app := NewApp()
	root := NewContainer()
	root.SetLayout(VBox(0))
	tt := newTabTaker(capture)
	b := NewButton("other")
	root.Add(tt)
	root.Add(b)
	app.SetContent(root)
	app.setFocus(tt)
	return app, tt, b
}

func TestTabDeliveredToCapturingWidget(t *testing.T) {
	app, tt, _ := tabApp(t, true)

	app.dispatchKeyboard(render.InputState{KeysPressed: []render.Key{render.KeyTab}})
	app.dispatchKeyboard(render.InputState{
		KeysPressed: []render.Key{render.KeyTab},
		Modifiers:   render.ModifierSet(render.ModShift),
	})

	if app.focused != Widget(tt) {
		t.Errorf("focus moved away from a widget that captures Tab")
	}
	if len(tt.keys) != 2 || tt.keys[0] != render.KeyTab || tt.keys[1] != render.KeyTab {
		t.Errorf("widget received %v, want two Tab key events", tt.keys)
	}
}

func TestTabStillMovesFocusWhenNotCaptured(t *testing.T) {
	app, tt, b := tabApp(t, false)

	app.dispatchKeyboard(render.InputState{KeysPressed: []render.Key{render.KeyTab}})

	if app.focused != Widget(b) {
		t.Errorf("Tab should move focus on when the widget does not capture it")
	}
	if len(tt.keys) != 0 {
		t.Errorf("widget received %v, want nothing", tt.keys)
	}
}

func TestTabStillMovesFocusFromPlainWidgets(t *testing.T) {
	app, a, b := twoButtonApp(t)
	app.setFocus(a)
	app.dispatchKeyboard(render.InputState{KeysPressed: []render.Key{render.KeyTab}})
	if app.focused != Widget(b) {
		t.Errorf("widgets that do not implement TabCapturer must keep default Tab behaviour")
	}
}
