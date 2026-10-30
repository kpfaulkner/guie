package ui

import (
	"runtime"
	"testing"
	"testing/fstest"
)

func TestCurrentPlaceIndex(t *testing.T) {
	f := FSFileSystem(fstest.MapFS{})
	places := []fileDialogPlace{
		{"Home", "/home/me"},                // 0
		{"Desktop", "/home/me/Desktop"},     // 1
		{"Documents", "/home/me/Documents"}, // 2
		{"/", "/"},                          // 3
		{"D:", "/data"},                     // 4
	}
	cases := []struct {
		name, dir string
		want      int
	}{
		{"a place itself", "/home/me/Desktop", 1},
		{"home itself", "/home/me", 0},
		{"below a place", "/home/me/Documents/work", 2},
		{"deep below a place", "/home/me/Desktop/a/b/c/d", 1},
		{"under home, not under a sub-place", "/home/me/projects/guie", 0},
		{"a sibling with the same prefix is not inside", "/home/me/Desktopper", 0},
		{"another root", "/data/x", 4},
		{"the second root itself", "/data", 4},
		{"the filesystem root is a place", "/", 3},
		{"under the filesystem root only", "/etc/ssh", 3},
		{"above home falls to the root place", "/home", 3},
	}
	for _, c := range cases {
		if got := currentPlaceIndex(f, places, c.dir); got != c.want {
			t.Errorf("%s: currentPlaceIndex(%q) = %d (%s), want %d (%s)", c.name, c.dir, got, nameAt(places, got), c.want, nameAt(places, c.want))
		}
	}
}

func nameAt(places []fileDialogPlace, i int) string {
	if i < 0 || i >= len(places) {
		return "none"
	}
	return places[i].name
}

func TestCurrentPlaceIndexNoMatch(t *testing.T) {
	f := FSFileSystem(fstest.MapFS{})
	homeOnly := []fileDialogPlace{{"Home", "/home/me"}}
	if got := currentPlaceIndex(f, homeOnly, "/etc"); got != -1 {
		t.Errorf("a folder outside every place = %d, want -1", got)
	}
	if got := currentPlaceIndex(f, nil, "/home/me"); got != -1 {
		t.Errorf("no places = %d, want -1", got)
	}
}

func TestCurrentPlaceIndexFirstOfEqualPathsWins(t *testing.T) {
	f := FSFileSystem(fstest.MapFS{})
	places := []fileDialogPlace{{"Home", "/h"}, {"Also home", "/h"}}
	if got := currentPlaceIndex(f, places, "/h/x"); got != 0 {
		t.Errorf("got %d, want 0", got)
	}
}

func TestCurrentPlaceIndexOSPaths(t *testing.T) {
	f := OSFileSystem()
	base := t.TempDir()
	sub := f.Join(base, "docs")
	places := []fileDialogPlace{{"Home", base}, {"Docs", sub}}
	if got := currentPlaceIndex(f, places, f.Join(f.Join(sub, "deep"), "er")); got != 1 {
		t.Errorf("below Docs = %d, want 1", got)
	}
	if got := currentPlaceIndex(f, places, f.Join(base, "other")); got != 0 {
		t.Errorf("below Home = %d, want 0", got)
	}
	parent, _ := f.Parent(base)
	if got := currentPlaceIndex(f, places, parent); got != -1 {
		t.Errorf("above every place = %d, want -1", got)
	}
}

func TestSamePath(t *testing.T) {
	if !samePath("/a/b", "/a/b") {
		t.Error("identical paths must match")
	}
	if samePath("/a/b", "/a/c") {
		t.Error("different paths must not match")
	}
	// Case folds only where the filesystem does.
	if got, want := samePath("C:/Users", "c:/users"), runtime.GOOS == "windows"; got != want {
		t.Errorf("case-insensitive match = %v, want %v on %s", got, want, runtime.GOOS)
	}
}
