package ui

import (
	"syscall"
)

var procGetLogicalDrives = syscall.NewLazyDLL("kernel32.dll").NewProc("GetLogicalDrives")

// osRoots lists the drive letters Windows reports as present. It reads the
// GetLogicalDrives bitmask instead of probing A:..Z: with Stat, which stalls on
// empty removable drives and unreachable network mappings.
func osRoots() []FileRoot {
	mask, _, _ := procGetLogicalDrives.Call()
	var roots []FileRoot
	for i := 0; i < 26; i++ {
		if mask&(1<<uint(i)) == 0 {
			continue
		}
		letter := string(rune('A' + i))
		roots = append(roots, FileRoot{Name: letter + ":", Path: letter + `:\`})
	}
	if len(roots) == 0 {
		roots = []FileRoot{{Name: `C:`, Path: `C:\`}}
	}
	return roots
}
