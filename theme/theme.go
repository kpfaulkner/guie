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

	// Status colours, for toasts and anything else that reports an outcome. A
	// nil colour means the built-in default (see WithDefaults); text drawn on
	// them is OnPrimary. Info is the exception: nil means "use Primary", which is
	// what info toasts have always done.
	Success color.Color
	Warning color.Color
	Error   color.Color
	Info    color.Color

	// Scrim dims the page behind a modal dialog. Nil means the built-in default, a
	// black at about 63% opacity.
	Scrim color.Color
}

// Theme bundles the colour palette with the default font/size and the default
// corner radius used by controls.
type Theme struct {
	Palette      Palette
	Font         render.FontFace // default text face; may be nil until set
	FontSize     float64
	CornerRadius float64 // rounding of buttons, fields, etc.

	// The fields below were added after the ones above. Every one of them treats
	// its zero value as "the default", so a Theme written before they existed
	// behaves exactly as it did; WithDefaults fills them in.

	// States is how far hover, pressed and similar states shift a widget's own
	// colour.
	States StateTints
	// Density multiplies the padding and gaps inside every widget: 1 is the default,
	// below 1 is more compact, above 1 more spacious. It does not scale the font.
	Density float64
	// Spacing is the gap that layouts built with ui.ThemeGap resolve to, before
	// Density is applied.
	Spacing float64
	// BorderWidth is the thickness of control outlines. FocusWidth is the keyboard
	// focus ring. To have no outline, make Palette.Border transparent.
	BorderWidth float64
	FocusWidth  float64
	// Motion controls the hover, press and focus fades and the tab underline slide.
	Motion Motion
	// NoShadows turns off the soft drop shadows under popups, tooltips and toasts.
	// (App.SetShadows also does; either one off means off.)
	NoShadows bool
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
		Success:    defaultSuccess,
		Warning:    defaultWarning,
		Error:      defaultError,
		Scrim:      defaultScrim,
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
	}.WithDefaults()
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
		Success:    defaultSuccess,
		Warning:    defaultWarning,
		Error:      defaultError,
		Scrim:      defaultScrim,
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
	// Density, spacing and motion are the user's preferences, not part of a look, so
	// they survive the toggle. Borders, focus width and state tints belong to the
	// preset and come from the new one.
	n.Density, n.Spacing, n.Motion = t.Density, t.Spacing, t.Motion
	if t.Density == 0 {
		n.Density = defaultDensity
	}
	if t.Spacing == 0 {
		n.Spacing = defaultSpacing
	}
	if t.Motion.Duration == 0 {
		n.Motion.Duration = defaultFadeSeconds
	}
	return n
}

// StateTints says how far a widget's state shifts its own base colour, as the
// fraction of the way toward white (on a dark base) or black (on a light one). It
// is an amount, not a colour, because one hover colour cannot suit a filled
// button, a list row and a table header, which all start from different bases.
// Zero fields take the defaults shown.
type StateTints struct {
	Hover    float64 // 0.06: a hovered row, cell, tab or menu item
	Pressed  float64 // 0.14: a pressed flat button
	Header   float64 // 0.05: the table header band (striping uses half)
	Emphasis float64 // 0.09: a hovered header cell, the tooltip body
}

// Motion controls the short fades that follow hover, press and focus, and the
// tab underline's slide.
type Motion struct {
	// Duration is the length of a fade in seconds; zero means 0.12.
	Duration float64
	// Reduced makes widgets jump straight to their end state, for people who are
	// bothered by movement. Motion the application starts itself (App.Animate, a
	// toast sliding in, a spinner) is its own and is not affected.
	Reduced bool
}

// Built-in defaults for the fields a Theme may leave zero.
const (
	defaultDensity     = 1.0
	defaultSpacing     = 8.0
	defaultBorderWidth = 1.0
	defaultFocusWidth  = 2.0
	defaultFadeSeconds = 0.12
)

var (
	// The status colours have not changed from the ones toasts have always used,
	// except Warning, which is a little darker so white text on it reads clearly
	// (the old amber gave about 3.6:1, under the 4.5:1 for text).
	defaultSuccess = color.RGBA{R: 0x2E, G: 0x7D, B: 0x32, A: 0xff}
	defaultWarning = color.RGBA{R: 0x94, G: 0x63, B: 0x00, A: 0xff}
	defaultError   = color.RGBA{R: 0xC6, G: 0x28, B: 0x28, A: 0xff}
	defaultScrim   = color.RGBA{A: 0xA0}
)

