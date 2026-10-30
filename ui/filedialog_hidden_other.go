//go:build !windows

package ui

import "io/fs"

// attributeHidden has no attribute to read away from Windows, where a file is
// hidden by its name alone.
func attributeHidden(fs.FileInfo) (hidden, known bool) { return false, false }
