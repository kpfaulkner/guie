// Package theme defines the color palette and font defaults used by widgets.
// Widgets resolve colors through these palette roles and may override
// individual values per widget.
package theme

import (
	"image/color"

	"github.com/kpfaulkner/uiframework/render"
)

// Palette is the set of named colors a theme provides. Widgets fall back to
// these when their own color fields are unset.
type Palette struct {
	Background color.Color // window/app backdrop
	Surface    color.Color // panels, cards, containers
	Primary    color.Color // primary action color (e.g. buttons)
	OnPrimary  color.Color // text/icons drawn on Primary
	Text       color.Color // default text
	TextMuted  color.Color // secondary/disabled-ish text
	Border     color.Color // outlines and separators
	Accent     color.Color // highlights, hover, focus
	Disabled   color.Color // disabled widget fill
}

// Theme bundles the color palette with the default font and its size.
type Theme struct {
	Palette  Palette
	Font     render.FontFace // default text face; may be nil until set
	FontSize float64
}

// DefaultPalette returns the built-in dark palette.
func DefaultPalette() Palette {
	return Palette{
		Background: color.RGBA{R: 0x1e, G: 0x1e, B: 0x28, A: 0xff},
		Surface:    color.RGBA{R: 0x2b, G: 0x2b, B: 0x3a, A: 0xff},
		Primary:    color.RGBA{R: 0x4a, G: 0x6f, B: 0xa5, A: 0xff},
		OnPrimary:  color.RGBA{R: 0xf0, G: 0xf0, B: 0xf0, A: 0xff},
		Text:       color.RGBA{R: 0xf0, G: 0xf0, B: 0xf0, A: 0xff},
		TextMuted:  color.RGBA{R: 0x9a, G: 0x9a, B: 0xb0, A: 0xff},
		Border:     color.RGBA{R: 0x10, G: 0x10, B: 0x18, A: 0xff},
		Accent:     color.RGBA{R: 0x5d, G: 0x86, B: 0xc4, A: 0xff},
		Disabled:   color.RGBA{R: 0x55, G: 0x55, B: 0x66, A: 0xff},
	}
}

// Default returns the built-in theme: the dark palette at the default font
// size. The Font is left nil here so this package stays backend-independent;
// the App fills in a default font from the backend if one is not supplied.
func Default() Theme {
	return Theme{
		Palette:  DefaultPalette(),
		FontSize: 14,
	}
}
