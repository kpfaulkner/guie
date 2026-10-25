package guitest_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kpfaulkner/guie/guitest"
	"github.com/kpfaulkner/guie/render"
	"github.com/kpfaulkner/guie/ui"
)

func wg(x, y, w, h int) render.WindowGeometry {
	return render.WindowGeometry{X: x, Y: y, Width: w, Height: h, HasPosition: true}
}

func TestWithWindowGeometryIsAppliedAtStartup(t *testing.T) {
	want := wg(40, 60, 700, 500)
	h := guitest.New(300, 200, ui.WithWindowGeometry(want))
	if got := h.AppliedWindowGeometry(); got != want {
		t.Errorf("applied = %+v, want %+v", got, want)
	}
}

func TestNoGeometryAppliesNothing(t *testing.T) {
	h := guitest.New(300, 200)
	if got := h.AppliedWindowGeometry(); got != (render.WindowGeometry{}) {
		t.Errorf("applied = %+v, want the zero value", got)
	}
	if _, ok := h.App.WindowGeometry(); ok {
		t.Error("no geometry should be known before the window reports one")
	}
}

func TestAppTracksWindowGeometry(t *testing.T) {
	h := guitest.New(300, 200)
	h.SetWindowGeometry(wg(10, 20, 640, 480), true).Step()
	if g, ok := h.App.WindowGeometry(); !ok || g != wg(10, 20, 640, 480) {
		t.Fatalf("geometry = %+v ok=%v", g, ok)
	}
	h.SetWindowGeometry(wg(30, 40, 800, 600), true).Step()
	if g, _ := h.App.WindowGeometry(); g != wg(30, 40, 800, 600) {
		t.Errorf("after a move/resize geometry = %+v", g)
	}
}

func TestMaximizedKeepsRestoredGeometry(t *testing.T) {
	h := guitest.New(300, 200)
	h.SetWindowGeometry(wg(10, 20, 640, 480), true).Step()

	max := wg(0, 0, 2560, 1440)
	max.Maximized = true
	h.SetWindowGeometry(max, true).Step()
	g, _ := h.App.WindowGeometry()
	want := wg(10, 20, 640, 480)
	want.Maximized = true
	if g != want {
		t.Errorf("while maximized: %+v, want the restored size with Maximized set (%+v)", g, want)
	}

	// Restoring clears the flag and takes the live geometry again.
	h.SetWindowGeometry(wg(15, 25, 700, 500), true).Step()
	if g, _ := h.App.WindowGeometry(); g != wg(15, 25, 700, 500) {
		t.Errorf("after restore: %+v", g)
	}
}

func TestMinimizedKeepsLastGeometry(t *testing.T) {
	h := guitest.New(300, 200)
	h.SetWindowGeometry(wg(10, 20, 640, 480), true).Step()
	h.SetWindowGeometry(wg(-32000, -32000, 160, 28), false).Step() // minimized
	if g, _ := h.App.WindowGeometry(); g != wg(10, 20, 640, 480) {
		t.Errorf("minimized window overwrote the saved geometry: %+v", g)
	}
}

func TestWindowStateFileRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "window.json") // directory is created on save

	// First run: nothing saved yet, so nothing is applied; the user then moves the
	// window, and exiting writes it.
	h := guitest.New(300, 200, ui.WithWindowStateFile(path))
	if got := h.AppliedWindowGeometry(); got != (render.WindowGeometry{}) {
		t.Fatalf("first run applied %+v, want none", got)
	}
	h.SetWindowGeometry(wg(50, 70, 900, 650), true).Step()
	if err := h.App.Run(); err != nil { // the harness driver returns at once, so this is "exit"
		t.Fatal(err)
	}

	// Second run restores it.
	h2 := guitest.New(300, 200, ui.WithWindowStateFile(path))
	if got := h2.AppliedWindowGeometry(); got != wg(50, 70, 900, 650) {
		t.Errorf("second run applied %+v, want the saved geometry", got)
	}
	if g, ok := h2.App.WindowGeometry(); !ok || g != wg(50, 70, 900, 650) {
		t.Errorf("restored geometry not remembered: %+v ok=%v", g, ok)
	}
}

func TestWindowStateFileSavesMaximizedWithRestoredSize(t *testing.T) {
	path := filepath.Join(t.TempDir(), "w.json")
	h := guitest.New(300, 200, ui.WithWindowStateFile(path))
	h.SetWindowGeometry(wg(10, 20, 640, 480), true).Step()
	max := wg(0, 0, 2560, 1440)
	max.Maximized = true
	h.SetWindowGeometry(max, true).Step()
	h.App.Run()

	got, ok := ui.LoadWindowGeometry(path)
	want := wg(10, 20, 640, 480)
	want.Maximized = true
	if !ok || got != want {
		t.Errorf("saved %+v ok=%v, want %+v", got, ok, want)
	}
}

func TestWindowStateFileNotWrittenWithoutGeometry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "w.json")
	h := guitest.New(300, 200, ui.WithWindowStateFile(path))
	h.App.Run() // never saw a geometry
	if _, err := os.Stat(path); err == nil {
		t.Error("a state file was written although no geometry was ever known")
	}
}

func TestLoadWindowGeometryRejectsBadFiles(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	if _, ok := ui.LoadWindowGeometry(filepath.Join(dir, "missing.json")); ok {
		t.Error("missing file should not load")
	}
	if _, ok := ui.LoadWindowGeometry(write("junk.json", "not json {")); ok {
		t.Error("corrupt file should not load")
	}
	if _, ok := ui.LoadWindowGeometry(write("empty.json", "{}")); ok {
		t.Error("an empty geometry should not count as saved state")
	}
}

func TestLoadWindowGeometrySanitizes(t *testing.T) {
	p := filepath.Join(t.TempDir(), "w.json")
	body := `{"x":999999,"y":5,"width":99999,"height":600,"hasPosition":true,"maximized":true}`
	os.WriteFile(p, []byte(body), 0o644)
	g, ok := ui.LoadWindowGeometry(p)
	if !ok {
		t.Fatal("file with a usable flag should still load")
	}
	if g.Width != 0 || g.Height != 0 {
		t.Errorf("an absurd width must clear the whole size, got %dx%d", g.Width, g.Height)
	}
	if g.HasPosition || g.X != 0 || g.Y != 0 {
		t.Errorf("an absurd x must clear the position, got %+v", g)
	}
	if !g.Maximized {
		t.Error("the maximized flag should survive")
	}
}

func TestSaveWindowGeometryRoundTripAndNoTempLeftBehind(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "w.json")
	want := wg(1, 2, 800, 600)
	if err := ui.SaveWindowGeometry(p, want); err != nil {
		t.Fatal(err)
	}
	if err := ui.SaveWindowGeometry(p, want); err != nil { // overwrite
		t.Fatal(err)
	}
	got, ok := ui.LoadWindowGeometry(p)
	if !ok || got != want {
		t.Errorf("round trip = %+v ok=%v", got, ok)
	}
	if _, err := os.Stat(p + ".tmp"); err == nil {
		t.Error("temporary file left behind")
	}
}
