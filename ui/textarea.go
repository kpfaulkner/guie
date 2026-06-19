package ui

import (
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
// Lines are split on '\n'; there is no soft word-wrap or horizontal scrolling
// yet, so text past the right edge of a long line is clipped.
type TextArea struct {
	BaseWidget
	lines    [][]rune // text split into lines
	caretRow int
	caretCol int
	focused  bool
	hover    bool
	scrollY  float64

	placeholder string
	onChange    func(string)

	font render.FontFace
}

// TextAreaOption configures a TextArea.
type TextAreaOption func(*TextArea)

// TextAreaPlaceholder sets text shown when the area is empty and unfocused.
func TextAreaPlaceholder(s string) TextAreaOption {
	return func(t *TextArea) { t.placeholder = s }
}

// OnTextAreaChange registers a handler called whenever the text changes.
func OnTextAreaChange(fn func(string)) TextAreaOption {
	return func(t *TextArea) { t.onChange = fn }
}

// NewTextArea returns an empty TextArea configured by opts.
func NewTextArea(opts ...TextAreaOption) *TextArea {
	t := &TextArea{BaseWidget: NewBase(), lines: [][]rune{{}}}
	for _, o := range opts {
		o(t)
	}
	return t
}

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
	t.fireChange()
}

func (t *TextArea) face() render.FontFace {
	if t.font != nil {
		return t.font
	}
	return t.appTheme().Font
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

func (t *TextArea) insertRune(r rune) {
	ln := t.line()
	ln = append(ln, 0)
	copy(ln[t.caretCol+1:], ln[t.caretCol:])
	ln[t.caretCol] = r
	t.lines[t.caretRow] = ln
	t.caretCol++
	t.fireChange()
}

func (t *TextArea) insertNewline() {
	ln := t.line()
	tail := append([]rune{}, ln[t.caretCol:]...)
	t.lines[t.caretRow] = ln[:t.caretCol]
	// insert the tail as a new line after the current row
	t.lines = append(t.lines, nil)
	copy(t.lines[t.caretRow+2:], t.lines[t.caretRow+1:])
	t.lines[t.caretRow+1] = tail
	t.caretRow++
	t.caretCol = 0
	t.fireChange()
}

func (t *TextArea) deleteBack() {
	if t.caretCol > 0 {
		ln := t.line()
		t.lines[t.caretRow] = append(ln[:t.caretCol-1], ln[t.caretCol:]...)
		t.caretCol--
		t.fireChange()
		return
	}
	if t.caretRow > 0 {
		prev := t.lines[t.caretRow-1]
		t.caretCol = len(prev)
		t.lines[t.caretRow-1] = append(prev, t.line()...)
		t.lines = append(t.lines[:t.caretRow], t.lines[t.caretRow+1:]...)
		t.caretRow--
		t.fireChange()
	}
}

func (t *TextArea) deleteForward() {
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

func (t *TextArea) clampCol() {
	if t.caretCol > len(t.line()) {
		t.caretCol = len(t.line())
	}
}

// Draw renders the box, the visible lines (or placeholder), the caret, and a
// scrollbar thumb when the content overflows.
func (t *TextArea) Draw(canvas render.Canvas) {
	pal := t.appTheme().Palette
	f := t.face()
	if f == nil {
		return
	}
	b := t.Bounds()
	canvas.FillRect(b, pal.Surface)
	border := pal.Border
	if t.focused {
		border = pal.Accent
	}
	canvas.StrokeRect(b, border, 1)

	inner := b.Inset(textAreaPadding)
	lh := t.lineHeight()

	if t.isEmpty() && !t.focused && t.placeholder != "" {
		canvas.PushClip(inner)
		canvas.DrawText(t.placeholder, geom.Point{X: inner.X, Y: inner.Y}, f, pal.TextMuted)
		canvas.PopClip()
		return
	}

	t.scrollCaretIntoView(inner.H)

	canvas.PushClip(inner)
	for i, ln := range t.lines {
		y := inner.Y + float64(i)*lh - t.scrollY
		if y+lh < inner.Y || y > inner.Y+inner.H {
			continue
		}
		canvas.DrawText(string(ln), geom.Point{X: inner.X, Y: y}, f, pal.Text)
	}
	if t.focused {
		caretX := inner.X + f.Measure(string(t.line()[:t.caretCol])).W
		caretY := inner.Y + float64(t.caretRow)*lh - t.scrollY
		canvas.DrawLine(
			geom.Point{X: caretX, Y: caretY + 1},
			geom.Point{X: caretX, Y: caretY + lh - 1},
			pal.Text, 1,
		)
	}
	canvas.PopClip()

	if t.contentHeight() > inner.H {
		t.drawScrollbar(canvas, b, inner.H)
	}
}

func (t *TextArea) isEmpty() bool {
	return len(t.lines) == 1 && len(t.lines[0]) == 0
}

func (t *TextArea) contentHeight() float64 { return float64(len(t.lines)) * t.lineHeight() }

func (t *TextArea) maxScroll(viewH float64) float64 {
	return maxF(0, t.contentHeight()-viewH)
}

// scrollCaretIntoView adjusts scrollY so the caret line is visible.
func (t *TextArea) scrollCaretIntoView(viewH float64) {
	lh := t.lineHeight()
	top := float64(t.caretRow) * lh
	bottom := top + lh
	if top < t.scrollY {
		t.scrollY = top
	} else if bottom > t.scrollY+viewH {
		t.scrollY = bottom - viewH
	}
	t.clampScroll(viewH)
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
	pal := t.appTheme().Palette
	ch := t.contentHeight()
	thumbH := maxF(minThumb, viewH*viewH/ch)
	var off float64
	if m := t.maxScroll(viewH); m > 0 {
		off = (t.scrollY / m) * (b.H - thumbH)
	}
	x := b.X + b.W - scrollbarWidth
	canvas.FillRect(geom.Rect{X: x, Y: b.Y, W: scrollbarWidth, H: b.H}, pal.Background)
	canvas.FillRect(geom.Rect{X: x, Y: b.Y + off, W: scrollbarWidth, H: thumbH}, pal.Accent)
}

// HandleEvent edits the text and scrolls in response to input.
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
	case EventFocusGained:
		t.focused = true
		return true
	case EventFocusLost:
		t.focused = false
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
		return t.handleKey(ev.Key)
	}
	return false
}

func (t *TextArea) handleKey(k render.Key) bool {
	switch k {
	case render.KeyEnter:
		t.insertNewline()
	case render.KeyBackspace:
		t.deleteBack()
	case render.KeyDelete:
		t.deleteForward()
	case render.KeyLeft:
		t.moveLeft()
	case render.KeyRight:
		t.moveRight()
	case render.KeyUp:
		if t.caretRow > 0 {
			t.caretRow--
			t.clampCol()
		}
	case render.KeyDown:
		if t.caretRow < len(t.lines)-1 {
			t.caretRow++
			t.clampCol()
		}
	case render.KeyHome:
		t.caretCol = 0
	case render.KeyEnd:
		t.caretCol = len(t.line())
	default:
		return false
	}
	return true
}

func (t *TextArea) moveLeft() {
	if t.caretCol > 0 {
		t.caretCol--
	} else if t.caretRow > 0 {
		t.caretRow--
		t.caretCol = len(t.line())
	}
}

func (t *TextArea) moveRight() {
	if t.caretCol < len(t.line()) {
		t.caretCol++
	} else if t.caretRow < len(t.lines)-1 {
		t.caretRow++
		t.caretCol = 0
	}
}
