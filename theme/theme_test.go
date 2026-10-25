package theme

import (
	"image/color"
	"math"
	"testing"
)

func TestDefaultPaletteIsPopulated(t *testing.T) {
	p := DefaultPalette()
	colours := map[string]interface{ RGBA() (r, g, b, a uint32) }{
		"Background": p.Background, "Surface": p.Surface, "Primary": p.Primary,
		"OnPrimary": p.OnPrimary, "Text": p.Text, "TextMuted": p.TextMuted,
		"Border": p.Border, "Accent": p.Accent, "Disabled": p.Disabled,
	}
	for name, c := range colours {
		if c == nil {
			t.Errorf("palette colour %s is nil", name)
			continue
		}
		if _, _, _, a := c.RGBA(); a == 0 {
			t.Errorf("palette colour %s is fully transparent", name)
		}
	}
}

func TestDefaultTheme(t *testing.T) {
	th := Default()
	if th.FontSize != 14 {
		t.Errorf("default font size: got %v want 14", th.FontSize)
	}
	if th.Font != nil {
		t.Error("default theme font should be nil (filled in by the App from the backend)")
	}
	if th.Palette != DefaultPalette() {
		t.Error("default theme palette should equal DefaultPalette()")
	}
}

func TestLightPaletteIsPopulated(t *testing.T) {
	p := LightPalette()
	for name, c := range map[string]interface{ RGBA() (r, g, b, a uint32) }{
		"Background": p.Background, "Surface": p.Surface, "Primary": p.Primary,
		"OnPrimary": p.OnPrimary, "Text": p.Text, "TextMuted": p.TextMuted,
		"Border": p.Border, "Accent": p.Accent, "Disabled": p.Disabled,
	} {
		if c == nil {
			t.Errorf("light palette colour %s is nil", name)
		} else if _, _, _, a := c.RGBA(); a == 0 {
			t.Errorf("light palette colour %s is fully transparent", name)
		}
	}
}

func TestIsDark(t *testing.T) {
	if !DefaultPalette().IsDark() {
		t.Error("default palette should be dark")
	}
	if LightPalette().IsDark() {
		t.Error("light palette should not be dark")
	}
}

func TestToggledFlipsAndKeepsMetrics(t *testing.T) {
	d := Dark()
	d.FontSize, d.CornerRadius = 18, 12
	l := d.Toggled()
	if l.Palette.IsDark() {
		t.Error("toggling dark should give light")
	}
	if l.FontSize != 18 || l.CornerRadius != 12 {
		t.Errorf("toggle lost metrics: size %v radius %v", l.FontSize, l.CornerRadius)
	}
	if !l.Toggled().Palette.IsDark() {
		t.Error("toggling light should give dark")
	}
}

// --- the model beyond the nine colours ---

// lin converts an 8-bit-per-channel (16-bit here) sRGB channel to linear light.
func lin(v uint32) float64 {
	c := float64(v) / 0xffff
	if c <= 0.03928 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}

func luminance(c color.Color) float64 {
	r, g, b, _ := c.RGBA()
	return 0.2126*lin(r) + 0.7152*lin(g) + 0.0722*lin(b)
}

