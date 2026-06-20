package ui

import (
	"testing"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
)

// keyRec is a focusable test widget that records the KeyDown events it receives.
type keyRec struct {
	BaseWidget
	got []render.Key
}

func (k *keyRec) Focusable() bool { return true }
func (k *keyRec) HandleEvent(ev *Event) bool {
	if ev.Type == EventKeyDown {
		k.got = append(k.got, ev.Key)
	}
	return true
}

func primarySet(extra ...render.Modifier) render.ModifierSet {
	m := render.ModPrimary
	for _, e := range extra {
		m |= e
	}
	return render.ModifierSet(m)
}

func TestAcceleratorFiresAndConsumes(t *testing.T) {
	app := NewApp()
	rec := &keyRec{BaseWidget: NewBase()}
	app.SetContent(rec)
	app.setFocus(rec)

	fired := 0
	app.AddAccelerator(render.KeyS, primarySet(), func() { fired++ })

	app.dispatchKeyboard(render.InputState{
		KeysPressed: []render.Key{render.KeyS},
		Modifiers:   primarySet(),
	})

	if fired != 1 {
		t.Fatalf("accelerator should fire once, got %d", fired)
	}
	if len(rec.got) != 0 {
		t.Fatalf("focused widget must not receive an accelerated key, got %v", rec.got)
	}
}

func TestAcceleratorMatchesPrimaryRegardlessOfConcreteModifier(t *testing.T) {
	app := NewApp()
	fired := 0
	app.AddAccelerator(render.KeyS, primarySet(), func() { fired++ })

	// Windows/Linux: Control is the primary, so the frame carries both bits.
	app.dispatchKeyboard(render.InputState{
		KeysPressed: []render.Key{render.KeyS},
		Modifiers:   render.ModifierSet(render.ModControl | render.ModPrimary),
	})
	// macOS: Command is the primary, frame carries Meta + Primary.
	app.dispatchKeyboard(render.InputState{
		KeysPressed: []render.Key{render.KeyS},
		Modifiers:   render.ModifierSet(render.ModMeta | render.ModPrimary),
	})

	if fired != 2 {
		t.Fatalf("a ModPrimary accelerator should fire on both platforms, got %d", fired)
	}
}

func TestAcceleratorRequiresExactModifiers(t *testing.T) {
	app := NewApp()
	fired := 0
	app.AddAccelerator(render.KeyS, primarySet(), func() { fired++ })

	app.dispatchKeyboard(render.InputState{
		KeysPressed: []render.Key{render.KeyS},
		Modifiers:   primarySet(render.ModShift),
	})
	if fired != 0 {
		t.Fatalf("Primary+Shift must not fire a Primary-only accelerator, got %d", fired)
	}
}

func TestRightClickOpensContextMenuAndChoosingRuns(t *testing.T) {
	app := NewApp()
	app.resize(400, 300)

	chosen := ""
	w := &keyRec{BaseWidget: NewBase()}
	w.SetContextMenu(
		NewMenuItem("Cut", func() { chosen = "Cut" }),
		NewMenuItem("Copy", func() { chosen = "Copy" }),
		NewMenuItem("Paste", func() { chosen = "Paste" }),
	)
	app.SetContent(w)

	right := render.ButtonSet(0).Set(render.MouseRight)
	app.dispatchPointer(render.InputState{
		MousePos:     geom.Point{X: 50, Y: 50},
		MousePressed: right,
		MouseDown:    right,
	})

	if len(app.overlays) != 1 {
		t.Fatalf("right-click should open one context-menu popup, got %d overlays", len(app.overlays))
	}
	panel, ok := app.overlays[0].content.(*Container)
	if !ok {
		t.Fatal("context menu content should be a *Container")
	}
	rows := panel.Children()
	if len(rows) != 3 {
		t.Fatalf("expected 3 menu rows, got %d", len(rows))
	}

	// Choosing the "Copy" row runs its action and closes the menu.
	rows[1].(*menuRow).onClick()
	if chosen != "Copy" {
		t.Fatalf("expected Copy chosen, got %q", chosen)
	}
	if len(app.overlays) != 0 {
		t.Fatalf("context menu should close after a choice, got %d overlays", len(app.overlays))
	}
}

func TestNoContextMenuWithoutItems(t *testing.T) {
	app := NewApp()
	app.resize(400, 300)
	w := &keyRec{BaseWidget: NewBase()} // no context menu set
	app.SetContent(w)

	right := render.ButtonSet(0).Set(render.MouseRight)
	app.dispatchPointer(render.InputState{
		MousePos:     geom.Point{X: 50, Y: 50},
		MousePressed: right,
		MouseDown:    right,
	})
	if len(app.overlays) != 0 {
		t.Fatalf("a widget without a context menu must not open a popup, got %d overlays", len(app.overlays))
	}
}
