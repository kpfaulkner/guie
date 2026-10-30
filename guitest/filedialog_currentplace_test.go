package guitest_test

import (
	"image/color"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/kpfaulkner/guie/guitest"
	"github.com/kpfaulkner/guie/render"
	"github.com/kpfaulkner/guie/ui"
)

func sameRGBA(a, b color.Color) bool {
	ar, ag, ab, aa := a.RGBA()
	br, bg, bb, ba := b.RGBA()
	return ar == br && ag == bg && ab == bb && aa == ba
}

// isCurrentPlace reports whether the places pane marks the place called name: its
// label is drawn in the on-primary colour and a primary-coloured fill sits behind it.
func (e *fdEnv) isCurrentPlace(name string) bool {
	e.t.Helper()
	probe := ui.NewLabel("x") // unmounted widgets resolve colours against the default theme
	primary := probe.ColourOf(ui.RolePrimary)
	onPrimary := probe.ColourOf(ui.RoleOnPrimary)

	// The last one drawn: the path bar can show the same text as a place ("/"), and
	// the pane is drawn after it.
	var text *guitest.Op
	for _, op := range e.h.Frame().OpsOfKind(guitest.OpDrawText) {
		if op.Text == name {
			op := op
			text = &op
		}
	}
	if text == nil {
		e.t.Fatalf("place %q is not drawn; drawn: %v", name, e.h.Frame().Texts())
	}
	coloured := sameRGBA(text.Colour, onPrimary)
	filled := false
	for _, op := range e.h.Frame().OpsOfKind(guitest.OpFillRoundRect) {
		if sameRGBA(op.Colour, primary) && op.Rect.Contains(text.A) {
			filled = true
		}
	}
	if coloured != filled {
		e.t.Errorf("place %q is half marked: on-primary text = %v, primary fill behind = %v", name, coloured, filled)
	}
	return coloured && filled
}

// wantCurrentPlace checks that exactly one place is marked, and it is name; an
// empty name means none is.
func (e *fdEnv) wantCurrentPlace(name string) {
	e.t.Helper()
	var marked []string
	for _, p := range []string{"Home", "Desktop", "Documents", "Downloads", "/", "D:"} {
		if e.drawn(p) && e.isCurrentPlace(p) {
			marked = append(marked, p)
		}
	}
	switch {
	case name == "" && len(marked) != 0:
		e.t.Errorf("no place should be marked, got %v", marked)
	case name != "" && (len(marked) != 1 || marked[0] != name):
		e.t.Errorf("marked places = %v, want just [%s]", marked, name)
	}
}

func currentPlaceFixture() fstest.MapFS {
	m := placesFixture()
	m["home/me/Documents/work/w.txt"] = &fstest.MapFile{Data: []byte("w")}
	m["home/me/Documents/work/deeper/x.txt"] = &fstest.MapFile{Data: []byte("x")}
	m["home/me/projects/p.txt"] = &fstest.MapFile{Data: []byte("p")}
	return m
}

func openCurrentPlace(t *testing.T, opts ...ui.FileDialogOption) *fdEnv {
	t.Helper()
	pfs := placesFS{
		FileSystem: ui.FSFileSystem(currentPlaceFixture()),
		home:       "/home/me",
		roots:      []ui.FileRoot{{Name: "/", Path: "/"}, {Name: "D:", Path: "/data"}},
	}
	return newHarnessFor(t, ui.FileOpen, append([]ui.FileDialogOption{ui.FileDialogFS(pfs)}, opts...)...)
}

func TestFileDialogPlacesMarkTheStartingPlace(t *testing.T) {
	openCurrentPlace(t).wantCurrentPlace("Home")
}

func TestFileDialogPlacesMarkFollowsClicks(t *testing.T) {
	e := openCurrentPlace(t)
	e.click("Documents")
	e.wantCurrentPlace("Documents")
	e.click("D:")
	e.wantCurrentPlace("D:")
	e.click("Desktop")
	e.wantCurrentPlace("Desktop")
	e.click("Home")
	e.wantCurrentPlace("Home")
	e.click("/")
	e.wantCurrentPlace("/")
}

func TestFileDialogPlacesMarkFollowsBrowsing(t *testing.T) {
	e := openCurrentPlace(t, ui.FileDialogStart("/home/me/Documents"))
	e.wantCurrentPlace("Documents")

	e.doubleClick("work/") // a folder inside Documents is still Documents
	e.wantDrawn("w.txt")
	e.wantCurrentPlace("Documents")
	e.doubleClick("deeper/")
	e.wantCurrentPlace("Documents")

	e.click("Up")
	e.click("Up")
	e.wantCurrentPlace("Documents") // back at Documents itself
	e.click("Up")
	e.wantCurrentPlace("Home") // one up from Documents is the home folder
	e.click("Up")
	e.wantCurrentPlace("/") // above home, only the filesystem root contains it
}

func TestFileDialogPlacesMarkHomeForOtherFoldersUnderIt(t *testing.T) {
	e := openCurrentPlace(t)
	e.doubleClick("projects/")
	e.wantDrawn("p.txt")
	e.wantCurrentPlace("Home")
}

func TestFileDialogPlacesMarkFollowsTheKeyboardAndPathBar(t *testing.T) {
	e := openCurrentPlace(t)
	e.h.TypeKey(render.KeyBackspace) // Backspace in the listing goes up: out of home
	e.wantCurrentPlace("/")

	e.click("/home") // the path bar, which now shows /home
	e.keyWith(fdPrimary, render.KeyA)
	e.h.TypeText("/data")
	e.h.Step()
	e.h.TypeKey(render.KeyEnter)
	e.wantDrawn("disk.bin")
	e.wantCurrentPlace("D:")
}

func TestFileDialogPlacesMarkSurvivesAFailedNavigation(t *testing.T) {
	pfs := placesFS{
		FileSystem: failFS2{ui.FSFileSystem(currentPlaceFixture())},
		home:       "/home/me",
		roots:      []ui.FileRoot{{Name: "/", Path: "/"}},
	}
	e := newHarnessFor(t, ui.FileOpen, ui.FileDialogFS(pfs))
	e.wantCurrentPlace("Home")
	e.doubleClick("projects/") // refused: the dialog stays where it was
	e.wantCurrentPlace("Home")
	e.wantDrawn("hello.txt")
}

// failFS2 refuses to list /home/me/projects, as a permission error would.
type failFS2 struct{ ui.FileSystem }

func (f failFS2) ReadDir(dir string) ([]fs.FileInfo, error) {
	if dir == "/home/me/projects" {
		return nil, &fs.PathError{Op: "open", Path: dir, Err: fs.ErrPermission}
	}
	return f.FileSystem.ReadDir(dir)
}