// contrast is the WCAG contrast ratio of two opaque colours, 1 to 21.
func contrast(a, b color.Color) float64 {
	la, lb := luminance(a), luminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

func TestContrastHelper(t *testing.T) {
	if c := contrast(color.White, color.Black); math.Abs(c-21) > 0.01 {
		t.Errorf("white on black = %v, want 21", c)
	}
	if c := contrast(color.White, color.White); math.Abs(c-1) > 0.001 {
		t.Errorf("white on white = %v, want 1", c)
	}
}

func TestPresetsClearContrastTargets(t *testing.T) {
	presets := []struct {
		name string
		th   Theme
		text float64 // the target for ordinary text
	}{
		{"dark", Dark(), 4.5},
		{"light", Light(), 4.5},
		{"high contrast", HighContrast(), 7},
	}
	for _, p := range presets {
		pal := p.th.Palette
		info := pal.Info
		if info == nil {
			info = pal.Primary // nil Info means Primary
		}
		checks := []struct {
			what string
			fg   color.Color
			bg   color.Color
			min  float64
		}{
			{"Text on Background", pal.Text, pal.Background, p.text},
			{"Text on Surface", pal.Text, pal.Surface, p.text},
			{"OnPrimary on Primary", pal.OnPrimary, pal.Primary, p.text},
			{"TextMuted on Surface", pal.TextMuted, pal.Surface, 3},
			{"Accent on Surface (focus ring)", pal.Accent, pal.Surface, 3},
			{"OnPrimary on Success", pal.OnPrimary, pal.Success, p.text},
			{"OnPrimary on Warning", pal.OnPrimary, pal.Warning, p.text},
			{"OnPrimary on Error", pal.OnPrimary, pal.Error, p.text},
			{"OnPrimary on Info", pal.OnPrimary, info, p.text},
		}
		for _, c := range checks {
			if got := contrast(c.fg, c.bg); got < c.min {
				t.Errorf("%s: %s = %.2f, want at least %.1f", p.name, c.what, got, c.min)
			}
		}
	}
}

func TestZeroValueThemeResolvesToDefault(t *testing.T) {
	// The shape callers wrote before the new fields existed.
	old := Theme{Palette: DefaultPalette(), FontSize: 14, CornerRadius: 5}
	if got, want := old.WithDefaults(), Default(); got != want {
		t.Errorf("old-style theme resolves to\n%+v\nwant\n%+v", got, want)
	}
	// And a bare palette with nothing else set gets every metric.
	bare := Theme{Palette: DefaultPalette()}.WithDefaults()
	if bare.Density != 1 || bare.Spacing != 8 || bare.BorderWidth != 1 || bare.FocusWidth != 2 ||
		bare.Motion.Duration != 0.12 || bare.States != DefaultStates() {
		t.Errorf("bare theme metrics not defaulted: %+v", bare)
	}
}

func TestWithDefaultsKeepsWhatIsSet(t *testing.T) {
	th := Theme{
		Palette: DefaultPalette(), Density: 0.8, Spacing: 12, BorderWidth: 3, FocusWidth: 4,
		States: StateTints{Hover: 0.5},
		Motion: Motion{Duration: 0.3, Reduced: true},
	}
	th.Palette.Success = color.RGBA{R: 1, G: 2, B: 3, A: 255}
	got := th.WithDefaults()
	if got.Density != 0.8 || got.Spacing != 12 || got.BorderWidth != 3 || got.FocusWidth != 4 {
		t.Errorf("set metrics overwritten: %+v", got)
	}
	if got.States.Hover != 0.5 || got.States.Pressed != DefaultStates().Pressed {
		t.Errorf("states = %+v, want Hover kept and the rest defaulted", got.States)
	}
	if got.Motion.Duration != 0.3 || !got.Motion.Reduced {
		t.Errorf("motion = %+v, want kept", got.Motion)
	}
	if c := got.Palette.Success; c != (color.RGBA{R: 1, G: 2, B: 3, A: 255}) {
		t.Errorf("a set status colour was replaced: %v", c)
	}
	if got.Palette.Error == nil || got.Palette.Scrim == nil || got.Palette.Warning == nil {
		t.Error("unset status colours and scrim should be filled")
	}
	if got.Palette.Info != nil {
		t.Error("Info nil means Primary and must be left nil")
	}
	// Idempotent.
	if got != got.WithDefaults() {
		t.Error("WithDefaults is not idempotent")
	}
}

func TestDefaultsMatchWhatTheCodeUsedToHardCode(t *testing.T) {
	th := Default()
	if th.States != (StateTints{Hover: 0.06, Pressed: 0.14, Header: 0.05, Emphasis: 0.09}) {
		t.Errorf("default tints %+v are not today's constants", th.States)
	}
	if th.Density != 1 || th.BorderWidth != 1 || th.FocusWidth != 2 || th.Motion.Duration != 0.12 || th.Motion.Reduced {
		t.Errorf("default metrics changed: %+v", th)
	}
	if th.Palette.Success != color.Color(color.RGBA{R: 0x2E, G: 0x7D, B: 0x32, A: 0xff}) ||
		th.Palette.Error != color.Color(color.RGBA{R: 0xC6, G: 0x28, B: 0x28, A: 0xff}) {
		t.Error("success/error defaults are not the colours toasts have always used")
	}
}

func TestLightAndDarkShareMetrics(t *testing.T) {
	d, l := Dark(), Light()
	d.Palette, l.Palette = Palette{}, Palette{}
	if d != l {
		t.Errorf("dark and light differ beyond their palettes:\n%+v\n%+v", d, l)
	}
	if Light().Palette.Success == nil || Light().Palette.Scrim == nil {
		t.Error("the light theme should have the status colours and scrim filled")
	}
}

func TestToggledKeepsDensityAndMotion(t *testing.T) {
	d := Dark().WithDensity(0.8).WithReducedMotion()
	d.Spacing = 12
	l := d.Toggled()
	if l.Density != 0.8 || l.Spacing != 12 || !l.Motion.Reduced {
		t.Errorf("toggle lost preferences: density %v spacing %v motion %+v", l.Density, l.Spacing, l.Motion)
	}
	// Borders and tints belong to the preset: leaving high contrast sheds them.
	back := HighContrast().Toggled()
	if back.BorderWidth != 1 || back.FocusWidth != 2 || back.States != DefaultStates() {
		t.Errorf("leaving high contrast kept its look: border %v focus %v states %+v", back.BorderWidth, back.FocusWidth, back.States)
	}
	// A theme written before the new fields toggles to a complete one.
	old := Theme{Palette: DefaultPalette(), FontSize: 14, CornerRadius: 5}.Toggled()
	if old.Density != 1 || old.Spacing != 8 || old.Motion.Duration != 0.12 {
		t.Errorf("toggling an old-style theme left zero metrics: %+v", old)
	}
}

func TestHighContrastIsStrongerThanDefault(t *testing.T) {
	hc, d := HighContrast(), Default()
	if hc.BorderWidth <= d.BorderWidth || hc.FocusWidth <= d.FocusWidth {
		t.Errorf("border %v focus %v should exceed the default %v / %v", hc.BorderWidth, hc.FocusWidth, d.BorderWidth, d.FocusWidth)
	}
	if hc.States.Hover <= d.States.Hover || hc.States.Pressed <= d.States.Pressed {
		t.Errorf("state tints %+v should exceed the default %+v", hc.States, d.States)
	}
	if !hc.Motion.Reduced {
		t.Error("high contrast turns the fades off")
	}
	if !hc.Palette.IsDark() {
		t.Error("high contrast is a dark page")
	}
	if hc.Font != nil {
		t.Error("presets leave the font to the App")
	}
}

func TestWithDensityAndReducedMotionReturnCopies(t *testing.T) {
	d := Dark()
	c := d.WithDensity(1.25).WithReducedMotion()
	if d.Density != 1 || d.Motion.Reduced {
		t.Error("the original theme was modified")
	}
	if c.Density != 1.25 || !c.Motion.Reduced {
		t.Errorf("copy = density %v reduced %v", c.Density, c.Motion.Reduced)
	}
	if d.WithDensity(0).Density != 1 || d.WithDensity(-2).Density != 1 {
		t.Error("a zero or negative density must be ignored")
	}
}

// The focus ring is drawn in Accent over the control's own fill. On a filled
// button that needs Accent to stand apart from Primary. High contrast is built to,
// and is held to it. The two ordinary presets are NOT, and this records it rather
// than hiding it: the light theme's Accent *is* Primary (invisible on a button) and
// the dark theme's is 1.8:1. The fix is in how the ring is drawn, not in repainting the
// presets: ui's strokeFocusable now adds a halo in the page colour just inside the
// ring (guitest/theme_step2_test.go checks it is drawn), so the ring is bounded by
// the page on one side and a contrasting line on the other whatever the fill. The
// colour ratio below stays low on purpose; it is the reason the halo exists.
func TestFocusRingStandsApartFromAFilledButton(t *testing.T) {
	if c := contrast(HighContrast().Palette.Accent, HighContrast().Palette.Primary); c < 3 {
		t.Errorf("high contrast: Accent on Primary = %.2f, want at least 3", c)
	}
	known := map[string]Theme{"dark": Dark(), "light": Light()}
	for name, th := range known {
		if c := contrast(th.Palette.Accent, th.Palette.Primary); c >= 3 {
			t.Errorf("%s now clears 3:1 (%.2f): remove it from the known gaps", name, c)
		} else {
			t.Logf("known gap: %s focus ring on a filled button is %.2f:1", name, c)
		}
	}
}
