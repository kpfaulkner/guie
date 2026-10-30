package ui

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"runtime"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// FileFilter is one entry in a file dialog's filter dropdown.
type FileFilter struct {
	// Name is the label shown in the dropdown ("Images").
	Name string
	// Patterns are path.Match patterns ("*.png") tried case-insensitively against
	// a file's base name. A file matches if any pattern does. An empty list
	// matches every file. Directories are never filtered out.
	Patterns []string
}

// matches reports whether a file called name passes the filter.
func (f FileFilter) matches(name string) bool {
	if len(f.Patterns) == 0 {
		return true
	}
	name = strings.ToLower(name)
	for _, p := range f.Patterns {
		if ok, err := path.Match(strings.ToLower(p), name); err == nil && ok {
			return true
		}
	}
	return false
}

// fileSortKey is the column a file listing is ordered by.
type fileSortKey int

const (
	sortByName fileSortKey = iota
	sortBySize
	sortByModified
)

// isHidden reports whether a dialog should treat an entry as hidden. On Windows
// that is the file's hidden attribute, as in Explorer: ".gitignore" is shown and
// a folder like "$Recycle.Bin" is not. Everywhere else, and for any filesystem
// that carries no Windows attributes (an io/fs.FS), it is the leading-dot
// convention.
func isHidden(info fs.FileInfo) bool {
	if hidden, known := attributeHidden(info); known {
		return hidden
	}
	return strings.HasPrefix(info.Name(), ".")
}

// visibleEntries returns the entries of a directory that a dialog should list:
// hidden ones only when showHidden is set, files only when they pass filter,
// and no files at all when dirsOnly is set. The input is not modified.
func visibleEntries(entries []fs.FileInfo, filter FileFilter, showHidden, dirsOnly bool) []fs.FileInfo {
	out := make([]fs.FileInfo, 0, len(entries))
	for _, e := range entries {
		if !showHidden && isHidden(e) {
			continue
		}
		if !e.IsDir() && (dirsOnly || !filter.matches(e.Name())) {
			continue
		}
		out = append(out, e)
	}
	return out
}

// sortEntries orders entries in place: directories first whatever the
// direction, then by key. Ties fall back to name ascending, so the order is
// total and a re-sort never shuffles equal rows. Directories have no meaningful
// size, so under sortBySize they are ordered by name alone.
func sortEntries(entries []fs.FileInfo, key fileSortKey, asc bool) {
	sort.SliceStable(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		if a.IsDir() != b.IsDir() {
			return a.IsDir()
		}
		var c int
		switch key {
		case sortBySize:
			if !a.IsDir() {
				c = cmpInt64(a.Size(), b.Size())
			}
		case sortByModified:
			c = a.ModTime().Compare(b.ModTime())
		default:
			c = compareNames(a.Name(), b.Name())
		}
		if c != 0 {
			if asc {
				return c < 0
			}
			return c > 0
		}
		return compareNames(a.Name(), b.Name()) < 0
	})
}

func cmpInt64(a, b int64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

// compareNames orders file names the way a file manager does: naturally (see
// naturalCompare), then, where that finds no difference, by plain string
// comparison, so "a" and "A", or "a01" and "a1", still have a fixed order and the
// sort is total.
func compareNames(a, b string) int {
	if c := naturalCompare(a, b); c != 0 {
		return c
	}
	return strings.Compare(a, b)
}

// naturalCompare compares two names ignoring case, treating each run of ASCII
// digits as one number: "file2" sorts before "file10", and "img09" before
// "img10". Numbers are compared by value however long they are, with no integer
// to overflow. A digit and a non-digit compare as their characters do. It works
// on the strings in place, without allocating, because sorting a large
// directory calls it O(n log n) times.
func naturalCompare(a, b string) int {
	for a != "" && b != "" {
		if isASCIIDigit(a[0]) && isASCIIDigit(b[0]) {
			i, j := digitRunEnd(a), digitRunEnd(b)
			if c := compareDigitRuns(a[:i], b[:j]); c != 0 {
				return c
			}
			a, b = a[i:], b[j:]
			continue
		}
		ra, na := utf8.DecodeRuneInString(a)
		rb, nb := utf8.DecodeRuneInString(b)
		if la, lb := unicode.ToLower(ra), unicode.ToLower(rb); la != lb {
			if la < lb {
				return -1
			}
			return 1
		}
		a, b = a[na:], b[nb:]
	}
	switch {
	case a != "": // b ran out first, so it is the prefix
		return 1
	case b != "":
		return -1
	}
	return 0
}

func isASCIIDigit(c byte) bool { return '0' <= c && c <= '9' }

// digitRunEnd returns the length of the run of ASCII digits at the start of s.
func digitRunEnd(s string) int {
	i := 0
	for i < len(s) && isASCIIDigit(s[i]) {
		i++
	}
	return i
}

// compareDigitRuns compares two non-empty runs of digits by numeric value.
// Leading zeros do not count, so "007" equals "7"; whichever has more digits
// after them is larger, and equal lengths compare as text.
func compareDigitRuns(x, y string) int {
	x, y = strings.TrimLeft(x, "0"), strings.TrimLeft(y, "0")
	if len(x) != len(y) {
		return cmpInt64(int64(len(x)), int64(len(y)))
	}
	return strings.Compare(x, y)
}

// humanSize formats a byte count for the Size column: "512 B", "3.1 KB",
// "1.0 MB". Units are powers of 1024.
func humanSize(n int64) string {
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	units := []string{"KB", "MB", "GB", "TB", "PB", "EB"}
	v := float64(n) / 1024
	i := 0
	for v >= 1024 && i < len(units)-1 {
		v /= 1024
		i++
	}
	return fmt.Sprintf("%.1f %s", v, units[i])
}

// withDefaultExt appends ext to name if name has no extension of its own. ext
// may be given with or without its leading dot. An empty ext leaves name alone.
func withDefaultExt(name, ext string) string {
	if ext == "" || path.Ext(name) != "" {
		return name
	}
	return name + "." + strings.TrimPrefix(ext, ".")
}

// saveTarget works out where a Save dialog would write. dir and name are the
// current directory and the text in the name field; ext is the default
// extension. exists reports an existing file at the target (the caller then
// asks to replace it). An existing directory is an error, since it cannot be
// replaced by a file; so is an empty name. A target that cannot be inspected
// for any reason other than not existing is an error too.
func saveTarget(fsys FileSystem, dir, name, ext string) (target string, exists bool, err error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", false, fmt.Errorf("enter a file name")
	}
	target = fsys.Resolve(dir, withDefaultExt(name, ext))
	info, err := fsys.Stat(target)
	switch {
	case err == nil && info.IsDir():
		return "", false, fmt.Errorf("%q is a folder", name)
	case err == nil:
		return target, true, nil
	case isNotExist(err):
		return target, false, nil
	}
	return "", false, err
}

