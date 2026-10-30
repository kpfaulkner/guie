package ui

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// FileRoot is a top-level place a file dialog can start from: a drive on
// Windows, "/" elsewhere, or the root of an io/fs.FS.
type FileRoot struct {
	// Name is the label shown to the user ("C:", "/").
	Name string
	// Path is the root's path in the FileSystem's own form ("C:\", "/", ".").
	Path string
}

// FileSystem is what a file dialog browses. It exists so the dialog never
// touches package os or path/filepath itself: path separators and roots differ
// between the OS and an io/fs.FS, and a dialog tied to os could be neither
// tested headlessly nor run under WASM. See OSFileSystem and FSFileSystem.
//
// Paths are opaque to the dialog: it only passes back what a FileSystem handed
// it, or what Resolve returned for text the user typed.
//
// The dialog calls ReadDir on a background goroutine, so a slow directory does
// not freeze the window, while it calls everything else on the UI goroutine. A
// FileSystem must therefore allow ReadDir to run alongside its other methods.
type FileSystem interface {
	// Roots lists the top-level places (drives). It never returns an empty slice.
	Roots() []FileRoot
	// Home returns the user's home directory, or "" if there is none.
	Home() string
	// ReadDir lists dir, unsorted and unfiltered. Symbolic links are followed, so
	// a link to a directory reports IsDir; a link that cannot be followed is
	// reported as the link itself. Entries that cannot be inspected are skipped.
	ReadDir(dir string) ([]fs.FileInfo, error)
	// Stat describes path, following symbolic links.
	Stat(path string) (fs.FileInfo, error)
	// Join returns the path of the entry name inside dir.
	Join(dir, name string) string
	// Resolve turns text a user typed into a clean path: p is taken as is if it is
	// absolute, otherwise it is relative to dir. It does not touch the disk.
	Resolve(dir, p string) string
	// Parent returns the directory containing path, and false if path is a root.
	Parent(path string) (string, bool)
}

// DirMaker is implemented by a FileSystem that can create directories. A dialog
// offers a New folder button only when its FileSystem is one, so a read-only
// FileSystem (FSFileSystem, which wraps a read-only io/fs.FS) needs nothing.
// OSFileSystem is one. Mkdir creates a single directory, not its parents, and
// returns an error wrapping fs.ErrExist if path is already there.
type DirMaker interface {
	Mkdir(path string) error
}

// OSFileSystem returns a FileSystem over the operating system's files, using
// the host's path semantics (drive letters and backslashes on Windows).
func OSFileSystem() FileSystem { return osFileSystem{} }

type osFileSystem struct{}

func (osFileSystem) Roots() []FileRoot { return osRoots() }

func (osFileSystem) Home() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return h
}

func (osFileSystem) ReadDir(dir string) ([]fs.FileInfo, error) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := make([]fs.FileInfo, 0, len(ents))
	for _, e := range ents {
		info, err := e.Info()
		if err != nil {
			continue // vanished between ReadDir and Info
		}
		if info.Mode()&fs.ModeSymlink != 0 {
			if target, err := os.Stat(filepath.Join(dir, e.Name())); err == nil {
				info = target
			}
		}
		out = append(out, info)
	}
	return out, nil
}

func (osFileSystem) Stat(p string) (fs.FileInfo, error) { return os.Stat(p) }

func (osFileSystem) Mkdir(p string) error { return os.Mkdir(p, 0o755) }

func (osFileSystem) Join(dir, name string) string { return filepath.Join(dir, name) }

func (osFileSystem) Resolve(dir, p string) string {
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	return filepath.Clean(filepath.Join(dir, p))
}

func (osFileSystem) Parent(p string) (string, bool) {
	p = filepath.Clean(p)
	parent := filepath.Dir(p)
	if parent == p {
		return p, false
	}
	return parent, true
}

// FSFileSystem returns a FileSystem over any io/fs.FS: an embed.FS, the files
// of an OS drop, a browser filesystem under WASM, or fstest.MapFS in tests.
//
// Its paths are slash-separated and rooted, like an absolute OS path: "/" is the
// root and "/docs/a.txt" a file in it. That keeps what a user sees and retypes
// in the dialog's path bar unambiguous. A path handed back to the caller is
// therefore not directly usable with the fs.FS: strip the leading slash first
// (strings.TrimPrefix(p, "/")).
func FSFileSystem(fsys fs.FS) FileSystem { return fsFileSystem{fsys: fsys} }

type fsFileSystem struct{ fsys fs.FS }

// rel turns one of this filesystem's rooted paths into the unrooted form io/fs
// needs, with "." for the root. It never escapes the root.
func (fsFileSystem) rel(p string) string {
	p = strings.TrimPrefix(path.Clean("/"+p), "/")
	if p == "" {
		return "."
	}
	return p
}

func (fsFileSystem) Roots() []FileRoot { return []FileRoot{{Name: "/", Path: "/"}} }

func (fsFileSystem) Home() string { return "" }

func (f fsFileSystem) ReadDir(dir string) ([]fs.FileInfo, error) {
	ents, err := fs.ReadDir(f.fsys, f.rel(dir))
	if err != nil {
		return nil, err
	}
	out := make([]fs.FileInfo, 0, len(ents))
	for _, e := range ents {
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.Mode()&fs.ModeSymlink != 0 {
			if target, err := fs.Stat(f.fsys, f.rel(path.Join(dir, e.Name()))); err == nil {
				info = target
			}
		}
		out = append(out, info)
	}
	return out, nil
}

func (f fsFileSystem) Stat(p string) (fs.FileInfo, error) { return fs.Stat(f.fsys, f.rel(p)) }

func (fsFileSystem) Join(dir, name string) string { return path.Join("/", dir, name) }

func (fsFileSystem) Resolve(dir, p string) string {
	if strings.HasPrefix(p, "/") {
		return path.Clean(p)
	}
	return path.Clean(path.Join("/", dir, p))
}

func (fsFileSystem) Parent(p string) (string, bool) {
	p = path.Clean("/" + p)
	if p == "/" {
		return p, false
	}
	return path.Dir(p), true
}
