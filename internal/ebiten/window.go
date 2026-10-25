package ebitenbackend

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kpfaulkner/guie/render"
)

const (
	// minVisibleW is how much of a restored window's width must stay on its
	// monitor, so a saved position from a larger or since-removed monitor never
	// leaves the window somewhere it cannot be reached.
	minVisibleW = 96
	// minVisibleBelowTop is how much must stay below the monitor's top edge, so the
	// title bar can still be grabbed.
	minVisibleBelowTop = 48
)

// WindowGeometry satisfies render.WindowController. It reads the live window, so
// it is only valid once the loop is running (EBiten panics on WindowPosition
// before then); the framework only calls it from its hooks. A minimized or
// fullscreen window has no geometry worth saving.
func (d *Driver) WindowGeometry() (render.WindowGeometry, bool) {
	if ebiten.IsWindowMinimized() || ebiten.IsFullscreen() {
		return render.WindowGeometry{}, false
	}
	x, y := ebiten.WindowPosition()
	w, h := ebiten.WindowSize()
	return render.WindowGeometry{
		X: x, Y: y, Width: w, Height: h,
		HasPosition: true,
		Maximized:   ebiten.IsWindowMaximized(),
	}, true
}

// applyGeometry sets the window size and position before the loop starts, which
// EBiten allows. Clamping to the monitor waits for place, because the monitor the
// window lands on is not known until the loop is running.
func applyGeometry(g render.WindowGeometry) {
	if g.Width > 0 && g.Height > 0 {
		ebiten.SetWindowSize(g.Width, g.Height)
	}
	if g.HasPosition {
		ebiten.SetWindowPosition(g.X, g.Y)
	}
}

// place runs once, on the first Update, when the window and its monitor exist.
// It pulls a restored window back onto the monitor and then maximizes it if it
// was maximized (MaximizeWindow needs the window to be up).
func place(g render.WindowGeometry) {
	mw, mh := ebiten.Monitor().Size()
	w, h := ebiten.WindowSize()
	if g.Width > 0 && g.Height > 0 && (w > mw || h > mh) {
		nw, nh := min(w, mw), min(h, mh)
		ebiten.SetWindowSize(nw, nh)
		w, h = nw, nh
	}
	if g.HasPosition {
		x, y := ebiten.WindowPosition()
		if nx, ny := clampToMonitor(x, y, w, h, mw, mh); nx != x || ny != y {
			ebiten.SetWindowPosition(nx, ny)
		}
	}
	if g.Maximized {
		ebiten.MaximizeWindow()
	}
}

// clampToMonitor moves a w x h window at (x, y) so that enough of it, including
// its title bar, is inside a mw x mh monitor. Positions already fine are returned
// unchanged.
func clampToMonitor(x, y, w, h, mw, mh int) (int, int) {
	vis := min(minVisibleW, w)
	x = max(x, vis-w)  // at most w-vis of the window may hang off the left
	x = min(x, mw-vis) // at least vis must remain before the right edge
	y = max(y, 0)      // the title bar never above the top
	y = min(y, mh-min(minVisibleBelowTop, h))
	return x, y
}
