package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
)

func near(a, b float64) bool { return a-b < 0.001 && b-a < 0.001 }

func TestThumbRectGeometry(t *testing.T) {
	track := geom.Rect{X: 100, Y: 10, W: scrollbarWidth, H: 200}

	// Nothing to scroll: the thumb fills the track.
	if th := thumbRect(track, true, scrollMetrics{content: 150, view: 200}); th != track {
		t.Errorf("fitting content: thumb %v, want the whole track %v", th, track)
	}
	// Half visible: half-length thumb at the top, then at the bottom at max offset.
	m := scrollMetrics{content: 400, view: 200}
	if th := thumbRect(track, true, m); !near(th.H, 100) || !near(th.Y, 10) || th.X != track.X || th.W != track.W {
		t.Errorf("top thumb %v, want 100 tall at y=10", th)
	}
	m.offset = 200
	if th := thumbRect(track, true, m); !near(th.Y+th.H, track.Y+track.H) {
		t.Errorf("bottom thumb %v should end at the track end %v", th, track.Y+track.H)
	}
	// A huge document keeps a grabbable minimum thumb.
	if th := thumbRect(track, true, scrollMetrics{content: 1e6, view: 200}); th.H != minThumb {
		t.Errorf("min thumb = %v, want %v", th.H, minThumb)
	}
	// A track shorter than the minimum never gets a thumb longer than itself.
	short := geom.Rect{W: scrollbarWidth, H: 10}
	if th := thumbRect(short, true, scrollMetrics{content: 1e6, view: 10}); th.H > 10 {
		t.Errorf("thumb %v overflows a 10px track", th.H)
	}
	// Horizontal swaps the axes.
	htrack := geom.Rect{X: 0, Y: 50, W: 200, H: scrollbarWidth}
	hm := scrollMetrics{content: 400, view: 200, offset: 100}
	th := thumbRect(htrack, false, hm)
	if !near(th.W, 100) || !near(th.X, 50) || th.H != scrollbarWidth || th.Y != 50 {
		t.Errorf("horizontal thumb %v, want 100 wide at x=50", th)
	}
}

func TestBarDragProportionalAndCaptured(t *testing.T) {
	track := geom.Rect{X: 0, Y: 0, W: scrollbarWidth, H: 200}
	m := scrollMetrics{content: 400, view: 200} // thumb 100 tall, travel 100, max offset 200
	var d barDrag
	at := func(ty EventType, y float64) *Event {
		return &Event{Type: ty, Button: render.MouseLeft, Pos: geom.Point{X: 5, Y: y}}
	}

	if _, _, ok := d.handle(at(EventPointerDown, 50), track, true, true, m); !ok || !d.active {
		t.Fatal("a press on the thumb should start a drag")
	}
	// Drag 50px down: half the travel, so half the max offset (200 * 50/100 = 100).
	off, changed, ok := d.handle(at(EventPointerMove, 100), track, true, true, m)
	if !ok || !changed || !near(off, 100) {
		t.Errorf("drag to y=100: off=%v changed=%v ok=%v, want 100", off, changed, ok)
	}
	// Far past the end clamps, and the pointer need not stay on the bar.
	off, _, _ = d.handle(at(EventPointerMove, 5000), track, true, true, m)
	if !near(off, 200) {
		t.Errorf("drag past the end: off=%v, want clamped 200", off)
	}
	off, _, _ = d.handle(at(EventPointerMove, -5000), track, true, true, m)
	if off != 0 {
		t.Errorf("drag past the start: off=%v, want 0", off)
	}
	if _, _, ok := d.handle(at(EventPointerUp, 0), track, true, true, m); !ok || d.active {
		t.Error("release should end the drag and be consumed")
	}
	if _, _, ok := d.handle(at(EventPointerMove, 80), track, true, true, m); ok {
		t.Error("moves after the drag ended are not the bar's")
	}
}

func TestBarGutterPagesTowardsClick(t *testing.T) {
	track := geom.Rect{X: 0, Y: 0, W: scrollbarWidth, H: 200}
	m := scrollMetrics{content: 1000, view: 200, offset: 300}
	var d barDrag
	press := func(y float64) float64 {
		off, _, ok := d.handle(&Event{Type: EventPointerDown, Button: render.MouseLeft, Pos: geom.Point{X: 5, Y: y}}, track, true, true, m)
		if !ok {
			t.Fatal("a press on the gutter is the bar's")
		}
		return off
	}
	th := thumbRect(track, true, m)
	if off := press(th.Y + th.H + 5); !near(off, 500) {
		t.Errorf("click below the thumb: off=%v, want one view down (500)", off)
	}
	if off := press(th.Y - 5); !near(off, 100) {
		t.Errorf("click above the thumb: off=%v, want one view up (100)", off)
	}
	if d.active {
		t.Error("a gutter click must not start a drag")
	}
}

