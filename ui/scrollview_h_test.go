package ui

import (
	"testing"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
)

// hView returns a 200x120 ScrollView over a 600x300 stub, laid out.
func hView(t *testing.T, hscroll bool) (*ScrollView, Widget) {
	t.Helper()
	app := NewApp()
	sv := NewScrollView()
	content := newStub(600, 300)
	app.SetContent(sv)
	sv.SetContent(content)
	sv.SetHorizontalScroll(hscroll)
	sv.SetBounds(geom.Rect{X: 10, Y: 20, W: 200, H: 120})
	sv.Layout()
	return sv, content
}

func TestScrollViewHorizontalOffByDefault(t *testing.T) {
	sv, content := hView(t, false)
	if sv.hbarShown() {
		t.Error("no horizontal bar unless horizontal scrolling is turned on")
	}
	if got, want := content.Bounds().W, sv.viewport().W; got != want {
		t.Errorf("content width = %v, want the view width %v (stretched, as before)", got, want)
	}
	if got := sv.MinSize().W; got != 600+scrollbarWidth {
		t.Errorf("MinSize width = %v, want the content width plus the gutter", got)
	}
	sv.HandleEvent(&Event{Type: EventWheel, Wheel: geom.Point{X: -3}})
	if sv.offsetX != 0 {
		t.Error("a sideways wheel must do nothing while horizontal scrolling is off")
	}
}

func TestScrollViewHorizontalLayout(t *testing.T) {
	sv, content := hView(t, true)
	if !sv.hbarShown() {
		t.Fatal("content wider than the view should show the horizontal bar")
	}
	vp := sv.viewport()
	if vp.W != 200-scrollbarWidth || vp.H != 120-scrollbarWidth {
		t.Errorf("viewport = %vx%v, want the bounds minus both bars (%vx%v)", vp.W, vp.H, 200-scrollbarWidth, 120-scrollbarWidth)
	}
	if content.Bounds().W != 600 {
		t.Errorf("content width = %v, want its natural 600", content.Bounds().W)
	}
	if got := sv.MinSize(); got.W >= 600 {
		t.Errorf("MinSize width = %v: with horizontal scrolling the view must not demand the content width", got.W)
	}

	// The two gutters stop short of the corner.
	v, h := sv.track(), sv.trackH()
	if v.Y+v.H > h.Y || h.X+h.W > v.X {
		t.Errorf("tracks overlap the corner: vertical %v horizontal %v", v, h)
	}
}

func TestScrollViewHorizontalNeededOnlyWhenWider(t *testing.T) {
	app := NewApp()
	sv := NewScrollView()
	app.SetContent(sv)
	sv.SetContent(newStub(80, 300)) // narrower than the view
	sv.SetHorizontalScroll(true)
	sv.SetBounds(geom.Rect{W: 200, H: 120})
	sv.Layout()
	if sv.hbarShown() {
		t.Error("content that fits across needs no horizontal bar")
	}
	if got := sv.viewport().H; got != 120 {
		t.Errorf("viewport height = %v, want the full 120 without a bar", got)
	}
	if sv.contentWidth() != sv.viewport().W {
		t.Errorf("narrow content should still fill the view width, got %v", sv.contentWidth())
	}
}

func TestScrollViewHorizontalThumbDragMovesContent(t *testing.T) {
	sv, content := hView(t, true)
	tr := sv.trackH()
	press := geom.Point{X: tr.X + 3, Y: tr.Y + 4} // on the thumb at offset 0
	sv.HandleEvent(&Event{Type: EventPointerDown, Button: render.MouseLeft, Pos: press})
	sv.HandleEvent(&Event{Type: EventPointerMove, Pos: geom.Point{X: press.X + 30, Y: press.Y + 60}}) // drifting off the bar is fine
	if sv.offsetX <= 0 {
		t.Fatalf("dragging the horizontal thumb left offsetX at %v", sv.offsetX)
	}
	if sv.offsetY != 0 {
		t.Errorf("a horizontal drag moved the vertical offset to %v", sv.offsetY)
	}
	sv.HandleEvent(&Event{Type: EventPointerUp, Button: render.MouseLeft, Pos: press})
	sv.Layout()
	if got, want := content.Bounds().X, sv.viewport().X-sv.offsetX; got != want {
		t.Errorf("content x = %v, want %v: the content must shift left with the offset", got, want)
	}

	// Dragging all the way stops at the far edge.
	sv.offsetX = 0 // back to the start, so the press below lands on the thumb again
	sv.HandleEvent(&Event{Type: EventPointerDown, Button: render.MouseLeft, Pos: press})
	sv.HandleEvent(&Event{Type: EventPointerMove, Pos: geom.Point{X: press.X + 5000, Y: press.Y}})
	sv.HandleEvent(&Event{Type: EventPointerUp, Button: render.MouseLeft, Pos: press})
	if want := 600 - sv.viewport().W; sv.offsetX != want {
		t.Errorf("offsetX at the end = %v, want %v", sv.offsetX, want)
	}
}

