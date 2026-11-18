package ebitenbackend

import "github.com/hajimehoshi/ebiten/v2"

// WindowFocused satisfies render.AttentionController.
func (d *Driver) WindowFocused() bool { return ebiten.IsFocused() }

// RequestAttention satisfies render.AttentionController. Only Windows can do it
// for now, see attention_windows.go.
func (d *Driver) RequestAttention() bool { return requestAttention() }
