package ui

import (
	"testing"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
)

// treeFixture builds a mounted Tree with a known shape:
//
//	A            (parent: A1, A2)
//	B            (leaf)
//
// It is mounted via an App so the theme font is available, and given fixed
// bounds so row hit-testing works without a layout pass.
func treeFixture(t *testing.T) (*App, *Tree, *TreeNode, *TreeNode, *TreeNode, *TreeNode) {
	t.Helper()
	a1 := NewTreeNode("A1")
	a2 := NewTreeNode("A2")
	a := NewTreeNode("A", a1, a2)
	b := NewTreeNode("B")

	tree := NewTree(a, b)
	app := NewApp()
	app.SetContent(tree)
	tree.SetBounds(geom.Rect{X: 0, Y: 0, W: 200, H: 400})
	return app, tree, a, a1, a2, b
}

func rowCenterY(tr *Tree, i int) float64 {
	return tr.Bounds().Y + float64(i)*tr.RowHeight() + tr.RowHeight()/2
}

func TestTreeFlattenRespectsExpansion(t *testing.T) {
	_, tree, a, a1, a2, b := treeFixture(t)

	rows := tree.rows()
	if len(rows) != 2 || rows[0].node != a || rows[1].node != b {
		t.Fatalf("collapsed tree should show only top-level nodes, got %d rows", len(rows))
	}

	tree.Expand(a)
	rows = tree.rows()
	if len(rows) != 4 {
		t.Fatalf("expanding A should reveal its children: want 4 rows, got %d", len(rows))
	}
	if rows[1].node != a1 || rows[2].node != a2 || rows[1].depth != 1 {
		t.Fatalf("children should follow A at depth 1, got %+v", rows)
	}
	_ = a2

	tree.Collapse(a)
	if len(tree.rows()) != 2 {
		t.Fatal("collapsing A should hide its children again")
	}
}

func TestTreeChevronClickTogglesRowClickSelects(t *testing.T) {
	_, tree, a, _, _, _ := treeFixture(t)

	// Click within the chevron column of row 0 (depth 0 → x in [pad, pad+chevron]).
	tree.HandleEvent(&Event{Type: EventClick, Pos: geom.Point{X: treeRowPad + 2, Y: rowCenterY(tree, 0)}})
	if !a.Expanded() {
		t.Fatal("clicking the chevron should expand the node")
	}
	if tree.Selected() == a {
		t.Fatal("a chevron click should toggle, not select")
	}

	// Click the label area of row 0 → selects A (without collapsing).
	tree.HandleEvent(&Event{Type: EventClick, Pos: geom.Point{X: 120, Y: rowCenterY(tree, 0)}})
	if tree.Selected() != a {
		t.Fatalf("clicking the row label should select it, got %v", tree.Selected())
	}
	if !a.Expanded() {
		t.Fatal("selecting via the label must not collapse the node")
	}
}

func TestTreeKeyboardNavigation(t *testing.T) {
	_, tree, a, a1, _, b := treeFixture(t)

	// With nothing selected, Down selects the first row.
	tree.HandleEvent(&Event{Type: EventKeyDown, Key: render.KeyDown})
	if tree.Selected() != a {
		t.Fatalf("Down should select the first node, got %v", tree.Selected())
	}

	// Right expands A.
	tree.HandleEvent(&Event{Type: EventKeyDown, Key: render.KeyRight})
	if !a.Expanded() {
		t.Fatal("Right on a collapsed parent should expand it")
	}

	// Down moves into the first child.
	tree.HandleEvent(&Event{Type: EventKeyDown, Key: render.KeyDown})
	if tree.Selected() != a1 {
		t.Fatalf("Down should move to the first child A1, got %v", tree.Selected())
	}

	// Left from a child hops to its parent.
	tree.HandleEvent(&Event{Type: EventKeyDown, Key: render.KeyLeft})
	if tree.Selected() != a {
		t.Fatalf("Left on a child should move to the parent, got %v", tree.Selected())
	}

	// Left again collapses A.
	tree.HandleEvent(&Event{Type: EventKeyDown, Key: render.KeyLeft})
	if a.Expanded() {
		t.Fatal("Left on an expanded parent should collapse it")
	}

	// End selects the last visible row (B, since A is collapsed).
	tree.HandleEvent(&Event{Type: EventKeyDown, Key: render.KeyEnd})
	if tree.Selected() != b {
		t.Fatalf("End should select the last row, got %v", tree.Selected())
	}
}

func TestTreeActivateLeafFiresCallback(t *testing.T) {
	_, tree, a, _, _, b := treeFixture(t)
	var activated *TreeNode
	tree.OnActivate(func(n *TreeNode) { activated = n })

	// Enter on a parent toggles instead of activating.
	tree.SetSelected(a)
	tree.HandleEvent(&Event{Type: EventKeyDown, Key: render.KeyEnter})
	if activated != nil {
		t.Fatal("Enter on a parent should toggle, not activate")
	}
	if !a.Expanded() {
		t.Fatal("Enter on a collapsed parent should expand it")
	}

	// Enter on a leaf activates it.
	tree.SetSelected(b)
	tree.HandleEvent(&Event{Type: EventKeyDown, Key: render.KeyEnter})
	if activated != b {
		t.Fatalf("Enter on a leaf should activate it, got %v", activated)
	}
}

func TestTreeOnSelectFiresOnChangeOnly(t *testing.T) {
	_, tree, a, _, _, _ := treeFixture(t)
	calls := 0
	tree.OnSelect(func(*TreeNode) { calls++ })
	tree.SetSelected(a)
	tree.SetSelected(a) // unchanged → no fire
	if calls != 1 {
		t.Fatalf("OnSelect should fire once per change, got %d", calls)
	}
}

// --- Toast ---

func TestToastShowAndAutoExpire(t *testing.T) {
	app := NewApp()
	app.ShowToast("hello", ToastDuration(1.0))
	if len(app.toasts) != 1 {
		t.Fatalf("ShowToast should add a toast, have %d", len(app.toasts))
	}
	app.advanceToasts(0.5)
	if len(app.toasts) != 1 {
		t.Fatal("toast should still be active mid-life")
	}
	app.advanceToasts(0.6) // total 1.1 > 1.0
	if len(app.toasts) != 0 {
		t.Fatal("toast should be removed after its duration elapses")
	}
}

func TestToastFadeAlpha(t *testing.T) {
	app := NewApp()
	to := app.ShowToast("x", ToastDuration(1.0))

	if to.alpha() != 0 {
		t.Fatalf("alpha should start at 0 (fade-in), got %v", to.alpha())
	}
	app.advanceToasts(toastFadeIn) // fully faded in
	if a := to.alpha(); a < 0.99 {
		t.Fatalf("alpha should reach ~1 after fade-in, got %v", a)
	}
	// Advance into the fade-out window: elapsed 0.8, remaining 0.2 of 0.4 → 0.5.
	app.advanceToasts(0.8 - toastFadeIn)
	if a := to.alpha(); a < 0.4 || a > 0.6 {
		t.Fatalf("alpha mid-fade-out should be ~0.5, got %v", a)
	}
}

func TestToastDismissFadesOut(t *testing.T) {
	app := NewApp()
	to := app.ShowToast("x", ToastDuration(10.0))
	app.advanceToasts(1.0)
	to.Dismiss()
	app.advanceToasts(toastFadeOut + 0.01) // past the shortened end
	if len(app.toasts) != 0 {
		t.Fatal("a dismissed toast should expire after its fade-out")
	}
}
