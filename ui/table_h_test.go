package ui

import (
	"fmt"
	"testing"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
)

// hTable returns a Table with five columns, w x h, with nrows rows.
func hTable(t *testing.T, hscroll bool, w, h float64, nrows int, cols ...Column) *Table {
	t.Helper()
	if cols == nil {
		cols = []Column{{Title: "A"}, {Title: "B"}, {Title: "C"}, {Title: "D"}, {Title: "E"}}
	}
	app := NewApp()
	tb := NewTable(cols)
	app.SetContent(tb)
	tb.SetHorizontalScroll(hscroll)
	tb.SetBounds(geom.Rect{X: 10, Y: 20, W: w, H: h})
	for i := 0; i < nrows; i++ {
		row := make([]string, len(cols))
		for c := range row {
			row[c] = fmt.Sprintf("r%dc%d", i, c)
		}
		tb.AddRow(row...)
	}
	return tb
}

func sum(xs []float64) (s float64) {
	for _, x := range xs {
		s += x
	}
	return
}

func TestFlexWithMins(t *testing.T) {
	cases := []struct {
		name    string
		total   float64
		weights []int
		mins    []float64
		want    []float64
	}{
		{"no minimum bites", 300, []int{1, 1, 1}, []float64{0, 0, 0}, []float64{100, 100, 100}},
		{"a wide minimum pins one column", 500, []int{1, 1, 1}, []float64{300, 0, 0}, []float64{300, 100, 100}},
		{"pinning cascades", 300, []int{1, 1, 1}, []float64{140, 80, 0}, []float64{140, 80, 80}},
		{"weights still apply to the rest", 400, []int{1, 3}, []float64{0, 0}, []float64{100, 300}},
		{"exactly the minimums", 240, []int{1, 1, 1}, []float64{80, 80, 80}, []float64{80, 80, 80}},
	}
	for _, c := range cases {
		got := flexWithMins(c.total, c.weights, c.mins)
		for i := range got {
			if !near(got[i], c.want[i]) {
				t.Errorf("%s: widths %v, want %v", c.name, got, c.want)
				break
			}
		}
		if !near(sum(got), c.total) {
			t.Errorf("%s: widths %v sum to %v, want %v", c.name, got, sum(got), c.total)
		}
	}
}

func TestTableHorizontalOffByDefault(t *testing.T) {
	tb := hTable(t, false, 300, 150, 3, Column{Title: "A", MinWidth: 200}, Column{Title: "B", MinWidth: 200})
	g := tb.geomInfo()
	if g.hOver || tb.maxOffsetX() != 0 {
		t.Error("no horizontal bar unless horizontal scrolling is on, whatever MinWidth says")
	}
	if w := tb.colWidths(); !near(w[0], 150) || !near(w[1], 150) {
		t.Errorf("widths %v: columns should split the width by weight, ignoring MinWidth", w)
	}
	tb.HandleEvent(&Event{Type: EventWheel, Wheel: geom.Point{X: -2}})
	tb.HandleEvent(&Event{Type: EventKeyDown, Key: render.KeyRight})
	if tb.offsetX != 0 {
		t.Error("sideways input must do nothing while horizontal scrolling is off")
	}
}

func TestTableHorizontalFitsWhenWideEnough(t *testing.T) {
	tb := hTable(t, true, 600, 150, 3) // 5 x 80 = 400 < 600
	g := tb.geomInfo()
	if g.hOver {
		t.Fatal("columns that fit need no bar")
	}
	w := tb.colWidths()
	if !near(sum(w), g.viewW) || !near(w[0], 120) {
		t.Errorf("widths %v: spare width should still be shared by weight (want 5 x 120)", w)
	}
	if g.bodyH != 150-tb.rowHeight() {
		t.Errorf("body height %v: without a bar the body keeps all its height", g.bodyH)
	}
}

func TestTableHorizontalOverflow(t *testing.T) {
	tb := hTable(t, true, 300, 150, 3) // 400 > 300
	g := tb.geomInfo()
	if !g.hOver {
		t.Fatal("columns wider than the table should show the bar")
	}
	if !near(g.totalW, 400) || !near(g.viewW, 300) || !near(tb.maxOffsetX(), 100) {
		t.Errorf("totalW=%v viewW=%v maxOffsetX=%v, want 400, 300, 100", g.totalW, g.viewW, tb.maxOffsetX())
	}
	if want := 150 - tb.rowHeight() - scrollbarWidth; !near(g.bodyH, want) {
		t.Errorf("body height %v, want %v: the bar takes a strip off the body", g.bodyH, want)
	}
	for c, w := range tb.colWidths() {
		if w < defaultColMinWidth-0.001 {
			t.Errorf("column %d is %v wide, under the minimum %v", c, w, float64(defaultColMinWidth))
		}
	}
}

