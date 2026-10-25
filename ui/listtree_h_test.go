package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
)

var wideItem = strings.Repeat("W", 80)

// hList returns a List of n short rows plus one very wide one, w x h.
func hList(t *testing.T, hscroll bool, w, h float64, n int) *List {
	t.Helper()
	items := make([]string, 0, n+1)
	for i := 0; i < n; i++ {
		items = append(items, fmt.Sprintf("row %d", i))
	}
	items = append(items, wideItem)
	app := NewApp()
	l := NewList(items)
	app.SetContent(l)
	l.SetHorizontalScroll(hscroll)
	l.SetBounds(geom.Rect{X: 10, Y: 20, W: w, H: h})
	return l
}

func TestCarveBars(t *testing.T) {
	g := carveBars(200, 100, 150, 80, true)
	if g.vOver || g.hOver || g.viewW != 200 || g.viewH != 100 {
		t.Errorf("everything fits: %+v", g)
	}
	g = carveBars(200, 100, 150, 500, true)
	if !g.vOver || g.hOver || g.viewW != 200-scrollbarWidth {
		t.Errorf("tall only: %+v", g)
	}
	g = carveBars(200, 100, 500, 80, true)
	if g.vOver || !g.hOver || g.viewH != 100-scrollbarWidth {
		t.Errorf("wide only: %+v", g)
	}
	// 90px of rows fit in 100, but not once the horizontal bar takes 12.
	g = carveBars(200, 100, 500, 95, true)
	if !g.vOver || !g.hOver || g.viewW != 200-scrollbarWidth || g.viewH != 100-scrollbarWidth {
		t.Errorf("each bar forces the other: %+v", g)
	}
	// Horizontal scrolling off: content width is ignored.
	g = carveBars(200, 100, 5000, 80, false)
	if g.hOver || g.viewH != 100 {
		t.Errorf("hscroll off: %+v", g)
	}
}

func TestListHorizontalOffByDefault(t *testing.T) {
	l := hList(t, false, 150, 100, 3)
	if l.geomInfo().hOver || l.maxOffsetX() != 0 {
		t.Error("no horizontal bar unless horizontal scrolling is on")
	}
	l.HandleEvent(&Event{Type: EventWheel, Wheel: geom.Point{X: -2}})
	l.HandleEvent(&Event{Type: EventKeyDown, Key: render.KeyRight})
	if l.offsetX != 0 {
		t.Error("sideways input must do nothing while horizontal scrolling is off")
	}
	if l.MinSize().W < 100 {
		t.Errorf("MinSize width %v: without scrolling the list still asks for its widest row", l.MinSize().W)
	}
}

func TestListHorizontalOverflow(t *testing.T) {
	l := hList(t, true, 150, 100, 3)
	g := l.geomInfo()
	if !g.hOver {
		t.Fatal("a row wider than the view should show the bar")
	}
	if !near(g.viewH, 100-scrollbarWidth) {
		t.Errorf("view height %v, want the bar's strip taken off", g.viewH)
	}
	if want := l.widest() - g.viewW; !near(l.maxOffsetX(), want) || want <= 0 {
		t.Errorf("maxOffsetX = %v, want %v", l.maxOffsetX(), want)
	}
	if l.MinSize().W >= 100+scrollbarWidth {
		t.Errorf("MinSize width %v: with scrolling the list must not demand its widest row", l.MinSize().W)
	}
}

func TestListWidthCacheTracksItemEdits(t *testing.T) {
	l := hList(t, true, 150, 100, 3)
	w0 := l.widest()
	l.items[0] = strings.Repeat("W", 200) // the caller edits its own slice
	if l.widest() <= w0 {
		t.Error("an edited item should be re-measured")
	}
	l.items[0] = "row 0"
	if got := l.widest(); !near(got, w0) {
		t.Errorf("width %v after restoring the item, want %v", got, w0)
	}
	l.items = append(l.items, strings.Repeat("W", 400))
	if l.widest() <= w0 {
		t.Error("a longer appended item should be measured")
	}
}

