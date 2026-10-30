package guitest_test

import (
	"io/fs"
	"sync"
	"testing"
	"time"

	"github.com/kpfaulkner/guie/guitest"
	"github.com/kpfaulkner/guie/render"
	"github.com/kpfaulkner/guie/ui"
)

// gateFS holds ReadDir of chosen directories until the test lets them go, as a
// slow network share would.
type gateFS struct {
	ui.FileSystem
	mu    sync.Mutex
	gates map[string]chan struct{}
	seen  chan string // directories whose read has started
	reads map[string]int
}

func newGateFS(inner ui.FileSystem) *gateFS {
	return &gateFS{FileSystem: inner, gates: map[string]chan struct{}{}, seen: make(chan string, 64), reads: map[string]int{}}
}

// readCount is how many times dir has been read.
func (g *gateFS) readCount(dir string) int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.reads[dir]
}

// hold makes reads of dir wait; the returned func lets them through.
func (g *gateFS) hold(dir string) (release func()) {
	ch := make(chan struct{})
	g.mu.Lock()
	g.gates[dir] = ch
	g.mu.Unlock()
	var once sync.Once
	return func() { once.Do(func() { close(ch) }) }
}

func (g *gateFS) ReadDir(dir string) ([]fs.FileInfo, error) {
	g.mu.Lock()
	ch := g.gates[dir]
	g.reads[dir]++
	g.mu.Unlock()
	if ch != nil {
		g.seen <- dir
		<-ch
	}
	return g.FileSystem.ReadDir(dir)
}

// waitStarted blocks until a read of dir is in flight.
func (g *gateFS) waitStarted(t *testing.T, dir string) {
	t.Helper()
	deadline := time.After(5 * time.Second)
	for {
		select {
		case d := <-g.seen:
			if d == dir {
				return
			}
		case <-deadline:
			t.Fatalf("no read of %q started", dir)
		}
	}
}

func openGated(t *testing.T) (*fdEnv, *gateFS) {
	t.Helper()
	g := newGateFS(ui.FSFileSystem(fdFS()))
	h := guitest.New(800, 600)
	h.SetContent(ui.NewContainer())
	e := &fdEnv{t: t, h: h}
	e.popup = h.App.ShowFileDialog(ui.FileOpen, func(paths []string, ok bool) {
		e.calls++
		e.paths, e.ok = paths, ok
	}, ui.FileDialogFS(g))
	h.Step()
	return e, g
}

func TestFileDialogLoadKeepsTheOldListingUntilTheNewOneArrives(t *testing.T) {
	e, g := openGated(t)
	e.wantDrawn("docs/", "notes.txt")

	release := g.hold("/docs")
	defer release()
	e.h.HoldAsync(true)
	e.doubleClick("docs/")
	g.waitStarted(t, "/docs")

	// Still reading: the dialog answers, and still shows where it was.
	e.h.Step()
	e.wantDrawn("docs/", "notes.txt")
	e.wantAbsent("a.txt")

	release()
	e.h.HoldAsync(false)
	e.h.Step()
	e.wantDrawn("a.txt", "b.md")
	e.wantAbsent("notes.txt")
}

func TestFileDialogLoadHintAppearsOnlyWhenSlow(t *testing.T) {
	e, g := openGated(t)
	release := g.hold("/docs")
	defer release()
	e.h.HoldAsync(true)
	e.doubleClick("docs/")
	g.waitStarted(t, "/docs")

	e.h.Step()
	if e.h.Frame().TextContaining("Loading") {
		t.Error("a read that has only just started must not show a loading message")
	}
	time.Sleep(300 * time.Millisecond)
	e.h.Step()
	if !e.h.Frame().TextContaining("Loading /docs") {
		t.Errorf("a slow read should say it is loading; drawn: %v", e.h.Frame().Texts())
	}

	release()
	e.h.HoldAsync(false)
	e.h.Step()
	if e.h.Frame().TextContaining("Loading") {
		t.Errorf("the message should go once the listing arrives; drawn: %v", e.h.Frame().Texts())
	}
	e.wantDrawn("a.txt")
}

func TestFileDialogQuickLoadNeverShowsTheHint(t *testing.T) {
	e, _ := openGated(t)
	e.doubleClick("docs/")
	time.Sleep(300 * time.Millisecond) // past the hint delay; the read was long finished
	e.h.Step()
	if e.h.Frame().TextContaining("Loading") {
		t.Errorf("a finished read must not show the message later; drawn: %v", e.h.Frame().Texts())
	}
}

