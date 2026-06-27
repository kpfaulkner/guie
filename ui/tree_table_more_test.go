package ui

import "testing"

func TestTreeNodeChildrenParent(t *testing.T) {
	c1 := NewTreeNode("c1")
	c2 := NewTreeNode("c2")
	parent := NewTreeNode("p", c1, c2)

	if got := parent.Children(); len(got) != 2 || got[0] != c1 || got[1] != c2 {
		t.Fatalf("Children: got %v", got)
	}
	if c1.Parent() != parent {
		t.Error("Parent should point back to the node that adopted it")
	}
	if NewTreeNode("orphan").Parent() != nil {
		t.Error("a top-level node should have no parent")
	}
}

func TestTreeSetRootsFontMinSize(t *testing.T) {
	tr := NewTree()
	tr.SetFont(DefaultFont(14))
	if tr.face() == nil {
		t.Fatal("SetFont should make a face available")
	}

	tr.SetSelected(NewTreeNode("ignored")) // selection on an absent node
	tr.SetRoots(NewTreeNode("A"), NewTreeNode("B"))
	if tr.Selected() != nil {
		t.Error("SetRoots should clear the selection")
	}
	if n := len(tr.rows()); n != 2 {
		t.Errorf("rows after SetRoots: got %d, want 2", n)
	}
	if !tr.Focusable() {
		t.Error("an enabled tree should be focusable")
	}
	if ms := tr.MinSize(); ms.H <= 0 || ms.W <= 0 {
		t.Errorf("MinSize with a font should be positive, got %+v", ms)
	}
}

func TestMenuBarSetFont(t *testing.T) {
	mb := NewMenuBar()
	f := DefaultFont(14)
	mb.SetFont(f)
	if mb.face() != f {
		t.Error("SetFont should make the given face the menu bar's face")
	}
}

func TestTableAccessors(t *testing.T) {
	tbl := NewTable([]Column{{Title: "A"}, {Title: "B"}})
	tbl.SetFont(DefaultFont(14))
	tbl.SetSortable(false)

	got := -1
	tbl.OnSelect(func(i int) { got = i })

	tbl.AddRow("1", "2")
	tbl.AddRow("3", "4")
	if tbl.RowCount() != 2 {
		t.Fatalf("RowCount: got %d, want 2", tbl.RowCount())
	}

	if r := tbl.Row(0); len(r) != 2 || r[0] != "1" || r[1] != "2" {
		t.Errorf("Row(0): got %v, want [1 2]", r)
	}
	if tbl.Row(-1) != nil || tbl.Row(99) != nil {
		t.Error("Row out of range should return nil")
	}

	tbl.SetSelected(1)
	if tbl.Selected() != 1 || got != 1 {
		t.Errorf("SetSelected(1): selected=%d onSelect=%d", tbl.Selected(), got)
	}

	if !tbl.Focusable() {
		t.Error("an enabled table should be focusable")
	}
	if ms := tbl.MinSize(); ms.W <= 0 || ms.H <= 0 {
		t.Errorf("MinSize with a font should be positive, got %+v", ms)
	}
}
