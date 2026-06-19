package ui

import (
	"os"

	"github.com/kpfaulkner/guie/geom"
	ebitenbackend "github.com/kpfaulkner/guie/internal/ebiten"
	"github.com/kpfaulkner/guie/render"
)

// LoadImageBytes decodes image data (PNG, JPEG or GIF) into a render.Image that
// can be drawn by widgets or shown with an Image widget.
func LoadImageBytes(data []byte) (render.Image, error) {
	return ebitenbackend.LoadImageBytes(data)
}

// LoadImage reads an image file (PNG, JPEG or GIF) and decodes it.
func LoadImage(path string) (render.Image, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return LoadImageBytes(data)
}

// ImageFit controls how an Image widget scales its picture within its bounds.
type ImageFit int

const (
	// FitContain scales the image to fit inside the bounds, preserving aspect
	// ratio, and centers it. This is the default.
	FitContain ImageFit = iota
	// FitStretch fills the bounds exactly, ignoring aspect ratio.
	FitStretch
	// FitNone draws the image at its native size, centered (and clipped to the
	// bounds if larger).
	FitNone
)

// Image is a widget that displays a render.Image, scaled per its ImageFit.
type Image struct {
	BaseWidget
	img render.Image
	fit ImageFit
}

// NewImage returns an Image widget showing img (which may be nil), using
// FitContain by default.
func NewImage(img render.Image) *Image {
	return &Image{BaseWidget: NewBase(), img: img, fit: FitContain}
}

// SetImage replaces the displayed image and requests a re-layout.
func (i *Image) SetImage(img render.Image) {
	i.img = img
	i.Invalidate()
}

// SetFit sets how the image is scaled within the widget's bounds.
func (i *Image) SetFit(f ImageFit) {
	i.fit = f
	i.Invalidate()
}

// MinSize returns the image's native size (zero if there is no image).
func (i *Image) MinSize() geom.Size {
	if i.img == nil {
		return geom.Size{}
	}
	return i.img.Size()
}

// Draw renders the image scaled per the fit mode, clipped to the widget bounds.
func (i *Image) Draw(c render.Canvas) {
	if i.img == nil {
		return
	}
	c.PushClip(i.Bounds())
	c.DrawImage(i.img, fitRect(i.fit, i.Bounds(), i.img.Size()))
	c.PopClip()
}

// fitRect computes the destination rectangle for an image of size img drawn
// within bounds according to fit.
func fitRect(fit ImageFit, bounds geom.Rect, img geom.Size) geom.Rect {
	switch fit {
	case FitStretch:
		return bounds
	case FitNone:
		return centered(bounds, img.W, img.H)
	default: // FitContain
		if img.W <= 0 || img.H <= 0 {
			return bounds
		}
		scale := bounds.W / img.W
		if s := bounds.H / img.H; s < scale {
			scale = s
		}
		return centered(bounds, img.W*scale, img.H*scale)
	}
}

// centered returns a w×h rectangle centered within bounds.
func centered(bounds geom.Rect, w, h float64) geom.Rect {
	return geom.Rect{
		X: bounds.X + (bounds.W-w)/2,
		Y: bounds.Y + (bounds.H-h)/2,
		W: w,
		H: h,
	}
}