func TestTableBothBarsInteract(t *testing.T) {
	// Rows overflow only once the horizontal bar has taken its strip: 4 rows fit
	// exactly in the body, and the bar's 12px pushes the last one out.
	tb := hTable(t, true, 300, 0, 0)
	rh := tb.rowHeight()
	tb.SetBounds(geom.Rect{X: 10, Y: 20, W: 300, H: rh * 5})
	for i := 0; i < 4; i++ {
		tb.AddRow("a", "b", "c", "d", "e")
	}
	g := tb.geomInfo()
	if !g.hOver || !g.vOver {
		t.Fatalf("hOver=%v vOver=%v, want both: the bar's strip must make the rows overflow", g.hOver, g.vOver)
	}
	if !near(g.viewW, 300-scrollbarWidth) {
		t.Errorf("viewW=%v, want the vertical gutter taken off (%v)", g.viewW, 300-scrollbarWidth)
	}
	if !near(g.totalW, 400) {
		t.Errorf("totalW=%v, want 400", g.totalW)
	}
	if tb.overflow() != g.vOver || tb.bodyHeight() != g.bodyH {
		t.Error("overflow()/bodyHeight() must agree with geomInfo")
	}
	// The vertical track stops above the corner.
	v, h := tb.scrollTrack(), tb.scrollTrackH()
	if v.Y+v.H > h.Y+0.001 || h.X+h.W > v.X+0.001 {
		t.Errorf("tracks overlap: vertical %v horizontal %v", v, h)
	}
}

func TestTableMinWidthOption(t *testing.T) {
	tb := hTable(t, true, 500, 150, 2, Column{Title: "ID", MinWidth: 40}, Column{Title: "Name", MinWidth: 300}, Column{Title: "Notes"})
	w := tb.colWidths()
	if w[0] < 40 || w[1] < 300 || w[2] < defaultColMinWidth {
		t.Errorf("widths %v violate the minimums 40 / 300 / %v", w, float64(defaultColMinWidth))
	}
	// 40 + 300 + 80 = 420 < 500: fits, the spare is shared by weight.
	if tb.geomInfo().hOver || !near(sum(w), 500) {
		t.Errorf("widths %v should fill the 500px with no bar", w)
	}
}

func TestTableHorizontalThumbDragShiftsColumns(t *testing.T) {
	tb := hTable(t, true, 300, 150, 3)
	tr := tb.scrollTrackH()
	press := geom.Point{X: tr.X + 3, Y: tr.Y + 4}
	tb.HandleEvent(&Event{Type: EventPointerDown, Button: render.MouseLeft, Pos: press})
	tb.HandleEvent(&Event{Type: EventPointerMove, Pos: geom.Point{X: press.X + 200, Y: press.Y - 80}})
	tb.HandleEvent(&Event{Type: EventPointerUp, Button: render.MouseLeft, Pos: press})
	if !near(tb.offsetX, 100) {
		t.Fatalf("dragging the thumb to the end: offsetX = %v, want the max 100", tb.offsetX)
	}
	xs := tb.colX(tb.colWidths())
	if !near(xs[0], tb.Bounds().X-100) {
		t.Errorf("first column x = %v, want %v: columns shift left with the offset", xs[0], tb.Bounds().X-100)
	}
	if tb.offset != 0 || tb.Selected() != -1 {
		t.Error("a horizontal drag must not scroll vertically or select anything")
	}
}

func TestTableHeaderAndRowsFollowTheScroll(t *testing.T) {
	tb := hTable(t, true, 300, 150, 3)
	tb.offsetX = 100 // columns now start at x = 10-100
	widths := tb.colWidths()
	xs := tb.colX(widths)

	// A header click in column 3's scrolled position hits column 3, not column 0.
	hx := xs[3] + widths[3]/2
	hy := tb.Bounds().Y + tb.rowHeight()/2
	if got := tb.headerColAt(geom.Point{X: hx, Y: hy}); got != 3 {
		t.Errorf("header at x=%v hit column %d, want 3", hx, got)
	}

	// A body click selects its row, wherever it has scrolled to.
	by := tb.Bounds().Y + tb.rowHeight()*1.5
	tb.HandleEvent(&Event{Type: EventPointerDown, Button: render.MouseLeft, Pos: geom.Point{X: hx, Y: by}})
	tb.HandleEvent(&Event{Type: EventClick, Button: render.MouseLeft, Pos: geom.Point{X: hx, Y: by}})
	if tb.Selected() != 0 {
		t.Errorf("selected %d, want row 0", tb.Selected())
	}
}

