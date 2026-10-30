package ui

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestValidFolderName(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"reports", "reports", false},
		{"  reports  ", "reports", false},
		{"My Photos 2026", "My Photos 2026", false},
		{".config", ".config", false},
		{"a.b", "a.b", false},
		{"", "", true},
		{"   ", "", true},
		{"a/b", "", true},
		{`a\b`, "", true},
		{"/abs", "", true},
		{".", "", true},
		{"..", "", true},
		{"a\x00b", "", true},
	}
	for _, c := range cases {
		got, err := validFolderName(c.in)
		if (err != nil) != c.wantErr || got != c.want {
			t.Errorf("validFolderName(%q) = (%q, %v), want (%q, err=%v)", c.in, got, err, c.want, c.wantErr)
		}
	}
}

func TestUniqueName(t *testing.T) {
	taken := func(names ...string) func(string) bool {
		return func(n string) bool {
			for _, t := range names {
				if t == n {
					return true
				}
			}
			return false
		}
	}
	cases := []struct {
		taken []string
		want  string
	}{
		{nil, "New folder"},
		{[]string{"other"}, "New folder"},
		{[]string{"New folder"}, "New folder (2)"},
		{[]string{"New folder", "New folder (2)"}, "New folder (3)"},
		{[]string{"New folder", "New folder (3)"}, "New folder (2)"}, // fills the first gap
	}
	for _, c := range cases {
		if got := uniqueName("New folder", taken(c.taken...)); got != c.want {
			t.Errorf("taken %v: got %q, want %q", c.taken, got, c.want)
		}
	}
}

func TestOSFileSystemIsADirMaker(t *testing.T) {
	if _, ok := OSFileSystem().(DirMaker); !ok {
		t.Error("OSFileSystem should create directories")
	}
	if _, ok := FSFileSystem(fstest.MapFS{}).(DirMaker); ok {
		t.Error("FSFileSystem wraps a read-only fs.FS and must not claim to create directories")
	}
}

func TestOSFileSystemMkdir(t *testing.T) {
	f := OSFileSystem().(DirMaker)
	dir := t.TempDir()

	p := filepath.Join(dir, "made")
	if err := f.Mkdir(p); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(p); err != nil || !info.IsDir() {
		t.Errorf("directory not created: %v, %v", info, err)
	}

	if err := f.Mkdir(p); !errors.Is(err, fs.ErrExist) {
		t.Errorf("second Mkdir = %v, want an fs.ErrExist error", err)
	}
	if err := f.Mkdir(filepath.Join(dir, "no", "parent")); err == nil {
		t.Error("Mkdir must not create missing parents")
	}
}
