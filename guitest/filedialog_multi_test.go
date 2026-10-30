package guitest_test

import (
	"testing"

	"github.com/kpfaulkner/guie/render"
	"github.com/kpfaulkner/guie/ui"
)

const (
	fdShift   = render.ModifierSet(render.ModShift)
	fdPrimary = render.ModifierSet(render.ModPrimary | render.ModControl)
)

// clickWith clicks the first thing drawn as text with modifiers held.
func (e *fdEnv) clickWith(mods render.ModifierSet, text string) {
	e.t.Helper()
	e.h.SetModifiers(mods)
	e.click(text)
	e.h.SetModifiers(0)
}

// keyWith presses a key with modifiers held.
func (e *fdEnv) keyWith(mods render.ModifierSet, k render.Key) {
	e.t.Helper()
	e.h.SetModifiers(mods)
	e.h.TypeKey(k)
	e.h.SetModifiers(0)
}

func openMulti(t *testing.T, opts ...ui.FileDialogOption) *fdEnv {
	t.Helper()
	return openFD(t, fdFS(), append([]ui.FileDialogOption{ui.FileDialogMultiple()}, opts...)...)
}

func TestFileDialogMultipleCtrlClickTwoFiles(t *testing.T) {
	e := openMulti(t)
	e.click("readme.md")
	e.clickWith(fdPrimary, "notes.txt")
	e.wantDrawn(`"notes.txt" "readme.md"`) // listing order, quoted
	e.click("Open")
	e.wantResult(true, "/notes.txt", "/readme.md")
	e.wantClosed()
}

func TestFileDialogMultipleShiftClickRange(t *testing.T) {
	e := openMulti(t)
	e.click("notes.txt")
	e.clickWith(fdShift, "readme.md")
	e.wantDrawn(`"notes.txt" "pic.png" "readme.md"`)
	e.click("Open")
	e.wantResult(true, "/notes.txt", "/pic.png", "/readme.md")
}

func TestFileDialogMultipleSingleSelectionShowsPlainName(t *testing.T) {
	e := openMulti(t)
	e.click("pic.png")
	n := 0
	for _, s := range e.h.Frame().Texts() {
		if s == "pic.png" {
			n++
		}
	}
	if n != 2 {
		t.Errorf("pic.png drawn %d times, want 2 (row and unquoted name field)", n)
	}
	e.click("Open")
	e.wantResult(true, "/pic.png")
}

func TestFileDialogMultipleSelectAllSkipsDirectories(t *testing.T) {
	e := openMulti(t)
	e.click("notes.txt")
	e.keyWith(fdPrimary, render.KeyA) // the listing has focus after a click
	e.click("Open")
	e.wantResult(true, "/notes.txt", "/pic.png", "/readme.md")
}

func TestFileDialogMultipleOnlyADirSelectedOpensIt(t *testing.T) {
	e := openMulti(t)
	e.click("docs/")
	e.click("Open")
	e.wantDrawn("a.txt")
	if e.calls != 0 {
		t.Fatalf("calls = %d, want 0", e.calls)
	}
}

func TestFileDialogMultipleDoubleClickTakesOnlyThatFile(t *testing.T) {
	e := openMulti(t)
	e.click("notes.txt")
	e.clickWith(fdPrimary, "pic.png")
	e.doubleClick("readme.md")
	e.wantResult(true, "/readme.md")
}

func TestFileDialogMultipleEnterTakesWholeSelection(t *testing.T) {
	e := openMulti(t)
	e.h.TypeKey(render.KeyDown) // docs/
	e.h.TypeKey(render.KeyDown) // notes.txt
	e.keyWith(fdShift, render.KeyDown)
	e.wantDrawn(`"notes.txt" "pic.png"`)
	e.h.TypeKey(render.KeyEnter)
	e.wantResult(true, "/notes.txt", "/pic.png")
}

func TestFileDialogMultipleTypedNameOverridesSelection(t *testing.T) {
	e := openMulti(t)
	e.click("notes.txt")
	e.clickWith(fdPrimary, "readme.md")
	e.focusName()
	e.keyWith(fdPrimary, render.KeyA) // select all in the field
	e.h.TypeText("pic.png")
	e.h.Step()
	e.click("Open")
	e.wantResult(true, "/pic.png")
}

func TestFileDialogMultipleSelectionSurvivesSort(t *testing.T) {
	e := openMulti(t)
	e.click("notes.txt")
	e.clickWith(fdPrimary, "readme.md")
	e.click("Size") // readme (5) now sorts above notes (20)
	e.wantOrder("docs/", "readme.md", "pic.png", "notes.txt")
	e.click("Open")
	e.wantResult(true, "/readme.md", "/notes.txt")
}

func TestFileDialogMultipleFilterDropsHiddenFilesFromSelection(t *testing.T) {
	e := openMulti(t, ui.FileDialogFilters(
		ui.FileFilter{Name: "All files"},
		ui.FileFilter{Name: "Text", Patterns: []string{"*.txt"}},
	))
	e.click("notes.txt")
	e.clickWith(fdPrimary, "pic.png")
	e.wantDrawn(`"notes.txt" "pic.png"`)
	e.click("All files")
	e.click("Text")
	e.wantAbsent("pic.png", `"notes.txt" "pic.png"`)
	e.click("Open") // only notes.txt is still listed, so only it is returned
	e.wantResult(true, "/notes.txt")
}

func TestFileDialogMultipleNavigatingClearsSelection(t *testing.T) {
	e := openMulti(t)
	e.click("notes.txt")
	e.clickWith(fdPrimary, "pic.png")
	e.doubleClick("docs/")
	e.wantDrawn("a.txt")
	e.wantAbsent(`"notes.txt" "pic.png"`)
	e.click("Open") // nothing selected, nothing typed
	if e.calls != 0 || !e.h.Frame().TextContaining("Choose a file") {
		t.Errorf("calls=%d; drawn: %v", e.calls, e.h.Frame().Texts())
	}
}

func TestFileDialogWithoutMultipleIgnoresCtrlClick(t *testing.T) {
	e := openFD(t, fdFS())
	e.click("notes.txt")
	e.clickWith(fdPrimary, "pic.png")
	e.click("Open")
	e.wantResult(true, "/pic.png")
}

func TestFileDialogMultipleIgnoredBySaveAndSelectDir(t *testing.T) {
	e := openMode(t, ui.FileSave, fdFS(), ui.FileDialogMultiple())
	e.click("notes.txt")
	e.clickWith(fdPrimary, "pic.png")
	e.click("Save")
	if !e.h.Frame().TextContaining(`"pic.png" already exists`) {
		t.Errorf("Save takes one name; drawn: %v", e.h.Frame().Texts())
	}

	d := openMode(t, ui.FileSelectDir, fdFS(), ui.FileDialogMultiple())
	d.click("docs/")
	d.click("Select")
	d.wantResult(true, "/docs")
}
