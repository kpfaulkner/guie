package guitest_test

import (
	"io/fs"
	"reflect"
	"testing"
	"testing/fstest"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/guitest"
	"github.com/kpfaulkner/guie/render"
	"github.com/kpfaulkner/guie/ui"
)

// fdFS is the filesystem the file dialog tests browse. File sizes differ so a
// size sort has something to order.
func fdFS() fstest.MapFS {
	return fstest.MapFS{
		"readme.md":         {Data: []byte("12345")},
		"notes.txt":         {Data: []byte("12345678901234567890")},
		"pic.png":           {Data: []byte("1234567890")},
		".hidden":           {Data: []byte("x")},
		"docs/a.txt":        {Data: []byte("a")},
		"docs/b.md":         {Data: []byte("b")},
		"docs/sub/deep.txt": {Data: []byte("d")},
	}
}

// failFS refuses to open one path, as a permission error would.
type failFS struct {
	fs.FS
	bad string
}

func (f failFS) Open(name string) (fs.File, error) {
	if name == f.bad {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrPermission}
	}
	return f.FS.Open(name)
}

// fdEnv is an open file dialog under a harness, and what its callback saw.
type fdEnv struct {
	t     *testing.T
	h     *guitest.Harness
	popup *ui.Popup
	calls int
	paths []string
	ok    bool
}

func openFD(t *testing.T, fsys fs.FS, opts ...ui.FileDialogOption) *fdEnv {
	t.Helper()
	return openMode(t, ui.FileOpen, fsys, opts...)
}

func openMode(t *testing.T, mode ui.FileDialogMode, fsys fs.FS, opts ...ui.FileDialogOption) *fdEnv {
	t.Helper()
	h := guitest.New(800, 600)
	h.SetContent(ui.NewContainer())
	e := &fdEnv{t: t, h: h}
	opts = append([]ui.FileDialogOption{ui.FileDialogFS(ui.FSFileSystem(fsys))}, opts...)
	e.popup = h.App.ShowFileDialog(mode, func(paths []string, ok bool) {
		e.calls++
		e.paths, e.ok = paths, ok
	}, opts...)
	h.Step()
	return e
}

// pos returns a point just inside the first thing drawn as text.
func (e *fdEnv) pos(text string) geom.Point {
	e.t.Helper()
	for _, op := range e.h.Frame().OpsOfKind(guitest.OpDrawText) {
		if op.Text == text {
			return geom.Point{X: op.A.X + 3, Y: op.A.Y + 3}
		}
	}
	e.t.Fatalf("%q is not drawn; drawn: %v", text, e.h.Frame().Texts())
	return geom.Point{}
}

func (e *fdEnv) click(text string) {
	e.t.Helper()
	p := e.pos(text)
	e.h.Click(p.X, p.Y)
}

func (e *fdEnv) doubleClick(text string) {
	e.t.Helper()
	p := e.pos(text)
	e.h.Click(p.X, p.Y)
	e.h.Click(p.X, p.Y)
}

func (e *fdEnv) drawn(text string) bool { return e.h.Frame().HasText(text) }

// index returns where text was first drawn in the frame's draw order, or -1.
func (e *fdEnv) index(text string) int {
	for i, s := range e.h.Frame().Texts() {
		if s == text {
			return i
		}
	}
	return -1
}

func (e *fdEnv) wantDrawn(names ...string) {
	e.t.Helper()
	for _, n := range names {
		if !e.drawn(n) {
			e.t.Errorf("%q should be listed; drawn: %v", n, e.h.Frame().Texts())
		}
	}
}

func (e *fdEnv) wantAbsent(names ...string) {
	e.t.Helper()
	for _, n := range names {
		if e.drawn(n) {
			e.t.Errorf("%q should not be listed; drawn: %v", n, e.h.Frame().Texts())
		}
	}
}

// wantOrder checks that the names are drawn in this order, top to bottom.
func (e *fdEnv) wantOrder(names ...string) {
	e.t.Helper()
	last := -1
	for _, n := range names {
		i := e.index(n)
		if i < 0 {
			e.t.Fatalf("%q is not drawn; drawn: %v", n, e.h.Frame().Texts())
		}
		if i < last {
			e.t.Errorf("%q is drawn above an earlier name in %v; drawn: %v", n, names, e.h.Frame().Texts())
		}
		last = i
	}
}

