package ui

import (
	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
)

const (
	splitterThickness = 6
	splitterMinPane   = 24 // default minimum size of each pane along the split axis
)

// SplitPane lays out two child panes separated by a draggable divider. The
// Direction is the axis along which the panes are arranged: Horizontal places
// them left/right (a vertical divider), Vertical places them top/bottom (a
// horizontal divider). Dragging the divider changes the split ratio, clamped so
// neither pane shrinks below its minimum.
type SplitPane struct {
	BaseWidget
	direction        geom.Direction
	first, second    Widget
	ratio            float64 // first pane's fraction of the available space
	minFirst, minSec float64
	hover, dragging  bool
}

// SplitOption configures a SplitPane.
type SplitOption func(*SplitPane)

// SplitRatio sets the initial fraction (0..1) given to the first pane.
func SplitRatio(r float64) SplitOption { return func(s *SplitPane) { s.ratio = clamp01(r) } }

// SplitMinSizes sets the minimum size (along the split axis) of each pane.
func SplitMinSizes(first, second float64) SplitOption {
	return func(s *SplitPane) { s.minFirst, s.minSec = first, second }
}

// NewSplitPane returns a SplitPane arranging first and second along dir.
func NewSplitPane(dir geom.Direction, first, second Widget, opts ...SplitOption) *SplitPane {
	s := &SplitPane{
		BaseWidget: NewBase(),
		direction:  dir,
		first:      first,
		second:     second,
		ratio:      0.5,
		minFirst:   splitterMinPane,
		minSec:     splitterMinPane,
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

// HSplit returns a horizontal SplitPane (panes side by side).
func HSplit(first, second Widget, opts ...SplitOption) *SplitPane {
	return NewSplitPane(geom.Horizontal, first, second, opts...)
}

// VSplit returns a vertical SplitPane (panes stacked).
func VSplit(first, second Widget, opts ...SplitOption) *SplitPane {
	return NewSplitPane(geom.Vertical, first, second, opts...)
}

// Ratio returns the current split fraction held by the first pane.
func (s *SplitPane) Ratio() float64 { return s.ratio }

// Children returns the two panes (the divider is self-drawn, not a child).
func (s *SplitPane) Children() []Widget {
	var out []Widget
	if s.first != nil {
		out = append(out, s.first)
	}
	if s.second != nil {
		out = append(out, s.second)
	}
	return out
}

func (s *SplitPane) mount(self, parent Widget, ctx *treeContext) {
	s.BaseWidget.mount(self, parent, ctx)
	if s.first != nil {
		s.first.mount(s.first, self, ctx)
	}
	if s.second != nil {
		s.second.mount(s.second, self, ctx)
	}
}

// sizes splits total (the extent along the split axis) into the two pane sizes,
// honoring the minimums.
func (s *SplitPane) sizes(total float64) (first, second float64) {
	avail := total - splitterThickness
	if avail < 0 {
		avail = 0
	}
	fs := s.ratio * avail
	if fs < s.minFirst {
		fs = s.minFirst
	}
	if fs > avail-s.minSec {
		fs = avail - s.minSec
	}
	if fs < 0 {
		fs = 0
	}
	if fs > avail {
		fs = avail
	}
	return fs, avail - fs
}

// rects returns the first-pane, divider and second-pane rectangles.
func (s *SplitPane) rects() (first, divider, second geom.Rect) {
	b := s.Bounds()
	if s.direction == geom.Horizontal {
		fs, ss := s.sizes(b.W)
		first = geom.Rect{X: b.X, Y: b.Y, W: fs, H: b.H}
		divider = geom.Rect{X: b.X + fs, Y: b.Y, W: splitterThickness, H: b.H}
		second = geom.Rect{X: b.X + fs + splitterThickness, Y: b.Y, W: ss, H: b.H}
	} else {
		fs, ss := s.sizes(b.H)
		first = geom.Rect{X: b.X, Y: b.Y, W: b.W, H: fs}
		divider = geom.Rect{X: b.X, Y: b.Y + fs, W: b.W, H: splitterThickness}
		second = geom.Rect{X: b.X, Y: b.Y + fs + splitterThickness, W: b.W, H: ss}
	}
	return
}

// MinSize accounts for both panes plus the divider along the split axis, and the
// larger pane on the cross axis.
func (s *SplitPane) MinSize() geom.Size {
	var fm, sm geom.Size
	if s.first != nil {
		fm = s.first.MinSize()
	}
	if s.second != nil {
		sm = s.second.MinSize()
	}
	if s.direction == geom.Horizontal {
		return geom.Size{
			W: maxF(s.minFirst, fm.W) + maxF(s.minSec, sm.W) + splitterThickness,
			H: maxF(fm.H, sm.H),
		}
	}
	return geom.Size{
		W: maxF(fm.W, sm.W),
		H: maxF(s.minFirst, fm.H) + maxF(s.minSec, sm.H) + splitterThickness,
	}
}

// Layout positions the two panes around the divider.
func (s *SplitPane) Layout() {
	first, _, second := s.rects()
	if s.first != nil {
		s.first.SetBounds(first)
		s.first.Layout()
	}
	if s.second != nil {
		s.second.SetBounds(second)
		s.second.Layout()
	}
}

// Draw paints each pane (clipped to its area) and the divider.
func (s *SplitPane) Draw(canvas render.Canvas) {
	first, divider, second := s.rects()

	if s.first != nil && s.first.Visible() {
		canvas.PushClip(first)
		s.first.Draw(canvas)
		canvas.PopClip()
	}
	if s.second != nil && s.second.Visible() {
		canvas.PushClip(second)
		s.second.Draw(canvas)
		canvas.PopClip()
	}

	col := s.ColorOf(RoleBorder)
	if s.hover || s.dragging {
		col = s.ColorOf(RoleAccent)
	}
	canvas.FillRect(divider, col)
}

// HandleEvent drives divider hover and drag-to-resize.
func (s *SplitPane) HandleEvent(ev *Event) bool {
	switch ev.Type {
	case EventPointerLeave:
		s.hover = false
		return true
	case EventPointerMove:
		if s.dragging {
			s.dragTo(ev.Pos)
			return true
		}
		_, divider, _ := s.rects()
		s.hover = divider.Contains(ev.Pos)
		return true
	case EventPointerDown:
		if _, divider, _ := s.rects(); divider.Contains(ev.Pos) {
			s.dragging = true
			return true
		}
	case EventPointerUp:
		if s.dragging {
			s.dragging = false
			return true
		}
	}
	return false
}

// dragTo sets the split ratio so the divider follows the pointer.
func (s *SplitPane) dragTo(pos geom.Point) {
	b := s.Bounds()
	var pointerMain, start, total float64
	if s.direction == geom.Horizontal {
		pointerMain, start, total = pos.X, b.X, b.W
	} else {
		pointerMain, start, total = pos.Y, b.Y, b.H
	}
	avail := total - splitterThickness
	if avail <= 0 {
		return
	}
	fs := pointerMain - start - splitterThickness/2
	if fs < s.minFirst {
		fs = s.minFirst
	}
	if fs > avail-s.minSec {
		fs = avail - s.minSec
	}
	if fs < 0 {
		fs = 0
	}
	s.ratio = fs / avail
	s.Invalidate()
}
