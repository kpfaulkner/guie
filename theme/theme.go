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
