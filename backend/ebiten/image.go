package ebitenbackend

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kpfaulkner/uiframework/geom"
	"github.com/kpfaulkner/uiframework/render"
)

// imageHandle is the backend's concrete render.Image, wrapping an *ebiten.Image.
type imageHandle struct {
	img *ebiten.Image
}

// Size returns the pixel dimensions of the image.
func (h *imageHandle) Size() geom.Size {
	b := h.img.Bounds()
	return geom.Size{W: float64(b.Dx()), H: float64(b.Dy())}
}

// NewImage wraps an existing *ebiten.Image as a render.Image.
func NewImage(img *ebiten.Image) render.Image {
	return &imageHandle{img: img}
}