func (e *fdEnv) wantResult(ok bool, paths ...string) {
	e.t.Helper()
	if e.calls != 1 {
		e.t.Fatalf("onResult called %d times, want exactly 1", e.calls)
	}
	if e.ok != ok || !reflect.DeepEqual(e.paths, paths) && !(len(paths) == 0 && len(e.paths) == 0) {
		e.t.Errorf("result = (%v, %v), want (%v, %v)", e.paths, e.ok, paths, ok)
	}
}

func (e *fdEnv) wantClosed() {
	e.t.Helper()
	e.h.Step()
	if e.drawn("Open file") || e.drawn("Save file") || e.drawn("Select folder") {
		e.t.Error("dialog should be closed")
	}
}

func TestFileDialogListsDirectoriesFirst(t *testing.T) {
	e := openFD(t, fdFS())
	e.wantDrawn("Open file", "docs/", "notes.txt", "pic.png", "readme.md")
	e.wantAbsent(".hidden")
	e.wantOrder("docs/", "notes.txt", "pic.png", "readme.md")
}

func TestFileDialogShowHidden(t *testing.T) {
	e := openFD(t, fdFS(), ui.FileDialogShowHidden(true))
	e.wantDrawn(".hidden")
}

func TestFileDialogDoubleClickDirNavigates(t *testing.T) {
	e := openFD(t, fdFS())
	e.doubleClick("docs/")
	e.wantDrawn("a.txt", "b.md", "sub/")
	e.wantAbsent("readme.md")
	e.wantDrawn("/docs") // the path field
	if e.calls != 0 {
		t.Errorf("navigating must not report a result, calls = %d", e.calls)
	}
}

func TestFileDialogUpButtonAndSelectsWhereYouWere(t *testing.T) {
	e := openFD(t, fdFS(), ui.FileDialogStart("docs/sub"))
	e.wantDrawn("deep.txt")
	e.click("Up")
	e.wantDrawn("a.txt", "b.md", "sub/")
	e.click("Up")
	e.wantDrawn("readme.md")
	e.wantAbsent("a.txt")

	// At the root there is nowhere further up; Up must not move or fail.
	e.click("Up")
	e.wantDrawn("readme.md")
}

func TestFileDialogBackspaceInListingGoesUp(t *testing.T) {
	e := openFD(t, fdFS(), ui.FileDialogStart("docs"))
	e.wantDrawn("a.txt")
	e.h.TypeKey(render.KeyBackspace)
	e.wantDrawn("readme.md")
}

func TestFileDialogDoubleClickFileResolves(t *testing.T) {
	e := openFD(t, fdFS())
	e.doubleClick("readme.md")
	e.wantResult(true, "/readme.md")
	e.wantClosed()
}

func TestFileDialogSelectFillsNameAndOpenConfirms(t *testing.T) {
	e := openFD(t, fdFS())
	e.click("notes.txt")
	// Row and name field both draw it now.
	n := 0
	for _, s := range e.h.Frame().Texts() {
		if s == "notes.txt" {
			n++
		}
	}
	if n != 2 {
		t.Errorf("notes.txt drawn %d times, want 2 (row and name field)", n)
	}
	e.click("Open")
	e.wantResult(true, "/notes.txt")
	e.wantClosed()
}

func TestFileDialogSelectingDirClearsNameAndOpenEntersIt(t *testing.T) {
	e := openFD(t, fdFS())
	e.click("notes.txt")
	e.click("docs/")
	e.click("Open")
	e.wantDrawn("a.txt")
	if e.calls != 0 {
		t.Errorf("Open on a directory must navigate, not resolve; calls = %d", e.calls)
	}
}

