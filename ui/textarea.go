package ui

import (
	"image/color"
	"strings"

	"github.com/kpfaulkner/uiframework/geom"
	"github.com/kpfaulkner/uiframework/render"
)

// textAreaPadding is the inner padding around a text area's content.
var textAreaPadding = geom.Insets{Top: 6, Right: 8, Bottom: 6, Left: 8}

// defaultTextAreaRows is the number of lines a text area is tall by default.
const defaultTextAreaRows = 4

// TextArea is a multi-line text editor. It is focusable and edits its content
// in response to typed runes and editing keys: Enter inserts a newline,
// Backspace/Delete join lines at the ends, and the arrow keys plus Home/End
// move the caret. The view scrolls vertically (wheel or to follow the caret).
//
// It supports a selection: click and drag, or hold Shift while moving the
// caret, to select a range across lines; Ctrl+A selects all. Typing,
// Backspace/Delete and inserting replace the selection.
//
// Lines are split on '\n'; there is no soft word-wrap or horizontal scrolling
// yet, so text past the right edge of a long line is clipped.
type TextArea struct {
	BaseWidget
	lines     [][]rune
	caretRow  int
	caretCol  int
	anchorRow int
	anchorCol int
	focused   bool
	hover     bool
	dragging  bool
	scrollY   float64

	wrap      bool
	wrapWidth float64 // content width used for the last wrap pass

	placeholder string
	onChange    func(string)

	font render.FontFace
}

// visRow is a visual row: a segment [start,end) of logical line lr. With wrap
// off there is exactly one visRow per logical line.
type visRow struct {
	lr, start, end int
}

// rows computes the visual rows for the current text and wrap width.
func (t *TextArea) rows() []visRow {
	var out []visRow
	for lr := range t.lines {
		for _, seg := range t.wrapSegments(t.lines[lr]) {
			out = append(out, visRow{lr: lr, start: seg[0], end: seg[1]})
		}
	}
	return out
}

// wrapSegments splits a logical line into [start,end) segments that each fit
// within wrapWidth, breaking at spaces where possible. With wrap off (or before
// the first draw) it returns the whole line as one segment.
func (t *TextArea) wrapSegments(line []rune) [][2]int {
	f := t.face()
	if !t.wrap || t.wrapWidth <= 0 || len(line) == 0 || f == nil {
		return [][2]int{{0, len(line)}}
	}
	var segs [][2]int
	start, lastSpace := 0, -1
	for i := 0; i < len(line); i++ {
		if line[i] == ' ' {
			lastSpace = i
		}
		if f.Measure(string(line[start:i+1])).W > t.wrapWidth && i > start {
			brk := i
			if lastSpace > start {
				brk = lastSpace + 1 // break just after the last space
			}
			segs = append(segs, [2]int{start, brk})
			start = brk
			lastSpace = -1
			i = start - 1 // re-examine from the new segment start
		}
	}
	return append(segs, [2]int{start, len(line)})
}

// caretVisualIndex returns the index in rows of the visual row holding the caret.
func (t *TextArea) caretVisualIndex(rows []visRow) int {
	for i, r := range rows {
		if r.lr != t.caretRow {
			continue
		}
		lastSeg := i+1 >= len(rows) || rows[i+1].lr != r.lr
		if t.caretCol < r.end || (t.caretCol == r.end && lastSeg) {
			return i
		}
	}
	for i := len(rows) - 1; i >= 0; i-- {
		if rows[i].lr == t.caretRow {
			return i
		}
	}
	return 0
}

// caretXIn returns the caret's x offset (from the content left edge) within row r.
func (t *TextArea) caretXIn(r visRow) float64 {
	f := t.face()
	if f == nil {
		return 0
	}
	return f.Measure(string(t.lines[r.lr][r.start:t.caretCol])).W
}

// TextAreaOption configures a TextArea.
type TextAreaOption func(*TextArea)

// TextAreaPlaceholder sets text shown when the area is empty and unfocused.
func TextAreaPlaceholder(s string) TextAreaOption {
	return func(t *TextArea) { t.placeholder = s }
}

// TextAreaWrap enables soft word-wrapping: long logical lines are wrapped to the
// content width across multiple visual rows.
func TextAreaWrap() TextAreaOption {
	return func(t *TextArea) { t.wrap = true }
}