func TestBarIgnoresWhenHiddenOrOtherButtons(t *testing.T) {
	track := geom.Rect{X: 0, Y: 0, W: scrollbarWidth, H: 200}
	m := scrollMetrics{content: 400, view: 200}
	var d barDrag
	down := &Event{Type: EventPointerDown, Button: render.MouseLeft, Pos: geom.Point{X: 5, Y: 5}}
	if _, _, ok := d.handle(down, track, false, true, m); ok {
		t.Error("a hidden bar must not take presses")
	}
	right := &Event{Type: EventPointerDown, Button: render.MouseRight, Pos: geom.Point{X: 5, Y: 5}}
	if _, _, ok := d.handle(right, track, true, true, m); ok {
		t.Error("only the left button drives a bar")
	}
	off := &Event{Type: EventPointerDown, Button: render.MouseLeft, Pos: geom.Point{X: 500, Y: 5}}
	if _, _, ok := d.handle(off, track, true, true, m); ok {
		t.Error("a press off the track is not the bar's")
	}
	click := &Event{Type: EventClick, Pos: geom.Point{X: 5, Y: 5}}
	if _, _, ok := d.handle(click, track, true, true, m); !ok {
		t.Error("the click ending a press on the bar should be swallowed")
	}
}

func TestScrollBarWidget(t *testing.T) {
	for _, dir := range []geom.Direction{geom.Vertical, geom.Horizontal} {
		name := map[geom.Direction]string{geom.Vertical: "vertical", geom.Horizontal: "horizontal"}[dir]
		t.Run(name, func(t *testing.T) {
			sb := NewScrollBar(dir)
			vert := dir == geom.Vertical
			if vert {
				sb.SetBounds(geom.Rect{X: 0, Y: 0, W: scrollbarWidth, H: 200})
			} else {
				sb.SetBounds(geom.Rect{X: 0, Y: 0, W: 200, H: scrollbarWidth})
			}
			sb.SetRange(400, 200) // thumb 100 long, travel 100
			var got []float64
			sb.OnScroll(func(o float64) { got = append(got, o) })
			pt := func(along float64) geom.Point {
				if vert {
					return geom.Point{X: 5, Y: along}
				}
				return geom.Point{X: along, Y: 5}
			}
			ev := func(ty EventType, along float64) *Event {
				return &Event{Type: ty, Button: render.MouseLeft, Pos: pt(along)}
			}

			sb.HandleEvent(ev(EventPointerDown, 50))
			sb.HandleEvent(ev(EventPointerMove, 100))
			sb.HandleEvent(ev(EventPointerUp, 100))
			if !near(sb.Offset(), 100) || len(got) != 1 || !near(got[0], 100) {
				t.Fatalf("drag: offset=%v callbacks=%v, want 100 once", sb.Offset(), got)
			}

			// SetOffset moves the thumb without calling back, and clamps.
			sb.SetOffset(9999)
			if !near(sb.Offset(), 200) || len(got) != 1 {
				t.Errorf("SetOffset: offset=%v callbacks=%v, want 200 and no callback", sb.Offset(), got)
			}
			// SetRange re-clamps an offset that no longer fits.
			sb.SetRange(300, 200)
			if !near(sb.Offset(), 100) {
				t.Errorf("SetRange: offset=%v, want clamped to 100", sb.Offset())
			}

			// The wheel scrolls (the horizontal bar also takes a sideways wheel).
			sb.SetOffset(0)
			got = nil
			w := geom.Point{Y: -1}
			if !vert {
				w = geom.Point{X: -1}
			}
			sb.HandleEvent(&Event{Type: EventWheel, Wheel: w})
			if len(got) != 1 || !near(got[0], wheelStep) {
				t.Errorf("wheel: callbacks=%v, want [%v]", got, float64(wheelStep))
			}

			sb.SetEnabled(false)
			if sb.HandleEvent(ev(EventPointerDown, 5)) {
				t.Error("a disabled bar takes no events")
			}
		})
	}
}

// scroller adapts the built-in scrolling widgets to one test.
type scroller struct {
	name    string
	w       Widget
	track   geom.Rect
	offset  func() float64
	clicked func() bool // whether a click was acted on (selection etc.)
}

