// Package theme defines the colour palette and font defaults used by widgets.
// Widgets resolve colours through these palette roles and may override
// individual values per widget.
package theme

import (
	"image/color"

	"github.com/kpfaulkner/guie/render"
)

// Palette is the set of named colours a theme provides. Widgets fall back to
// these when their own colour fields are unset.
type Palette struct {
	Background color.Color // window/app backdrop
	Surface    color.Color // panels, cards, containers
	Primary    color.Color // primary action colour (e.g. buttons)
	OnPrimary  color.Color // text/icons drawn on Primary
	Text       color.Color // default text
	TextMuted  color.Color // secondary/disabled-ish text
	Border     color.Color // outlines and separators
	Accent     color.Color // highlights, hover, focus
	Disabled   color.Color // disabled widget fill
}

// Theme bundles the colour palette with the default font/size and the default
// corner radius used by controls.
type Theme struct {
	Palette      Palette
	Font         render.FontFace // default text face; may be nil until set
	FontSize     float64
	CornerRadius float64 // rounding of buttons, fields, etc.
}

// DefaultPalette returns the built-in dark palette. Background and Surface are
// distinct elevations; Border is a soft, low-contrast separator (not a harsh
// outline); Accent is reserved for focus and selection.
func DefaultPalette() Palette {
	return Palette{
		Background: color.RGBA{R: 0x1a, G: 0x1a, B: 0x22, A: 0xff},
		Surface:    color.RGBA{R: 0x26, G: 0x26, B: 0x32, A: 0xff},
		Primary:    color.RGBA{R: 0x4a, G: 0x6f, B: 0xa5, A: 0xff},
		OnPrimary:  color.RGBA{R: 0xf4, G: 0xf6, B: 0xfb, A: 0xff},
		Text:       color.RGBA{R: 0xea, G: 0xea, B: 0xf2, A: 0xff},
		TextMuted:  color.RGBA{R: 0x9a, G: 0x9a, B: 0xb0, A: 0xff},
		Border:     color.RGBA{R: 0x3a, G: 0x3a, B: 0x48, A: 0xff},
		Accent:     color.RGBA{R: 0x6f, G: 0x9a, B: 0xe0, A: 0xff},
		Disabled:   color.RGBA{R: 0x44, G: 0x44, B: 0x52, A: 0xff},
	}
}

// Default returns the built-in theme: the dark palette at the default font size
// with softly rounded controls. The Font is left nil here so this package stays
// backend-independent; the App fills in a default font from the backend if one
// is not supplied.
func Default() Theme {
	return Theme{
		Palette:      DefaultPalette(),
		FontSize:     14,
		CornerRadius: 5,
	}
}

// LightPalette returns the built-in light palette: a near-white backdrop with
// white surfaces, dark text and the same blue family as the dark theme. Border
// and Disabled are soft greys; Accent matches Primary so focus and selection
// stay clearly visible on light surfaces.
func LightPalette() Palette {
	return Palette{
		Background: color.RGBA{R: 0xf4, G: 0xf5, B: 0xf8, A: 0xff},
		Surface:    color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff},
		Primary:    color.RGBA{R: 0x3b, G: 0x6e, B: 0xa5, A: 0xff},
		OnPrimary:  color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff},
		Text:       color.RGBA{R: 0x1b, G: 0x1d, B: 0x22, A: 0xff},
		TextMuted:  color.RGBA{R: 0x6b, G: 0x6f, B: 0x7a, A: 0xff},
		Border:     color.RGBA{R: 0xd0, G: 0xd3, B: 0xda, A: 0xff},
		Accent:     color.RGBA{R: 0x3b, G: 0x6e, B: 0xa5, A: 0xff},
		Disabled:   color.RGBA{R: 0xc8, G: 0xcc, B: 0xd4, A: 0xff},
	}
}

// Dark returns the built-in dark theme. It is the same as Default and exists so
// Dark and Light read as a pair.
func Dark() Theme { return Default() }

// Light returns the built-in light theme: the light palette with the same font
// size and corner radius as the dark theme.
func Light() Theme {
	t := Default()
	t.Palette = LightPalette()
	return t
}

// IsDark reports whether the palette is a dark one, judged by the luminance of
// its Background. Useful for a toggle that flips to the opposite theme.
func (p Palette) IsDark() bool {
	r, g, b, _ := p.Background.RGBA()
	// Rec. 601 luma on 16-bit channels; below mid-grey counts as dark.
	return (299*r+587*g+114*b)/1000 < 0x8000
}

// Toggled returns the light theme if t is dark, otherwise the dark theme. The
// font, font size and corner radius of t are carried over so a toggle does not
// reset them.
func (t Theme) Toggled() Theme {
	n := Dark()
	if t.Palette.IsDark() {
		n = Light()
	}
	n.Font, n.FontSize, n.CornerRadius = t.Font, t.FontSize, t.CornerRadius
	return n
}
