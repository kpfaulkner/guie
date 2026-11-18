package ebitenbackend

import "image"

// largestIcon is the biggest of the icon images, the one to scale down from
// where the system wants a single image (the macOS Dock). Nil for none.
func largestIcon(icons []image.Image) image.Image {
	var best image.Image
	for _, img := range icons {
		if img == nil {
			continue
		}
		if best == nil || img.Bounds().Dx()*img.Bounds().Dy() > best.Bounds().Dx()*best.Bounds().Dy() {
			best = img
		}
	}
	return best
}
