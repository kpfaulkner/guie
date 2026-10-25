package ui

import (
	"testing"

	"github.com/kpfaulkner/guie/theme"
)

// A theme written before the optional fields existed must still produce a complete
// theme in the App, and look the same as the default.
func TestAppFillsInOptionalThemeFields(t *testing.T) {
	old := theme.Theme{Palette: theme.DefaultPalette(), FontSize: 14, CornerRadius: 5}
	app := NewApp(WithTheme(old))
	got := app.Theme()
	got.Font = nil // the App supplies the font
	if want := theme.Default(); got != want {
		t.Errorf("app theme from an old-style literal:\n%+v\nwant\n%+v", got, want)
	}
}

func TestSetThemeFillsInAndKeepsFont(t *testing.T) {
	app := NewApp()
	font := app.Theme().Font

	app.SetTheme(theme.Theme{Palette: theme.LightPalette()}) // bare: no size, font or metrics
	got := app.Theme()
	if got.Font != font || got.FontSize == 0 {
		t.Errorf("font/size not kept: font changed=%v size=%v", got.Font != font, got.FontSize)
	}
	if got.Density != 1 || got.BorderWidth != 1 || got.FocusWidth != 2 || got.Motion.Duration != 0.12 {
		t.Errorf("metrics not defaulted: %+v", got)
	}
	if got.Palette.Success == nil || got.Palette.Scrim == nil {
		t.Error("status colours and scrim not defaulted")
	}
}

func TestSetThemeIsTheThemeYouPass(t *testing.T) {
	app := NewApp()
	app.SetTheme(theme.Dark().WithDensity(0.8))
	app.SetTheme(theme.Light()) // density left at 1 by the preset: not carried over from before
	if d := app.Theme().Density; d != 1 {
		t.Errorf("density = %v after SetTheme(Light()), want the new theme's 1", d)
	}
}

func TestHighContrastRunsThroughTheApp(t *testing.T) {
	app := NewApp(WithTheme(theme.HighContrast()))
	got := app.Theme()
	if got.BorderWidth != 2 || got.FocusWidth != 3 || !got.Motion.Reduced {
		t.Errorf("high contrast metrics lost: %+v", got)
	}
	if got.Palette != theme.HighContrastPalette() {
		t.Error("high contrast palette was altered on the way in")
	}
}
