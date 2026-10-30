package ui

import (
	"io/fs"
	"reflect"
	"testing"
	"testing/fstest"
	"time"
)

type fakeInfo struct {
	name string
	size int64
	dir  bool
	mod  time.Time
}

func (f fakeInfo) Name() string       { return f.name }
func (f fakeInfo) Size() int64        { return f.size }
func (f fakeInfo) Mode() fs.FileMode  { return 0o644 }
func (f fakeInfo) ModTime() time.Time { return f.mod }
func (f fakeInfo) IsDir() bool        { return f.dir }
func (f fakeInfo) Sys() any           { return nil }

func file(name string, size int64, mod time.Time) fs.FileInfo {
	return fakeInfo{name: name, size: size, mod: mod}
}

func dir(name string, mod time.Time) fs.FileInfo { return fakeInfo{name: name, dir: true, mod: mod} }

func namesOf(es []fs.FileInfo) []string {
	out := make([]string, len(es))
	for i, e := range es {
		out[i] = e.Name()
	}
	return out
}

func TestFileFilterMatches(t *testing.T) {
	cases := []struct {
		f    FileFilter
		name string
		want bool
	}{
		{FileFilter{}, "anything.bin", true},
		{FileFilter{Patterns: []string{"*.png"}}, "a.png", true},
		{FileFilter{Patterns: []string{"*.png"}}, "A.PNG", true},
		{FileFilter{Patterns: []string{"*.PNG"}}, "a.png", true},
		{FileFilter{Patterns: []string{"*.png"}}, "a.jpg", false},
		{FileFilter{Patterns: []string{"*.png", "*.jpg"}}, "a.jpg", true},
		{FileFilter{Patterns: []string{"*.png"}}, "png", false},
		{FileFilter{Patterns: []string{"data?.csv"}}, "data1.csv", true},
		{FileFilter{Patterns: []string{"[unclosed"}}, "x", false}, // bad pattern never matches
	}
	for _, c := range cases {
		if got := c.f.matches(c.name); got != c.want {
			t.Errorf("%v.matches(%q) = %v, want %v", c.f.Patterns, c.name, got, c.want)
		}
	}
}