// NewTextArea returns an empty TextArea configured by opts.
func NewTextArea(opts ...TextAreaOption) *TextArea {
	t := &TextArea{BaseWidget: NewBase(), lines: [][]rune{{}}}
	for _, o := range opts {
		o(t)
	}
	return t
}

// OnChange registers the handler invoked whenever the text changes.
func (t *TextArea) OnChange(fn func(string)) { t.onChange = fn }

// Text returns the full contents with lines joined by '\n'.
func (t *TextArea) Text() string {
	parts := make([]string, len(t.lines))
	for i, ln := range t.lines {
		parts[i] = string(ln)
	}
	return strings.Join(parts, "\n")
}

// SetText replaces the contents and moves the caret to the end.
func (t *TextArea) SetText(s string) {
	raw := strings.Split(s, "\n")
	t.lines = make([][]rune, len(raw))
	for i, ln := range raw {
		t.lines[i] = []rune(ln)
	}
	if len(t.lines) == 0 {
		t.lines = [][]rune{{}}
	}
	t.caretRow = len(t.lines) - 1
	t.caretCol = len(t.lines[t.caretRow])
	t.collapse()
	t.fireChange()
}

// --- rune-offset addressing (for find/replace and programmatic selection) ---
//
// Offsets index the text as returned by Text(): each '\n' between lines counts
// as one rune.

// posToOffset converts a (row, col) caret position to a rune offset.
func (t *TextArea) posToOffset(row, col int) int {
	off := 0
	for i := 0; i < row && i < len(t.lines); i++ {
		off += len(t.lines[i]) + 1 // +1 for the newline
	}
	return off + col
}

// offsetToPos converts a rune offset to a (row, col) position, clamped to range.
func (t *TextArea) offsetToPos(off int) (int, int) {
	if off < 0 {
		off = 0
	}
	for i := range t.lines {
		if off <= len(t.lines[i]) {
			return i, off
		}
		off -= len(t.lines[i]) + 1
	}
	last := len(t.lines) - 1
	return last, len(t.lines[last])
}

// CaretOffset returns the caret position as a rune offset into Text().
func (t *TextArea) CaretOffset() int { return t.posToOffset(t.caretRow, t.caretCol) }

// SelectionRange returns the current selection as rune offsets [start, end) into
// Text(); start == end when there is no selection.
func (t *TextArea) SelectionRange() (start, end int) {
	sr, sc, er, ec := t.selRange()
	return t.posToOffset(sr, sc), t.posToOffset(er, ec)
}

// SelectRange selects the text between rune offsets start and end (order
// independent) and places the caret at the end. The selected range is scrolled
// into view on the next frame.
func (t *TextArea) SelectRange(start, end int) {
	if start > end {
		start, end = end, start
	}
	sr, sc := t.offsetToPos(start)
	er, ec := t.offsetToPos(end)
	t.anchorRow, t.anchorCol = sr, sc
	t.caretRow, t.caretCol = er, ec
	t.Invalidate()
}

// Find searches for sub starting at rune offset from. If found, it selects the
// match and returns its offset and true; otherwise it returns -1 and false.
func (t *TextArea) Find(sub string, from int) (int, bool) {
	if sub == "" {
		return -1, false
	}
	hay := []rune(t.Text())
	needle := []rune(sub)
	idx := runeIndexFrom(hay, needle, from)
	if idx < 0 {
		return -1, false
	}
	t.SelectRange(idx, idx+len(needle))
	return idx, true
}

