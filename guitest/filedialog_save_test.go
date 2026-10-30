package guitest_test

import (
	"testing"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/guitest"
	"github.com/kpfaulkner/guie/render"
	"github.com/kpfaulkner/guie/ui"
)

// focusedFieldHarness returns a harness whose content is one text field that
// has keyboard focus.
func focusedFieldHarness(t *testing.T) (*guitest.Harness, *ui.TextField) {
	t.Helper()
	h := guitest.New(800, 600)
	field := ui.NewTextField()
	root := ui.NewContainer()
	root.SetLayout(ui.VBox(0))
	root.Add(field)
	h.SetContent(root)
	h.Step()
	b := field.Bounds()
	h.Click(b.X+5, b.Y+5)
	return h, field
}

// textPos returns a point just inside the first thing drawn as text.
func textPos(t *testing.T, h *guitest.Harness, text string) geom.Point {
	t.Helper()
	for _, op := range h.Frame().OpsOfKind(guitest.OpDrawText) {
		if op.Text == text {
			return geom.Point{X: op.A.X + 3, Y: op.A.Y + 3}
		}
	}
	t.Fatalf("%q is not drawn; drawn: %v", text, h.Frame().Texts())
	return geom.Point{}
}

// focusName clicks into the name field of an open dialog. The field is found by
// its label so the click lands in it whatever it holds.
func (e *fdEnv) focusName() {
	e.t.Helper()
	label := "File name:"
	if !e.drawn(label) {
		label = "Folder:"
	}
	p := e.pos(label)
	// The field is to the right of its label: step past the label.
	e.h.Click(p.X+120, p.Y)
}

// typeName types text into the name field.
func (e *fdEnv) typeName(text string) {
	e.t.Helper()
	e.focusName()
	e.h.TypeText(text)
	e.h.Step() // runes are delivered after keys within a frame
}

func (e *fdEnv) wantConfirm() {
	e.t.Helper()
	if !e.drawn("Replace file?") {
		e.t.Fatalf("want the replace prompt; drawn: %v", e.h.Frame().Texts())
	}
}

func TestFileSaveTitleAndButtons(t *testing.T) {
	e := openMode(t, ui.FileSave, fdFS())
	e.wantDrawn("Save file", "Save", "Cancel", "File name:")
}

func TestFileSavePrefilledNameSavesNewFile(t *testing.T) {
	e := openMode(t, ui.FileSave, fdFS(), ui.FileDialogName("report.txt"))
	e.wantDrawn("report.txt")
	e.click("Save")
	e.wantResult(true, "/report.txt")
	e.wantClosed()
}

func TestFileSaveTypingReplacesSuggestedName(t *testing.T) {
	e := openMode(t, ui.FileSave, fdFS(), ui.FileDialogName("report.txt"))
	e.h.TypeText("x") // the suggestion is selected and the field has focus
	e.h.Step()
	e.click("Save")
	e.wantResult(true, "/x")
}

func TestFileSaveEnterInNameField(t *testing.T) {
	e := openMode(t, ui.FileSave, fdFS())
	e.h.TypeText("fresh.txt") // the name field has focus when saving
	e.h.Step()
	e.h.TypeKey(render.KeyEnter)
	e.wantResult(true, "/fresh.txt")
}

func TestFileSaveDefaultExt(t *testing.T) {
	t.Run("added", func(t *testing.T) {
		e := openMode(t, ui.FileSave, fdFS(), ui.FileDialogDefaultExt("txt"))
		e.h.TypeText("fresh")
		e.h.Step()
		e.click("Save")
		e.wantResult(true, "/fresh.txt")
	})
	t.Run("leading dot accepted", func(t *testing.T) {
		e := openMode(t, ui.FileSave, fdFS(), ui.FileDialogDefaultExt(".txt"))
		e.h.TypeText("fresh")
		e.h.Step()
		e.click("Save")
		e.wantResult(true, "/fresh.txt")
	})
	t.Run("own extension kept", func(t *testing.T) {
		e := openMode(t, ui.FileSave, fdFS(), ui.FileDialogDefaultExt("txt"))
		e.h.TypeText("fresh.md")
		e.h.Step()
		e.click("Save")
		e.wantResult(true, "/fresh.md")
	})
	t.Run("prompt names the real file", func(t *testing.T) {
		e := openMode(t, ui.FileSave, fdFS(), ui.FileDialogDefaultExt("txt"))
		e.h.TypeText("notes") // notes.txt exists
		e.h.Step()
		e.click("Save")
		e.wantConfirm()
		if !e.h.Frame().TextContaining(`"notes.txt" already exists`) {
			t.Errorf("prompt should name notes.txt; drawn: %v", e.h.Frame().Texts())
		}
	})
}

