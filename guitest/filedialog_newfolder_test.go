package guitest_test

import (
	"io/fs"
	"path"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/guitest"
	"github.com/kpfaulkner/guie/render"
	"github.com/kpfaulkner/guie/ui"
)

// writableFS is an FSFileSystem that can also create directories, which it does
// by adding directory entries to the MapFS it wraps.
type writableFS struct {
	ui.FileSystem
	m fstest.MapFS
}

func newWritable(m fstest.MapFS) writableFS { return writableFS{FileSystem: ui.FSFileSystem(m), m: m} }

func (w writableFS) Mkdir(p string) error {
	rel := strings.TrimPrefix(path.Clean(p), "/")
	if _, err := fs.Stat(w.m, rel); err == nil {
		return &fs.PathError{Op: "mkdir", Path: p, Err: fs.ErrExist}
	}
	w.m[rel] = &fstest.MapFile{Mode: fs.ModeDir | 0o755}
	return nil
}

func (w writableFS) hasDir(rel string) bool {
	info, err := fs.Stat(w.m, rel)
	return err == nil && info.IsDir()
}

func openWritable(t *testing.T, mode ui.FileDialogMode, w writableFS, opts ...ui.FileDialogOption) *fdEnv {
	t.Helper()
	return openMode(t, mode, fstest.MapFS{}, append([]ui.FileDialogOption{ui.FileDialogFS(w)}, opts...)...)
}

// clickLast clicks the last thing drawn as text, which is the top-most when
// several widgets draw the same label.
func (e *fdEnv) clickLast(text string) {
	e.t.Helper()
	var found bool
	var p geom.Point
	for _, op := range e.h.Frame().OpsOfKind(guitest.OpDrawText) {
		if op.Text == text {
			p, found = geom.Point{X: op.A.X + 3, Y: op.A.Y + 3}, true
		}
	}
	if !found {
		e.t.Fatalf("%q is not drawn; drawn: %v", text, e.h.Frame().Texts())
	}
	e.h.Click(p.X, p.Y)
}

// count returns how many times text is drawn.
func (e *fdEnv) count(text string) int {
	n := 0
	for _, s := range e.h.Frame().Texts() {
		if s == text {
			n++
		}
	}
	return n
}

func TestFileDialogNewFolderButtonOnlyWhereItCanWork(t *testing.T) {
	t.Run("save, writable", func(t *testing.T) {
		openWritable(t, ui.FileSave, newWritable(fdFS())).wantDrawn("New folder")
	})
	t.Run("select folder, writable", func(t *testing.T) {
		openWritable(t, ui.FileSelectDir, newWritable(fdFS())).wantDrawn("New folder")
	})
	t.Run("open, writable", func(t *testing.T) {
		openWritable(t, ui.FileOpen, newWritable(fdFS())).wantAbsent("New folder")
	})
	t.Run("save, read-only", func(t *testing.T) {
		openMode(t, ui.FileSave, fdFS()).wantAbsent("New folder")
	})
}

func TestFileDialogNewFolderCreatesAndSelectsIt(t *testing.T) {
	w := newWritable(fdFS())
	e := openWritable(t, ui.FileSave, w)
	e.click("New folder")
	e.wantDrawn("New folder name:")
	if e.count("New folder") != 2 { // the button and the suggestion in the field
		t.Errorf("the field should hold the suggestion New folder; drawn: %v", e.h.Frame().Texts())
	}

	e.h.TypeText("reports") // the suggestion is selected, so this replaces it
	e.h.Step()
	e.h.TypeKey(render.KeyEnter)

	e.wantAbsent("New folder name:")
	e.wantDrawn("reports/", "Save file")
	if !w.hasDir("reports") {
		t.Error("the folder was not created")
	}
	if !e.h.Frame().TextContaining(`Created folder "reports"`) {
		t.Errorf("want a confirmation; drawn: %v", e.h.Frame().Texts())
	}
	if e.calls != 0 {
		t.Errorf("creating a folder must not resolve the dialog, calls = %d", e.calls)
	}
}

func TestFileDialogNewFolderCreateButton(t *testing.T) {
	w := newWritable(fdFS())
	e := openWritable(t, ui.FileSave, w)
	e.click("New folder")
	e.h.TypeText("viabutton")
	e.h.Step()
	e.click("Create")
	e.wantDrawn("viabutton/")
	if !w.hasDir("viabutton") {
		t.Error("the folder was not created")
	}
}

func TestFileDialogNewFolderCancelAndEscapeCreateNothing(t *testing.T) {
	for _, how := range []string{"button", "escape"} {
		w := newWritable(fdFS())
		e := openWritable(t, ui.FileSave, w)
		e.click("New folder")
		e.h.TypeText("nope")
		e.h.Step()
		if how == "button" {
			// Two Cancel buttons are drawn; the prompt's is the last, on top.
			e.clickLast("Cancel")
		} else {
			e.h.TypeKey(render.KeyEscape)
		}
		e.wantAbsent("New folder name:")
		e.wantDrawn("Save file") // the dialog is still there
		if w.hasDir("nope") || e.calls != 0 {
			t.Errorf("%s: hasDir=%v calls=%d", how, w.hasDir("nope"), e.calls)
		}
	}
}

