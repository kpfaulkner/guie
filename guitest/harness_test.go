package guitest_test

import (
	"testing"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/guitest"
	"github.com/kpfaulkner/guie/render"
	"github.com/kpfaulkner/guie/ui"
)

func TestClickFiresButton(t *testing.T) {
	h := guitest.New(200, 100)
	clicked := 0
	btn := ui.NewButton("Save")
	btn.OnClick(func() { clicked++ })
	h.SetContent(btn) // root fills the 200x100 surface

	h.Click(100, 50)
	if clicked != 1 {
		t.Fatalf("expected the button to fire once, got %d", clicked)
	}
}

func TestLabelDrawsText(t *testing.T) {
	h := guitest.New(200, 60)
	h.SetContent(ui.NewLabel("Hello, guie!"))

	rec := h.Step()
	if !rec.HasText("Hello, guie!") {
		t.Fatalf("label text was not drawn; texts = %v", rec.Texts())
	}
}

func TestTextFieldFocusAndType(t *testing.T) {
	h := guitest.New(240, 60)
	tf := ui.NewTextField()
	h.SetContent(tf)

	h.Click(40, 30) // focus the field
	h.TypeText("hello").Step()

	if tf.Text() != "hello" {
		t.Fatalf("typed text not captured, got %q", tf.Text())
	}
}

func TestListClickSelectsRow(t *testing.T) {
	h := guitest.New(200, 200)
	list := ui.NewList([]string{"alpha", "beta", "gamma"})
	got := -1
	list.OnSelect(func(i int) { got = i })
	h.SetContent(list)

	h.Step() // lay out so RowHeight/bounds are known
	rh := list.RowHeight()
	h.Click(20, rh*1+rh/2) // click row index 1

	if got != 1 || list.Selected() != 1 {
		t.Fatalf("expected row 1 selected, OnSelect=%d Selected=%d", got, list.Selected())
	}
}

func TestKeyboardActivatesFocusedButton(t *testing.T) {
	h := guitest.New(200, 80)
	fired := 0
	btn := ui.NewButton("Go")
	btn.OnClick(func() { fired++ })
	h.SetContent(btn)

	h.Click(100, 40) // focus
	fired = 0        // ignore the click activation; test keyboard
	h.TypeKey(render.KeyEnter)
	if fired != 1 {
		t.Fatalf("Enter on a focused button should activate it once, got %d", fired)
	}
}

func TestResizeChangesRecordingSize(t *testing.T) {
	h := guitest.New(200, 100)
	h.SetContent(ui.NewLabel("x"))
	if rec := h.Step(); rec.Size != (geom.Size{W: 200, H: 100}) {
		t.Fatalf("recording size = %v, want 200x100", rec.Size)
	}
	h.Resize(320, 240)
	if rec := h.Step(); rec.Size != (geom.Size{W: 320, H: 240}) {
		t.Fatalf("recording size after resize = %v, want 320x240", rec.Size)
	}
}

// noopGhost is a custom drag ghost so a harness-driven drag doesn't hit the
// default snapshot ghost (which would allocate a GPU RenderTarget).
type noopGhost struct{}

func (noopGhost) DrawGhost(c render.Canvas, at geom.Point) {}

func TestDragBetweenWidgets(t *testing.T) {
	h := guitest.New(200, 100)

	left := ui.NewContainer()
	left.SetDragGhost(noopGhost{})
	left.SetDragSource(func() *ui.DragData { return &ui.DragData{Type: "item", Value: 42} })

	right := ui.NewContainer()
	var dropped *ui.DragData
	right.SetDropTarget(func(d ui.DragData) bool { return d.Type == "item" })
	right.OnDrop(func(d ui.DragData, _ geom.Point) bool { dropped = &d; return true })

	row := ui.NewContainer()
	row.SetLayout(ui.HBox(0))
	row.Add(left, ui.Weight(1), ui.Align(geom.AlignStretch))
	row.Add(right, ui.Weight(1), ui.Align(geom.AlignStretch))
	h.SetContent(row)

	// Left half center → right half center.
	h.Drag(50, 50, 150, 50)

	if dropped == nil {
		t.Fatal("expected a drop on the right panel")
	}
	if dropped.Value.(int) != 42 {
		t.Fatalf("dropped payload = %v, want 42", dropped.Value)
	}
}

func TestRecordingHelpers(t *testing.T) {
	h := guitest.New(120, 40)
	h.SetContent(ui.NewLabel("ABCD"))
	rec := h.Step()

	if n := rec.Count(guitest.OpDrawText); n == 0 {
		t.Fatal("expected at least one text op")
	}
	if !rec.TextContaining("BC") {
		t.Fatal("TextContaining should find the substring")
	}
	// Deterministic font: width is rune count * advance (0.6 * font size).
	w := h.Font.Measure("ABCD").W
	if w <= 0 {
		t.Fatalf("headless font should measure positive width, got %v", w)
	}
}