func TestFileSaveExistingFileAsksBeforeReplacing(t *testing.T) {
	e := openMode(t, ui.FileSave, fdFS())
	e.h.TypeText("notes.txt")
	e.h.Step()
	e.click("Save")
	e.wantConfirm()
	if e.calls != 0 {
		t.Fatalf("must not resolve before the user agrees; calls = %d", e.calls)
	}
	e.click("Replace")
	e.wantResult(true, "/notes.txt")
	e.wantClosed()
}

func TestFileSaveReplacePromptDefaultsToCancel(t *testing.T) {
	e := openMode(t, ui.FileSave, fdFS())
	e.h.TypeText("notes.txt")
	e.h.Step()
	e.click("Save")
	e.wantConfirm()

	// A stray Enter must not replace the file.
	e.h.TypeKey(render.KeyEnter)
	e.h.Step()
	if e.calls != 0 {
		t.Fatalf("Enter on the prompt should cancel, not replace; calls = %d", e.calls)
	}
	e.wantDrawn("Save file") // back in the dialog
	if e.drawn("Replace file?") {
		t.Error("the prompt should be gone")
	}
}

func TestFileSaveCancellingThePromptKeepsTheDialogUsable(t *testing.T) {
	e := openMode(t, ui.FileSave, fdFS())
	e.h.TypeText("notes.txt")
	e.h.Step()
	e.click("Save")
	e.wantConfirm()
	e.h.TypeKey(render.KeyEscape) // closes only the prompt, the top popup
	e.h.Step()
	if e.calls != 0 || !e.drawn("Save file") || e.drawn("Replace file?") {
		t.Fatalf("calls=%d; drawn: %v", e.calls, e.h.Frame().Texts())
	}

	// Pick another name and carry on.
	e.focusName() // clicking Save moved focus to the button
	e.h.TypeKey(render.KeyEnd)
	e.h.TypeText("2")
	e.h.Step()
	e.click("Save")
	e.wantResult(true, "/notes.txt2")
}

func TestFileSaveNoOverwritePromptReturnsExisting(t *testing.T) {
	e := openMode(t, ui.FileSave, fdFS(), ui.FileDialogNoOverwritePrompt())
	e.h.TypeText("notes.txt")
	e.h.Step()
	e.click("Save")
	e.wantResult(true, "/notes.txt")
}

func TestFileSaveDoubleClickExistingFileAsksToo(t *testing.T) {
	e := openMode(t, ui.FileSave, fdFS())
	e.doubleClick("pic.png")
	e.wantConfirm()
	if e.calls != 0 {
		t.Fatalf("calls = %d, want 0 before agreeing", e.calls)
	}
	e.click("Replace")
	e.wantResult(true, "/pic.png")
}

func TestFileSaveEmptyNameAsksForOne(t *testing.T) {
	e := openMode(t, ui.FileSave, fdFS())
	e.click("Save")
	if !e.h.Frame().TextContaining("Enter a file name") {
		t.Errorf("want a prompt; drawn: %v", e.h.Frame().Texts())
	}
	if e.calls != 0 {
		t.Errorf("calls = %d, want 0", e.calls)
	}
}

func TestFileSaveTypedFolderNameOpensItAndClearsName(t *testing.T) {
	e := openMode(t, ui.FileSave, fdFS())
	e.h.TypeText("docs")
	e.h.Step()
	e.click("Save")
	e.wantDrawn("a.txt", "b.md")
	if e.calls != 0 {
		t.Fatalf("a folder name must navigate, calls = %d", e.calls)
	}
	e.typeName("new") // clicking Save moved focus to the button; the field was cleared
	e.click("Save")
	e.wantResult(true, "/docs/new")
}

func TestFileSaveNameSurvivesBrowsing(t *testing.T) {
	e := openMode(t, ui.FileSave, fdFS())
	e.h.TypeText("draft")
	e.h.Step()
	e.click("docs/") // selecting a folder leaves the name alone
	e.wantDrawn("draft")
	e.doubleClick("docs/") // so does going into it
	e.wantDrawn("a.txt", "draft")
	e.click("Save")
	e.wantResult(true, "/docs/draft")
}

func TestFileSaveClickingFileFillsName(t *testing.T) {
	e := openMode(t, ui.FileSave, fdFS())
	e.click("readme.md")
	e.click("Save") // the name is now readme.md, which exists
	e.wantConfirm()
}

