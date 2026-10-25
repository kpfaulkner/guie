package ui

import "unicode"

// undoDepth is the most steps a text widget remembers; the oldest are dropped.
const undoDepth = 500

// editKind says how an edit may be merged into the previous undo step.
type editKind int

const (
	kindOther     editKind = iota // paste, cut, selection replace, Enter: never merges
	kindType                      // a typed rune
	kindBackspace                 // one rune deleted backwards
	kindDelete                    // one rune deleted forwards
)

// textEdit is one recorded change: at position at, the runes removed were
// replaced by the runes inserted. before and after are the caret/selection
// either side, so undo and redo restore it too. P is the position type (a rune
// index, or a row/column) and S the selection type.
type textEdit[P any, S comparable] struct {
	at                P
	removed, inserted []rune
	before, after     S
	kind              editKind
}

// editStack is a text widget's undo/redo history. It stores edits, not
// snapshots, so a step costs what changed rather than the whole text.
type editStack[P any, S comparable] struct {
	undo, redo []textEdit[P, S]
	// sealed stops the next edit merging into the top step. It is set by
	// anything that is not an edit (focus loss, undo, redo).
	sealed bool
}

// add records e, merging it into the top step when the two read as one action.
// Any new edit invalidates the redo stack.
func (s *editStack[P, S]) add(e textEdit[P, S]) {
	s.redo = nil
	if n := len(s.undo); n > 0 && !s.sealed && mergeEdit(&s.undo[n-1], e) {
		return
	}
	s.undo = append(s.undo, e)
	if len(s.undo) > undoDepth {
		copy(s.undo, s.undo[1:])
		s.undo = s.undo[:undoDepth]
	}
	s.sealed = false
}

// seal ends the current step: the next edit starts a new one.
func (s *editStack[P, S]) seal() { s.sealed = true }

// clear forgets all history.
func (s *editStack[P, S]) clear() { s.undo, s.redo, s.sealed = nil, nil, false }

func (s *editStack[P, S]) popUndo() (textEdit[P, S], bool) {
	n := len(s.undo)
	if n == 0 {
		return textEdit[P, S]{}, false
	}
	e := s.undo[n-1]
	s.undo = s.undo[:n-1]
	return e, true
}

func (s *editStack[P, S]) popRedo() (textEdit[P, S], bool) {
	n := len(s.redo)
	if n == 0 {
		return textEdit[P, S]{}, false
	}
	e := s.redo[n-1]
	s.redo = s.redo[:n-1]
	return e, true
}

// mergeEdit folds e into top if they are the same mergeable kind and e starts
// exactly where top left the caret (so any caret or selection change in between
// keeps them apart). Typing also splits at a word boundary: a whitespace rune
// typed after a non-whitespace one starts a new step.
func mergeEdit[P any, S comparable](top *textEdit[P, S], e textEdit[P, S]) bool {
	if e.kind == kindOther || top.kind != e.kind || e.before != top.after {
		return false
	}
	switch e.kind {
	case kindType:
		if n := len(top.inserted); n > 0 && len(e.inserted) > 0 &&
			unicode.IsSpace(e.inserted[0]) && !unicode.IsSpace(top.inserted[n-1]) {
			return false
		}
		top.inserted = append(top.inserted, e.inserted...)
	case kindBackspace:
		top.removed = append(append([]rune(nil), e.removed...), top.removed...)
		top.at = e.at
	case kindDelete:
		top.removed = append(top.removed, e.removed...)
	default:
		return false
	}
	top.after = e.after
	return true
}
