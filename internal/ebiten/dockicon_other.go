//go:build !darwin

package ebitenbackend

import "image"

// setDockIcon does nothing outside macOS: there the window icon
// (ebiten.SetWindowIcon) is what the taskbar shows.
func setDockIcon([]image.Image) {}
