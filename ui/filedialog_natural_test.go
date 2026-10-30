package ui

import (
	"io/fs"
	"math/rand"
	"reflect"
	"sort"
	"testing"
	"time"
)

func TestNaturalCompare(t *testing.T) {
	cases := []struct {
		a, b string
		want int // sign of the result
	}{
		// The point of it: numbers by value.
		{"file2", "file10", -1},
		{"file10", "file2", 1},
		{"file9", "file10", -1},
		{"img09", "img10", -1},
		{"a1b2", "a1b10", -1},
		{"a2b1", "a10b1", -1},
		{"1", "2", -1},
		{"2", "10", -1},
		{"10", "9", 1},

		// Equal numbers, however they are written.
		{"a007", "a7", 0},
		{"a0", "a00", 0},
		{"a0b", "a00b", 0},
		{"file1", "file1", 0},

		// Longer than any integer type holds.
		{"v99999999999999999999999", "v100000000000000000000000", -1},
		{"v100000000000000000000000", "v99999999999999999999999", 1},
		{"v123456789012345678901234567890", "v123456789012345678901234567891", -1},

		// Case is ignored.
		{"apple", "Banana", -1},
		{"Apple", "apple", 0},
		{"README", "readme", 0},
		{"B2", "a10", 1},

		// Prefixes come first.
		{"file", "file1", -1},
		{"file1", "file", 1},
		{"", "a", -1},
		{"", "", 0},

		// A digit against a character compares as characters do.
		{"a1", "ab", -1}, // '1' < 'b'
		{"a-", "a1", -1}, // '-' < '1'
		{"a1", "a_", -1}, // '1' < '_'
		{"1a", "a1", -1},

		// Not just ASCII.
		{"éa2", "éa10", -1},
		{"Éa", "éa", 0}, // E acute either case
	}
	for _, c := range cases {
		got := naturalCompare(c.a, c.b)
		if sign(got) != c.want {
			t.Errorf("naturalCompare(%q, %q) = %d, want sign %d", c.a, c.b, got, c.want)
		}
		if back := naturalCompare(c.b, c.a); sign(back) != -c.want {
			t.Errorf("naturalCompare(%q, %q) = %d, want sign %d (not antisymmetric)", c.b, c.a, back, -c.want)
		}
	}
}

func sign(n int) int {
	switch {
	case n < 0:
		return -1
	case n > 0:
		return 1
	}
	return 0
}

func TestCompareNamesIsATotalOrderOnDistinctNames(t *testing.T) {
	// Natural order alone calls "a01", "a1" and "A1" equal; compareNames must not.
	for _, pair := range [][2]string{{"a01", "a1"}, {"a1", "A1"}, {"a", "A"}, {"a00", "a0"}} {
		if compareNames(pair[0], pair[1]) == 0 || compareNames(pair[0], pair[1]) != -compareNames(pair[1], pair[0]) {
			t.Errorf("compareNames(%q, %q) must be nonzero and antisymmetric", pair[0], pair[1])
		}
	}
	if compareNames("same", "same") != 0 {
		t.Error("identical names must compare equal")
	}
}

// TestCompareNamesIsTransitive checks, over every short string from an alphabet
// chosen to hit the awkward cases (case, leading zeros, digits beside symbols),
// that compareNames is a consistent ordering: sort would misbehave otherwise.
func TestCompareNamesIsTransitive(t *testing.T) {
	alphabet := []string{"a", "A", "0", "1", "-"}
	names := []string{""}
	frontier := []string{""}
	for n := 0; n < 3; n++ {
		var next []string
		for _, p := range frontier {
			for _, c := range alphabet {
				next = append(next, p+c)
			}
		}
		names = append(names, next...)
		frontier = next
	}
	for _, a := range names {
		for _, b := range names {
			ab, ba := sign(compareNames(a, b)), sign(compareNames(b, a))
			if ab != -ba {
				t.Fatalf("not antisymmetric: %q vs %q gives %d and %d", a, b, ab, ba)
			}
			if ab != 0 {
				continue
			}
			if a != b {
				t.Fatalf("distinct names %q and %q compare equal", a, b)
			}
		}
	}
	for _, a := range names {
		for _, b := range names {
			if compareNames(a, b) > 0 {
				continue
			}
			for _, c := range names {
				if compareNames(b, c) <= 0 && compareNames(a, c) > 0 {
					t.Fatalf("not transitive: %q <= %q <= %q but %q > %q", a, b, c, a, c)
				}
			}
		}
	}
}

