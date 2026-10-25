package ebitenbackend

import "testing"

func TestClampToMonitor(t *testing.T) {
	// A 800x600 window on a 1920x1080 monitor.
	cases := []struct {
		name       string
		x, y       int
		wantX, wnY int
	}{
		{"inside is untouched", 100, 80, 100, 80},
		{"far right pulled back", 5000, 80, 1920 - minVisibleW, 80},
		{"far left pulled back", -5000, 80, minVisibleW - 800, 80},
		{"above the top pulled down", 100, -200, 100, 0},
		{"far below pulled up", 100, 5000, 100, 1080 - minVisibleBelowTop},
		{"partly off left is allowed", -300, 0, -300, 0},
		{"corner at the edge is allowed", 1920 - minVisibleW, 1080 - minVisibleBelowTop, 1920 - minVisibleW, 1080 - minVisibleBelowTop},
	}
	for _, c := range cases {
		x, y := clampToMonitor(c.x, c.y, 800, 600, 1920, 1080)
		if x != c.wantX || y != c.wnY {
			t.Errorf("%s: clamp(%d,%d) = (%d,%d), want (%d,%d)", c.name, c.x, c.y, x, y, c.wantX, c.wnY)
		}
	}
}

func TestClampToMonitorTinyWindow(t *testing.T) {
	// A window narrower than the visible margin must still stay fully reachable.
	x, y := clampToMonitor(5000, 5000, 40, 20, 1000, 800)
	if x != 1000-40 || y != 800-20 {
		t.Errorf("tiny window clamped to (%d,%d), want (960,780)", x, y)
	}
}
