package guitest_test

import (
	"testing"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/guitest"
	"github.com/kpfaulkner/guie/render"
	"github.com/kpfaulkner/guie/theme"
	"github.com/kpfaulkner/guie/ui"
)

// canvasHarness mounts cw filling a 200x100 window.
func canvasHarness(cw ui.Widget) *guitest.Harness {
	h := guitest.New(200, 100)
	root := ui.NewContainer()
	root.SetLayout(ui.NewStack())
	root.Add(cw, ui.Align(geom.AlignStretch))
	h.SetContent(root)
	h.Step()
	return h
}

func TestCanvasWidgetDrawsClippedToBounds(t *testing.T) {
	var got geom.Rect
	calls := 0
	cw := ui.NewCanvasWidget(func(c render.Canvas, b geom.Rect) {
		calls++
		got = b
		c.DrawText("plot", geom.Point{X: b.X, Y: b.Y}, nil, theme.DefaultPalette().Text)
	})
	h := canvasHarness(cw)
	rec := h.Step()

	if calls == 0 {
		t.Fatal("draw callback never ran")
	}
	if got != cw.Bounds() {
		t.Errorf("draw bounds %v, want %v", got, cw.Bounds())
	}
	if !rec.HasText("plot") {
		t.Error("callback drawing not recorded")
	}
	// The callback's text op must sit between a PushClip of the bounds and a PopClip.
	ops := rec.Ops
	pushed := -1
	for i, op := range ops {
		if op.Kind == guitest.OpPushClip && op.Rect == cw.Bounds() {
			pushed = i
		}
		if op.Kind == guitest.OpDrawText && op.Text == "plot" {
			if pushed < 0 {
				t.Fatal("callback drew before its bounds clip was pushed")
			}
			for _, later := range ops[i:] {
				if later.Kind == guitest.OpPopClip {
					return
				}
			}
			t.Fatal("no PopClip after callback drawing")
		}
	}
}

func TestCanvasWidgetNilAndSetDraw(t *testing.T) {
	cw := ui.NewCanvasWidget(nil)
	h := canvasHarness(cw)
	h.Step() // must not panic

	cw.SetDraw(func(c render.Canvas, b geom.Rect) {
		c.DrawText("late", geom.Point{X: b.X, Y: b.Y}, nil, theme.DefaultPalette().Text)
	})
	if !h.Step().HasText("late") {
		t.Error("SetDraw renderer not used")
	}
}

func TestCanvasWidgetPointerCaptureAndToLocal(t *testing.T) {
	var types []ui.EventType
	var downPos geom.Point
	var cw *ui.CanvasWidget
	cw = ui.NewCanvasWidget(nil, ui.CanvasOnEvent(func(ev *ui.Event) bool {
		switch ev.Type {
		case ui.EventPointerDown:
			downPos = cw.ToLocal(ev.Pos)
			types = append(types, ev.Type)
		case ui.EventPointerMove, ui.EventPointerUp:
			types = append(types, ev.Type)
		}
		return true
	}))
	h := canvasHarness(cw)

	b := cw.Bounds()
	h.MoveMouse(b.X+10, b.Y+20).PressMouse(render.MouseLeft).Step()
	h.MoveMouse(b.X+b.W+500, b.Y+20).Step() // far outside: capture keeps delivering
	h.ReleaseMouse(render.MouseLeft).Step()

	if downPos != (geom.Point{X: 10, Y: 20}) {
		t.Errorf("ToLocal(down) = %v, want {10 20}", downPos)
	}
	var down, move, up bool
	for _, ty := range types {
		down = down || ty == ui.EventPointerDown
		move = move || ty == ui.EventPointerMove
		up = up || ty == ui.EventPointerUp
	}
	if !down || !move || !up {
		t.Errorf("capture: down=%v move=%v up=%v, want all true (%v)", down, move, up, types)
	}
}

