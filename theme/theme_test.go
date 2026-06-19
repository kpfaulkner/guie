package theme

import "testing"

func TestDefaultPaletteIsPopulated(t *testing.T) {
	p := DefaultPalette()
	colors := map[string]interface{ RGBA() (r, g, b, a uint32) }{
		"Background": p.Background, "Surface": p.Surface, "Primary": p.Primary,
		"OnPrimary": p.OnPrimary, "Text": p.Text, "TextMuted": p.TextMuted,
		"Border": p.Border, "Accent": p.Accent, "Disabled": p.Disabled,
	}
	for name, c := range colors {
		if c == nil {
			t.Errorf("palette color %s is nil", name)
			continue
		}
		if _, _, _, a := c.RGBA(); a == 0 {
			t.Errorf("palette color %s is fully transparent", name)
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
