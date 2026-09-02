package ui

import (
	"strings"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
)

// textFieldPadding is the inner padding around a text field's content.
var textFieldPadding = geom.Insets{Top: 5, Right: 8, Bottom: 5, Left: 8}

// maskRune is the glyph a masked field shows in place of each real rune. It is
// not configurable: a MaskRune option can be added later without breaking
// anything, whereas an option shipped now would be frozen.
const maskRune = '•' // BULLET

// caretIndexForX returns the caret index in line whose boundary is nearest the
// local x coordinate (x measured from the start of the text).
func caretIndexForX(line []rune, x float64, f render.FontFace) int {
	if x <= 0 || f == nil {
		return 0
	}
	prev := 0.0
	for i := 1; i <= len(line); i++ {
		w := f.Measure(string(line[:i])).W
		if x < (prev+w)/2 {
			return i - 1
		}
		prev = w
	}
	return len(line)
}

// TextField is a single-line text input. It is focusable and edits its content
// in response to typed runes and editing keys (Backspace, Delete, Left/Right,
// Home/End). Enter invokes OnSubmit. The view scrolls horizontally to keep the
// caret visible.
//
// It supports a selection: click and drag, or hold Shift while moving the
// caret, to select a range; Ctrl+A selects all. Typing, Backspace/Delete and
// inserting replace the selection.
type TextField struct {
	BaseWidget
	runes    []rune
	caret    int // caret index (0..len)
	anchor   int // selection anchor; equal to caret means no selection
	focused  bool
	hover    bool
	dragging bool
	scrollX  float64

	preedit      string // IME composition (uncommitted) shown inline at the caret
	preeditCaret int    // caret position within preedit, in runes

	masked      bool // display mask runes instead of the real text
	placeholder string
	onChange    func(string)
	onSubmit    func(string)

	font render.FontFace
}

// TextFieldOption configures a TextField.
type TextFieldOption func(*TextField)

// Placeholder sets text shown when the field is empty and unfocused.
func Placeholder(s string) TextFieldOption { return func(t *TextField) { t.placeholder = s } }

// Masked constructs the field showing one mask glyph per character instead of
// its contents, for passwords, API keys and other credentials. Text and
// OnChange still carry the real text: masking is presentation, not storage.
func Masked() TextFieldOption { return func(t *TextField) { t.masked = true } }

// NewTextField returns an empty TextField.
func NewTextField(opts ...TextFieldOption) *TextField {
	t := &TextField{BaseWidget: NewBase()}
	for _, o := range opts {
		o(t)
	}
	return t
}

// OnChange registers the handler invoked whenever the text changes.
func (t *TextField) OnChange(fn func(string)) { t.onChange = fn }

// OnSubmit registers the handler invoked when Enter is pressed.
func (t *TextField) OnSubmit(fn func(string)) { t.onSubmit = fn }

// Text returns the current contents.
func (t *TextField) Text() string { return string(t.runes) }

// SetText replaces the contents and moves the caret to the end.
func (t *TextField) SetText(s string) {
	t.runes = []rune(s)
	t.caret = len(t.runes)
	t.anchor = t.caret
	t.fireChange()
}

func (t *TextField) face() render.FontFace {
	if t.font != nil {
		return t.font
	}
	return t.appTheme().Font
}

// SetFont overrides the field's font face (nil falls back to the theme font).
func (t *TextField) SetFont(f render.FontFace) {
	t.font = f
	t.Invalidate()
}

// SetMasked turns masking on or off at runtime, so an app can offer a "Show"
// toggle next to a credential field.
func (t *TextField) SetMasked(v bool) {
	t.masked = v
	t.Invalidate()
}

// maskRunes returns one mask rune per rune in rs while masked, otherwise rs
// itself. The result must not be mutated: unmasked it aliases rs.
func (t *TextField) maskRunes(rs []rune) []rune {
	if !t.masked || len(rs) == 0 {
		return rs
	}
	out := make([]rune, len(rs))
	for i := range out {
		out[i] = maskRune
	}
	return out
}

// display returns what the field shows: the real runes, or one mask rune per
// real rune when masked.
//
// Contract: display() returns exactly len(t.runes) runes. That 1:1 rule is why
// masking is a display-only change - every caret index, selection bound and
// scroll offset stays valid with no index translation anywhere. A change that
// collapses or pads the mask breaks the caret silently.
func (t *TextField) display() []rune { return t.maskRunes(t.runes) }

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

// --- selection helpers ---

func (t *TextField) hasSelection() bool { return t.anchor != t.caret }

// selRange returns the selection bounds [lo, hi) in order.
func (t *TextField) selRange() (lo, hi int) {
	if t.anchor <= t.caret {
		return t.anchor, t.caret
	}
	return t.caret, t.anchor
}

func (t *TextField) deleteSelection() {
	lo, hi := t.selRange()
	t.runes = append(t.runes[:lo], t.runes[hi:]...)
	t.caret = lo
	t.anchor = lo
}

