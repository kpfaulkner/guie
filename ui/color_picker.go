package ui

import (
	"fmt"
	"image/color"
	"math"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
)

// color picker layout constants.
const (
	cpPad      = 6
	cpSwatchH  = 36
	cpTrackH   = 18
	cpGap      = 8
	cpChannels = 3 // hue, saturation, value
)

// cpKeyStep is the keyboard adjustment per Left/Right press.
const cpKeyStep = 0.02

// ColorPicker is an HSV color picker: a preview swatch (with the hex value) above
// three gradient sliders for hue, saturation and value. Drag or click a track to
// set that channel; while focused, Up/Down choose the active channel and
// Left/Right adjust it. OnChange fires with the new color whenever it changes.
//
// It is built from 1D gradient tracks (not a 2D area) so it needs no gradient
// primitive or offscreen surface, and works headlessly. Alpha is not edited; the
// reported color is fully opaque.
type ColorPicker struct {
	BaseWidget
	h, s, v  float64
	dragging bool
	hasFocus bool
	active   int // channel being dragged / keyboard-active (0..2)
	font     render.FontFace
	onChange func(color.Color)
}

// ColorPickerOption configures a ColorPicker.
type ColorPickerOption func(*ColorPicker)

// ColorPickerValue sets the initial color.
func ColorPickerValue(c color.Color) ColorPickerOption {
	return func(p *ColorPicker) { p.h, p.s, p.v = rgbToHSV(c) }
}

// NewColorPicker returns a ColorPicker (defaulting to red) configured by opts.
func NewColorPicker(opts ...ColorPickerOption) *ColorPicker {
	p := &ColorPicker{BaseWidget: NewBase(), h: 0, s: 1, v: 1}
	for _, o := range opts {
		o(p)
	}
	return p
}

// OnChange registers the handler invoked with the new color when it changes.
func (p *ColorPicker) OnChange(fn func(color.Color)) { p.onChange = fn }

// Color returns the currently selected (opaque) color.
func (p *ColorPicker) Color() color.Color { return hsvToRGB(p.h, p.s, p.v) }

// SetColor sets the color (converted to HSV) and fires OnChange if it changed.
func (p *ColorPicker) SetColor(c color.Color) {
	h, s, v := rgbToHSV(c)
	if h == p.h && s == p.s && v == p.v {
		return
	}
	p.h, p.s, p.v = h, s, v
	p.fireChange()
	p.Invalidate()
}

func (p *ColorPicker) fireChange() {
	if p.onChange != nil {
		p.onChange(p.Color())
	}
}

func (p *ColorPicker) face() render.FontFace {
	if p.font != nil {
		return p.font
	}
	return p.appTheme().Font
}

// SetFont overrides the picker's font face (nil falls back to the theme font).
func (p *ColorPicker) SetFont(f render.FontFace) {
	p.font = f
	p.Invalidate()
}

// Focusable reports whether the picker can take focus (only when enabled).
func (p *ColorPicker) Focusable() bool { return p.Enabled() }

// MinSize returns the picker's footprint: swatch over three tracks.
func (p *ColorPicker) MinSize() geom.Size {
	return geom.Size{
		W: 220,
		H: cpSwatchH + cpGap + cpChannels*cpTrackH + (cpChannels-1)*cpGap + 2*cpPad,
	}
}

func (p *ColorPicker) inner() geom.Rect { return p.Bounds().Inset(geom.UniformInsets(cpPad)) }

func (p *ColorPicker) swatchRect() geom.Rect {
	in := p.inner()
	return geom.Rect{X: in.X, Y: in.Y, W: in.W, H: cpSwatchH}
}

// trackRect returns the rectangle of channel i's slider track.
func (p *ColorPicker) trackRect(i int) geom.Rect {
	in := p.inner()
	y := in.Y + cpSwatchH + cpGap + float64(i)*(cpTrackH+cpGap)
	return geom.Rect{X: in.X, Y: y, W: in.W, H: cpTrackH}
}

func (p *ColorPicker) channel(i int) float64 {
	switch i {
	case 0:
		return p.h
	case 1:
		return p.s
	default:
		return p.v
	}
}

// gradientColor returns the color shown at position t in channel i's track.
func (p *ColorPicker) gradientColor(i int, t float64) color.Color {
	switch i {
	case 0:
		return hsvToRGB(t, 1, 1)
	case 1:
		return hsvToRGB(p.h, t, p.v)
	default:
		return hsvToRGB(p.h, p.s, t)
	}
}

// Draw paints the preview swatch (with hex) and the three gradient tracks.
func (p *ColorPicker) Draw(canvas render.Canvas) {
	b := p.Bounds()
	rad := p.cornerRadius()
	canvas.FillRoundRect(b, rad, p.ColorOf(RoleSurface))
	canvas.StrokeRoundRect(b, rad, p.ColorOf(RoleBorder), 1)

	// Preview swatch with the hex value in a contrasting color.
	sw := p.swatchRect()
	cur := p.Color()
	canvas.FillRoundRect(sw, 4, cur)
	canvas.StrokeRoundRect(sw, 4, p.ColorOf(RoleBorder), 1)
	if f := p.face(); f != nil {
		hex := hexOf(cur)
		tw := f.Measure(hex).W
		canvas.DrawText(hex, geom.Point{X: sw.X + (sw.W-tw)/2, Y: vCenterY(f, sw.Y, sw.H)}, f, contrastColor(cur))
	}

	for i := 0; i < cpChannels; i++ {
		p.drawTrack(canvas, i)
	}
}

