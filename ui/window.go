package ui

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// TitleBarHeight is the height in pixels of a window's draggable title bar.
const TitleBarHeight = 24

// Window is a top-level, movable frame with a title bar and a content area.
// Widgets are added to its Content container, whose coordinate origin sits just
// below the title bar.
type Window struct {
	BaseWidget

	Title           string
	Content         *Container
	TitleBarColor   color.Color
	BackgroundColor color.Color
	BorderColor     color.Color

	dragging   bool
	dragAnchor Point // cursor offset from the window origin when dragging began
}

// NewWindow returns a window with the given title occupying r. Its Content
// container fills the area below the title bar.
func NewWindow(title string, r Rect) *Window {
	w := &Window{
		BaseWidget:      NewBase(r),
		Title:           title,
		TitleBarColor:   DefaultTitleBar,
		BackgroundColor: DefaultWindowBackground,
		BorderColor:     DefaultBorder,
	}
	w.Content = NewContainer(Rect{X: 0, Y: TitleBarHeight, W: r.W, H: r.H - TitleBarHeight})
	return w
}

// Update advances the window's content.
func (w *Window) Update() error { return w.Content.Update() }

// Draw renders the window frame, title bar, and content.
func (w *Window) Draw(dst *ebiten.Image, origin Point) {
	abs := w.bounds.Add(origin)

	vector.FillRect(dst, float32(abs.X), float32(abs.Y), float32(abs.W), float32(abs.H), w.BackgroundColor, false)
	vector.FillRect(dst, float32(abs.X), float32(abs.Y), float32(abs.W), float32(TitleBarHeight), w.TitleBarColor, false)
	ebitenutil.DebugPrintAt(dst, w.Title, abs.X+6, abs.Y+4)

	w.Content.Draw(dst, abs.Origin())

	if w.BorderColor != nil {
		vector.StrokeRect(dst, float32(abs.X), float32(abs.Y), float32(abs.W), float32(abs.H), 1, w.BorderColor, false)
	}
}

// HandleEvent implements title-bar dragging and otherwise delegates to the
// content container.
func (w *Window) HandleEvent(ev *Event, origin Point) bool {
	abs := w.bounds.Add(origin)
	titleBar := Rect{X: abs.X, Y: abs.Y, W: abs.W, H: TitleBarHeight}

	switch ev.Type {
	case MouseDown:
		if ev.Button == ebiten.MouseButtonLeft && titleBar.Contains(ev.Pos) {
			w.dragging = true
			w.dragAnchor = ev.Pos.Sub(abs.Origin())
			return true
		}
	case MouseMove:
		if w.dragging {
			// Convert the cursor's absolute position back into parent-relative
			// bounds, accounting for the grab anchor.
			w.bounds.X = ev.Pos.X - origin.X - w.dragAnchor.X
			w.bounds.Y = ev.Pos.Y - origin.Y - w.dragAnchor.Y
			return true
		}
	case MouseUp:
		if w.dragging && ev.Button == ebiten.MouseButtonLeft {
			w.dragging = false
			return true
		}
	}

	return w.Content.HandleEvent(ev, abs.Origin())
}
