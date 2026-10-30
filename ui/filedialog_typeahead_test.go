package ui

import (
	"testing"
	"time"
)

func TestTypeAheadAccumulatesAndExpires(t *testing.T) {
	var ta typeAhead
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	if p, fresh := ta.add('r', t0); p != "r" || !fresh {
		t.Errorf("first letter = %q fresh=%v", p, fresh)
	}
	if p, fresh := ta.add('e', t0.Add(300*time.Millisecond)); p != "re" || fresh {
		t.Errorf("second letter = %q fresh=%v, want re, not fresh", p, fresh)
	}
	// A pause longer than the timeout starts over.
	if p, fresh := ta.add('n', t0.Add(300*time.Millisecond+typeAheadTimeout+time.Millisecond)); p != "n" || !fresh {
		t.Errorf("after a pause = %q fresh=%v, want n, fresh", p, fresh)
	}
	// The timeout runs from the last letter, not the first.
	now := t0.Add(5 * time.Second)
	ta.add('a', now)
	for i := 0; i < 5; i++ {
		now = now.Add(800 * time.Millisecond)
		ta.add('b', now)
	}
	if p, _ := ta.add('c', now.Add(800*time.Millisecond)); p != "abbbbbc" {
		t.Errorf("steady typing should keep the prefix, got %q", p)
	}
}

func TestTypeAheadMatch(t *testing.T) {
	names := []string{"docs", "Notes.txt", "pic.png", "photos", "readme.md"}
	cases := []struct {
		name   string
		start  int
		prefix string
		want   int
	}{
		{"first match from the top", 0, "p", 2},
		{"case-insensitive both ways", 0, "N", 1},
		{"lower prefix, mixed-case name", 0, "notes", 1},
		{"start after the first match finds the next", 3, "p", 3},
		{"wraps round", 4, "d", 0},
		{"wraps to an earlier match", 3, "pi", 2},
		{"no match", 0, "z", -1},
		{"empty prefix", 0, "", -1},
		{"start out of range begins at the top", 99, "d", 0},
		{"negative start begins at the top", -1, "r", 4},
	}
	for _, c := range cases {
		if got := typeAheadMatch(names, c.start, c.prefix); got != c.want {
			t.Errorf("%s: typeAheadMatch(start=%d, %q) = %d, want %d", c.name, c.start, c.prefix, got, c.want)
		}
	}
	if typeAheadMatch(nil, 0, "a") != -1 {
		t.Error("no names should give -1")
	}
}

func TestAllSameRune(t *testing.T) {
	if !allSameRune([]rune("aaa")) || allSameRune([]rune("aab")) || !allSameRune([]rune("a")) {
		t.Error("allSameRune wrong")
	}
}
