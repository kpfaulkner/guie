package ui

import (
	"io/fs"
	"syscall"
)

// attributeHidden reports whether info carries the Windows "hidden" attribute.
// known is false when info did not come from the Windows filesystem (an
// fstest.MapFS, an embed.FS), so the caller falls back to the dot convention.
func attributeHidden(info fs.FileInfo) (hidden, known bool) {
	a, ok := info.Sys().(*syscall.Win32FileAttributeData)
	if !ok || a == nil {
		return false, false
	}
	return a.FileAttributes&syscall.FILE_ATTRIBUTE_HIDDEN != 0, true
}