func TestListHorizontalDragWheelKeys(t *testing.T) {
	l := hList(t, true, 150, 100, 3)
	tr := l.scrollTrackH()
	press := geom.Point{X: tr.X + 3, Y: tr.Y + 4}
	l.HandleEvent(&Event{Type: EventPointerDown, Button: render.MouseLeft, Pos: press})
	l.HandleEvent(&Event{Type: EventPointerMove, Pos: geom.Point{X: press.X + 5000, Y: press.Y - 60}})
	l.HandleEvent(&Event{Type: EventPointerUp, Button: render.MouseLeft, Pos: press})
	if !near(l.offsetX, l.maxOffsetX()) || l.offset != 0 {
		t.Fatalf("drag to the end: offsetX=%v want %v, offset=%v", l.offsetX, l.maxOffsetX(), l.offset)
	}

	l.offsetX = 0
	l.HandleEvent(&Event{Type: EventWheel, Wheel: geom.Point{X: -1}})
	if l.offsetX != wheelStep {
		t.Errorf("sideways wheel: offsetX = %v", l.offsetX)
	}
	l.HandleEvent(&Event{Type: EventWheel, Wheel: geom.Point{Y: -1}, Modifiers: render.ModifierSet(render.ModShift)})
	if l.offsetX != 2*wheelStep {
		t.Errorf("shift+wheel: offsetX = %v", l.offsetX)
	}
	l.HandleEvent(&Event{Type: EventKeyDown, Key: render.KeyLeft})
	l.HandleEvent(&Event{Type: EventKeyDown, Key: render.KeyLeft})
	l.HandleEvent(&Event{Type: EventKeyDown, Key: render.KeyLeft})
	if l.offsetX != 0 {
		t.Errorf("Left past the start: offsetX = %v, want 0", l.offsetX)
	}
	if !l.HandleEvent(&Event{Type: EventKeyDown, Key: render.KeyRight}) || l.offsetX != wheelStep {
		t.Errorf("Right: offsetX = %v", l.offsetX)
	}
}

func TestListBarStripIsNotARow(t *testing.T) {
	l := hList(t, true, 150, 100, 40) // plenty of rows, so one would sit under the strip
	p := geom.Point{X: 40, Y: l.scrollTrackH().Y + 4}
	if got := l.rowAt(p.Y); got != -1 {
		t.Errorf("rowAt in the horizontal bar = %d, want -1", got)
	}
	l.HandleEvent(&Event{Type: EventPointerDown, Button: render.MouseLeft, Pos: p})
	l.HandleEvent(&Event{Type: EventPointerUp, Button: render.MouseLeft, Pos: p})
	l.HandleEvent(&Event{Type: EventClick, Button: render.MouseLeft, Pos: p})
	if l.Selected() != -1 {
		t.Errorf("clicking the bar selected row %d", l.Selected())
	}
}

func TestListKeyboardScrollKeepsSelectionVisibleWithBar(t *testing.T) {
	l := hList(t, true, 150, 100, 40)
	for i := 0; i < 25; i++ {
		l.HandleEvent(&Event{Type: EventKeyDown, Key: render.KeyDown})
	}
	rh := l.RowHeight()
	sel := float64(l.Selected())
	bottom := (sel+1)*rh - l.offset
	if bottom > l.geomInfo().viewH+0.001 {
		t.Errorf("selected row ends at %v, below the view height %v: it is hidden by the bar", bottom, l.geomInfo().viewH)
	}
}

func TestListHorizontalClampsAndResets(t *testing.T) {
	l := hList(t, true, 150, 100, 3)
	l.offsetX = l.maxOffsetX()
	l.SetBounds(geom.Rect{X: 10, Y: 20, W: 2000, H: 100}) // everything fits now
	l.clamp()
	if l.offsetX != 0 {
		t.Errorf("offsetX = %v after widening, want 0", l.offsetX)
	}
	l.SetBounds(geom.Rect{X: 10, Y: 20, W: 150, H: 100})
	l.offsetX = 30
	l.SetHorizontalScroll(false)
	if l.offsetX != 0 || l.geomInfo().hOver {
		t.Errorf("turning it off left offsetX=%v hOver=%v", l.offsetX, l.geomInfo().hOver)
	}
}

// hTree returns a Tree whose deepest node has a very long label.
func hTree(t *testing.T, hscroll bool, w, h float64) (*Tree, *TreeNode, *TreeNode) {
	t.Helper()
	deep := NewTreeNode(wideItem)
	mid := NewTreeNode("mid", deep)
	root := NewTreeNode("root", mid)
	app := NewApp()
	tr := NewTree(root)
	app.SetContent(tr)
	tr.SetHorizontalScroll(hscroll)
	tr.SetBounds(geom.Rect{X: 10, Y: 20, W: w, H: h})
	tr.Expand(root)
	tr.Expand(mid)
	return tr, root, deep
}

