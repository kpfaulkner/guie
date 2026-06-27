package ui

import (
	"testing"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
)

func TestBaseWidgetDefaults(t *testing.T) {
	b := NewBase()
	// Leaf defaults: zero MinSize, no-op Layout/Draw.
	if b.MinSize() != (geom.Size{}) {
		t.Error("BaseWidget.MinSize should default to zero")
	}
	b.Layout()  // no-op
	b.Draw(nil) // no-op (must not touch the canvas)

	b.SetVisible(false)
	if b.Visible() {
		t.Error("SetVisible(false) should hide the widget")
	}
	b.SetVisible(true)

	// RequestFocus before mounting is a safe no-op (nil ctx guard).
	b.RequestFocus()
}

func TestRequestFocusMounted(t *testing.T) {
	a := NewApp()
	btn := NewButton("x")
	root := NewContainer()
	root.Add(btn)
	a.SetContent(root)

	btn.RequestFocus()
	if a.focused != Widget(btn) {
		t.Error("RequestFocus on a mounted widget should focus it")
	}
}

func TestContainerRemove(t *testing.T) {
	c := NewContainer()
	w1 := NewLabel("a")
	w2 := NewLabel("b")
	c.Add(w1)
	c.Add(w2)

	c.Remove(w1)
	if got := c.Children(); len(got) != 1 || got[0] != w2 {
		t.Fatalf("Remove should drop only w1, got %v", got)
	}
	// Removing a non-child is a no-op.
	c.Remove(NewLabel("z"))
	if len(c.Children()) != 1 {
		t.Error("removing a non-child should not change the children")
	}
}

func TestPopupClose(t *testing.T) {
	a := NewApp()
	p := a.ShowModal(NewLabel("dialog"))
	if len(a.overlays) != 1 {
		t.Fatalf("ShowModal should open one overlay, got %d", len(a.overlays))
	}
	a.Close(p)
	if len(a.overlays) != 0 {
		t.Errorf("Close should dismiss the overlay, got %d", len(a.overlays))
	}
}

func TestStackMeasure(t *testing.T) {
	s := NewStack()
	got := s.Measure(items(newStub(10, 20), newStub(30, 5)))
	if got != (geom.Size{W: 30, H: 20}) {
		t.Errorf("Stack.Measure should return the largest extents, got %+v", got)
	}
}

func TestTextareaIntHelpers(t *testing.T) {
	if maxI(3, 5) != 5 || maxI(5, 3) != 5 {
		t.Error("maxI should return the larger int")
	}
	if minI(3, 5) != 3 || minI(5, 3) != 3 {
		t.Error("minI should return the smaller int")
	}
}

func TestLoadFontError(t *testing.T) {
	if _, err := LoadFont("does-not-exist.ttf", 14); err == nil {
		t.Error("LoadFont on a missing path should return an error")
	}
}

func TestTextFieldMoveRight(t *testing.T) {
	tf := NewTextField()
	tf.SetText("ab")
	keyDown(tf, render.KeyLeft)  // caret off the end
	keyDown(tf, render.KeyRight) // moveRight
	// No assertion on internal caret state; the point is to exercise moveRight
	// without panicking and leave the text intact.
	if tf.Text() != "ab" {
		t.Errorf("arrow keys should not change the text, got %q", tf.Text())
	}
}

func TestTextAreaCutSelection(t *testing.T) {
	a := NewApp()
	ta := NewTextArea()
	a.SetContent(ta) // mount so the clipboard is available
	ta.SetText("hello")

	primaryKey(ta, render.KeyA) // select all
	primaryKey(ta, render.KeyX) // cut
	if ta.Text() != "" {
		t.Errorf("cut over a full selection should empty the text, got %q", ta.Text())
	}
}