// setCaret moves the caret, optionally extending the selection (keeping the
// anchor); otherwise it collapses the selection.
func (t *TextField) setCaret(pos int, extend bool) {
	if pos < 0 {
		pos = 0
	}
	if pos > len(t.runes) {
		pos = len(t.runes)
	}
	t.caret = pos
	if !extend {
		t.anchor = pos
	}
}

func (t *TextField) selectAll() {
	t.anchor = 0
	t.caret = len(t.runes)
}

// --- editing ---

func (t *TextField) insert(r rune) {
	if t.hasSelection() {
		t.deleteSelection()
	}
	t.runes = append(t.runes, 0)
	copy(t.runes[t.caret+1:], t.runes[t.caret:])
	t.runes[t.caret] = r
	t.caret++
	t.anchor = t.caret
	t.fireChange()
}

func (t *TextField) deleteBack() {
	if t.hasSelection() {
		t.deleteSelection()
		t.fireChange()
		return
	}
	if t.caret == 0 {
		return
	}
	t.runes = append(t.runes[:t.caret-1], t.runes[t.caret:]...)
	t.caret--
	t.anchor = t.caret
	t.fireChange()
}

func (t *TextField) deleteForward() {
	if t.hasSelection() {
		t.deleteSelection()
		t.fireChange()
		return
	}
	if t.caret >= len(t.runes) {
		return
	}
	t.runes = append(t.runes[:t.caret], t.runes[t.caret+1:]...)
	t.fireChange()
}

// insertString inserts s at the caret, replacing any selection.
func (t *TextField) insertString(s string) {
	if s == "" {
		return
	}
	if t.hasSelection() {
		t.deleteSelection()
	}
	rs := []rune(s)
	out := make([]rune, 0, len(t.runes)+len(rs))
	out = append(out, t.runes[:t.caret]...)
	out = append(out, rs...)
	out = append(out, t.runes[t.caret:]...)
	t.runes = out
	t.caret += len(rs)
	t.anchor = t.caret
	t.fireChange()
}

func (t *TextField) selectedText() string {
	lo, hi := t.selRange()
	return string(t.runes[lo:hi])
}

func (t *TextField) copySelection() {
	if t.masked {
		return
	}
	if cb := t.clipboard(); cb != nil && t.hasSelection() {
		cb.WriteText(t.selectedText())
	}
}

func (t *TextField) cutSelection() {
	// Cut is a no-op while masked, not "delete without copying": the whole
	// operation is suppressed, and revealing (SetMasked(false)) re-enables it.
	if t.masked {
		return
	}
	if !t.hasSelection() {
		return
	}
	t.copySelection()
	t.deleteSelection()
	t.fireChange()
}

func (t *TextField) paste() {
	cb := t.clipboard()
	if cb == nil {
		return
	}
	// A single-line field flattens any newlines from the pasted text.
	t.insertString(strings.ReplaceAll(cb.ReadText(), "\n", " "))
}

// caretIndexAt maps an absolute x coordinate to a caret index.
func (t *TextField) caretIndexAt(absX float64) int {
	inner := t.Bounds().Inset(textFieldPadding)
	return caretIndexForX(t.display(), absX-inner.X+t.scrollX, t.face())
}

// --- IME preedit ---

// composing reports whether an IME composition is active.
func (t *TextField) composing() bool { return t.preedit != "" }

// setPreedit stores the IME preedit. Starting a composition (first non-empty
// preedit) replaces any selection, so the eventual commit lands at the caret.
func (t *TextField) setPreedit(text string, caret int) {
	if text != "" && t.preedit == "" && t.hasSelection() {
		t.deleteSelection()
	}
	t.preedit = text
	n := len([]rune(text))
	if caret < 0 {
		caret = 0
	}
	if caret > n {
		caret = n
	}
	t.preeditCaret = caret
	t.Invalidate()
}

// caretVisualWidth returns the caret's x offset from the text origin, including
// the preedit prefix while composing.
func (t *TextField) caretVisualWidth(f render.FontFace) float64 {
	d := t.display()
	w := f.Measure(string(d[:t.caret])).W
	if t.composing() {
		pr := t.maskRunes([]rune(t.preedit))
		w += f.Measure(string(pr[:t.preeditCaret])).W
	}
	return w
}

// imeCaretRect reports the caret rectangle in absolute coordinates for IME
// candidate-window placement.
func (t *TextField) imeCaretRect() (geom.Rect, bool) {
	f := t.face()
	if !t.focused || f == nil {
		return geom.Rect{}, false
	}
	inner := t.Bounds().Inset(textFieldPadding)
	x := inner.X - t.scrollX + t.caretVisualWidth(f)
	return geom.Rect{X: x, Y: inner.Y, W: 1, H: inner.H}, true
}