func TestNaturalCompareDoesNotAllocate(t *testing.T) {
	a, b := "holiday photos 2026 IMG_00123.JPG", "holiday photos 2026 IMG_00124.JPG"
	if n := testing.AllocsPerRun(100, func() { compareNames(a, b) }); n != 0 {
		t.Errorf("compareNames allocates %v times per call, want 0", n)
	}
}

func TestSortEntriesByNameIsNatural(t *testing.T) {
	now := time.Now()
	mk := func(names ...string) []fs.FileInfo {
		out := make([]fs.FileInfo, len(names))
		for i, n := range names {
			out[i] = file(n, 1, now)
		}
		return out
	}
	es := mk("file10.txt", "file2.txt", "File1.txt", "file1b.txt", "file20.txt", "file3.txt")
	sortEntries(es, sortByName, true)
	if got, want := namesOf(es), []string{"File1.txt", "file1b.txt", "file2.txt", "file3.txt", "file10.txt", "file20.txt"}; !reflect.DeepEqual(got, want) {
		t.Errorf("ascending: got %v, want %v", got, want)
	}
	sortEntries(es, sortByName, false)
	if got, want := namesOf(es), []string{"file20.txt", "file10.txt", "file3.txt", "file2.txt", "file1b.txt", "File1.txt"}; !reflect.DeepEqual(got, want) {
		t.Errorf("descending: got %v, want %v", got, want)
	}

	// Directories still come first, and are in natural order too.
	mixed := []fs.FileInfo{file("a2", 1, now), dir("d10", now), dir("d2", now), file("a10", 1, now)}
	sortEntries(mixed, sortByName, true)
	if got, want := namesOf(mixed), []string{"d2", "d10", "a2", "a10"}; !reflect.DeepEqual(got, want) {
		t.Errorf("dirs first: got %v, want %v", got, want)
	}
}

func TestSortEntriesTiesBySizeFallBackToNaturalName(t *testing.T) {
	now := time.Now()
	es := []fs.FileInfo{file("f10", 5, now), file("f9", 5, now), file("f100", 5, now)}
	sortEntries(es, sortBySize, false)
	if got, want := namesOf(es), []string{"f9", "f10", "f100"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestSortEntriesBigDirectoryIsSortedAndFast(t *testing.T) {
	now := time.Now()
	rng := rand.New(rand.NewSource(1))
	const n = 50000
	es := make([]fs.FileInfo, n)
	for i, v := range rng.Perm(n) {
		es[i] = file("IMG_"+itoa(v)+".jpg", 1, now)
	}
	start := time.Now()
	sortEntries(es, sortByName, true)
	if d := time.Since(start); d > 2*time.Second {
		t.Errorf("sorting %d names took %v", n, d)
	}
	if !sort.SliceIsSorted(es, func(i, j int) bool { return compareNames(es[i].Name(), es[j].Name()) < 0 }) {
		t.Error("not sorted")
	}
	if es[0].Name() != "IMG_0.jpg" || es[n-1].Name() != "IMG_"+itoa(n-1)+".jpg" {
		t.Errorf("ends = %q ... %q", es[0].Name(), es[n-1].Name())
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for ; n > 0; n /= 10 {
		b = append([]byte{byte('0' + n%10)}, b...)
	}
	return string(b)
}