func TestFileDialogNewFolderExistingNameShowsErrorAndStaysOpen(t *testing.T) {
	w := newWritable(fdFS())
	e := openWritable(t, ui.FileSave, w)
	e.click("New folder")
	e.h.TypeText("docs")
	e.h.Step()
	e.h.TypeKey(render.KeyEnter)
	if !e.h.Frame().TextContaining(`"docs" already exists`) {
		t.Fatalf("want the error; drawn: %v", e.h.Frame().Texts())
	}
	e.wantDrawn("New folder name:")

	// Fix the name in place and try again.
	e.h.TypeText("2")
	e.h.Step()
	e.h.TypeKey(render.KeyEnter)
	e.wantAbsent("New folder name:")
	e.wantDrawn("docs2/")
}

func TestFileDialogNewFolderRefusesBadNames(t *testing.T) {
	cases := []struct {
		name, typed, want string
	}{
		{"separator", "a/b", "cannot contain"},
		{"backslash", `a\b`, "cannot contain"},
		{"dots", "..", "not a valid folder name"},
	}
	for _, c := range cases {
		w := newWritable(fdFS())
		e := openWritable(t, ui.FileSave, w)
		e.click("New folder")
		e.h.TypeText(c.typed)
		e.h.Step()
		e.h.TypeKey(render.KeyEnter)
		if !e.h.Frame().TextContaining(c.want) {
			t.Errorf("%s: want %q; drawn: %v", c.name, c.want, e.h.Frame().Texts())
		}
		e.wantDrawn("New folder name:")
	}

	// An empty name.
	w := newWritable(fdFS())
	e := openWritable(t, ui.FileSave, w)
	e.click("New folder")
	e.h.TypeKey(render.KeyBackspace) // deletes the selected suggestion
	e.h.TypeKey(render.KeyEnter)
	if !e.h.Frame().TextContaining("Enter a folder name") {
		t.Errorf("want a prompt for a name; drawn: %v", e.h.Frame().Texts())
	}
}

func TestFileDialogNewFolderSuggestsAFreeName(t *testing.T) {
	m := fdFS()
	m["New folder/x"] = &fstest.MapFile{Data: []byte("x")}
	m["New folder (2)/x"] = &fstest.MapFile{Data: []byte("x")}
	e := openWritable(t, ui.FileSave, newWritable(m))
	e.click("New folder")
	e.wantDrawn("New folder (3)")
}

func TestFileDialogNewFolderInSubdirectory(t *testing.T) {
	w := newWritable(fdFS())
	e := openWritable(t, ui.FileSave, w, ui.FileDialogStart("docs"))
	e.click("New folder")
	e.h.TypeText("inner")
	e.h.Step()
	e.h.TypeKey(render.KeyEnter)
	e.wantDrawn("inner/", "a.txt")
	if !w.hasDir("docs/inner") || w.hasDir("inner") {
		t.Errorf("created in the wrong place: docs/inner=%v inner=%v", w.hasDir("docs/inner"), w.hasDir("inner"))
	}
}

func TestFileDialogNewFolderKeepsTheTypedSaveName(t *testing.T) {
	e := openWritable(t, ui.FileSave, newWritable(fdFS()))
	e.h.TypeText("draft") // the name field has focus when saving
	e.h.Step()
	e.click("New folder")
	e.h.TypeText("sub")
	e.h.Step()
	e.h.TypeKey(render.KeyEnter)
	e.wantDrawn("sub/", "draft")
	e.click("Save")
	e.wantResult(true, "/draft")
}

func TestFileDialogNewFolderSelectDirChoosesIt(t *testing.T) {
	e := openWritable(t, ui.FileSelectDir, newWritable(fdFS()))
	e.click("New folder")
	e.h.TypeText("fresh")
	e.h.Step()
	e.h.TypeKey(render.KeyEnter)
	e.click("Select") // the new folder is the selected row
	e.wantResult(true, "/fresh")
}

func TestFileDialogNewFolderFocusGoesToListing(t *testing.T) {
	e := openWritable(t, ui.FileSelectDir, newWritable(fdFS()))
	e.click("New folder")
	e.h.TypeText("fresh")
	e.h.Step()
	e.h.TypeKey(render.KeyEnter)
	e.h.TypeKey(render.KeyEnter) // activates the selected new folder: browses into it
	e.wantDrawn("/fresh")        // the path field
	if e.calls != 0 {
		t.Errorf("calls = %d, want 0", e.calls)
	}
}

func TestFileDialogNewFolderClosingTheDialogTakesThePromptWithIt(t *testing.T) {
	w := newWritable(fdFS())
	e := openWritable(t, ui.FileSave, w)
	e.click("New folder")
	e.wantDrawn("New folder name:")
	e.h.App.Close(e.popup)
	e.wantResult(false)
	e.h.Step()
	e.wantAbsent("New folder name:", "Save file")
}

func TestFileDialogNewFolderFocusReturnsToTheButtonOnCancel(t *testing.T) {
	e := openWritable(t, ui.FileSelectDir, newWritable(fdFS()))
	e.click("New folder")
	e.h.TypeKey(render.KeyEscape)
	e.wantAbsent("New folder name:")
	// The button that opened the prompt has focus again, so Enter activates it.
	e.h.TypeKey(render.KeyEnter)
	e.wantDrawn("New folder name:")
	if e.calls != 0 {
		t.Errorf("calls = %d, want 0", e.calls)
	}
}
