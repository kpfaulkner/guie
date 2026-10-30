package ui

import (
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"syscall"
	"testing"
	"time"
)

// winInfo is a fakeInfo carrying Windows file attributes, as os reports them.
type winInfo struct {
	fakeInfo
	attrs uint32
}

func (w winInfo) Sys() any { return &syscall.Win32FileAttributeData{FileAttributes: w.attrs} }

func TestIsHiddenUsesTheAttributeOnWindows(t *testing.T) {
	now := time.Now()
	cases := []struct {
		name string
		info fs.FileInfo
		want bool
	}{
		{"attribute, no dot", winInfo{file("secret.txt", 1, now).(fakeInfo), syscall.FILE_ATTRIBUTE_HIDDEN}, true},
		{"attribute and dot", winInfo{file(".git", 1, now).(fakeInfo), syscall.FILE_ATTRIBUTE_HIDDEN | syscall.FILE_ATTRIBUTE_DIRECTORY}, true},
		{"dot only is shown, as in Explorer", winInfo{file(".gitignore", 1, now).(fakeInfo), syscall.FILE_ATTRIBUTE_ARCHIVE}, false},
		{"plain", winInfo{file("a.txt", 1, now).(fakeInfo), syscall.FILE_ATTRIBUTE_NORMAL}, false},
		{"no attributes known: dot rule", file(".env", 1, now), true},
		{"no attributes known, no dot", file("a.txt", 1, now), false},
	}
	for _, c := range cases {
		if got := isHidden(c.info); got != c.want {
			t.Errorf("%s: isHidden = %v, want %v", c.name, got, c.want)
		}
	}
}

// setHidden sets the Windows hidden attribute on path.
func setHidden(t *testing.T, path string) {
	t.Helper()
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	attrs, err := syscall.GetFileAttributes(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := syscall.SetFileAttributes(p, attrs|syscall.FILE_ATTRIBUTE_HIDDEN); err != nil {
		t.Fatalf("cannot set the hidden attribute: %v", err)
	}
}

func TestOSFileSystemHiddenAttributeEndToEnd(t *testing.T) {
	dir := t.TempDir()
	write := func(name string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	write("plain.txt")
	write(".dotfile")
	setHidden(t, write("secret.txt"))
	if err := os.Mkdir(filepath.Join(dir, "hiddendir"), 0o755); err != nil {
		t.Fatal(err)
	}
	setHidden(t, filepath.Join(dir, "hiddendir"))

	entries, err := OSFileSystem().ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	names := func(es []fs.FileInfo) []string {
		out := namesOf(es)
		sortStrings(out)
		return out
	}
	if got, want := names(visibleEntries(entries, FileFilter{}, false, false)), []string{".dotfile", "plain.txt"}; !reflect.DeepEqual(got, want) {
		t.Errorf("hidden off: got %v, want %v", got, want)
	}
	if got, want := names(visibleEntries(entries, FileFilter{}, true, false)), []string{".dotfile", "hiddendir", "plain.txt", "secret.txt"}; !reflect.DeepEqual(got, want) {
		t.Errorf("hidden on: got %v, want %v", got, want)
	}
	// A folder dialog lists directories only, and honours the attribute there too.
	if got := names(visibleEntries(entries, FileFilter{}, false, true)); len(got) != 0 {
		t.Errorf("dirs only, hidden off: got %v, want none", got)
	}
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
