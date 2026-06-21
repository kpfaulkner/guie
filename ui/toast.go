package ui

import (
	"image/color"
	"strings"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
)

// toast layout & timing constants.
const (
	toastPad         = 10  // inner padding around the message
	toastGap         = 8   // vertical gap between stacked toasts
	toastMargin      = 16  // distance from the surface edges
	toastMaxWidth    = 360 // toasts wider than this are not produced (text is not wrapped)
	toastFadeIn      = 0.18
	toastFadeOut     = 0.40
	toastDefaultSecs = 3.0
)

// ToastKind selects a toast's accent color and conveys severity.
type ToastKind int

const (
	// ToastInfo is a neutral, informational toast (uses the theme primary).
	ToastInfo ToastKind = iota
	// ToastSuccess indicates an operation succeeded (green).
	ToastSuccess
	// ToastWarning indicates something needs attention (amber).
	ToastWarning
	// ToastError indicates an operation failed (red).
	ToastError
)

// Toast is a transient, non-interactive notification shown by the App. It fades
// in, holds, then fades out and removes itself after its duration. Obtain one
// from App.ShowToast; call Dismiss to remove it early (with a fade-out).
type Toast struct {
	message  string
	kind     ToastKind
	duration float64
	elapsed  float64
}

// ToastOption configures a toast at creation.
type ToastOption func(*Toast)

// ToastDuration sets how long (in seconds) the toast is shown before it fades
// out. Values <= 0 fall back to the default.
func ToastDuration(seconds float64) ToastOption {
	return func(t *Toast) {
		if seconds > 0 {
			t.duration = seconds
		}
	}
}

// WithToastKind sets the toast's severity/color.
func WithToastKind(k ToastKind) ToastOption { return func(t *Toast) { t.kind = k } }

// ShowToast displays a transient notification with the given message and returns
// its handle. It is safe to call from event handlers and frame callbacks.
func (a *App) ShowToast(message string, opts ...ToastOption) *Toast {
	t := &Toast{message: message, kind: ToastInfo, duration: toastDefaultSecs}
	for _, o := range opts {
		o(t)
	}
	a.toasts = append(a.toasts, t)
	return t
}

// Dismiss removes the toast early, fading it out rather than cutting it abruptly.
func (t *Toast) Dismiss() {
	if end := t.elapsed + toastFadeOut; end < t.duration {
		t.duration = end
	}
}

// alpha returns the toast's current opacity in [0,1] from its fade-in/out ramps.
func (t *Toast) alpha() float64 {
	if t.elapsed < toastFadeIn {
		return t.elapsed / toastFadeIn
	}
	if rem := t.duration - t.elapsed; rem < toastFadeOut {
		return maxF(0, rem/toastFadeOut)
	}
	return 1
}

// advanceToasts ages the active toasts by dt and drops any that have expired.
// Called once per frame from update.
func (a *App) advanceToasts(dt float64) {
	if len(a.toasts) == 0 {
		return
	}
	kept := a.toasts[:0]
	for _, t := range a.toasts {
		t.elapsed += dt
		if t.elapsed < t.duration {
			kept = append(kept, t)
		}
	}
	// Zero out the tail so dropped toasts aren't retained by the backing array.
	for i := len(kept); i < len(a.toasts); i++ {
		a.toasts[i] = nil
	}
	a.toasts = kept
}

// toastColors returns the background, border and text colors for a kind, using
// the theme where it makes sense so info toasts follow the palette.
func (a *App) toastColors(k ToastKind) (bg, border, text color.Color) {
	pal := a.theme.Palette
	switch k {
	case ToastSuccess:
		return color.NRGBA{R: 0x2E, G: 0x7D, B: 0x32, A: 0xff}, color.NRGBA{R: 0x1B, G: 0x5E, B: 0x20, A: 0xff}, color.White
	case ToastWarning:
		return color.NRGBA{R: 0xB7, G: 0x7A, B: 0x00, A: 0xff}, color.NRGBA{R: 0x8C, G: 0x5E, B: 0x00, A: 0xff}, color.White
	case ToastError:
		return color.NRGBA{R: 0xC6, G: 0x28, B: 0x28, A: 0xff}, color.NRGBA{R: 0x99, G: 0x1B, B: 0x1B, A: 0xff}, color.White
	default: // ToastInfo
		return pal.Primary, pal.Border, pal.OnPrimary
	}
}

// drawToasts paints the active toasts stacked from the bottom-right corner
// upward (newest at the bottom), each at its current opacity. Toasts are not
// hit-tested, so pointer input passes through to the widgets beneath them.
func (a *App) drawToasts(c render.Canvas) {
	f := a.theme.Font
	if f == nil || len(a.toasts) == 0 {
		return
	}
	rad := a.theme.CornerRadius
	y := a.surfaceSize.H - toastMargin
	for i := len(a.toasts) - 1; i >= 0; i-- {
		t := a.toasts[i]
		sz := toastTextSize(f, t.message)
		w := sz.W + 2*toastPad
		h := sz.H + 2*toastPad
		x := a.surfaceSize.W - toastMargin - w
		top := y - h
		rect := geom.Rect{X: x, Y: top, W: w, H: h}

		al := t.alpha()
		bg, border, text := a.toastColors(t.kind)
		if a.shadows && al > 0 {
			drawShadow(c, rect, rad)
		}
		c.FillRoundRect(rect, rad, withAlpha(bg, al))
		c.StrokeRoundRect(rect, rad, withAlpha(border, al), 1)
		inner := geom.Rect{X: x + toastPad, Y: top + toastPad, W: sz.W, H: sz.H}
		drawText(c, t.message, inner, geom.AlignStart, f, withAlpha(text, al))

		y = top - toastGap
	}
}

// toastTextSize measures a (possibly multi-line) message: widest line by the
// total block height.
func toastTextSize(f render.FontFace, s string) geom.Size {
	m := f.Metrics()
	lines := strings.Split(s, "\n")
	var w float64
	for _, ln := range lines {
		w = maxF(w, f.Measure(ln).W)
	}
	if w > toastMaxWidth {
		w = toastMaxWidth
	}
	h := m.LineHeight*float64(len(lines)-1) + m.Ascent + m.Descent
	return geom.Size{W: w, H: h}
}

// withAlpha returns c scaled to the given opacity in [0,1]. a >= 1 returns c
// unchanged; values below 0 are treated as fully transparent.
func withAlpha(c color.Color, a float64) color.Color {
	if a >= 1 {
		return c
	}
	if a < 0 {
		a = 0
	}
	nc := color.NRGBAModel.Convert(c).(color.NRGBA)
	nc.A = uint8(float64(nc.A) * a)
	return nc
}