// DefaultStates returns the built-in state tints.
func DefaultStates() StateTints {
	return StateTints{Hover: 0.06, Pressed: 0.14, Header: 0.05, Emphasis: 0.09}
}

// WithDefaults returns a copy of t with every unset (zero) optional field filled
// in with its default, so a Theme written before those fields existed, or one that
// only sets a few of them, is complete. Palette.Info is left alone: nil there means
// "use Primary". The font and the nine base palette colours are not touched.
func (t Theme) WithDefaults() Theme {
	s := DefaultStates()
	if t.States.Hover == 0 {
		t.States.Hover = s.Hover
	}
	if t.States.Pressed == 0 {
		t.States.Pressed = s.Pressed
	}
	if t.States.Header == 0 {
		t.States.Header = s.Header
	}
	if t.States.Emphasis == 0 {
		t.States.Emphasis = s.Emphasis
	}
	if t.Density == 0 {
		t.Density = defaultDensity
	}
	if t.Spacing == 0 {
		t.Spacing = defaultSpacing
	}
	if t.BorderWidth == 0 {
		t.BorderWidth = defaultBorderWidth
	}
	if t.FocusWidth == 0 {
		t.FocusWidth = defaultFocusWidth
	}
	if t.Motion.Duration == 0 {
		t.Motion.Duration = defaultFadeSeconds
	}
	if t.Palette.Success == nil {
		t.Palette.Success = defaultSuccess
	}
	if t.Palette.Warning == nil {
		t.Palette.Warning = defaultWarning
	}
	if t.Palette.Error == nil {
		t.Palette.Error = defaultError
	}
	if t.Palette.Scrim == nil {
		t.Palette.Scrim = defaultScrim
	}
	return t
}

// WithDensity returns a copy of t with the given Density: about 0.8 is compact,
// 1.25 spacious. Values at or below zero are ignored.
func (t Theme) WithDensity(d float64) Theme {
	if d > 0 {
		t.Density = d
	}
	return t
}

// WithReducedMotion returns a copy of t with the fades turned off.
func (t Theme) WithReducedMotion() Theme {
	t.Motion.Reduced = true
	return t
}

// HighContrastPalette returns the palette used by HighContrast: black, white and
// two strongly different accents, every pair chosen to clear a 7:1 contrast ratio.
func HighContrastPalette() Palette {
	return Palette{
		Background: color.RGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xff},
		Surface:    color.RGBA{R: 0x0b, G: 0x0b, B: 0x0b, A: 0xff},
		Primary:    color.RGBA{R: 0x00, G: 0x3f, B: 0xa3, A: 0xff},
		OnPrimary:  color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff},
		Text:       color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff},
		TextMuted:  color.RGBA{R: 0xd0, G: 0xd0, B: 0xd0, A: 0xff},
		Border:     color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff},
		Accent:     color.RGBA{R: 0xff, G: 0xff, B: 0x00, A: 0xff},
		Disabled:   color.RGBA{R: 0x4d, G: 0x4d, B: 0x4d, A: 0xff},
		// Status fills dark enough for white text at 7:1.
		Success: color.RGBA{R: 0x00, G: 0x5a, B: 0x1e, A: 0xff},
		Warning: color.RGBA{R: 0x7a, G: 0x4f, B: 0x00, A: 0xff},
		Error:   color.RGBA{R: 0xa8, G: 0x00, B: 0x1c, A: 0xff},
		// A heavier dim behind dialogs, so the page really recedes.
		Scrim: color.RGBA{A: 0xd0},
	}
}

// HighContrast returns a theme built to be unmistakable rather than pretty: a
// black page, white text and outlines, a yellow focus ring and selection, 2 px
// borders and a 3 px focus ring, bigger state shifts, and no fades.
func HighContrast() Theme {
	t := Default()
	t.Palette = HighContrastPalette()
	t.States = StateTints{Hover: 0.18, Pressed: 0.30, Header: 0.14, Emphasis: 0.24}
	t.BorderWidth = 2
	t.FocusWidth = 3
	t.NoShadows = true
	t.Motion.Reduced = true
	return t.WithDefaults()
}
