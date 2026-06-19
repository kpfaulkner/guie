package ui

import (
	"github.com/kpfaulkner/uiframework/geom"
	"github.com/kpfaulkner/uiframework/render"
)

// textFieldPadding is the inner padding around a text field's content.
var textFieldPadding = geom.Insets{Top: 5, Right: 8, Bottom: 5, Left: 8}

// TextField is a single-line text input. It is focusable and edits its content
// in response to typed runes and the usual editing keys (Backspace, Delete,
// Left/Right, Home/End). Enter invokes OnSubmit. The view scrolls horizontally
// to keep the caret visible.
type TextField struct {
	BaseWidget
	runes   []rune
	caret   int // caret index in runes (0..len)
	focused bool
	hover   bool
	scrollX float64

	placeholder string
	onChange    func(string)
	onSubmit    func(string)

	font render.FontFace
}

// TextFieldOption configures a TextField.
type TextFieldOption func(*TextField)

// Placeholder sets text shown when the field is empty and unfocused.
func Placeholder(s string) TextFieldOption { return func(t *TextField) { t.placeholder = s } }

// OnTextChange registers a handler called whenever the text changes.
func OnTextChange(fn func(string)) TextFieldOption { return func(t *TextField) { t.onChange = fn } }

// OnSubmit registers a handler called when Enter is pressed.
func OnSubmit(fn func(string)) TextFieldOption { return func(t *TextField) { t.onSubmit = fn } }

// NewTextField returns an empty TextField.
func NewTextField(opts ...TextFieldOption) *TextField {
	t := &TextField{BaseWidget: NewBase()}
	for _, o := range opts {
		o(t)
	}
	return t
}

// Text returns the current contents.
func (t *TextField) Text() string { return string(t.runes) }

// SetText replaces the contents and moves the caret to the end.
func (t *TextField) SetText(s string) {
	t.runes = []rune(s)
	t.caret = len(t.runes)
	t.fireChange()
}

func (t *TextField) face() render.FontFace {
	if t.font != nil {
		return t.font
	}
	return t.appTheme().Font
}

// Focusable reports whether the field can take focus (only when enabled).
func (t *TextField) Focusable() bool { return t.Enabled() }

// MinSize returns one line tall and a default width, including padding.
func (t *TextField) MinSize() geom.Size {
	f := t.face()
	var h float64
	if f != nil {
		h = f.Measure("Ag").H
	}
	const defaultWidth = 160
	return geom.Size{
		W: defaultWidth,
		H: h + textFieldPadding.Top + textFieldPadding.Bottom,
	}
}

func (t *TextField) fireChange() {
	if t.onChange != nil {
		t.onChange(string(t.runes))
	}
}

func (t *TextField) insert(r rune) {
	t.runes = append(t.runes, 0)
	copy(t.runes[t.caret+1:], t.runes[t.caret:])
	t.runes[t.caret] = r
	t.caret++
	t.fireChange()
}

func (t *TextField) deleteBack() {
	if t.caret == 0 {
		return
	}
	t.runes = append(t.runes[:t.caret-1], t.runes[t.caret:]...)
	t.caret--
	t.fireChange()
}

func (t *TextField) deleteForward() {
	if t.caret >= len(t.runes) {
		return
	}
	t.runes = append(t.runes[:t.caret], t.runes[t.caret+1:]...)
	t.fireChange()
}

// Draw renders the box, the text (or placeholder), and the caret when focused.
func (t *TextField) Draw(canvas render.Canvas) {
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

	inner := b.Inset(textFieldPadding)
	canvas.PushClip(inner)
	defer canvas.PopClip()

	lineH := f.Measure("Ag").H
	y := inner.Y + (inner.H-lineH)/2

	if len(t.runes) == 0 && !t.focused && t.placeholder != "" {
		canvas.DrawText(t.placeholder, geom.Point{X: inner.X, Y: y}, f, pal.TextMuted)
		return
	}

	t.updateScroll(f, inner.W)
	canvas.DrawText(string(t.runes), geom.Point{X: inner.X - t.scrollX, Y: y}, f, pal.Text)

	if t.focused {
		caretX := inner.X - t.scrollX + f.Measure(string(t.runes[:t.caret])).W
		canvas.DrawLine(
			geom.Point{X: caretX, Y: inner.Y + 2},
			geom.Point{X: caretX, Y: inner.Y + inner.H - 2},
			pal.Text, 1,
		)
	}
}

// updateScroll adjusts scrollX so the caret stays within the visible width.
func (t *TextField) updateScroll(f render.FontFace, width float64) {
	caretW := f.Measure(string(t.runes[:t.caret])).W
	if caretW-t.scrollX > width {
		t.scrollX = caretW - width
	}
	if caretW-t.scrollX < 0 {
		t.scrollX = caretW
	}
	if t.scrollX < 0 {
		t.scrollX = 0
	}
}

// HandleEvent edits the text in response to keys and typed runes.
func (t *TextField) HandleEvent(ev *Event) bool {
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
		t.caret = len(t.runes)
		return true
	case EventFocusLost:
		t.focused = false
		return true
	case EventText:
		if ev.Rune >= 0x20 && ev.Rune != 0x7f {
			t.insert(ev.Rune)
			return true
		}
	case EventKeyDown:
		return t.handleKey(ev.Key)
	}
	return false
}

func (t *TextField) handleKey(k render.Key) bool {
	switch k {
	case render.KeyBackspace:
		t.deleteBack()
	case render.KeyDelete:
		t.deleteForward()
	case render.KeyLeft:
		if t.caret > 0 {
			t.caret--
		}
	case render.KeyRight:
		if t.caret < len(t.runes) {
			t.caret++
		}
	case render.KeyHome:
		t.caret = 0
	case render.KeyEnd:
		t.caret = len(t.runes)
	case render.KeyEnter:
		if t.onSubmit != nil {
			t.onSubmit(string(t.runes))
		}
	default:
		return false
	}
	return true
}
