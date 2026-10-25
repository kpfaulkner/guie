package ui

import "unicode"

// runeClass groups runes for double-click word selection: a word is a run of
// letters, digits and underscores; whitespace and punctuation each select as
// their own run, as in most editors.
func runeClass(r rune) int {
	switch {
	case unicode.IsSpace(r):
		return 0
	case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_':
		return 1
	default:
		return 2
	}
}

// wordBounds returns the [lo, hi) run of same-class runes around index idx in
// rs. idx is a caret position, so it sits between two runes: the word-class rune
// beside it wins (right first), else the one on the right, else (at the end of
// the text) the one on the left. Empty text gives an empty range.
func wordBounds(rs []rune, idx int) (lo, hi int) {
	if len(rs) == 0 {
		return 0, 0
	}
	if idx >= len(rs) {
		idx = len(rs) - 1
	}
	if idx < 0 {
		idx = 0
	}
	if idx > 0 && runeClass(rs[idx]) != 1 && runeClass(rs[idx-1]) == 1 {
		idx-- // the click landed just past the end of a word
	}
	c := runeClass(rs[idx])
	lo, hi = idx, idx+1
	for lo > 0 && runeClass(rs[lo-1]) == c {
		lo--
	}
	for hi < len(rs) && runeClass(rs[hi]) == c {
		hi++
	}
	return lo, hi
}