func TestFileDialogLaterNavigationSupersedesASlowOne(t *testing.T) {
	e, g := openGated(t)
	slowRelease := g.hold("/docs")
	defer slowRelease()
	e.h.HoldAsync(true)
	e.doubleClick("docs/") // slow, and still in flight
	g.waitStarted(t, "/docs")

	// Go somewhere else while that read hangs: Up from a dialog that has not moved
	// yet is a no-op at the root, so use the path bar.
	e.click("/")
	e.keyWith(fdPrimary, render.KeyA)
	e.h.TypeText("/docs/sub")
	e.h.Step()
	e.h.TypeKey(render.KeyEnter)
	// The hung read is still running, so Step must not wait for it: step by hand
	// until the new listing lands.
	deadline := time.Now().Add(5 * time.Second)
	for !e.drawn("deep.txt") {
		if time.Now().After(deadline) {
			t.Fatalf("/docs/sub never listed while /docs hung; drawn: %v", e.h.Frame().Texts())
		}
		time.Sleep(time.Millisecond)
		e.h.Step()
	}

	// The slow one finally comes back; it must not drag the dialog to /docs.
	slowRelease()
	e.h.HoldAsync(false)
	e.h.Step() // waits for the released read, then delivers (and drops) its result
	e.wantDrawn("deep.txt")
	e.wantAbsent("a.txt")
	if e.h.Err() != nil {
		t.Errorf("harness error: %v", e.h.Err())
	}
}

func TestFileDialogClosingDuringALoadIsClean(t *testing.T) {
	e, g := openGated(t)
	release := g.hold("/docs")
	e.h.HoldAsync(true)
	e.doubleClick("docs/")
	g.waitStarted(t, "/docs")

	e.h.TypeKey(render.KeyEscape)
	e.wantResult(false)

	release()
	e.h.HoldAsync(false)
	e.h.Step() // the late result must be dropped, not applied to a closed dialog
	if e.h.Err() != nil {
		t.Errorf("harness error: %v", e.h.Err())
	}
	e.wantResult(false)
}

// A dialog closed while its start directory is still being read must not carry on
// down its list of fallbacks.
func TestFileDialogClosedDuringInitialLoadStopsFallingBack(t *testing.T) {
	g := newGateFS(ui.FSFileSystem(fdFS()))
	release := g.hold("/nope")
	h := guitest.New(800, 600)
	h.SetContent(ui.NewContainer())
	e := &fdEnv{t: t, h: h}
	h.HoldAsync(true)
	e.popup = h.App.ShowFileDialog(ui.FileOpen, func(paths []string, ok bool) {
		e.calls++
		e.paths, e.ok = paths, ok
	}, ui.FileDialogFS(g), ui.FileDialogStart("/nope")) // missing: the next try would be home or the root
	g.waitStarted(t, "/nope")
	h.Step()
	h.TypeKey(render.KeyEscape)
	e.wantResult(false)

	release()
	h.HoldAsync(false)
	h.Step()
	if n := g.readCount("/"); n != 0 {
		t.Errorf("the closed dialog went on to read the root %d time(s)", n)
	}
}

func TestFileDialogOKBeforeTheFirstListingDoesNothing(t *testing.T) {
	g := newGateFS(ui.FSFileSystem(fdFS()))
	release := g.hold("/")
	defer release()
	h := guitest.New(800, 600)
	h.SetContent(ui.NewContainer())
	e := &fdEnv{t: t, h: h}
	h.HoldAsync(true)
	e.popup = h.App.ShowFileDialog(ui.FileOpen, func(paths []string, ok bool) {
		e.calls++
		e.paths, e.ok = paths, ok
	}, ui.FileDialogFS(g))
	g.waitStarted(t, "/")
	h.Step()

	// A typed name would resolve against an empty directory and name something
	// that does not exist in the listing the user has not seen yet.
	e.click("File name")
	h.TypeText("readme.md")
	h.Step()
	e.click("Open")
	if e.calls != 0 {
		t.Fatalf("OK with no listing resolved the dialog: %v %v", e.paths, e.ok)
	}

	release()
	h.HoldAsync(false)
	h.Step()
	e.wantDrawn("readme.md", "docs/")
}
