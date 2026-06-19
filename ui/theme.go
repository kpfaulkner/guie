package ui

import "image/color"

// Default colors used by widgets when their own color fields are left unset.
// They are exported so applications can tweak the look globally before
// constructing widgets.
var (
	DefaultBackground       = color.RGBA{R: 0x1e, G: 0x1e, B: 0x28, A: 0xff}
	DefaultWindowBackground = color.RGBA{R: 0x2b, G: 0x2b, B: 0x3a, A: 0xff}
	DefaultTitleBar         = color.RGBA{R: 0x3d, G: 0x5a, B: 0x80, A: 0xff}
	DefaultBorder           = color.RGBA{R: 0x10, G: 0x10, B: 0x18, A: 0xff}
	DefaultText             = color.RGBA{R: 0xf0, G: 0xf0, B: 0xf0, A: 0xff}

	DefaultButton        = color.RGBA{R: 0x4a, G: 0x6f, B: 0xa5, A: 0xff}
	DefaultButtonHover   = color.RGBA{R: 0x5d, G: 0x86, B: 0xc4, A: 0xff}
	DefaultButtonPressed = color.RGBA{R: 0x35, G: 0x52, B: 0x7a, A: 0xff}
)
