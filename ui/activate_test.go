package ui

import (
	"testing"
	"time"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
)

func dbl(pos geom.Point) *Event {
	return &Event{Type: EventDoubleClick, Button: render.MouseLeft, Pos: pos}
}

func TestListDoubleClickActivatesRow(t *testing.T) {
	app := NewApp()
	l := NewList([]string{"a", "b", "c"})
	app.SetContent(l)
	l.SetBounds(geom.Rect{W: 200, H: 200})
	var sel, act []int
	l.OnSelect(func(i int) { sel = append(sel, i) })
	l.OnActivate(func(i int) { act = append(act, i) })

	y := l.Bounds().Y + l.RowHeight()*1.5 // row 1
	l.HandleEvent(dbl(geom.Point{X: 5, Y: y}))
	if len(act) != 1 || act[0] != 1 {
		t.Fatalf("activated = %v, want [1]", act)
	}
	if l.Selected() != 1 || len(sel) != 1 {
		t.Errorf("selected=%d selects=%v: a double-click should select its row once", l.Selected(), sel)
	}

	// Below the last row: nothing to activate.
	l.HandleEvent(dbl(geom.Point{X: 5, Y: l.Bounds().Y + l.RowHeight()*10}))
	if len(act) != 1 {
		t.Errorf("activated = %v, want unchanged", act)
	}

	// Enter activates the selected row too.
	l.HandleEvent(&Event{Type: EventKeyDown, Key: render.KeyEnter})
	if len(act) != 2 || act[1] != 1 {
		t.Errorf("after Enter activated = %v, want [1 1]", act)
	}
}

func TestTableDoubleClickActivatesRowNotHeader(t *testing.T) {
	app := NewApp()
	tb := NewTable([]Column{{Title: "Name"}, {Title: "Size"}})
	app.SetContent(tb)
	tb.SetBounds(geom.Rect{W: 300, H: 200})
	tb.AddRow("x", "1")
	tb.AddRow("y", "2")
	var act []int
	tb.OnActivate(func(i int) { act = append(act, i) })

	rh := tb.rowHeight()
	tb.HandleEvent(dbl(geom.Point{X: 5, Y: tb.Bounds().Y + rh*0.5})) // header band
	if len(act) != 0 {
		t.Fatalf("a header double-click activated %v", act)
	}
	tb.HandleEvent(dbl(geom.Point{X: 5, Y: tb.Bounds().Y + rh*2.5})) // second body row
	if len(act) != 1 || act[0] != 1 || tb.selected != 1 {
		t.Errorf("activated=%v selected=%d, want [1] and 1", act, tb.selected)
	}
	tb.HandleEvent(&Event{Type: EventKeyDown, Key: render.KeyEnter})
	if len(act) != 2 {
		t.Errorf("Enter should activate the selected row, got %v", act)
	}
}

func TestTreeDoubleClickOpensNodes(t *testing.T) {
	app := NewApp()
	leaf := NewTreeNode("leaf")
	parent := NewTreeNode("parent", leaf)
	tr := NewTree(parent)
	app.SetContent(tr)
	tr.SetBounds(geom.Rect{W: 300, H: 200})
	var act []*TreeNode
	tr.OnActivate(func(n *TreeNode) { act = append(act, n) })

	rh := tr.RowHeight()
	labelX := tr.Bounds().X + 200 // well right of the chevron
	row := func(i int) geom.Point { return geom.Point{X: labelX, Y: tr.Bounds().Y + rh*(float64(i)+0.5)} }

	// A double-click on a parent's label toggles it open, and does not fire OnActivate.
	tr.HandleEvent(dbl(row(0)))
	if !parent.expanded || len(act) != 0 {
		t.Fatalf("expanded=%v activated=%v, want open and none", parent.expanded, act)
	}
	tr.HandleEvent(dbl(row(0)))
	if parent.expanded {
		t.Error("a second double-click should close the parent again")
	}

	// A leaf fires OnActivate.
	tr.Expand(parent)
	tr.HandleEvent(dbl(row(1)))
	if len(act) != 1 || act[0] != leaf || tr.Selected() != leaf {
		t.Errorf("activated=%v selected=%v, want [leaf]", act, tr.Selected())
	}
}

func TestTreeDoubleClickOnChevronLeavesToggleToClicks(t *testing.T) {
	app := NewApp()
	parent := NewTreeNode("parent", NewTreeNode("kid"))
	tr := NewTree(parent)
	app.SetContent(tr)
	tr.SetBounds(geom.Rect{W: 300, H: 200})
	chevron := geom.Point{X: tr.Bounds().X + treeRowPad + 1, Y: tr.Bounds().Y + tr.RowHeight()/2}
	tr.HandleEvent(dbl(chevron))
	if parent.expanded {
		t.Error("the chevron is toggled by the two clicks; the double-click must not toggle again")
	}
}

func TestListDoubleClickEndToEnd(t *testing.T) {
	app := NewApp()
	l := NewList([]string{"a", "b", "c"})
	root := NewContainer()
	root.SetBounds(geom.Rect{W: 300, H: 300})
	l.SetBounds(geom.Rect{X: 10, Y: 10, W: 200, H: 150})
	root.Add(l)
	app.SetContent(root)
	got := -1
	l.OnActivate(func(i int) { got = i })
	now := time.Unix(1000, 0)
	app.now = func() time.Time { return now }

	pos := geom.Point{X: 20, Y: l.Bounds().Y + l.RowHeight()*2.5}
	dcClick(app, pos)
	now = now.Add(80 * time.Millisecond)
	dcClick(app, pos)
	if got != 2 {
		t.Errorf("activated row %d after double-click, want 2", got)
	}
}
