package ebitenbackend

import (
	"bytes"
	"image"
	"image/color"
	_ "image/gif"  // register GIF decoder
	_ "image/jpeg" // register JPEG decoder
	_ "image/png"  // register PNG decoder

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
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

func (h *imageHandle) ebitenImage() *ebiten.Image { return h.img }

// NewImage wraps an existing *ebiten.Image as a render.Image.
func NewImage(img *ebiten.Image) render.Image {
	return &imageHandle{img: img}
}

// renderTarget is the backend's concrete render.RenderTarget: an offscreen
// *ebiten.Image plus a persistent canvas bound to it at 1:1 scale.
type renderTarget struct {
	img *ebiten.Image
	cv  *canvas
}

// NewRenderTarget allocates an offscreen drawable surface of the given pixel
// size. It returns nil for non-positive dimensions.
func NewRenderTarget(width, height int) render.RenderTarget {
	if width <= 0 || height <= 0 {
		return nil
	}
	img := ebiten.NewImage(width, height)
	return &renderTarget{img: img, cv: newCanvas()}
}

func (t *renderTarget) Size() geom.Size {
	b := t.img.Bounds()
	return geom.Size{W: float64(b.Dx()), H: float64(b.Dy())}
}

func (t *renderTarget) ebitenImage() *ebiten.Image { return t.img }

// Canvas rebinds the target's canvas for drawing (resetting the clip stack to
// the full surface) and returns it. Existing pixels are preserved.
func (t *renderTarget) Canvas() render.Canvas {
	t.cv.reset(t.img, 1)
	return t.cv
}

func (t *renderTarget) Clear(c color.Color) { t.img.Fill(c) }

func (t *renderTarget) Dispose() { t.img.Deallocate() }

// LoadImageBytes decodes PNG/JPEG/GIF data into a render.Image.
func LoadImageBytes(data []byte) (render.Image, error) {
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	return &imageHandle{img: ebiten.NewImageFromImage(src)}, nil
}
