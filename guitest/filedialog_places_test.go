package guitest_test

import (
	"testing"
	"testing/fstest"

	"github.com/kpfaulkner/guie/render"
	"github.com/kpfaulkner/guie/ui"
)

// placesFS is an FSFileSystem with a home directory and extra roots, as the OS
// one has on a real machine.
type placesFS struct {
	ui.FileSystem
	home  string
	roots []ui.FileRoot
}

func (p placesFS) Home() string         { return p.home }
func (p placesFS) Roots() []ui.FileRoot { return p.roots }

func placesFixture() fstest.MapFS {
	return fstest.MapFS{
		"readme.md":              {Data: []byte("r")},
		"home/me/hello.txt":      {Data: []byte("h")},
		"home/me/Desktop/d.txt":  {Data: []byte("d")},
		"home/me/Documents/y.md": {Data: []byte("y")},
		"data/disk.bin":          {Data: []byte("b")},
	}
}

func placesFileSystem(roots ...ui.FileRoot) placesFS {
	return placesFS{FileSystem: ui.FSFileSystem(placesFixture()), home: "/home/me", roots: roots}
}

func openPlaces(t *testing.T, opts ...ui.FileDialogOption) *fdEnv {
	t.Helper()
	pfs := placesFileSystem(ui.FileRoot{Name: "/", Path: "/"}, ui.FileRoot{Name: "D:", Path: "/data"})
	h := newHarnessFor(t, ui.FileOpen, append([]ui.FileDialogOption{ui.FileDialogFS(pfs)}, opts...)...)
	return h
}

// newHarnessFor opens a dialog whose filesystem comes from the options alone.
func newHarnessFor(t *testing.T, mode ui.FileDialogMode, opts ...ui.FileDialogOption) *fdEnv {
	t.Helper()
	// openMode always prepends an FS option; a later option replaces it, so the
	// placeholder FS here is never used.
	return openMode(t, mode, fstest.MapFS{}, opts...)
}

func TestFileDialogPlacesListHomeItsFoldersAndRoots(t *testing.T) {
	e := openPlaces(t)
	e.wantDrawn("Home", "Desktop", "Documents", "/", "D:")
	e.wantAbsent("Downloads") // does not exist, so not offered
	e.wantAbsent("Drives")    // the old drives button is gone
	e.wantDrawn("hello.txt")  // opens in home when no start is given
}

func TestFileDialogPlacesNavigate(t *testing.T) {
	e := openPlaces(t)
	e.click("Documents")
	e.wantDrawn("y.md")
	e.wantAbsent("hello.txt")
	e.click("D:")
	e.wantDrawn("disk.bin")
	e.click("Desktop")
	e.wantDrawn("d.txt")
	e.click("Home")
	e.wantDrawn("hello.txt", "Desktop/") // the folder shows as a row too
}

func TestFileDialogPlacesClickMovesFocusToListing(t *testing.T) {
	e := openPlaces(t)
	e.click("Documents")
	e.h.TypeKey(render.KeyDown) // goes to the listing, not to the place button
	e.h.TypeKey(render.KeyEnter)
	e.wantResult(true, "/home/me/Documents/y.md")
}

func TestFileDialogNoPlacesPaneForASingleRootAndNoHome(t *testing.T) {
	e := openFD(t, fdFS())
	e.wantDrawn("readme.md")
	e.wantAbsent("Home", "Drives")
}

func TestFileDialogPlacesPaneWorksInFolderMode(t *testing.T) {
	pfs := placesFileSystem(ui.FileRoot{Name: "/", Path: "/"})
	e := newHarnessFor(t, ui.FileSelectDir, ui.FileDialogFS(pfs))
	e.wantDrawn("Home", "Desktop", "Documents")
	e.click("Documents")
	e.click("Select")
	e.wantResult(true, "/home/me/Documents")
}

func TestFileDialogShowHiddenCheckbox(t *testing.T) {
	e := openFD(t, fdFS())
	e.wantDrawn("Show hidden")
	e.wantAbsent(".hidden")
	e.click("Show hidden")
	e.wantDrawn(".hidden")
	e.click("Show hidden")
	e.wantAbsent(".hidden")
}