func TestCanvasWidgetKeysNeedFocusable(t *testing.T) {
	keys := 0
	plain := ui.NewCanvasWidget(nil, ui.CanvasOnEvent(func(ev *ui.Event) bool {
		if ev.Type == ui.EventKeyDown {
			keys++
		}
		return true
	}))
	h := canvasHarness(plain)
	h.Click(plain.Bounds().X+5, plain.Bounds().Y+5)
	h.TypeKey(render.KeyA)
	if keys != 0 || plain.Focusable() || plain.Focused() {
		t.Errorf("non-focusable canvas got keys=%d focusable=%v focused=%v", keys, plain.Focusable(), plain.Focused())
	}

	keys = 0
	runes := 0
	foc := ui.NewCanvasWidget(nil, ui.CanvasFocusable(), ui.CanvasOnEvent(func(ev *ui.Event) bool {
		switch ev.Type {
		case ui.EventKeyDown:
			keys++
		case ui.EventText:
			runes++
		}
		return true
	}))
	h = canvasHarness(foc)
	h.Click(foc.Bounds().X+5, foc.Bounds().Y+5)
	if !foc.Focused() {
		t.Fatal("click did not focus a CanvasFocusable widget")
	}
	h.TypeKey(render.KeyA)
	h.TypeText("x").Step()
	if keys == 0 || runes == 0 {
		t.Errorf("focused canvas got keys=%d runes=%d, want both > 0", keys, runes)
	}
	// Focus ring: a rounded stroke drawn on top.
	if h.Step().Count(guitest.OpStrokeRoundRect) == 0 {
		t.Error("focused canvas drew no focus ring")
	}
}

func TestCanvasWidgetConsumeAndBubble(t *testing.T) {
	clicked := 0
	cw := ui.NewCanvasWidget(nil)
	h := guitest.New(200, 100)
	root := &bubbleSpy{Container: ui.NewContainer(), hits: &clicked}
	root.SetLayout(ui.NewStack())
	root.Add(cw, ui.Align(geom.AlignStretch))
	h.SetContent(root)
	h.Step()

	// Unhandled (no callback): bubbles to the parent.
	h.Click(cw.Bounds().X+5, cw.Bounds().Y+5)
	if clicked == 0 {
		t.Fatal("unconsumed event did not bubble")
	}

	// Consumed: stops.
	clicked = 0
	cw.SetOnEvent(func(ev *ui.Event) bool { return true })
	h.Click(cw.Bounds().X+5, cw.Bounds().Y+5)
	if clicked != 0 {
		t.Error("consumed event still bubbled")
	}
}

func TestCanvasWidgetDisabledGetsNoEvents(t *testing.T) {
	n := 0
	cw := ui.NewCanvasWidget(nil, ui.CanvasOnEvent(func(*ui.Event) bool { n++; return true }))
	h := canvasHarness(cw)
	cw.SetEnabled(false)
	n = 0
	h.Click(cw.Bounds().X+5, cw.Bounds().Y+5)
	if n != 0 {
		t.Errorf("disabled canvas received %d events", n)
	}
}

func TestCanvasWidgetHover(t *testing.T) {
	cw := ui.NewCanvasWidget(nil)
	h := canvasHarness(cw)
	h.MoveMouse(cw.Bounds().X+5, cw.Bounds().Y+5).Step()
	if !cw.Hovered() {
		t.Error("Hovered false with pointer over widget")
	}
}

func TestCanvasWidgetFollowsTheme(t *testing.T) {
	var cw *ui.CanvasWidget
	cw = ui.NewCanvasWidget(func(c render.Canvas, b geom.Rect) {
		c.FillRect(b, cw.ColourOf(ui.RoleSurface))
	})
	h := canvasHarness(cw)
	dark := h.Step().FillsOfColour(theme.DefaultPalette().Surface)
	h.App.SetTheme(theme.Light())
	light := h.Step().FillsOfColour(theme.LightPalette().Surface)
	if len(dark) == 0 || len(light) == 0 {
		t.Errorf("surface fills: dark=%d light=%d, want both > 0", len(dark), len(light))
	}
}

// bubbleSpy counts pointer-downs that bubble up to it.
type bubbleSpy struct {
	*ui.Container
	hits *int
}

func (s *bubbleSpy) HandleEvent(ev *ui.Event) bool {
	if ev.Type == ui.EventPointerDown {
		*s.hits++
	}
	return false
}

func TestDoubleClickThroughHarness(t *testing.T) {
	n := 0
	cw := ui.NewCanvasWidget(nil)
	cw.OnDoubleClick(func(p geom.Point) bool { n++; return true })
	h := canvasHarness(cw)
	x, y := cw.Bounds().X+5, cw.Bounds().Y+5
	h.Click(x, y)
	h.Click(x, y)
	if n != 1 {
		t.Errorf("double-clicks = %d, want 1", n)
	}
}
