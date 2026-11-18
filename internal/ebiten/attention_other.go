//go:build !windows

package ebitenbackend

// requestAttention has no implementation off Windows yet: macOS needs
// [NSApp requestUserAttention:] and X11 needs _NET_WM_STATE_DEMANDS_ATTENTION,
// and Ebitengine exposes neither.
func requestAttention() bool { return false }
