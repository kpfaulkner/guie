package guitest_test

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/kpfaulkner/guie/ui"
)

// TestFileDialogWindowsHiddenAttribute drives the dialog over the real
// filesystem: a file with the hidden attribute is hidden, and a dotfile without
// it is shown, which is what Explorer does.
func TestFileDialogWindowsHiddenAttribute(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"plain.txt", ".dotfile", "secret.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	p, err := syscall.UTF16PtrFromString(filepath.Join(dir, "secret.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if err := syscall.SetFileAttributes(p, syscall.FILE_ATTRIBUTE_HIDDEN); err != nil {
		t.Fatalf("cannot set the hidden attribute: %v", err)
	}

	e := openFD(t, nil, ui.FileDialogFS(ui.OSFileSystem()), ui.FileDialogStart(dir))
	e.wantDrawn("plain.txt", ".dotfile")
	e.wantAbsent("secret.txt")

	e.click("Show hidden")
	e.wantDrawn("plain.txt", ".dotfile", "secret.txt")

	e.click("Show hidden")
	e.wantAbsent("secret.txt")
}
