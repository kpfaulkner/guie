package ui

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func testMapFS() fstest.MapFS {
	return fstest.MapFS{
		"top.md":         {Data: []byte("hello")},
		"a/b.txt":        {Data: []byte("bb")},
		"a/c/d.go":       {Data: []byte("package d")},
		"link":           {Data: []byte("a"), Mode: fs.ModeSymlink},
		"dangling":       {Data: []byte("nowhere"), Mode: fs.ModeSymlink},
		"a/c/.hidden":    {Data: []byte("")},
		"a/c/UPPER.TXT":  {Data: []byte("x")},
		"a/c/deep/e.txt": {Data: []byte("e")},
	}
}

func entryNames(infos []fs.FileInfo) map[string]bool {
	m := map[string]bool{}
	for _, i := range infos {
		m[i.Name()] = i.IsDir()
	}
	return m
}

func TestFSFileSystemReadDir(t *testing.T) {
	f := FSFileSystem(testMapFS())
	got, err := f.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	names := entryNames(got)
	if isDir, ok := names["a"]; !ok || !isDir {
		t.Errorf("a should be a dir, got %v", names)
	}
	if isDir, ok := names["top.md"]; !ok || isDir {
		t.Errorf("top.md should be a file, got %v", names)
	}
	if isDir, ok := names["link"]; !ok || !isDir {
		t.Errorf("a symlink to a dir should report IsDir, got %v", names)
	}
	if isDir, ok := names["dangling"]; !ok || isDir {
		t.Errorf("a dangling symlink should stay a (non-dir) entry, got %v", names)
	}
}

func TestFSFileSystemReadDirMissing(t *testing.T) {
	if _, err := FSFileSystem(testMapFS()).ReadDir("nope"); err == nil {
		t.Fatal("ReadDir of a missing dir should fail")
	}
}

func TestFSFileSystemStat(t *testing.T) {
	f := FSFileSystem(testMapFS())
	if info, err := f.Stat("a/b.txt"); err != nil || info.Size() != 2 {
		t.Errorf("Stat a/b.txt = %v, %v", info, err)
	}
	if _, err := f.Stat("a/zzz"); !isNotExist(err) {
		t.Errorf("Stat of a missing file should be not-exist, got %v", err)
	}
}

func TestFSFileSystemJoinResolveParent(t *testing.T) {
	f := FSFileSystem(testMapFS())
	join := []struct{ dir, name, want string }{
		{"/", "a", "/a"},
		{".", "a", "/a"},
		{"/a", "b.txt", "/a/b.txt"},
		{"a/c", "d.go", "/a/c/d.go"},
	}
	for _, c := range join {
		if got := f.Join(c.dir, c.name); got != c.want {
			t.Errorf("Join(%q,%q) = %q, want %q", c.dir, c.name, got, c.want)
		}
	}
	resolve := []struct{ dir, p, want string }{
		{"/a", "b.txt", "/a/b.txt"},
		{"/a", "/top.md", "/top.md"},
		{"/a/c", "..", "/a"},
		{"/a/c", "../b.txt", "/a/b.txt"},
		{"/a", "../..", "/"},
		{"/a", "../../../x", "/x"},
		{"/a", "/", "/"},
		{"/a", "", "/a"},
		{"/", "a//c/./d.go", "/a/c/d.go"},
		{"", "a", "/a"},
	}
	for _, c := range resolve {
		if got := f.Resolve(c.dir, c.p); got != c.want {
			t.Errorf("Resolve(%q,%q) = %q, want %q", c.dir, c.p, got, c.want)
		}
	}
	parent := []struct {
		p    string
		want string
		ok   bool
	}{
		{"/a/c/deep", "/a/c", true},
		{"/a", "/", true},
		{"a", "/", true},
		{"/", "/", false},
		{".", "/", false},
	}
	for _, c := range parent {
		got, ok := f.Parent(c.p)
		if got != c.want || ok != c.ok {
			t.Errorf("Parent(%q) = %q,%v want %q,%v", c.p, got, ok, c.want, c.ok)
		}
	}
}

