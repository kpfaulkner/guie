package ui

import (
	ebitenbackend "github.com/kpfaulkner/guie/internal/ebiten"
	"github.com/kpfaulkner/guie/render"
)

// DefaultFont returns the bundled default font at the given point size. Use it
// to change text size at runtime, e.g. app.SetFont(ui.DefaultFont(18)).
func DefaultFont(size float64) render.FontFace {
	return ebitenbackend.DefaultFont(size)
}

// LoadFontBytes builds a font face from in-memory TrueType/OpenType data at the
// given point size.
func LoadFontBytes(ttf []byte, size float64) (render.FontFace, error) {
	return ebitenbackend.NewFontFace(ttf, size)
}

// LoadFont reads a TrueType/OpenType font file and builds a face at the given
// point size. Each file is read and parsed once and kept: later faces of the
// same file, at any size, share it, so a program that loads a large font for
// several sizes or windows reads it once. A file changed on disk after its
// first load is not seen until the program restarts.
func LoadFont(path string, size float64) (render.FontFace, error) {
	return ebitenbackend.LoadFontFile(path, size)
}
