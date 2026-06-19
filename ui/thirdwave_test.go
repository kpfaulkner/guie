package ui

import (
	"testing"

	"github.com/kpfaulkner/uiframework/geom"
	"github.com/kpfaulkner/uiframework/render"
)

// clickAt drives a full press+release at pos through the app dispatcher.
func clickAt(app *App, pos geom.Point) {
	app.dispatchPointer(downAt(pos))
	app.dispatchPointer(upAt(pos))
}

func TestListClickSelects(t *testing.T) {
	sel := -1
	l := NewList([]string{"a", "b", "c"})
	l.OnSelect(func(i int) { sel = i })
	app := NewApp()
	app.SetContent(l)
	rh := l.RowHeight()
	l.SetBounds(geom.Rect{X: 0, Y: 0, W: 100, H: rh * 3})

	ev := Event{Type: EventClick, Pos: geom.Point{X: 5, Y: rh*1 + rh/2}}
	l.HandleEvent(&ev)
	if l.Selected() != 1 || sel != 1 {
		t.Fatalf("clicking row 1 should select it: selected=%d cb=%d", l.Selected(), sel)
	}
}

func TestListKeyboardNavigation(t *testing.T) {
	l := NewList([]string{"a", "b", "c"})
	app := NewApp()
	app.SetContent(l)
	l.SetBounds(geom.Rect{X: 0, Y: 0, W: 100, H: l.RowHeight() * 3})

	keyDown(l, render.KeyDown) // -1 → 0
	keyDown(l, render.KeyDown) // → 1
	if l.Selected() != 1 {
		t.Fatalf("two Downs should select index 1, got %d", l.Selected())
	}
	keyDown(l, render.KeyUp) // → 0
	if l.Selected() != 0 {
		t.Fatalf("Up should select index 0, got %d", l.Selected())
	}
	keyDown(l, render.KeyEnd)
	if l.Selected() != 2 {
		t.Fatalf("End should select the last index, got %d", l.Selected())
	}
}

func TestDropdownTogglesPopup(t *testing.T) {
	dd := NewDropdown([]string{"x", "y", "z"})
	app := NewApp()
	root := NewContainer()
	root.Add(dd)
	app.SetContent(root)
	root.SetBounds(geom.Rect{X: 0, Y: 0, W: 800, H: 600})
	dd.SetBounds(geom.Rect{X: 0, Y: 0, W: 120, H: 28})

	clickAt(app, geom.Point{X: 10, Y: 10})
	if len(app.overlays) != 1 || !dd.open {
		t.Fatalf("clicking the dropdown should open one popup (open=%v overlays=%d)", dd.open, len(app.overlays))
	}

	// Click outside the popup → dismissed.
	clickAt(app, geom.Point{X: 400, Y: 400})
	if len(app.overlays) != 0 || dd.open {
		t.Fatalf("clicking outside should close the popup (open=%v overlays=%d)", dd.open, len(app.overlays))
	}
}

func TestDropdownSelectsFromPopup(t *testing.T) {
	chosen := -1
	dd := NewDropdown([]string{"x", "y", "z"})
	dd.OnSelect(func(i int) { chosen = i })
	app := NewApp()
	root := NewContainer()
	root.Add(dd)
	app.SetContent(root)
	root.SetBounds(geom.Rect{X: 0, Y: 0, W: 800, H: 600})
	dd.SetBounds(geom.Rect{X: 0, Y: 0, W: 120, H: 28})

	clickAt(app, geom.Point{X: 10, Y: 10}) // open
	list := app.overlays[0].content.(*List)
	b := list.Bounds()
	rh := list.RowHeight()
	clickAt(app, geom.Point{X: b.X + 5, Y: b.Y + rh*2 + rh/2}) // choose index 2

	if dd.Selected() != 2 || chosen != 2 {
		t.Fatalf("choosing row 2 should select it: selected=%d cb=%d", dd.Selected(), chosen)
	}
	if len(app.overlays) != 0 {
		t.Fatalf("selecting should close the popup, overlays=%d", len(app.overlays))
	}
}

func TestMenuBarItemRunsAndCloses(t *testing.T) {
	ran := ""
	mb := NewMenuBar()
	mb.AddMenu("File",
		NewMenuItem("Open", func() { ran = "open" }),
		NewMenuItem("Quit", func() { ran = "quit" }),
	)
	app := NewApp()
	app.SetContent(mb)
	mb.SetBounds(geom.Rect{X: 0, Y: 0, W: 200, H: 28})

	mb.openMenu(0)
	if len(app.overlays) != 1 {
		t.Fatalf("opening a menu should push one popup, got %d", len(app.overlays))
	}

	panel := app.overlays[0].content.(*Container)
	quit := panel.Children()[1] // second item
	ev := Event{Type: EventClick, Pos: quit.Bounds().Center()}
	quit.HandleEvent(&ev)

	if ran != "quit" {
		t.Fatalf("clicking Quit should run its action, got %q", ran)
	}
	if len(app.overlays) != 0 {
		t.Fatalf("choosing an item should close the menu, overlays=%d", len(app.overlays))
	}
}

func TestEscapeClosesPopup(t *testing.T) {
	dd := NewDropdown([]string{"x", "y"})
	app := NewApp()
	root := NewContainer()
	root.Add(dd)
	app.SetContent(root)
	root.SetBounds(geom.Rect{X: 0, Y: 0, W: 800, H: 600})
	dd.SetBounds(geom.Rect{X: 0, Y: 0, W: 120, H: 28})

	clickAt(app, geom.Point{X: 10, Y: 10})
	if len(app.overlays) != 1 {
		t.Fatalf("expected popup open")
	}
	app.dispatchKeyboard(render.InputState{KeysPressed: []render.Key{render.KeyEscape}})
	if len(app.overlays) != 0 {
		t.Fatalf("Escape should close the popup, overlays=%d", len(app.overlays))
	}
}
