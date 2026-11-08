package ui

import (
	"testing"

	"github.com/kpfaulkner/guie/render"
)

func readOnlyArea(t *testing.T, text string) *TextArea {
	t.Helper()
	app := newMemApp()
	ta := NewTextArea(TextAreaReadOnly())
	app.SetContent(ta)
	ta.SetText(text)
	return ta
}

func taKeyMods(t *TextArea, k render.Key, mods render.ModifierSet) bool {
	ev := Event{Type: EventKeyDown, Key: k, Modifiers: mods}
	return t.HandleEvent(&ev)
}

const primary = render.ModifierSet(render.ModPrimary | render.ModControl)

func TestTextAreaReadOnlyIgnoresEdits(t *testing.T) {
	ta := readOnlyArea(t, "hello\nworld")
	changes := 0
	ta.OnChange(func(string) { changes++ })
	ta.clipboard().WriteText("pasted")

	typeStr(ta, "xyz")
	for _, k := range []render.Key{render.KeyEnter, render.KeyBackspace, render.KeyDelete} {
		if taKeyMods(ta, k, 0) {
			t.Errorf("key %v reported handled on a read-only area", k)
		}
	}
	ta.SelectRange(0, 5)
	for _, k := range []render.Key{render.KeyX, render.KeyV, render.KeyZ, render.KeyY} {
		if taKeyMods(ta, k, primary) {
			t.Errorf("Ctrl+%v reported handled on a read-only area", k)
		}
	}
	ta.HandleEvent(&Event{Type: EventComposition, Comp: render.Composition{Text: "ni", Caret: 2}})

	if ta.Text() != "hello\nworld" || changes != 0 || ta.composing() {
		t.Fatalf("read-only area changed: %q, %d changes, composing %v", ta.Text(), changes, ta.composing())
	}
	if ta.clipboard().ReadText() != "pasted" {
		t.Fatal("cut wrote to the clipboard")
	}
}

func TestTextAreaReadOnlyStillSelectsCopiesAndNavigates(t *testing.T) {
	ta := readOnlyArea(t, "hello\nworld")

	if !taKeyMods(ta, render.KeyA, primary) {
		t.Fatal("select all not handled")
	}
	if !taKeyMods(ta, render.KeyC, primary) {
		t.Fatal("copy not handled")
	}
	if got := ta.clipboard().ReadText(); got != "hello\nworld" {
		t.Fatalf("copied %q", got)
	}

	ta.SelectRange(0, 0)
	taKey(ta, render.KeyDown)
	taKeyMods(ta, render.KeyRight, render.ModifierSet(render.ModShift))
	if start, end := ta.SelectionRange(); start != 6 || end != 7 {
		t.Fatalf("navigation selection %d..%d, want 6..7", start, end)
	}
}

func TestTextAreaReadOnlyProgrammaticChangesStillWork(t *testing.T) {
	ta := readOnlyArea(t, "one")
	ta.SetText("two")
	ta.ReplaceRange(0, 1, "T")
	if ta.Text() != "Two" {
		t.Fatalf("got %q", ta.Text())
	}
}

func TestTextAreaSetReadOnlyToggles(t *testing.T) {
	ta := NewTextArea()
	if ta.ReadOnly() {
		t.Fatal("new area is read-only")
	}
	ta.SetReadOnly(true)
	typeStr(ta, "a")
	ta.SetReadOnly(false)
	typeStr(ta, "b")
	if ta.Text() != "b" || ta.ReadOnly() {
		t.Fatalf("got %q, read-only %v", ta.Text(), ta.ReadOnly())
	}
}

func TestTextAreaSetReadOnlyDropsPreedit(t *testing.T) {
	ta := NewTextArea()
	ta.HandleEvent(&Event{Type: EventComposition, Comp: render.Composition{Text: "ni", Caret: 2}})
	if !ta.composing() {
		t.Fatal("composition did not start")
	}
	ta.SetReadOnly(true)
	if ta.composing() {
		t.Fatal("preedit kept after SetReadOnly(true)")
	}
}
