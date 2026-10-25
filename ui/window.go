package ui

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/kpfaulkner/guie/render"
)

// WindowGeometry is a host window's size and position, so an application can put
// it back where the user left it. Position is relative to the monitor the window
// is on. Maximized windows record their restored size, not the maximized one.
type WindowGeometry = render.WindowGeometry

const (
	// maxWindowSize and maxWindowPos bound what a saved file may ask for, so a
	// corrupt or hand-edited file cannot produce an absurd window.
	maxWindowSize = 16384
	maxWindowPos  = 32768
)

// WithWindowGeometry opens the window at g: its size, and its position if
// g.HasPosition, maximized if g.Maximized. Zero Width and Height keep the size
// from WithSize. A position that would leave the window off its monitor is pulled
// back on screen by the backend. Backends without windows (or without a position)
// ignore the parts they cannot honour.
func WithWindowGeometry(g WindowGeometry) AppOption {
	return func(a *App) { a.setGeometry(sanitizeGeometry(g)) }
}

// WithWindowStateFile remembers the window across runs: it restores the geometry
// saved in path (if the file is readable and sane) and writes the final geometry
// back to it when Run returns. The file is JSON; its directory is created on
// save. Where path lives is the application's choice (os.UserConfigDir is the
// usual home); the framework stores no other preferences.
//
// Saved state takes precedence over WithSize, since "the size the user left it"
// is what an app asks this option for. Pass it after any option that should not
// be overridden by it, or use WithWindowGeometry for explicit control.
func WithWindowStateFile(path string) AppOption {
	return func(a *App) {
		a.statePath = path
		if g, ok := LoadWindowGeometry(path); ok {
			a.setGeometry(g)
		}
	}
}

// setGeometry records g as the geometry to open at, and as the initial
// restorable geometry, so a window that starts maximized still remembers the size
// to restore to.
func (a *App) setGeometry(g WindowGeometry) {
	a.cfg.Geometry = g
	if g != (WindowGeometry{}) {
		a.geom, a.haveGeom = g, true
	}
}

// WindowGeometry returns the window's last known restorable geometry, and false
// if the backend cannot report one or no frame has run yet. While the window is
// maximized it keeps the size and position from before it was maximized, with
// Maximized set; while minimized it keeps the last normal geometry.
func (a *App) WindowGeometry() (WindowGeometry, bool) { return a.geom, a.haveGeom }

// trackGeometry refreshes the remembered geometry from the backend. It is cheap
// enough to run every frame and once more as the window closes, so a move that
// produced no frame is still caught.
func (a *App) trackGeometry() {
	// Backends such as EBiten panic if asked for the window before the loop runs,
	// and tests drive update directly without ever calling Run.
	if a.windower == nil || !a.running {
		return
	}
	cur, ok := a.windower.WindowGeometry()
	if !ok || cur.Width <= 0 || cur.Height <= 0 {
		return // minimized, fullscreen or unreadable: keep the last normal geometry
	}
	if cur.Maximized {
		if a.haveGeom {
			a.geom.Maximized = true // keep the restored size and position
		} else {
			a.geom, a.haveGeom = cur, true
		}
		return
	}
	a.geom, a.haveGeom = cur, true
}

// saveWindowState writes the final geometry to the state file, if one was
// configured and a geometry is known. Failure is ignored: not being able to
// remember the window must never stop an app from exiting.
func (a *App) saveWindowState() {
	if a.statePath == "" || !a.haveGeom {
		return
	}
	_ = SaveWindowGeometry(a.statePath, a.geom)
}

// LoadWindowGeometry reads a geometry saved by SaveWindowGeometry. It returns
// false if the file is missing, unreadable, not JSON, or holds nothing usable, so
// callers can fall back to their defaults without distinguishing the cases.
func LoadWindowGeometry(path string) (WindowGeometry, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return WindowGeometry{}, false
	}
	var g WindowGeometry
	if err := json.Unmarshal(data, &g); err != nil {
		return WindowGeometry{}, false
	}
	g = sanitizeGeometry(g)
	if g == (WindowGeometry{}) {
		return WindowGeometry{}, false
	}
	return g, true
}

// SaveWindowGeometry writes g to path as JSON, creating the directory if needed.
// The write goes to a temporary file first and is renamed into place, so a crash
// mid-write cannot leave a truncated file for the next launch to choke on.
func SaveWindowGeometry(path string, g WindowGeometry) error {
	data, err := json.MarshalIndent(sanitizeGeometry(g), "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// sanitizeGeometry drops the parts of g that are not plausible: a size outside
// (0, maxWindowSize] is cleared (both axes together), and a position beyond
// +/-maxWindowPos is cleared.
func sanitizeGeometry(g WindowGeometry) WindowGeometry {
	if g.Width <= 0 || g.Height <= 0 || g.Width > maxWindowSize || g.Height > maxWindowSize {
		g.Width, g.Height = 0, 0
	}
	if !g.HasPosition || abs(g.X) > maxWindowPos || abs(g.Y) > maxWindowPos {
		g.X, g.Y, g.HasPosition = 0, 0, false
	}
	return g
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