func isNotExist(err error) bool { return errors.Is(err, fs.ErrNotExist) }

// formatModTime formats a modification time for the Modified column, in local
// time to the minute. The zero time (a FileSystem that has none) is blank.
func formatModTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Local().Format("2006-01-02 15:04")
}

// typeAheadTimeout is how long a pause ends the letters typed so far.
const typeAheadTimeout = time.Second

// typeAhead collects letters typed in a listing into a prefix, which a pause
// of typeAheadTimeout starts over.
type typeAhead struct {
	buf  []rune
	last time.Time
}

// pending reports whether letters typed at or before now are still being
// collected, that is, whether the next one would extend the prefix.
func (t *typeAhead) pending(now time.Time) bool {
	return len(t.buf) > 0 && now.Sub(t.last) <= typeAheadTimeout
}

// add appends r and returns the prefix typed so far. fresh is true when r began
// a new prefix, because it is the first letter or came after a pause.
func (t *typeAhead) add(r rune, now time.Time) (prefix string, fresh bool) {
	if len(t.buf) == 0 || now.Sub(t.last) > typeAheadTimeout {
		t.buf = t.buf[:0]
		fresh = true
	}
	t.buf = append(t.buf, r)
	t.last = now
	return string(t.buf), fresh
}

// typeAheadMatch returns the index of the first name at or after start (wrapping
// round to the beginning) that starts with prefix, ignoring case, or -1 if none
// does. A start outside the list begins at the top.
func typeAheadMatch(names []string, start int, prefix string) int {
	if len(names) == 0 || prefix == "" {
		return -1
	}
	if start < 0 || start >= len(names) {
		start = 0
	}
	prefix = strings.ToLower(prefix)
	for k := 0; k < len(names); k++ {
		i := (start + k) % len(names)
		if strings.HasPrefix(strings.ToLower(names[i]), prefix) {
			return i
		}
	}
	return -1
}

// allSameRune reports whether every rune in rs is the same one.
func allSameRune(rs []rune) bool {
	for _, r := range rs[1:] {
		if r != rs[0] {
			return false
		}
	}
	return true
}

// validFolderName checks a name typed for a new folder and returns it trimmed.
// It must be a single path element: separators are rejected rather than creating
// nested folders. Characters the filesystem itself forbids are left to its own
// error, which the dialog shows.
func validFolderName(name string) (string, error) {
	name = strings.TrimSpace(name)
	switch {
	case name == "":
		return "", errors.New("Enter a folder name.")
	case strings.ContainsAny(name, `/\`):
		return "", errors.New(`A folder name cannot contain / or \.`)
	case name == "." || name == "..":
		return "", fmt.Errorf("%q is not a valid folder name.", name)
	case strings.ContainsRune(name, 0):
		return "", errors.New("A folder name cannot contain a null character.")
	}
	return name, nil
}

// uniqueName returns base if exists reports it free, otherwise "base (2)",
// "base (3)" and so on: the suggestion for a new folder. Name comparison is the
// caller's: exists decides what counts as taken.
func uniqueName(base string, exists func(string) bool) string {
	if !exists(base) {
		return base
	}
	for n := 2; ; n++ {
		if name := fmt.Sprintf("%s (%d)", base, n); !exists(name) {
			return name
		}
	}
}

// currentPlaceIndex returns the index in places of the place that dir is in: the
// most specific one, meaning dir itself if it is a place, otherwise its nearest
// ancestor that is. So with Home and Documents as places, the Documents folder and
// everything below it are in Documents, and any other folder under home is in
// Home. It returns -1 if dir is under none of them. Where two places have the
// same path the earlier one wins.
func currentPlaceIndex(fsys FileSystem, places []fileDialogPlace, dir string) int {
	p := dir
	for depth := 0; depth < 4096; depth++ { // a path has no more parents than this
		for i, pl := range places {
			if samePath(pl.path, p) {
				return i
			}
		}
		parent, ok := fsys.Parent(p)
		if !ok {
			break
		}
		p = parent
	}
	return -1
}

// samePath reports whether two paths from the same FileSystem name the same
// place. Windows paths are not case-sensitive (a drive typed as "c:" is "C:"); no
// other filesystem here is told that, so elsewhere only an exact match counts.
func samePath(a, b string) bool {
	if a == b {
		return true
	}
	return runtime.GOOS == "windows" && strings.EqualFold(a, b)
}