func (p *ColorPicker) drawTrack(canvas render.Canvas, i int) {
	tr := p.trackRect(i)
	if tr.W <= 0 {
		return
	}
	strips := int(tr.W / 3)
	if strips < 1 {
		strips = 1
	}
	for k := 0; k < strips; k++ {
		t := (float64(k) + 0.5) / float64(strips)
		sx := tr.X + float64(k)/float64(strips)*tr.W
		sw := tr.W/float64(strips) + 1
		canvas.FillRect(geom.Rect{X: sx, Y: tr.Y, W: sw, H: tr.H}, p.gradientColor(i, t))
	}
	border := p.ColorOf(RoleBorder)
	if p.active == i && (p.dragging || p.focused()) {
		border = p.ColorOf(RoleAccent)
	}
	canvas.StrokeRect(tr, border, 1)

	// Handle.
	hx := tr.X + p.channel(i)*tr.W
	hr := geom.Rect{X: hx - 2, Y: tr.Y - 2, W: 4, H: tr.H + 4}
	canvas.FillRect(hr, color.White)
	canvas.StrokeRect(hr, color.NRGBA{A: 200}, 1)
}

// focused reports keyboard focus (BaseWidget has no focus flag, so the picker
// tracks it itself via events).
func (p *ColorPicker) focused() bool { return p.hasFocus }

// HandleEvent drives dragging, hover and keyboard channel selection/adjustment.
func (p *ColorPicker) HandleEvent(ev *Event) bool {
	if !p.Enabled() {
		return false
	}
	switch ev.Type {
	case EventPointerDown:
		if i := p.trackAt(ev.Pos); i >= 0 {
			p.active = i
			p.dragging = true
			p.setChannelFromX(i, ev.Pos.X)
		}
		return true
	case EventPointerMove:
		if p.dragging {
			p.setChannelFromX(p.active, ev.Pos.X)
			return true
		}
	case EventPointerUp:
		p.dragging = false
		return true
	case EventFocusGained:
		p.hasFocus = true
		return true
	case EventFocusLost:
		p.hasFocus = false
		p.dragging = false
		return true
	case EventKeyDown:
		return p.handleKey(ev.Key)
	}
	return false
}

func (p *ColorPicker) handleKey(k render.Key) bool {
	switch k {
	case render.KeyUp:
		p.active = (p.active + cpChannels - 1) % cpChannels
	case render.KeyDown:
		p.active = (p.active + 1) % cpChannels
	case render.KeyLeft:
		p.setChannel(p.active, p.channel(p.active)-cpKeyStep)
	case render.KeyRight:
		p.setChannel(p.active, p.channel(p.active)+cpKeyStep)
	default:
		return false
	}
	return true
}

// trackAt returns the channel index whose track contains pos, or -1.
func (p *ColorPicker) trackAt(pos geom.Point) int {
	for i := 0; i < cpChannels; i++ {
		if p.trackRect(i).Contains(pos) {
			return i
		}
	}
	return -1
}

func (p *ColorPicker) setChannelFromX(i int, x float64) {
	tr := p.trackRect(i)
	if tr.W <= 0 {
		return
	}
	p.setChannel(i, (x-tr.X)/tr.W)
}

// setChannel sets channel i to t (clamped) and fires OnChange if it changed.
func (p *ColorPicker) setChannel(i int, t float64) {
	t = clamp01(t)
	if t == p.channel(i) {
		return
	}
	switch i {
	case 0:
		p.h = t
	case 1:
		p.s = t
	default:
		p.v = t
	}
	p.fireChange()
	p.Invalidate()
}

// --- HSV <-> RGB helpers ---

// hsvToRGB converts hue/sat/value in [0,1] to an opaque color.
func hsvToRGB(h, s, v float64) color.NRGBA {
	h = math.Mod(h, 1)
	if h < 0 {
		h += 1
	}
	i := int(h * 6)
	f := h*6 - float64(i)
	pp := v * (1 - s)
	q := v * (1 - f*s)
	tt := v * (1 - (1-f)*s)
	var r, g, b float64
	switch i % 6 {
	case 0:
		r, g, b = v, tt, pp
	case 1:
		r, g, b = q, v, pp
	case 2:
		r, g, b = pp, v, tt
	case 3:
		r, g, b = pp, q, v
	case 4:
		r, g, b = tt, pp, v
	default:
		r, g, b = v, pp, q
	}
	return color.NRGBA{R: to8(r), G: to8(g), B: to8(b), A: 255}
}

// rgbToHSV converts a color to hue/sat/value in [0,1].
func rgbToHSV(c color.Color) (h, s, v float64) {
	nc := color.NRGBAModel.Convert(c).(color.NRGBA)
	r, g, b := float64(nc.R)/255, float64(nc.G)/255, float64(nc.B)/255
	max := math.Max(r, math.Max(g, b))
	min := math.Min(r, math.Min(g, b))
	v = max
	d := max - min
	if max > 0 {
		s = d / max
	}
	if d == 0 {
		return 0, s, v
	}
	switch max {
	case r:
		h = (g - b) / d
		if g < b {
			h += 6
		}
	case g:
		h = (b-r)/d + 2
	default:
		h = (r-g)/d + 4
	}
	return h / 6, s, v
}

func to8(v float64) uint8 {
	n := v*255 + 0.5
	if n < 0 {
		n = 0
	}
	if n > 255 {
		n = 255
	}
	return uint8(n)
}

func hexOf(c color.Color) string {
	nc := color.NRGBAModel.Convert(c).(color.NRGBA)
	return fmt.Sprintf("#%02X%02X%02X", nc.R, nc.G, nc.B)
}

// contrastColor returns black or white, whichever is more readable on c.
func contrastColor(c color.Color) color.Color {
	nc := color.NRGBAModel.Convert(c).(color.NRGBA)
	lum := 0.299*float64(nc.R) + 0.587*float64(nc.G) + 0.114*float64(nc.B)
	if lum > 140 {
		return color.Black
	}
	return color.White
}