func TestFileDialogShowHiddenOptionSetsTheCheckbox(t *testing.T) {
	e := openFD(t, fdFS(), ui.FileDialogShowHidden(true))
	e.wantDrawn(".hidden")
	e.click("Show hidden") // starts checked, so this hides them
	e.wantAbsent(".hidden")
}

func TestFileDialogHiddenToggleKeepsSelection(t *testing.T) {
	e := openFD(t, fdFS())
	e.click("pic.png")
	e.click("Show hidden")
	e.wantDrawn(".hidden")
	e.click("Open")
	e.wantResult(true, "/pic.png")
}

func TestFileDialogHiddenToggleInFolderMode(t *testing.T) {
	e := openMode(t, ui.FileSelectDir, fstest.MapFS{
		"vis/a":   {Data: []byte("a")},
		".git/b":  {Data: []byte("b")},
		"file.go": {Data: []byte("f")},
	})
	e.wantDrawn("vis/")
	e.wantAbsent(".git/")
	e.click("Show hidden")
	e.wantDrawn(".git/")
}

func TestFileDialogTypeAheadSelects(t *testing.T) {
	e := openFD(t, fdFS())
	e.h.TypeText("p") // the listing has focus when the dialog opens
	e.h.Step()
	e.click("Open")
	e.wantResult(true, "/pic.png")
}

func TestFileDialogTypeAheadMultipleLetters(t *testing.T) {
	e := openFD(t, fdFS())
	e.h.TypeText("r")
	e.h.Step()
	e.h.TypeText("e")
	e.h.Step()
	e.h.TypeKey(render.KeyEnter) // activates the selected row
	e.wantResult(true, "/readme.md")
}

func TestFileDialogTypeAheadMatchesDirectoriesToo(t *testing.T) {
	e := openFD(t, fdFS())
	e.h.TypeText("d")
	e.h.Step()
	e.h.TypeKey(render.KeyEnter)
	e.wantDrawn("a.txt", "b.md") // opened docs/
	if e.calls != 0 {
		t.Errorf("calls = %d, want 0", e.calls)
	}
}

func TestFileDialogTypeAheadNoMatchKeepsSelection(t *testing.T) {
	e := openFD(t, fdFS())
	e.click("notes.txt")
	e.h.TypeText("z")
	e.h.Step()
	e.click("Open")
	e.wantResult(true, "/notes.txt")
}

func TestFileDialogTypeAheadRepeatedLetterCycles(t *testing.T) {
	fsys := fstest.MapFS{
		"alpha":   {Data: []byte("1")},
		"apple":   {Data: []byte("1")},
		"avocado": {Data: []byte("1")},
		"banana":  {Data: []byte("1")},
	}
	e := openFD(t, fsys)
	for i := 0; i < 3; i++ { // alpha, apple, avocado
		e.h.TypeText("a")
		e.h.Step()
	}
	e.h.TypeKey(render.KeyEnter)
	e.wantResult(true, "/avocado")
}

func TestFileDialogTypingInNameFieldIsNotTypeAhead(t *testing.T) {
	e := openFD(t, fdFS())
	e.click("File name") // focus the name field
	e.h.TypeText("p")
	e.h.Step()
	e.click("Open") // the name is "p", which does not exist; nothing jumped
	if e.calls != 0 || !e.h.Frame().TextContaining("does not exist") {
		t.Errorf("calls=%d; drawn: %v", e.calls, e.h.Frame().Texts())
	}
}

func TestFileDialogTypeAheadWorksAfterUpButton(t *testing.T) {
	e := openFD(t, fdFS(), ui.FileDialogStart("docs/sub"))
	e.click("Up") // focus goes back to the listing
	e.h.TypeText("s")
	e.h.Step()
	e.h.TypeKey(render.KeyEnter) // sub/ is a directory, so Enter opens it
	e.wantDrawn("deep.txt")
}
