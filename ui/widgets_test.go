package ui

import (
	"testing"

	"github.com/kpfaulkner/uiframework/geom"
	"github.com/kpfaulkner/uiframework/render"
)

func click(w Widget) {
	ev := Event{Type: EventClick, Button: render.MouseLeft}
	w.HandleEvent(&ev)
}

func keyDown(w Widget, k render.Key) {
	ev := Event{Type: EventKeyDown, Key: k}
	w.HandleEvent(&ev)
}

func typeRune(w Widget, r rune) {
	ev := Event{Type: EventText, Rune: r}
	w.HandleEvent(&ev)
}

func TestCheckboxToggle(t *testing.T) {
	changes := []bool{}
	c := NewCheckbox("on", OnToggle(func(v bool) { changes = append(changes, v) }))

	click(c)
	if !c.IsChecked() {
		t.Fatal("click should check the box")
	}
	click(c)
	if c.IsChecked() {
		t.Fatal("second click should uncheck the box")
	}
	if len(changes) != 2 || changes[0] != true || changes[1] != false {
		t.Fatalf("OnToggle should fire true then false, got %v", changes)
	}
}

func TestCheckboxKeyboard(t *testing.T) {
	c := NewCheckbox("on")
	keyDown(c, render.KeySpace)
	if !c.IsChecked() {
		t.Fatal("Space should toggle the checkbox")
	}
	keyDown(c, render.KeyEnter)
	if c.IsChecked() {
		t.Fatal("Enter should toggle the checkbox back")
	}
}

func TestRadioGroupExclusive(t *testing.T) {
	g := NewRadioGroup()
	selectedIdx := -1
	g.OnChange(func(i int) { selectedIdx = i })
	a := NewRadioButton("a", g)
	b := NewRadioButton("b", g)
	c := NewRadioButton("c", g)

	click(b)
	if !b.IsSelected() || a.IsSelected() || c.IsSelected() {
		t.Fatal("only b should be selected")
	}
	if g.Selected() != 1 || selectedIdx != 1 {
		t.Fatalf("group should report index 1, got %d / %d", g.Selected(), selectedIdx)
	}

	click(c)
	if !c.IsSelected() || b.IsSelected() {
		t.Fatal("selecting c should deselect b")
	}
	if g.Selected() != 2 {
		t.Fatalf("group should report index 2, got %d", g.Selected())
	}
}

func TestTextFieldEditing(t *testing.T) {
	tf := NewTextField()
	// Focus places the caret at the end (empty → 0).
	tf.HandleEvent(&Event{Type: EventFocusGained})

	for _, r := range "abc" {
		typeRune(tf, r)
	}
	if tf.Text() != "abc" || tf.caret != 3 {
		t.Fatalf("after typing: %q caret %d", tf.Text(), tf.caret)
	}

	keyDown(tf, render.KeyLeft) // caret between b and c
	typeRune(tf, 'X')
	if tf.Text() != "abXc" {
		t.Fatalf("insert at caret: got %q", tf.Text())
	}

	keyDown(tf, render.KeyBackspace) // remove X
	if tf.Text() != "abc" {
		t.Fatalf("backspace: got %q", tf.Text())
	}

	keyDown(tf, render.KeyHome)
	keyDown(tf, render.KeyDelete) // remove leading a
	if tf.Text() != "bc" {
		t.Fatalf("home+delete: got %q", tf.Text())
	}
}

func TestTextFieldSubmit(t *testing.T) {
	got := ""
	tf := NewTextField(OnSubmit(func(s string) { got = s }))
	tf.SetText("hello")
	keyDown(tf, render.KeyEnter)
	if got != "hello" {
		t.Fatalf("OnSubmit should receive %q, got %q", "hello", got)
	}
}

func TestTextFieldRejectsControlRunes(t *testing.T) {
	tf := NewTextField()
	tf.HandleEvent(&Event{Type: EventFocusGained})
	typeRune(tf, '\n') // control rune ignored
	typeRune(tf, 'a')
	if tf.Text() != "a" {
		t.Fatalf("control runes should be ignored, got %q", tf.Text())
	}
}