func TestFSFileSystemAcceptsRootedPaths(t *testing.T) {
	f := FSFileSystem(testMapFS())
	for _, dir := range []string{"/", ".", "", "/a", "a"} {
		if _, err := f.ReadDir(dir); err != nil {
			t.Errorf("ReadDir(%q): %v", dir, err)
		}
	}
	if info, err := f.Stat("/a/b.txt"); err != nil || info.Size() != 2 {
		t.Errorf("Stat(/a/b.txt) = %v, %v", info, err)
	}
	// A path cannot climb out of the root, whatever it is spelled like.
	if _, err := f.Stat("/../../top.md"); err != nil {
		t.Errorf("Stat(/../../top.md) should clamp to the root: %v", err)
	}
}

func TestFSFileSystemRoots(t *testing.T) {
	f := FSFileSystem(testMapFS())
	r := f.Roots()
	if len(r) != 1 || r[0].Path != "/" {
		t.Errorf("Roots = %v", r)
	}
	if f.Home() != "" {
		t.Errorf("Home = %q, want empty", f.Home())
	}
}

func TestOSFileSystemReadDirAndStat(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("12345"), 0o644); err != nil {
		t.Fatal(err)
	}
	f := OSFileSystem()
	got, err := f.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	names := entryNames(got)
	if len(names) != 2 || !names["sub"] || names["f.txt"] {
		t.Errorf("ReadDir = %v", names)
	}
	info, err := f.Stat(f.Join(dir, "f.txt"))
	if err != nil || info.Size() != 5 {
		t.Errorf("Stat = %v, %v", info, err)
	}
	if _, err := f.ReadDir(f.Join(dir, "missing")); err == nil {
		t.Error("ReadDir of a missing dir should fail")
	}
}

func TestOSFileSystemSymlinkToDirIsDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "real"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(dir, "real"), filepath.Join(dir, "lnk")); err != nil {
		t.Skipf("cannot create symlinks here: %v", err)
	}
	if err := os.Symlink(filepath.Join(dir, "gone"), filepath.Join(dir, "broken")); err != nil {
		t.Skipf("cannot create symlinks here: %v", err)
	}
	got, err := OSFileSystem().ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	names := entryNames(got)
	if !names["lnk"] {
		t.Errorf("symlink to a dir should be IsDir, got %v", names)
	}
	if isDir, ok := names["broken"]; !ok || isDir {
		t.Errorf("broken symlink should be listed as a non-dir, got %v", names)
	}
}

func TestOSFileSystemResolveAndParent(t *testing.T) {
	f := OSFileSystem()
	base := t.TempDir()
	if got, want := f.Resolve(base, "x.txt"), filepath.Join(base, "x.txt"); got != want {
		t.Errorf("relative Resolve = %q, want %q", got, want)
	}
	abs := filepath.Join(base, "sub", "..", "y.txt")
	if got, want := f.Resolve("elsewhere", abs), filepath.Join(base, "y.txt"); got != want {
		t.Errorf("absolute Resolve = %q, want %q", got, want)
	}
	if got, want := f.Resolve(base, ".."), filepath.Dir(base); got != want {
		t.Errorf("Resolve .. = %q, want %q", got, want)
	}

	if p, ok := f.Parent(filepath.Join(base, "sub")); !ok || p != base {
		t.Errorf("Parent = %q,%v want %q,true", p, ok, base)
	}
	root := filepath.VolumeName(base) + string(filepath.Separator)
	if _, ok := f.Parent(root); ok {
		t.Errorf("Parent(%q) should report a root", root)
	}
}

func TestOSFileSystemRootsNotEmpty(t *testing.T) {
	r := OSFileSystem().Roots()
	if len(r) == 0 {
		t.Fatal("Roots must never be empty")
	}
	for _, x := range r {
		if x.Name == "" || x.Path == "" {
			t.Errorf("incomplete root %+v", x)
		}
	}
}