// runeIndexFrom returns the index of needle in hay at or after from, or -1.
func runeIndexFrom(hay, needle []rune, from int) int {
	if from < 0 {
		from = 0
	}
	for i := from; i+len(needle) <= len(hay); i++ {
		match := true
		for j := range needle {
			if hay[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

func (t *TextArea) face() render.FontFace {
	if t.font != nil {
		return t.font
	}
	return t.appTheme().Font
}

// SetFont overrides the area's font face (nil falls back to the theme font).
func (t *TextArea) SetFont(f render.FontFace) {
	t.font = f
	t.Invalidate()
}

// Focusable reports whether the area can take focus (only when enabled).
func (t *TextArea) Focusable() bool { return t.Enabled() }

func (t *TextArea) lineHeight() float64 {
	f := t.face()
	if f == nil {
		return 0
	}
	return f.Measure("Ag").H
}

// MinSize returns a default multi-line box size including padding.
func (t *TextArea) MinSize() geom.Size {
	return geom.Size{
		W: 200,
		H: t.lineHeight()*float64(defaultTextAreaRows) + textAreaPadding.Top + textAreaPadding.Bottom,
	}
}

func (t *TextArea) fireChange() {
	if t.onChange != nil {
		t.onChange(t.Text())
	}
}

func (t *TextArea) line() []rune { return t.lines[t.caretRow] }

// --- selection helpers ---

func (t *TextArea) collapse() {
	t.anchorRow, t.anchorCol = t.caretRow, t.caretCol
}

func (t *TextArea) hasSelection() bool {
	return t.anchorRow != t.caretRow || t.anchorCol != t.caretCol
}

// selRange returns the selection bounds in document order (sr,sc) .. (er,ec).
func (t *TextArea) selRange() (sr, sc, er, ec int) {
	if t.anchorRow < t.caretRow || (t.anchorRow == t.caretRow && t.anchorCol <= t.caretCol) {
		return t.anchorRow, t.anchorCol, t.caretRow, t.caretCol
	}
	return t.caretRow, t.caretCol, t.anchorRow, t.anchorCol
}

func (t *TextArea) deleteSelection() {
	sr, sc, er, ec := t.selRange()
	if sr == er {
		ln := t.lines[sr]
		t.lines[sr] = append(ln[:sc], ln[ec:]...)
	} else {
		head := append([]rune{}, t.lines[sr][:sc]...)
		merged := append(head, t.lines[er][ec:]...)
		t.lines = append(t.lines[:sr+1], t.lines[er+1:]...)
		t.lines[sr] = merged
	}
	t.caretRow, t.caretCol = sr, sc
	t.collapse()
}

func (t *TextArea) setCaret(row, col int, extend bool) {
	if row < 0 {
		row = 0
	}
	if row > len(t.lines)-1 {
		row = len(t.lines) - 1
	}
	if col < 0 {
		col = 0
	}
	if col > len(t.lines[row]) {
		col = len(t.lines[row])
	}
	t.caretRow, t.caretCol = row, col
	if !extend {
		t.collapse()
	}
}

func (t *TextArea) selectAll() {
	t.anchorRow, t.anchorCol = 0, 0
	t.caretRow = len(t.lines) - 1
	t.caretCol = len(t.lines[t.caretRow])
}

// --- editing ---

func (t *TextArea) insertRune(r rune) {
	if t.hasSelection() {
		t.deleteSelection()
	}
	ln := t.line()
	ln = append(ln, 0)
	copy(ln[t.caretCol+1:], ln[t.caretCol:])
	ln[t.caretCol] = r
	t.lines[t.caretRow] = ln
	t.caretCol++
	t.collapse()
	t.fireChange()
}

func (t *TextArea) insertNewline() {
	if t.hasSelection() {
		t.deleteSelection()
	}
	ln := t.line()
	tail := append([]rune{}, ln[t.caretCol:]...)
	t.lines[t.caretRow] = ln[:t.caretCol]
	t.lines = append(t.lines, nil)
	copy(t.lines[t.caretRow+2:], t.lines[t.caretRow+1:])
	t.lines[t.caretRow+1] = tail
	t.caretRow++
	t.caretCol = 0
	t.collapse()
	t.fireChange()
}

func (t *TextArea) deleteBack() {
	if t.hasSelection() {
		t.deleteSelection()
		t.fireChange()
		return
	}
	if t.caretCol > 0 {
		ln := t.line()
		t.lines[t.caretRow] = append(ln[:t.caretCol-1], ln[t.caretCol:]...)
		t.caretCol--
		t.collapse()
		t.fireChange()
		return
	}
	if t.caretRow > 0 {
		prev := t.lines[t.caretRow-1]
		t.caretCol = len(prev)
		t.lines[t.caretRow-1] = append(prev, t.line()...)
		t.lines = append(t.lines[:t.caretRow], t.lines[t.caretRow+1:]...)
		t.caretRow--
		t.collapse()
		t.fireChange()
	}
}

func (t *TextArea) deleteForward() {
	if t.hasSelection() {
		t.deleteSelection()
		t.fireChange()
		return
	}
	ln := t.line()
	if t.caretCol < len(ln) {
		t.lines[t.caretRow] = append(ln[:t.caretCol], ln[t.caretCol+1:]...)
		t.fireChange()
		return
	}
	if t.caretRow < len(t.lines)-1 {
		t.lines[t.caretRow] = append(ln, t.lines[t.caretRow+1]...)
		t.lines = append(t.lines[:t.caretRow+1], t.lines[t.caretRow+2:]...)
		t.fireChange()
	}
}

// selectedText returns the text within the current selection.
func (t *TextArea) selectedText() string {
	if !t.hasSelection() {
		return ""
	}
	sr, sc, er, ec := t.selRange()
	if sr == er {
		return string(t.lines[sr][sc:ec])
	}
	var b strings.Builder
	b.WriteString(string(t.lines[sr][sc:]))
	for i := sr + 1; i < er; i++ {
		b.WriteByte('\n')
		b.WriteString(string(t.lines[i]))
	}
	b.WriteByte('\n')
	b.WriteString(string(t.lines[er][:ec]))
	return b.String()
}

// insertString inserts s (which may contain newlines) at the caret, replacing
// any selection.
func (t *TextArea) insertString(s string) {
	if s == "" {
		return
	}
	if t.hasSelection() {
		t.deleteSelection()
	}
	parts := strings.Split(s, "\n")
	if len(parts) == 1 {
		rs := []rune(parts[0])
		ln := t.line()
		out := make([]rune, 0, len(ln)+len(rs))
		out = append(out, ln[:t.caretCol]...)
		out = append(out, rs...)
		out = append(out, ln[t.caretCol:]...)
		t.lines[t.caretRow] = out
		t.caretCol += len(rs)
	} else {
		ln := t.line()
		tail := append([]rune{}, ln[t.caretCol:]...)
		first := append([]rune{}, ln[:t.caretCol]...)
		first = append(first, []rune(parts[0])...)
		t.lines[t.caretRow] = first

		newLines := make([][]rune, len(parts)-1)
		for i := 1; i < len(parts); i++ {
			newLines[i-1] = []rune(parts[i])
		}
		lastIdx := len(newLines) - 1
		finalCol := len(newLines[lastIdx])
		newLines[lastIdx] = append(newLines[lastIdx], tail...)

		rest := append([][]rune{}, t.lines[t.caretRow+1:]...)
		t.lines = append(t.lines[:t.caretRow+1], newLines...)
		t.lines = append(t.lines, rest...)
		t.caretRow += len(parts) - 1
		t.caretCol = finalCol
	}
	t.collapse()
	t.fireChange()
}

func (t *TextArea) copySelection() {
	if cb := t.clipboard(); cb != nil && t.hasSelection() {
		cb.WriteText(t.selectedText())
	}
}

func (t *TextArea) cutSelection() {
	if !t.hasSelection() {
		return
	}
	t.copySelection()
	t.deleteSelection()
	t.fireChange()
}

func (t *TextArea) paste() {
	if cb := t.clipboard(); cb != nil {
		t.insertString(cb.ReadText())
	}
}

// caretAt maps an absolute point to a (row, col) caret position via the visual
// rows.
func (t *TextArea) caretAt(p geom.Point) (int, int) {
	inner := t.Bounds().Inset(textAreaPadding)
	lh := t.lineHeight()
	rows := t.rows()
	vi := 0
	if lh > 0 {
		vi = int((p.Y - inner.Y + t.scrollY) / lh)
	}
	if vi < 0 {
		vi = 0
	}
	if vi > len(rows)-1 {
		vi = len(rows) - 1
	}
	r := rows[vi]
	col := r.start + caretIndexForX(t.lines[r.lr][r.start:r.end], p.X-inner.X, t.face())
	return r.lr, col
}

// Draw renders the box, the selection, the visible lines (or placeholder), the
// caret, and a scrollbar thumb when the content overflows.
func (t *TextArea) Draw(canvas render.Canvas) {
	f := t.face()
	if f == nil {
		return
	}
	b := t.Bounds()
	rad := t.cornerRadius()
	canvas.FillRoundRect(b, rad, t.ColorOf(RoleSurface))
	border := t.ColorOf(RoleBorder)
	if t.focused {
		border = t.ColorOf(RoleAccent)
	}
	canvas.StrokeRoundRect(b, rad, border, 1)

	inner := b.Inset(textAreaPadding)
	lh := t.lineHeight()

	if t.isEmpty() && !t.focused && t.placeholder != "" {
		canvas.PushClip(inner)
		canvas.DrawText(t.placeholder, geom.Point{X: inner.X, Y: inner.Y}, f, t.ColorOf(RoleTextMuted))
		canvas.PopClip()
		return
	}

	if t.wrap {
		t.wrapWidth = inner.W
	}
	t.scrollCaretIntoView(inner.H)

	rows := t.rows()
	canvas.PushClip(inner)
	t.drawSelectionRows(canvas, inner, lh, rows, t.ColorOf(RolePrimary))
	for vi, r := range rows {
		y := inner.Y + float64(vi)*lh - t.scrollY
		if y+lh < inner.Y || y > inner.Y+inner.H {
			continue
		}
		canvas.DrawText(string(t.lines[r.lr][r.start:r.end]), geom.Point{X: inner.X, Y: y}, f, t.ColorOf(RoleText))
	}
	if t.focused {
		ci := t.caretVisualIndex(rows)
		r := rows[ci]
		caretX := inner.X + f.Measure(string(t.lines[r.lr][r.start:t.caretCol])).W
		caretY := inner.Y + float64(ci)*lh - t.scrollY
		canvas.DrawLine(
			geom.Point{X: caretX, Y: caretY + 1},
			geom.Point{X: caretX, Y: caretY + lh - 1},
			t.ColorOf(RoleText), 1,
		)
	}
	canvas.PopClip()

	if t.contentHeight() > inner.H {
		t.drawScrollbar(canvas, b, inner.H)
	}
}

// drawSelectionRows fills the selection highlight, clipped to each visual row.
func (t *TextArea) drawSelectionRows(canvas render.Canvas, inner geom.Rect, lh float64, rows []visRow, col color.Color) {
	if !t.hasSelection() {
		return
	}
	f := t.face()
	sr, sc, er, ec := t.selRange()
	for vi, r := range rows {
		if r.lr < sr || r.lr > er {
			continue
		}
		y := inner.Y + float64(vi)*lh - t.scrollY
		if y+lh < inner.Y || y > inner.Y+inner.H {
			continue
		}
		// Selected column span within this logical line.
		selStart, selEnd := 0, len(t.lines[r.lr])
		if r.lr == sr {
			selStart = sc
		}
		if r.lr == er {
			selEnd = ec
		}
		a, bcol := maxI(selStart, r.start), minI(selEnd, r.end)
		if a >= bcol {
			continue
		}
		x0 := inner.X + f.Measure(string(t.lines[r.lr][r.start:a])).W
		x1 := inner.X + f.Measure(string(t.lines[r.lr][r.start:bcol])).W
		if selEnd > r.end {
			x1 += 4 // selection continues past this row (newline or wrap)
		}
		canvas.FillRect(geom.Rect{X: x0, Y: y, W: x1 - x0, H: lh}, col)
	}
}

func (t *TextArea) isEmpty() bool {
	return len(t.lines) == 1 && len(t.lines[0]) == 0
}

func (t *TextArea) contentHeight() float64 { return float64(len(t.rows())) * t.lineHeight() }

func (t *TextArea) maxScroll(viewH float64) float64 {
	return maxF(0, t.contentHeight()-viewH)
}

// scrollCaretIntoView adjusts scrollY so the caret's visual row is visible.
func (t *TextArea) scrollCaretIntoView(viewH float64) {
	lh := t.lineHeight()
	ci := t.caretVisualIndex(t.rows())
	top := float64(ci) * lh
	bottom := top + lh
	if top < t.scrollY {
		t.scrollY = top
	} else if bottom > t.scrollY+viewH {
		t.scrollY = bottom - viewH
	}
	t.clampScroll(viewH)
}

func maxI(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minI(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (t *TextArea) clampScroll(viewH float64) {
	if t.scrollY < 0 {
		t.scrollY = 0
	}
	if m := t.maxScroll(viewH); t.scrollY > m {
		t.scrollY = m
	}
}

func (t *TextArea) drawScrollbar(canvas render.Canvas, b geom.Rect, viewH float64) {
	ch := t.contentHeight()
	thumbH := maxF(minThumb, viewH*viewH/ch)
	var off float64
	if m := t.maxScroll(viewH); m > 0 {
		off = (t.scrollY / m) * (b.H - thumbH)
	}
	x := b.X + b.W - scrollbarWidth
	canvas.FillRect(geom.Rect{X: x, Y: b.Y, W: scrollbarWidth, H: b.H}, t.ColorOf(RoleBackground))
	canvas.FillRect(geom.Rect{X: x, Y: b.Y + off, W: scrollbarWidth, H: thumbH}, t.ColorOf(RoleAccent))
}

// HandleEvent edits the text, manages the selection and scrolls in response to
// input.
func (t *TextArea) HandleEvent(ev *Event) bool {
	if !t.Enabled() {
		return false
	}
	switch ev.Type {
	case EventPointerEnter:
		t.hover = true
		return true
	case EventPointerLeave:
		t.hover = false
		return true
	case EventPointerDown:
		r, c := t.caretAt(ev.Pos)
		t.caretRow, t.caretCol = r, c
		t.collapse()
		t.dragging = true
		return true
	case EventPointerMove:
		if t.dragging {
			r, c := t.caretAt(ev.Pos)
			t.caretRow, t.caretCol = r, c // extend (anchor stays)
			return true
		}
	case EventPointerUp:
		t.dragging = false
		return true
	case EventFocusGained:
		t.focused = true
		return true
	case EventFocusLost:
		t.focused = false
		t.dragging = false
		return true
	case EventWheel:
		t.scrollY -= ev.Wheel.Y * wheelStep
		t.clampScroll(t.Bounds().Inset(textAreaPadding).H)
		return true
	case EventText:
		if ev.Rune >= 0x20 && ev.Rune != 0x7f {
			t.insertRune(ev.Rune)
			return true
		}
	case EventKeyDown:
		return t.handleKey(ev.Key, ev.Modifiers)
	}
	return false
}

func (t *TextArea) handleKey(k render.Key, mods render.ModifierSet) bool {
	extend := mods.Has(render.ModShift)
	if mods.Has(render.ModControl) {
		switch k {
		case render.KeyA:
			t.selectAll()
			return true
		case render.KeyC:
			t.copySelection()
			return true
		case render.KeyX:
			t.cutSelection()
			return true
		case render.KeyV:
			t.paste()
			return true
		}
	}
	switch k {
	case render.KeyEnter:
		t.insertNewline()
	case render.KeyBackspace:
		t.deleteBack()
	case render.KeyDelete:
		t.deleteForward()
	case render.KeyLeft:
		t.moveLeft(extend)
	case render.KeyRight:
		t.moveRight(extend)
	case render.KeyUp:
		t.moveVertical(-1, extend)
	case render.KeyDown:
		t.moveVertical(1, extend)
	case render.KeyHome:
		t.setCaret(t.caretRow, 0, extend)
	case render.KeyEnd:
		t.setCaret(t.caretRow, len(t.line()), extend)
	default:
		return false
	}
	return true
}

func (t *TextArea) moveLeft(extend bool) {
	if !extend && t.hasSelection() {
		sr, sc, _, _ := t.selRange()
		t.caretRow, t.caretCol = sr, sc
		t.collapse()
		return
	}
	if t.caretCol > 0 {
		t.setCaret(t.caretRow, t.caretCol-1, extend)
	} else if t.caretRow > 0 {
		t.setCaret(t.caretRow-1, len(t.lines[t.caretRow-1]), extend)
	}
}

func (t *TextArea) moveRight(extend bool) {
	if !extend && t.hasSelection() {
		_, _, er, ec := t.selRange()
		t.caretRow, t.caretCol = er, ec
		t.collapse()
		return
	}
	if t.caretCol < len(t.line()) {
		t.setCaret(t.caretRow, t.caretCol+1, extend)
	} else if t.caretRow < len(t.lines)-1 {
		t.setCaret(t.caretRow+1, 0, extend)
	}
}

// moveVertical moves the caret up or down by one visual row, preserving its x
// position. With wrap off this is equivalent to moving by logical line.
func (t *TextArea) moveVertical(delta int, extend bool) {
	rows := t.rows()
	ci := t.caretVisualIndex(rows)
	x := t.caretXIn(rows[ci])
	ni := ci + delta
	if ni < 0 || ni >= len(rows) {
		return
	}
	r := rows[ni]
	col := r.start + caretIndexForX(t.lines[r.lr][r.start:r.end], x, t.face())
	t.setCaret(r.lr, col, extend)
}
