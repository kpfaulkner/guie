//go:build !windows

package ebitenbackend

// refreshRate is not known off Windows yet: Ebitengine tracks it but does not
// export it, and asking the system needs CoreGraphics on macOS and XRandR on
// X11. 0 lets the framework use its fixed rate.
func refreshRate() int { return 0 }