func TestFileDialogKeyboardNavigateAndOpen(t *testing.T) {
	e := openFD(t, fdFS())
	e.h.TypeKey(render.KeyDown) // first row: docs/
	e.h.TypeKey(render.KeyEnter)
	e.wantDrawn("sub/", "a.txt")
	e.h.TypeKey(render.KeyDown) // sub/
	e.h.TypeKey(render.KeyDown) // a.txt
	e.h.TypeKey(render.KeyEnter)
	e.wantResult(true, "/docs/a.txt")
}

func TestFileDialogCancelEscapeAndCloseResolveOnce(t *testing.T) {
	t.Run("cancel button", func(t *testing.T) {
		e := openFD(t, fdFS())
		e.click("Cancel")
		e.wantResult(false)
		e.wantClosed()
	})
	t.Run("escape", func(t *testing.T) {
		e := openFD(t, fdFS())
		e.h.TypeKey(render.KeyEscape)
		e.wantResult(false)
		e.wantClosed()
	})
	t.Run("App.Close", func(t *testing.T) {
		e := openFD(t, fdFS())
		e.h.App.Close(e.popup)
		e.wantResult(false)
	})
	t.Run("confirm then close does not resolve twice", func(t *testing.T) {
		e := openFD(t, fdFS())
		e.doubleClick("readme.md")
		e.h.TypeKey(render.KeyEscape)
		e.wantResult(true, "/readme.md")
	})
}

func TestFileDialogTypedNameEnter(t *testing.T) {
	e := openFD(t, fdFS())
	e.click("File name") // the placeholder
	e.h.TypeText("docs/b.md")
	e.h.Step() // runes are delivered after keys within a frame
	e.h.TypeKey(render.KeyEnter)
	e.wantResult(true, "/docs/b.md")
}

func TestFileDialogTypedNameThatIsADirNavigates(t *testing.T) {
	e := openFD(t, fdFS())
	e.click("File name")
	e.h.TypeText("docs")
	e.h.Step() // runes are delivered after keys within a frame
	e.h.TypeKey(render.KeyEnter)
	e.wantDrawn("b.md")
	if e.calls != 0 {
		t.Errorf("a directory name must navigate, calls = %d", e.calls)
	}
}

func TestFileDialogTypedMissingNameShowsErrorAndStaysOpen(t *testing.T) {
	e := openFD(t, fdFS())
	e.click("File name")
	e.h.TypeText("nope.txt")
	e.h.Step() // runes are delivered after keys within a frame
	e.h.TypeKey(render.KeyEnter)
	if !e.h.Frame().TextContaining("does not exist") {
		t.Errorf("want a does-not-exist message; drawn: %v", e.h.Frame().Texts())
	}
	if e.calls != 0 {
		t.Errorf("calls = %d, want 0", e.calls)
	}
	e.wantDrawn("Open file")
}

func TestFileDialogEmptyNameOpenAsksForAFile(t *testing.T) {
	e := openFD(t, fdFS())
	e.click("Open")
	if !e.h.Frame().TextContaining("Choose a file") {
		t.Errorf("want a prompt; drawn: %v", e.h.Frame().Texts())
	}
	if e.calls != 0 {
		t.Errorf("calls = %d, want 0", e.calls)
	}
}

func TestFileDialogTypedPathInPathBar(t *testing.T) {
	e := openFD(t, fdFS(), ui.FileDialogStart("docs"))
	e.click("/docs") // the path field; the caret lands in it
	e.h.TypeKey(render.KeyEnd)
	e.h.TypeText("/sub")
	e.h.Step() // runes are delivered after keys within a frame
	e.h.TypeKey(render.KeyEnter)
	e.wantDrawn("deep.txt")
	e.wantAbsent("a.txt")
}

func TestFileDialogFilterHidesFilesNotDirs(t *testing.T) {
	e := openFD(t, fdFS(), ui.FileDialogFilters(
		ui.FileFilter{Name: "All files"},
		ui.FileFilter{Name: "Images", Patterns: []string{"*.png"}},
	))
	e.wantDrawn("readme.md", "pic.png", "docs/")

	e.click("All files") // opens the dropdown
	e.click("Images")
	e.wantDrawn("pic.png", "docs/")
	e.wantAbsent("readme.md", "notes.txt")
}