func TestFileSaveCancelResolvesOnce(t *testing.T) {
	e := openMode(t, ui.FileSave, fdFS())
	e.click("Cancel")
	e.wantResult(false)
	e.wantClosed()
}

func TestFileSaveClosingWithPromptOpenResolvesOnce(t *testing.T) {
	e := openMode(t, ui.FileSave, fdFS())
	e.h.TypeText("notes.txt")
	e.h.Step()
	e.click("Save")
	e.wantConfirm()
	e.h.App.Close(e.popup) // takes the prompt down with it
	e.wantResult(false)
	e.wantClosed()
	if e.drawn("Replace file?") {
		t.Error("the prompt should close with the dialog")
	}
}

func TestFileSaveFocusReturnsAfterReplace(t *testing.T) {
	h, field := focusedFieldHarness(t)
	var got []string
	h.App.ShowFileDialog(ui.FileSave, func(p []string, ok bool) { got = p },
		ui.FileDialogFS(ui.FSFileSystem(fdFS())), ui.FileDialogName("notes.txt"))
	h.Step()
	p := textPos(t, h, "Save")
	h.Click(p.X, p.Y)
	p = textPos(t, h, "Replace")
	h.Click(p.X, p.Y)
	if len(got) != 1 || got[0] != "/notes.txt" {
		t.Fatalf("got %v", got)
	}
	h.TypeText("z")
	h.Step()
	if field.Text() != "z" {
		t.Errorf("focus should return to the field that had it; text = %q", field.Text())
	}
}

func TestFileSelectDirListsOnlyDirectories(t *testing.T) {
	e := openMode(t, ui.FileSelectDir, fdFS(),
		ui.FileDialogFilters(ui.FileFilter{Name: "Everything"}, ui.FileFilter{Name: "Other"}))
	e.wantDrawn("Select folder", "docs/", "Select", "Folder:")
	e.wantAbsent("readme.md", "notes.txt", "pic.png", "Everything") // no files, no filter dropdown
}

func TestFileSelectDirOKWithNothingSelectedReturnsCurrent(t *testing.T) {
	e := openMode(t, ui.FileSelectDir, fdFS())
	e.click("Select")
	e.wantResult(true, "/")
	e.wantClosed()
}

func TestFileSelectDirSelectedRowIsReturned(t *testing.T) {
	e := openMode(t, ui.FileSelectDir, fdFS())
	e.click("docs/")
	e.wantDrawn("docs") // in the name field
	e.click("Select")
	e.wantResult(true, "/docs")
}

func TestFileSelectDirDoubleClickBrowsesInto(t *testing.T) {
	e := openMode(t, ui.FileSelectDir, fdFS())
	e.doubleClick("docs/")
	e.wantDrawn("sub/")
	if e.calls != 0 {
		t.Fatalf("double-click must browse, not choose; calls = %d", e.calls)
	}
	e.click("Select") // nothing selected: the folder we are in
	e.wantResult(true, "/docs")
}

func TestFileSelectDirNestedBrowse(t *testing.T) {
	e := openMode(t, ui.FileSelectDir, fdFS(), ui.FileDialogStart("docs/sub"))
	e.click("Select")
	e.wantResult(true, "/docs/sub")
}

func TestFileSelectDirTypedName(t *testing.T) {
	t.Run("relative folder", func(t *testing.T) {
		e := openMode(t, ui.FileSelectDir, fdFS())
		e.typeName("docs/sub")
		e.h.TypeKey(render.KeyEnter)
		e.wantResult(true, "/docs/sub")
	})
	t.Run("missing", func(t *testing.T) {
		e := openMode(t, ui.FileSelectDir, fdFS())
		e.typeName("nope")
		e.h.TypeKey(render.KeyEnter)
		if !e.h.Frame().TextContaining("does not exist") || e.calls != 0 {
			t.Errorf("calls=%d; drawn: %v", e.calls, e.h.Frame().Texts())
		}
	})
	t.Run("a file is not a folder", func(t *testing.T) {
		e := openMode(t, ui.FileSelectDir, fdFS())
		e.typeName("readme.md")
		e.h.TypeKey(render.KeyEnter)
		if !e.h.Frame().TextContaining("is not a folder") || e.calls != 0 {
			t.Errorf("calls=%d; drawn: %v", e.calls, e.h.Frame().Texts())
		}
	})
}

func TestFileSelectDirEscapeResolvesOnce(t *testing.T) {
	e := openMode(t, ui.FileSelectDir, fdFS())
	e.h.TypeKey(render.KeyEscape)
	e.wantResult(false)
	e.wantClosed()
}