// Draw renders the box, the selection highlight, the text (or placeholder) and
// the caret when focused.
func (t *TextField) Draw(canvas render.Canvas) {
	f := t.face()
	if f == nil {
		return
	}
	b := t.Bounds()
	rad := t.cornerRadius()

	canvas.FillRoundRect(b, rad, t.ColourOf(RoleSurface))
	border := t.ColourOf(RoleBorder)
	if t.focused {
		border = t.ColourOf(RoleAccent)
	}
	canvas.StrokeRoundRect(b, rad, border, 1)

	inner := b.Inset(textFieldPadding)
	canvas.PushClip(inner)
	defer canvas.PopClip()

	y := vCenterY(f, inner.Y, inner.H)

	if len(t.runes) == 0 && !t.composing() && !t.focused && t.placeholder != "" {
		canvas.DrawText(t.placeholder, geom.Point{X: inner.X, Y: y}, f, t.ColourOf(RoleTextMuted))
		return
	}

	t.updateScroll(f, inner.W)

	// While composing, the selection has already been removed (setPreedit), so
	// only draw the highlight when not composing.
	if t.hasSelection() && !t.composing() {
		lo, hi := t.selRange()
		d := t.display()
		x0 := inner.X - t.scrollX + f.Measure(string(d[:lo])).W
		x1 := inner.X - t.scrollX + f.Measure(string(d[:hi])).W
		canvas.FillRect(geom.Rect{X: x0, Y: inner.Y, W: x1 - x0, H: inner.H}, t.ColourOf(RolePrimary))
	}

	textColour := t.ColourOf(RoleText)
	originX := inner.X - t.scrollX
	if t.composing() {
		// Insert the preedit into the displayed text at the caret and underline
		// it. The preedit is masked too: a field that silently refused typed
		// input would look broken, whereas dots appearing as you compose look
		// like every other password field.
		d := t.display()
		pre := string(d[:t.caret])
		post := string(d[t.caret:])
		pe := string(t.maskRunes([]rune(t.preedit)))
		canvas.DrawText(pre+pe+post, geom.Point{X: originX, Y: y}, f, textColour)
		ux0 := originX + f.Measure(pre).W
		ux1 := originX + f.Measure(pre+pe).W
		uy := inner.Y + inner.H - 2
		canvas.DrawLine(geom.Point{X: ux0, Y: uy}, geom.Point{X: ux1, Y: uy}, textColour, 1)
	} else {
		canvas.DrawText(string(t.display()), geom.Point{X: originX, Y: y}, f, textColour)
	}

	if t.focused {
		caretX := originX + t.caretVisualWidth(f)
		canvas.DrawLine(
			geom.Point{X: caretX, Y: inner.Y + 2},
			geom.Point{X: caretX, Y: inner.Y + inner.H - 2},
			textColour, 1,
		)
	}
}

// updateScroll adjusts scrollX so the caret (including any preedit) stays within
// the visible width.
func (t *TextField) updateScroll(f render.FontFace, width float64) {
	caretW := t.caretVisualWidth(f)
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

// HandleEvent edits the text and manages the selection in response to input.
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
	case EventPointerDown:
		idx := t.caretIndexAt(ev.Pos.X)
		t.caret = idx
		t.anchor = idx
		t.dragging = true
		return true
	case EventPointerMove:
		if t.dragging {
			t.caret = t.caretIndexAt(ev.Pos.X) // extend (anchor stays)
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
		t.preedit = ""
		t.preeditCaret = 0
		return true
	case EventComposition:
		t.setPreedit(ev.Comp.Text, ev.Comp.Caret)
		return true
	case EventText:
		if ev.Rune >= 0x20 && ev.Rune != 0x7f {
			t.insert(ev.Rune)
			return true
		}
	case EventKeyDown:
		return t.handleKey(ev.Key, ev.Modifiers)
	}
	return false
}

func (t *TextField) handleKey(k render.Key, mods render.ModifierSet) bool {
	extend := mods.Has(render.ModShift)
	if mods.Has(render.ModPrimary) {
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
	case render.KeyBackspace:
		t.deleteBack()
	case render.KeyDelete:
		t.deleteForward()
	case render.KeyLeft:
		t.moveLeft(extend)
	case render.KeyRight:
		t.moveRight(extend)
	case render.KeyHome:
		t.setCaret(0, extend)
	case render.KeyEnd:
		t.setCaret(len(t.runes), extend)
	case render.KeyEnter:
		if t.onSubmit != nil {
			t.onSubmit(string(t.runes))
		}
	default:
		return false
	}
	return true
}

func (t *TextField) moveLeft(extend bool) {
	if !extend && t.hasSelection() {
		lo, _ := t.selRange()
		t.caret, t.anchor = lo, lo
		return
	}
	t.setCaret(t.caret-1, extend)
}

func (t *TextField) moveRight(extend bool) {
	if !extend && t.hasSelection() {
		_, hi := t.selRange()
		t.caret, t.anchor = hi, hi
		return
	}
	t.setCaret(t.caret+1, extend)
}
