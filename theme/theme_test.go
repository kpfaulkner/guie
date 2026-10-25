package theme

import "testing"

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