func newScrollers(t *testing.T) []scroller {
	t.Helper()
	var out []scroller

	items := make([]string, 200)
	for i := range items {
		items[i] = fmt.Sprintf("row %d", i)
	}
	app := NewApp()
	l := NewList(items)
	app.SetContent(l)
	l.SetBounds(geom.Rect{W: 200, H: 120})
	out = append(out, scroller{"list", l, l.scrollTrack(), func() float64 { return l.offset }, func() bool { return l.Selected() >= 0 }})

	app2 := NewApp()
	var roots []*TreeNode
	for i := 0; i < 100; i++ {
		roots = append(roots, NewTreeNode(fmt.Sprintf("n%d", i)))
	}
	tr := NewTree(roots...)
	app2.SetContent(tr)
	tr.SetBounds(geom.Rect{W: 200, H: 120})
	out = append(out, scroller{"tree", tr, tr.scrollTrack(), func() float64 { return tr.offset }, func() bool { return tr.Selected() != nil }})

	app3 := NewApp()
	tb := NewTable([]Column{{Title: "A"}, {Title: "B"}})
	app3.SetContent(tb)
	tb.SetBounds(geom.Rect{W: 200, H: 120})
	for i := 0; i < 100; i++ {
		tb.AddRow(fmt.Sprint(i), "x")
	}
	out = append(out, scroller{"table", tb, tb.scrollTrack(), func() float64 { return tb.offset }, func() bool { return tb.selected >= 0 }})

	app4 := NewApp()
	ta := NewTextArea()
	app4.SetContent(ta)
	ta.SetBounds(geom.Rect{W: 200, H: 100})
	ta.SetText(strings.Repeat("line\n", 100) + "end")
	ta.caretRow, ta.caretCol = 0, 0
	ta.collapse()
	out = append(out, scroller{"textarea", ta, ta.scrollTrack(), func() float64 { return ta.scrollY },
		func() bool { return ta.caretRow != 0 || ta.hasSelection() }})

	app5 := NewApp()
	sv := NewScrollView()
	app5.SetContent(sv)
	sv.SetContent(newStub(100, 2000))
	sv.SetBounds(geom.Rect{W: 200, H: 120})
	sv.Layout()
	out = append(out, scroller{"scrollview", sv, sv.track(), func() float64 { return sv.offsetY }, func() bool { return false }})
	return out
}

func TestEveryScrollingWidgetDragsItsThumb(t *testing.T) {
	for _, s := range newScrollers(t) {
		t.Run(s.name, func(t *testing.T) {
			top := geom.Point{X: s.track.X + 4, Y: s.track.Y + 3} // on the thumb at offset 0
			s.w.HandleEvent(&Event{Type: EventPointerDown, Button: render.MouseLeft, Pos: top})
			s.w.HandleEvent(&Event{Type: EventPointerMove, Pos: geom.Point{X: top.X, Y: top.Y + 40}})
			if s.offset() <= 0 {
				t.Fatalf("dragging the thumb 40px down left the offset at %v", s.offset())
			}
			mid := s.offset()
			// The pointer can leave the bar entirely and the drag continues.
			s.w.HandleEvent(&Event{Type: EventPointerMove, Pos: geom.Point{X: top.X - 150, Y: top.Y + 70}})
			if s.offset() <= mid {
				t.Errorf("drag off the bar: offset %v, want more than %v", s.offset(), mid)
			}
			s.w.HandleEvent(&Event{Type: EventPointerUp, Button: render.MouseLeft, Pos: top})
			after := s.offset()
			s.w.HandleEvent(&Event{Type: EventPointerMove, Pos: geom.Point{X: top.X, Y: top.Y + 100}})
			if s.offset() != after {
				t.Error("moves after the release must not keep scrolling")
			}
		})
	}
}

func TestEveryScrollingWidgetPagesFromTheGutter(t *testing.T) {
	for _, s := range newScrollers(t) {
		t.Run(s.name, func(t *testing.T) {
			low := geom.Point{X: s.track.X + 4, Y: s.track.Y + s.track.H - 3} // below the thumb
			s.w.HandleEvent(&Event{Type: EventPointerDown, Button: render.MouseLeft, Pos: low})
			s.w.HandleEvent(&Event{Type: EventPointerUp, Button: render.MouseLeft, Pos: low})
			if s.offset() <= 0 {
				t.Errorf("a click in the gutter below the thumb left the offset at %v", s.offset())
			}
		})
	}
}

func TestClicksOnTheBarDoNotReachTheContent(t *testing.T) {
	for _, s := range newScrollers(t) {
		t.Run(s.name, func(t *testing.T) {
			p := geom.Point{X: s.track.X + 4, Y: s.track.Y + s.track.H/2}
			s.w.HandleEvent(&Event{Type: EventPointerDown, Button: render.MouseLeft, Pos: p})
			s.w.HandleEvent(&Event{Type: EventPointerUp, Button: render.MouseLeft, Pos: p})
			s.w.HandleEvent(&Event{Type: EventClick, Button: render.MouseLeft, Pos: p})
			if s.clicked() {
				t.Error("a click on the scrollbar selected or moved something underneath it")
			}
		})
	}
}

func TestNoBarWhenContentFits(t *testing.T) {
	app := NewApp()
	l := NewList([]string{"a", "b"})
	app.SetContent(l)
	l.SetBounds(geom.Rect{W: 200, H: 200})
	p := geom.Point{X: 200 - scrollbarWidth/2, Y: l.RowHeight() / 2}
	l.HandleEvent(&Event{Type: EventPointerDown, Button: render.MouseLeft, Pos: p})
	l.HandleEvent(&Event{Type: EventClick, Button: render.MouseLeft, Pos: p})
	if l.Selected() != 0 {
		t.Errorf("without overflow there is no bar, so the click should select row 0; selected %d", l.Selected())
	}
}
