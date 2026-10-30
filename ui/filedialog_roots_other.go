//go:build !windows

package ui

// osRoots returns the single filesystem root. Mount points live under it and
// are reached by navigating, so there is nothing to enumerate.
func osRoots() []FileRoot { return []FileRoot{{Name: "/", Path: "/"}} }
