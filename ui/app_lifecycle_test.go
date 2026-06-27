package ui

import (
	"image/color"
	"testing"

	"github.com/kpfaulkner/guie/render"
	"github.com/kpfaulkner/guie/theme"
)

func TestAppOptionsConfigureConfig(t *testing.T) {
	red := color.RGBA{R: 255, A: 255}
	a := NewApp(
		WithTitle("My Title"),
		WithSize(123, 456),
		WithResizable(false),
		WithBackground(red),
		WithShadows(false),
		WithDebugBounds(true),
	)
	if a.cfg.Title != "My Title" {
		t.Errorf("Title: got %q", a.cfg.Title)
	}
	if a.cfg.Width != 123 || a.cfg.Height != 456 {
		t.Errorf("Size: got %dx%d, want 123x456", a.cfg.Width, a.cfg.Height)
	}
	if a.cfg.Resizable {
		t.Error("WithResizable(false) should disable resizing")
	}
	if !sameColour(a.cfg.Background, red) {
		t.Errorf("Background: got %v, want red", a.cfg.Background)
	}
	if a.shadows {
		t.Error("WithShadows(false) should disable shadows")
	}
	if !a.debug {
		t.Error("WithDebugBounds(true) should enable debug bounds")
	}
}

func TestAppOptionDefaults(t *testing.T) {
	a := NewApp()
	if a.cfg.Title != "guie" || a.cfg.Width != 800 || a.cfg.Height != 600 || !a.cfg.Resizable {
		t.Errorf("unexpected defaults: %+v", a.cfg)
	}
	// Background defaults to the theme's background when unset.
	if !sameColour(a.cfg.Background, a.theme.Palette.Background) {
		t.Error("Background should default to the theme background")
	}
	if !a.shadows {
		t.Error("shadows should default on")
	}
	if a.clipboard == nil {
		t.Error("a default in-process clipboard should be installed")
	}
}

func TestWithThemeFontAndClipboard(t *testing.T) {
	custom := theme.Default()
	custom.CornerRadius = 99 // a marker we can assert on
	f := DefaultFont(18)
	cb := &memClipboard{}

	a := NewApp(WithTheme(custom), WithFont(f), WithClipboard(cb), WithFontSize(22))
	if a.theme.CornerRadius != 99 {
		t.Errorf("WithTheme not applied: CornerRadius=%v", a.theme.CornerRadius)
	}
	if a.theme.Font != f {
		t.Error("WithFont not applied")
	}
	if a.clipboard != cb {
		t.Error("WithClipboard not applied")
	}
	// WithFontSize sets the size (used only when no explicit face is given, but
	// the field is still recorded).
	if a.theme.FontSize != 22 {
		t.Errorf("WithFontSize: got %v, want 22", a.theme.FontSize)
	}
}

func TestAppDoRunsPending(t *testing.T) {
	a := NewApp()
	ran := 0
	a.Do(func() { ran++ })
	a.Do(func() { ran++ })
	a.Do(nil) // must be a safe no-op

	if ran != 0 {
		t.Fatal("Do should defer work, not run it immediately")
	}
	a.runPending()
	if ran != 2 {
		t.Errorf("runPending should run both queued funcs, ran=%d", ran)
	}

	// The queue is drained: a second runPending does nothing.
	a.runPending()
	if ran != 2 {
		t.Errorf("queue should be drained after runPending, ran=%d", ran)
	}
}

func TestAppQuitTerminatesUpdate(t *testing.T) {
	a := NewApp()
	a.Quit()
	if !a.quit.Load() {
		t.Error("Quit should set the quit flag")
	}
	if err := a.update(render.InputState{}); err != render.ErrTerminated {
		t.Errorf("update after Quit: got %v, want ErrTerminated", err)
	}
}

func TestRuntimeSetters(t *testing.T) {
	a := NewApp()
	a.SetShadows(false)
	if a.shadows {
		t.Error("SetShadows(false) should disable shadows")
	}
	a.SetDebugBounds(true)
	if !a.debug {
		t.Error("SetDebugBounds(true) should enable debug bounds")
	}

	custom := theme.Default()
	custom.CornerRadius = 42
	a.SetTheme(custom)
	if a.theme.CornerRadius != 42 {
		t.Errorf("SetTheme not applied: CornerRadius=%v", a.theme.CornerRadius)
	}
	// SetTheme keeps the existing font when the new theme leaves it nil.
	if a.theme.Font == nil {
		t.Error("SetTheme should retain the existing font when the new theme has none")
	}
}