func TestTableBarStripIsNotARow(t *testing.T) {
	tb := hTable(t, true, 300, 150, 40) // many rows, so one would be under the strip
	p := geom.Point{X: 50, Y: tb.scrollTrackH().Y + 4}
	if got := tb.rowAt(p.Y); got != -1 {
		t.Errorf("rowAt in the horizontal bar = %d, want -1", got)
	}
	tb.HandleEvent(&Event{Type: EventPointerDown, Button: render.MouseLeft, Pos: p})
	tb.HandleEvent(&Event{Type: EventPointerUp, Button: render.MouseLeft, Pos: p})
	tb.HandleEvent(&Event{Type: EventClick, Button: render.MouseLeft, Pos: p})
	if tb.Selected() != -1 {
		t.Errorf("clicking the bar selected row %d", tb.Selected())
	}
}

func TestTableHorizontalWheelAndKeys(t *testing.T) {
	tb := hTable(t, true, 300, 150, 40)

	tb.HandleEvent(&Event{Type: EventWheel, Wheel: geom.Point{X: -1}})
	if tb.offsetX != wheelStep || tb.offset != 0 {
		t.Errorf("sideways wheel: x=%v y=%v, want x=%v y=0", tb.offsetX, tb.offset, float64(wheelStep))
	}
	tb.HandleEvent(&Event{Type: EventWheel, Wheel: geom.Point{Y: -1}, Modifiers: render.ModifierSet(render.ModShift)})
	if tb.offsetX != 2*wheelStep || tb.offset != 0 {
		t.Errorf("shift+wheel: x=%v y=%v, want x=%v y=0", tb.offsetX, tb.offset, float64(2*wheelStep))
	}
	tb.HandleEvent(&Event{Type: EventWheel, Wheel: geom.Point{Y: -1}})
	if tb.offset != wheelStep {
		t.Errorf("plain wheel: y=%v, want %v", tb.offset, float64(wheelStep))
	}

	tb.offsetX = 0
	if !tb.HandleEvent(&Event{Type: EventKeyDown, Key: render.KeyRight}) || tb.offsetX != wheelStep {
		t.Errorf("Right: offsetX = %v, want %v", tb.offsetX, float64(wheelStep))
	}
	tb.HandleEvent(&Event{Type: EventKeyDown, Key: render.KeyLeft})
	tb.HandleEvent(&Event{Type: EventKeyDown, Key: render.KeyLeft})
	if tb.offsetX != 0 {
		t.Errorf("Left past the start: offsetX = %v, want clamped 0", tb.offsetX)
	}
}

func TestTableShiftWheelFallsBackToVerticalWhenNothingToScrollAcross(t *testing.T) {
	tb := hTable(t, true, 600, 150, 40) // columns fit
	tb.HandleEvent(&Event{Type: EventWheel, Wheel: geom.Point{Y: -1}, Modifiers: render.ModifierSet(render.ModShift)})
	if tb.offset != wheelStep {
		t.Errorf("with no horizontal range shift+wheel should scroll down, offset = %v", tb.offset)
	}
}

func TestTableHorizontalClampsAndResets(t *testing.T) {
	tb := hTable(t, true, 300, 150, 3)
	tb.offsetX = 100
	tb.SetBounds(geom.Rect{X: 10, Y: 20, W: 500, H: 150}) // now everything fits
	tb.clamp()
	if tb.offsetX != 0 {
		t.Errorf("offsetX = %v after widening, want clamped to 0", tb.offsetX)
	}
	tb.SetBounds(geom.Rect{X: 10, Y: 20, W: 300, H: 150})
	tb.offsetX = 60
	tb.SetHorizontalScroll(false)
	if tb.offsetX != 0 || tb.geomInfo().hOver {
		t.Errorf("turning it off left offsetX=%v hOver=%v", tb.offsetX, tb.geomInfo().hOver)
	}
}

func TestTableHorizontalMinSize(t *testing.T) {
	wide := hTable(t, false, 300, 150, 0)
	scroll := hTable(t, true, 300, 150, 0)
	if scroll.MinSize().W >= wide.MinSize().W && wide.MinSize().W > defaultColMinWidth {
		t.Errorf("MinSize width %v with scrolling vs %v without: scrolling must not demand more", scroll.MinSize().W, wide.MinSize().W)
	}
	if scroll.MinSize().W != defaultColMinWidth {
		t.Errorf("scrolling MinSize width = %v, want %v", scroll.MinSize().W, float64(defaultColMinWidth))
	}
}