func TestFileDialogSortByHeader(t *testing.T) {
	e := openFD(t, fdFS())
	e.click("Size")
	e.wantOrder("docs/", "readme.md", "pic.png", "notes.txt") // 5, 10, 20 bytes
	e.click("Size")
	e.wantOrder("docs/", "notes.txt", "pic.png", "readme.md") // dirs stay first
	e.click("Name")
	e.wantOrder("docs/", "notes.txt", "pic.png", "readme.md")
	e.click("Name")
	e.wantOrder("docs/", "readme.md", "pic.png", "notes.txt")
}

func TestFileDialogSortKeepsSelection(t *testing.T) {
	e := openFD(t, fdFS())
	e.click("pic.png")
	e.click("Size") // moves pic.png from row 3 to row 2
	e.click("Open")
	e.wantResult(true, "/pic.png")
}

func TestFileDialogUnreadableStartFallsBack(t *testing.T) {
	e := openFD(t, fdFS(), ui.FileDialogStart("missing/dir"))
	e.wantDrawn("readme.md")
}

func TestFileDialogReadDirErrorShowsStatusAndKeepsListing(t *testing.T) {
	e := openFD(t, failFS{FS: fdFS(), bad: "docs"})
	e.doubleClick("docs/")
	if !e.h.Frame().TextContaining("permission denied") {
		t.Errorf("want the error shown; drawn: %v", e.h.Frame().Texts())
	}
	e.wantDrawn("readme.md", "docs/") // still the old listing
	e.wantAbsent("a.txt")
	if e.calls != 0 {
		t.Errorf("calls = %d, want 0", e.calls)
	}
}

func TestFileDialogRestoresFocus(t *testing.T) {
	h := guitest.New(800, 600)
	field := ui.NewTextField()
	root := ui.NewContainer()
	root.SetLayout(ui.VBox(0))
	root.Add(field)
	h.SetContent(root)
	h.Step()
	b := field.Bounds()
	h.Click(b.X+5, b.Y+5) // focus the field

	h.App.ShowFileDialog(ui.FileOpen, nil, ui.FileDialogFS(ui.FSFileSystem(fdFS())))
	h.Step()
	h.TypeKey(render.KeyEscape)
	h.TypeText("x")
	h.Step()
	if field.Text() != "x" {
		t.Errorf("focus should return to the field after the dialog closes; text = %q", field.Text())
	}
}

func TestFileDialogUnknownModeCancels(t *testing.T) {
	h := guitest.New(800, 600)
	h.SetContent(ui.NewContainer())
	calls, ok := 0, true
	p := h.App.ShowFileDialog(ui.FileDialogMode(99), func(paths []string, o bool) { calls++; ok = o })
	if p != nil || calls != 1 || ok {
		t.Errorf("popup=%v calls=%d ok=%v, want nil popup and one cancel", p, calls, ok)
	}
}

func TestFileDialogFitsASmallSurface(t *testing.T) {
	h := guitest.New(300, 200)
	h.SetContent(ui.NewContainer())
	h.App.ShowFileDialog(ui.FileOpen, nil, ui.FileDialogFS(ui.FSFileSystem(fdFS())))
	h.Step()
	if !h.Frame().HasText("Open file") || !h.Frame().HasText("Cancel") {
		t.Errorf("dialog should still draw on a small surface; drawn: %v", h.Frame().Texts())
	}
}

func TestFileDialogSortsNamesNaturally(t *testing.T) {
	fsys := fstest.MapFS{
		"file10.txt": {Data: []byte("1")},
		"file2.txt":  {Data: []byte("1")},
		"File1.txt":  {Data: []byte("1")},
		"file20.txt": {Data: []byte("1")},
		"dir10/x":    {Data: []byte("1")},
		"dir9/x":     {Data: []byte("1")},
	}
	e := openFD(t, fsys)
	e.wantOrder("dir9/", "dir10/", "File1.txt", "file2.txt", "file10.txt", "file20.txt")
	e.click("Name") // the listing is already by name, so this reverses it
	e.wantOrder("dir10/", "dir9/", "file20.txt", "file10.txt", "file2.txt", "File1.txt")
}