func TestScrollViewWheelClamp(t *testing.T) {
	sv := NewScrollView()
	sv.SetContent(newStub(50, 300))
	sv.SetBounds(geom.Rect{X: 0, Y: 0, W: 100, H: 100})
	// viewport height 100, content 300 → max offset 200.

	wheel := func(dy float64) {
		ev := Event{Type: EventWheel, Wheel: geom.Point{Y: dy}}
		sv.HandleEvent(&ev)
	}

	wheel(-1) // scroll down by one step
	if sv.offsetY != wheelStep {
		t.Fatalf("one notch down should offset by %v, got %v", float64(wheelStep), sv.offsetY)
	}
	for i := 0; i < 50; i++ {
		wheel(-1)
	}
	if sv.offsetY != sv.maxOffset() {
		t.Fatalf("should clamp at max offset %v, got %v", sv.maxOffset(), sv.offsetY)
	}
	for i := 0; i < 50; i++ {
		wheel(1)
	}
	if sv.offsetY != 0 {
		t.Fatalf("should clamp at 0, got %v", sv.offsetY)
	}
}

func TestScrollViewConstrainedByLayout(t *testing.T) {
	// A ScrollView with tall content, given a weight in a VBox, must be
	// constrained to the available height (not grow to the content height),
	// otherwise there is nothing to scroll.
	app := NewApp()
	root := NewContainer()
	root.SetLayout(VBox(0))
	sv := NewScrollView()
	sv.SetContent(newStub(50, 1000))
	root.Add(sv, Weight(1))
	app.SetContent(root)

	app.resize(200, 300)
	app.layoutIfNeeded()

	if sv.Bounds().H > 300 {
		t.Fatalf("scroll view should be constrained to the viewport, got height %v", sv.Bounds().H)
	}
	if sv.maxOffset() <= 0 {
		t.Fatalf("tall content should be scrollable, maxOffset=%v", sv.maxOffset())
	}
}

func TestScrollViewLayoutOffsetsContent(t *testing.T) {
	content := newStub(50, 300)
	sv := NewScrollView()
	sv.SetContent(content)
	sv.SetBounds(geom.Rect{X: 0, Y: 0, W: 100, H: 100})
	sv.offsetY = 50
	sv.Layout()

	got := content.Bounds()
	want := geom.Rect{X: 0, Y: -50, W: 100 - scrollbarWidth, H: 300}
	if got != want {
		t.Fatalf("content bounds: got %+v want %+v", got, want)
	}
}

func TestSliderFromPointer(t *testing.T) {
	var last float64 = -1
	s := NewSlider(OnSlide(func(v float64) { last = v }))
	s.SetBounds(geom.Rect{X: 0, Y: 0, W: 100, H: 20})
	// track spans x0=8..x1=92 (handle radius 8).

	s.HandleEvent(&Event{Type: EventPointerDown, Pos: geom.Point{X: 50, Y: 10}})
	if !approx(s.Value(), 0.5) {
		t.Fatalf("midpoint press should be ~0.5, got %v", s.Value())
	}
	if !approx(last, 0.5) {
		t.Fatalf("OnSlide should fire with 0.5, got %v", last)
	}

	s.HandleEvent(&Event{Type: EventPointerMove, Pos: geom.Point{X: 0, Y: 10}})
	if s.Value() != 0 {
		t.Fatalf("dragging past the start should clamp to 0, got %v", s.Value())
	}
	s.HandleEvent(&Event{Type: EventPointerMove, Pos: geom.Point{X: 200, Y: 10}})
	if s.Value() != 1 {
		t.Fatalf("dragging past the end should clamp to 1, got %v", s.Value())
	}
}

func TestSliderKeyboard(t *testing.T) {
	s := NewSlider(SliderValue(0.5))
	keyDown(s, render.KeyRight)
	if !approx(s.Value(), 0.55) {
		t.Fatalf("Right should add a step, got %v", s.Value())
	}
	keyDown(s, render.KeyLeft)
	keyDown(s, render.KeyLeft)
	if !approx(s.Value(), 0.45) {
		t.Fatalf("two Lefts should subtract two steps, got %v", s.Value())
	}
}