func TestVisibleEntries(t *testing.T) {
	now := time.Now()
	in := []fs.FileInfo{
		dir("src", now), dir(".git", now),
		file("a.png", 1, now), file("b.txt", 1, now), file(".env", 1, now),
	}
	png := FileFilter{Patterns: []string{"*.png"}}

	cases := []struct {
		name     string
		filter   FileFilter
		hidden   bool
		dirsOnly bool
		want     []string
	}{
		{"all, no hidden", FileFilter{}, false, false, []string{"src", "a.png", "b.txt"}},
		{"all, hidden", FileFilter{}, true, false, []string{"src", ".git", "a.png", "b.txt", ".env"}},
		{"filter keeps dirs", png, false, false, []string{"src", "a.png"}},
		{"dirs only", FileFilter{}, false, true, []string{"src"}},
		{"dirs only, hidden", png, true, true, []string{"src", ".git"}},
	}
	for _, c := range cases {
		got := namesOf(visibleEntries(in, c.filter, c.hidden, c.dirsOnly))
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
	if len(in) != 5 || in[0].Name() != "src" {
		t.Error("visibleEntries must not modify its input")
	}
}

func TestSortEntriesDirsFirstBothDirections(t *testing.T) {
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mk := func() []fs.FileInfo {
		return []fs.FileInfo{
			file("b.txt", 20, t0.Add(2*time.Hour)),
			dir("zdir", t0),
			file("a.txt", 30, t0.Add(1*time.Hour)),
			dir("adir", t0.Add(3*time.Hour)),
			file("C.txt", 10, t0.Add(3*time.Hour)),
		}
	}
	cases := []struct {
		key  fileSortKey
		asc  bool
		want []string
	}{
		{sortByName, true, []string{"adir", "zdir", "a.txt", "b.txt", "C.txt"}},
		{sortByName, false, []string{"zdir", "adir", "C.txt", "b.txt", "a.txt"}},
		{sortBySize, true, []string{"adir", "zdir", "C.txt", "b.txt", "a.txt"}},
		{sortBySize, false, []string{"adir", "zdir", "a.txt", "b.txt", "C.txt"}},
		{sortByModified, true, []string{"zdir", "adir", "a.txt", "b.txt", "C.txt"}},
		{sortByModified, false, []string{"adir", "zdir", "C.txt", "b.txt", "a.txt"}},
	}
	for _, c := range cases {
		es := mk()
		sortEntries(es, c.key, c.asc)
		if got := namesOf(es); !reflect.DeepEqual(got, c.want) {
			t.Errorf("key=%d asc=%v: got %v, want %v", c.key, c.asc, got, c.want)
		}
	}
}

func TestSortEntriesNameTiesAreTotal(t *testing.T) {
	now := time.Now()
	for _, perm := range [][]fs.FileInfo{
		{file("a", 1, now), file("A", 1, now)},
		{file("A", 1, now), file("a", 1, now)},
	} {
		sortEntries(perm, sortByName, true)
		if got := namesOf(perm); !reflect.DeepEqual(got, []string{"A", "a"}) {
			t.Errorf("case-only difference should order deterministically, got %v", got)
		}
	}
	// Equal sizes fall back to name ascending, even when sorting descending.
	es := []fs.FileInfo{file("b", 5, now), file("a", 5, now), file("c", 5, now)}
	sortEntries(es, sortBySize, false)
	if got := namesOf(es); !reflect.DeepEqual(got, []string{"a", "b", "c"}) {
		t.Errorf("size ties should fall back to name ascending, got %v", got)
	}
}

func TestHumanSize(t *testing.T) {
	cases := []struct {
		n    int64
		want string
	}{
		{0, "0 B"},
		{1, "1 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{3174, "3.1 KB"},
		{1024 * 1024, "1.0 MB"},
		{1536 * 1024, "1.5 MB"},
		{5 * 1024 * 1024 * 1024, "5.0 GB"},
		{1 << 40, "1.0 TB"},
		{1<<63 - 1, "8.0 EB"},
	}
	for _, c := range cases {
		if got := humanSize(c.n); got != c.want {
			t.Errorf("humanSize(%d) = %q, want %q", c.n, got, c.want)
		}
	}
}

func TestFormatModTime(t *testing.T) {
	if got := formatModTime(time.Time{}); got != "" {
		t.Errorf("zero time = %q, want empty", got)
	}
	ts := time.Date(2026, 10, 30, 9, 5, 0, 0, time.Local)
	if got := formatModTime(ts); got != "2026-10-30 09:05" {
		t.Errorf("formatModTime = %q", got)
	}
}

func TestWithDefaultExt(t *testing.T) {
	cases := []struct{ name, ext, want string }{
		{"notes", "txt", "notes.txt"},
		{"notes", ".txt", "notes.txt"},
		{"notes.md", "txt", "notes.md"},
		{"notes", "", "notes"},
		{".bashrc", "txt", ".bashrc"}, // path.Ext sees the whole name as an extension
		{"archive.tar", "gz", "archive.tar"},
	}
	for _, c := range cases {
		if got := withDefaultExt(c.name, c.ext); got != c.want {
			t.Errorf("withDefaultExt(%q,%q) = %q, want %q", c.name, c.ext, got, c.want)
		}
	}
}

func TestSaveTarget(t *testing.T) {
	f := FSFileSystem(fstest.MapFS{
		"docs/exists.txt": {Data: []byte("x")},
		"docs/sub/f.txt":  {Data: []byte("x")},
	})
	cases := []struct {
		name       string
		dir, in    string
		ext        string
		wantTarget string
		wantExists bool
		wantErr    bool
	}{
		{"new file", "docs", "new", "txt", "/docs/new.txt", false, false},
		{"new, ext given", "docs", "new.md", "txt", "/docs/new.md", false, false},
		{"existing", "docs", "exists", "txt", "/docs/exists.txt", true, false},
		{"existing, exact", "docs", "exists.txt", "", "/docs/exists.txt", true, false},
		{"trims space", "docs", "  new  ", "txt", "/docs/new.txt", false, false},
		{"empty name", "docs", "   ", "txt", "", false, true},
		{"is a folder", "docs", "sub", "", "", false, true},
		{"typed absolute path", "docs/sub", "/docs/exists.txt", "", "/docs/exists.txt", true, false},
		{"typed relative path", "docs", "sub/f.txt", "", "/docs/sub/f.txt", true, false},
	}
	for _, c := range cases {
		target, exists, err := saveTarget(f, c.dir, c.in, c.ext)
		if (err != nil) != c.wantErr {
			t.Errorf("%s: err = %v, wantErr %v", c.name, err, c.wantErr)
			continue
		}
		if target != c.wantTarget || exists != c.wantExists {
			t.Errorf("%s: got (%q,%v), want (%q,%v)", c.name, target, exists, c.wantTarget, c.wantExists)
		}
	}
}
