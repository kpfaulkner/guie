package guitest_test

import (
	"testing"

	"github.com/kpfaulkner/guie/guitest"
	"github.com/kpfaulkner/guie/ui"
)

func TestTreeRendersExpandsAndSelects(t *testing.T) {
	h := guitest.New(220, 220)
	pal := h.App.Theme().Palette
	root := ui.NewTreeNode("Root", ui.NewTreeNode("Child1"), ui.NewTreeNode("Child2"))
	tree := ui.NewTree(root, ui.NewTreeNode("Leaf"))
	h.SetContent(tree)
	h.MoveMouse(-1, -1)

	rec := h.Step()
	if !rec.HasText("Root") || !rec.HasText("Leaf") {
		t.Fatal("collapsed tree should draw its top-level nodes")
	}
	if rec.HasText("Child1") {
		t.Fatal("a collapsed node's children should be hidden")
	}

	rh := tree.RowHeight()
	if rh <= 0 {
		t.Fatal("row height should be positive once mounted")
	}

	// Click the chevron of the first (collapsed) row to expand it.
	h.Click(10, rh/2)
	if !h.Step().HasText("Child1") {
		t.Error("expanding Root should reveal its children")
	}

	// Rows now: Root(0) Child1(1) Child2(2) Leaf(3). Click the Leaf label.
	h.Click(60, rh*3.5)
	if sel := tree.Selected(); sel == nil || sel.Label != "Leaf" {
		t.Fatalf("clicking the Leaf row should select it, got %v", tree.Selected())
	}
	if len(h.Step().FillsOfColour(pal.Primary)) == 0 {
		t.Error("the selected row should be filled with the primary colour")
	}
}

func TestTreeDrawsScrollbarWhenOverflowing(t *testing.T) {
	h := guitest.New(200, 40) // short viewport
	pal := h.App.Theme().Palette
	nodes := make([]*ui.TreeNode, 0, 20)
	for i := 0; i < 20; i++ {
		nodes = append(nodes, ui.NewTreeNode("row"))
	}
	h.SetContent(ui.NewTree(nodes...))
	h.MoveMouse(-1, -1)

	if len(h.Step().FillsOfColour(pal.Accent)) == 0 {
		t.Error("an overflowing tree should draw an accent scrollbar thumb")
	}
}

func TestTableRendersHeaderRowsAndSelects(t *testing.T) {
	h := guitest.New(320, 200)
	pal := h.App.Theme().Palette
	tbl := ui.NewTable([]ui.Column{{Title: "Name"}, {Title: "Age"}})
	tbl.AddRow("Alice", "30")
	tbl.AddRow("Bob", "25")
	got := -1
	tbl.OnSelect(func(i int) { got = i })
	h.SetContent(tbl)
	h.MoveMouse(-1, -1)

	rec := h.Step()
	if !rec.HasText("Name") || !rec.HasText("Alice") || !rec.HasText("Bob") {
		t.Fatal("table should draw its header and cell text")
	}
	if tbl.RowCount() != 2 {
		t.Fatalf("RowCount: got %d, want 2", tbl.RowCount())
	}

	rh := tbl.MinSize().H / 2 // MinSize height is two rows (header + one body)
	// Bands: [0,rh]=header, [rh,2rh]=Alice, [2rh,3rh]=Bob.
	h.Click(20, rh*2.5)
	if tbl.Selected() != 1 || got != 1 {
		t.Fatalf("clicking the second body row should select index 1, selected=%d onSelect=%d", tbl.Selected(), got)
	}
	if len(h.Step().FillsOfColour(pal.Primary)) == 0 {
		t.Error("the selected row should be filled with the primary colour")
	}
}

func TestTableDrawsScrollbarWhenOverflowing(t *testing.T) {
	h := guitest.New(300, 60)
	pal := h.App.Theme().Palette
	tbl := ui.NewTable([]ui.Column{{Title: "C"}})
	for i := 0; i < 20; i++ {
		tbl.AddRow("cell")
	}
	h.SetContent(tbl)
	h.MoveMouse(-1, -1)

	if len(h.Step().FillsOfColour(pal.Accent)) == 0 {
		t.Error("an overflowing table should draw an accent scrollbar thumb")
	}
}

func TestMenuBarOpensAndRunsItem(t *testing.T) {
	h := guitest.New(300, 200)
	opened := 0
	mb := ui.NewMenuBar()
	mb.AddMenu("File",
		ui.NewMenuItem("Open", func() { opened++ }),
		ui.NewMenuItem("Quit", func() {}),
	)

	// Keep the bar at its natural height at the top so its popup opens on-screen.
	root := ui.NewContainer()
	root.SetLayout(ui.VBox(0))
	root.Add(mb)
	root.Add(ui.NewContainer(), ui.Weight(1))
	h.SetContent(root)
	h.MoveMouse(-1, -1)

	if !h.Step().HasText("File") {
		t.Fatal("menu bar should draw its title")
	}

	bar := mb.Bounds()
	// Click the "File" title to open its menu.
	h.Click(bar.X+8, bar.Y+bar.H/2)
	if !h.Step().HasText("Open") {
		t.Fatalf("clicking a menu title should open its items")
	}

	// Click the first item ("Open"), just below the bar.
	h.Click(bar.X+8, bar.Y+bar.H+4)
	h.Step()
	if opened != 1 {
		t.Errorf("clicking a menu item should run its action once, got %d", opened)
	}
}