func TestTreeHorizontalOffByDefault(t *testing.T) {
	tr, _, _ := hTree(t, false, 150, 100)
	if tr.geomInfo().hOver || tr.maxOffsetX() != 0 {
		t.Error("no horizontal bar unless horizontal scrolling is on")
	}
	tr.HandleEvent(&Event{Type: EventWheel, Wheel: geom.Point{X: -2}})
	if tr.offsetX != 0 {
		t.Error("sideways wheel must do nothing while horizontal scrolling is off")
	}
}

func TestTreeHorizontalOverflowFollowsExpansion(t *testing.T) {
	tr, root, _ := hTree(t, true, 150, 100)
	g := tr.geomInfo()
	if !g.hOver || tr.maxOffsetX() <= 0 {
		t.Fatalf("the deep wide row should need the bar: %+v maxOffsetX=%v", g, tr.maxOffsetX())
	}
	if tr.MinSize().W >= 100+scrollbarWidth {
		t.Errorf("MinSize width %v: with scrolling the tree must not demand its widest row", tr.MinSize().W)
	}
	// Collapsed, only "root" is visible: nothing to scroll, and the offset resets.
	tr.offsetX = 50
	tr.Collapse(root)
	tr.clamp()
	if tr.geomInfo().hOver || tr.offsetX != 0 {
		t.Errorf("collapsed: hOver=%v offsetX=%v, want no bar and 0", tr.geomInfo().hOver, tr.offsetX)
	}
}

func TestTreeWidthIncludesIndentAndCachesLabels(t *testing.T) {
	tr, _, deep := hTree(t, true, 150, 100)
	w0 := tr.widest()
	f := tr.face()
	want := 2*treeIndent + treeChevron + treeGap + f.Measure(wideItem).W + 2*treeRowPad
	if !near(w0, want) {
		t.Errorf("widest = %v, want %v (two levels of indent)", w0, want)
	}
	deep.Label = strings.Repeat("W", 200) // a node's label is edited in place
	if tr.widest() <= w0 {
		t.Error("an edited label should be re-measured")
	}
}

func TestTreeHorizontalDragWheelAndHitTesting(t *testing.T) {
	tr, root, _ := hTree(t, true, 150, 100)
	tr.Collapse(root)
	tr.Expand(root)
	tr.Expand(root.Children()[0])
	tr.offsetX = 0
	tr.clamp()

	tb := tr.scrollTrackH()
	press := geom.Point{X: tb.X + 3, Y: tb.Y + 4}
	tr.HandleEvent(&Event{Type: EventPointerDown, Button: render.MouseLeft, Pos: press})
	tr.HandleEvent(&Event{Type: EventPointerMove, Pos: geom.Point{X: press.X + 5000, Y: press.Y - 40}})
	tr.HandleEvent(&Event{Type: EventPointerUp, Button: render.MouseLeft, Pos: press})
	if !near(tr.offsetX, tr.maxOffsetX()) || tr.maxOffsetX() <= 0 {
		t.Fatalf("drag to the end: offsetX = %v, max %v", tr.offsetX, tr.maxOffsetX())
	}

	// The chevron moves with the scroll: scrolled right, the root's chevron is off
	// the left edge and a click where it used to be does not toggle it.
	if tr.chevronHit(tr.Bounds().X+treeRowPad+1, 0) {
		t.Error("a scrolled-away chevron must not be hit at its old position")
	}
	tr.offsetX = 0
	if !tr.chevronHit(tr.Bounds().X+treeRowPad+1, 0) {
		t.Error("unscrolled, the chevron is at the left edge")
	}

	// A sideways wheel and Shift+wheel scroll; the bar strip is not a row.
	tr.offsetX = 0
	tr.HandleEvent(&Event{Type: EventWheel, Wheel: geom.Point{X: -1}})
	tr.HandleEvent(&Event{Type: EventWheel, Wheel: geom.Point{Y: -1}, Modifiers: render.ModifierSet(render.ModShift)})
	if tr.offsetX != 2*wheelStep {
		t.Errorf("offsetX = %v after two across-wheels, want %v", tr.offsetX, float64(2*wheelStep))
	}
	rows := tr.rows()
	if got := tr.rowAt(rows, tb.Y+4); got != -1 {
		t.Errorf("rowAt in the bar strip = %d, want -1", got)
	}
	tr.SetHorizontalScroll(false)
	if tr.offsetX != 0 {
		t.Errorf("turning it off left offsetX = %v", tr.offsetX)
	}
}