func TestScrollViewHorizontalGutterPages(t *testing.T) {
	sv, _ := hView(t, true)
	tr := sv.trackH()
	p := geom.Point{X: tr.X + tr.W - 3, Y: tr.Y + 4} // right of the thumb
	sv.HandleEvent(&Event{Type: EventPointerDown, Button: render.MouseLeft, Pos: p})
	sv.HandleEvent(&Event{Type: EventPointerUp, Button: render.MouseLeft, Pos: p})
	if want := sv.viewport().W; sv.offsetX != want {
		t.Errorf("a click right of the thumb: offsetX = %v, want one view (%v)", sv.offsetX, want)
	}
}

func TestScrollViewHorizontalWheel(t *testing.T) {
	sv, _ := hView(t, true)

	sv.HandleEvent(&Event{Type: EventWheel, Wheel: geom.Point{X: -1}})
	if sv.offsetX != wheelStep || sv.offsetY != 0 {
		t.Errorf("sideways wheel: x=%v y=%v, want x=%v y=0", sv.offsetX, sv.offsetY, float64(wheelStep))
	}

	sv.HandleEvent(&Event{Type: EventWheel, Wheel: geom.Point{Y: -1}, Modifiers: render.ModifierSet(render.ModShift)})
	if sv.offsetX != 2*wheelStep || sv.offsetY != 0 {
		t.Errorf("shift+wheel: x=%v y=%v, want x=%v y=0", sv.offsetX, sv.offsetY, float64(2*wheelStep))
	}

	sv.HandleEvent(&Event{Type: EventWheel, Wheel: geom.Point{Y: -1}})
	if sv.offsetY != wheelStep || sv.offsetX != 2*wheelStep {
		t.Errorf("plain wheel: x=%v y=%v, want y=%v only", sv.offsetX, sv.offsetY, float64(wheelStep))
	}
}

func TestScrollViewShiftWheelScrollsVerticallyWhenNothingToScrollAcross(t *testing.T) {
	app := NewApp()
	sv := NewScrollView()
	app.SetContent(sv)
	sv.SetContent(newStub(80, 900))
	sv.SetHorizontalScroll(true)
	sv.SetBounds(geom.Rect{W: 200, H: 120})
	sv.Layout()
	sv.HandleEvent(&Event{Type: EventWheel, Wheel: geom.Point{Y: -1}, Modifiers: render.ModifierSet(render.ModShift)})
	if sv.offsetY != wheelStep {
		t.Errorf("with no horizontal range shift+wheel should scroll vertically, offsetY = %v", sv.offsetY)
	}
}

func TestScrollViewBarsKeepTheirOwnEvents(t *testing.T) {
	sv, _ := hView(t, true)
	// Scrolled content extends under both bars; they must still win the hit test.
	for name, p := range map[string]geom.Point{
		"vertical bar":   {X: sv.track().X + 4, Y: sv.track().Y + 10},
		"horizontal bar": {X: sv.trackH().X + 10, Y: sv.trackH().Y + 4},
		"corner":         {X: sv.Bounds().X + sv.Bounds().W - 3, Y: sv.Bounds().Y + sv.Bounds().H - 3},
	} {
		if got := hitTest(sv, p); got != Widget(sv) {
			t.Errorf("%s: hit %v, want the ScrollView itself", name, got)
		}
	}
	if got := hitTest(sv, geom.Point{X: 20, Y: 30}); got == Widget(sv) {
		t.Error("a point inside the viewport should still reach the content")
	}
}

func TestScrollViewHorizontalClampsAfterContentShrinks(t *testing.T) {
	sv, _ := hView(t, true)
	sv.offsetX, sv.offsetY = 400, 150
	sv.SetContent(newStub(300, 130))
	sv.SetBounds(sv.Bounds())
	sv.Layout()
	if mx := sv.maxOffsetX(); sv.offsetX > mx {
		t.Errorf("offsetX %v beyond the new max %v", sv.offsetX, mx)
	}
	if my := sv.maxOffset(); sv.offsetY > my {
		t.Errorf("offsetY %v beyond the new max %v", sv.offsetY, my)
	}
}

func TestScrollViewTurningHorizontalOffResets(t *testing.T) {
	sv, content := hView(t, true)
	sv.offsetX = 150
	sv.SetHorizontalScroll(false)
	sv.Layout()
	if sv.offsetX != 0 || sv.hbarShown() {
		t.Errorf("offsetX=%v hbar=%v after turning it off, want 0 and none", sv.offsetX, sv.hbarShown())
	}
	if content.Bounds().X != sv.viewport().X {
		t.Errorf("content x = %v, want it back at the left edge %v", content.Bounds().X, sv.viewport().X)
	}
}
