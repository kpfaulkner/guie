package ui

import (
	"image/color"
	"math"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
)

// dragThreshold is how far (in logical pixels) the pointer must move while a
// drag source is pressed before the press is promoted to a drag, so a small
// jitter during a click does not start one.
const dragThreshold = 4.0

// ghostFade is the translucent white wash drawn over the source snapshot so the
// dragged ghost reads as a floating, in-flight copy rather than the real widget.
var ghostFade = color.NRGBA{R: 255, G: 255, B: 255, A: 110}

// DragData is the in-process payload of a drag operation. Type is a free-form
// tag drop targets match on ("row", "file", "tab", ...); Value carries the
// payload itself. Drag-and-drop is in-process only: Value is passed by reference
// to the drop target, not serialized.
type DragData struct {
	Type  string
	Value any
}

// DragGhost paints the visual that follows the cursor during a drag. at is the
// top-left position to draw at. Implement it for a fully custom ghost; by
// default a snapshot of the source widget is used (see SetDragGhost).
type DragGhost interface {
	DrawGhost(c render.Canvas, at geom.Point)
}

// dragConfig holds a widget's opt-in drag-and-drop configuration. It is
// allocated lazily by the first setter so widgets that never use drag-and-drop
// carry no extra cost.
type dragConfig struct {
	// source side
	provide   func() *DragData    // payload provider; nil return vetoes the drag
	ghost     DragGhost           // custom ghost, nil → default source snapshot
	onDragEnd func(accepted bool) // drag finished (dropped or cancelled)

	// target side
	accept  func(DragData) bool             // will this widget take the payload?
	onEnter func(DragData)                  // ghost entered the widget
	onLeave func()                          // ghost left the widget
	onOver  func(DragData, geom.Point)      // ghost moved while inside
	onDrop  func(DragData, geom.Point) bool // dropped here; return true if accepted
}

func (b *BaseWidget) ensureDrag() *dragConfig {
	if b.drag == nil {
		b.drag = &dragConfig{}
	}
	return b.drag
}

// dragCfg returns the widget's drag configuration, or nil if it never opted in.
// The dispatcher reads it off widgets via this accessor (the same pattern as
// ContextMenu), so drag-and-drop needs no methods on the Widget interface.
func (b *BaseWidget) dragCfg() *dragConfig { return b.drag }

// SetDragSource makes the widget draggable. provide is called once, when a press
// on the widget moves past the drag threshold, to produce the payload; returning
// nil cancels the drag (e.g. nothing is selected to drag).
func (b *BaseWidget) SetDragSource(provide func() *DragData) { b.ensureDrag().provide = provide }

// SetDragGhost overrides the visual dragged under the cursor. By default the
// framework snapshots the source widget; pass nil to restore that default.
func (b *BaseWidget) SetDragGhost(g DragGhost) { b.ensureDrag().ghost = g }

// OnDragEnd registers a callback fired when a drag that started on this source
// finishes, with accepted reporting whether a drop target took the payload.
func (b *BaseWidget) OnDragEnd(fn func(accepted bool)) { b.ensureDrag().onDragEnd = fn }

// SetDropTarget makes the widget a drop target. accept decides, per payload,
// whether the widget will receive it; only accepting targets get enter/over/drop
// callbacks and the drop.
func (b *BaseWidget) SetDropTarget(accept func(DragData) bool) { b.ensureDrag().accept = accept }

// OnDragEnter registers a callback fired when an accepted drag's ghost enters
// the widget (typically to show a drop highlight).
func (b *BaseWidget) OnDragEnter(fn func(d DragData)) { b.ensureDrag().onEnter = fn }

// OnDragLeave registers a callback fired when the ghost leaves the widget
// (typically to clear the highlight).
func (b *BaseWidget) OnDragLeave(fn func()) { b.ensureDrag().onLeave = fn }

// OnDragOver registers a callback fired each time the ghost moves while inside
// the widget, with the absolute cursor position (e.g. to show an insert line).
func (b *BaseWidget) OnDragOver(fn func(d DragData, pos geom.Point)) { b.ensureDrag().onOver = fn }

// OnDrop registers a callback fired when the payload is released over the widget.
// Return true if the drop was accepted (reported back to the source's OnDragEnd).
func (b *BaseWidget) OnDrop(fn func(d DragData, pos geom.Point) bool) { b.ensureDrag().onDrop = fn }

// dragOf returns the drag configuration of any widget (all widgets embed
// BaseWidget, so the accessor is always present), or nil if it never opted in.
func dragOf(w Widget) *dragConfig {
	if a, ok := w.(interface{ dragCfg() *dragConfig }); ok {
		return a.dragCfg()
	}
	return nil
}

// dragSourceOf returns the nearest widget at or above w that is a drag source.
func dragSourceOf(w Widget) Widget {
	for ; w != nil; w = w.Parent() {
		if c := dragOf(w); c != nil && c.provide != nil {
			return w
		}
	}
	return nil
}

// dropTargetOf returns the nearest widget at or above w that accepts data.
func dropTargetOf(w Widget, data DragData) Widget {
	for ; w != nil; w = w.Parent() {
		if c := dragOf(w); c != nil && c.accept != nil && c.accept(data) {
			return w
		}
	}
	return nil
}

// dragSession is the single in-flight drag tracked by the App.
type dragSession struct {
	source  Widget     // the drag source
	data    DragData   // payload (valid once started)
	over    Widget     // current accepting drop target, or nil
	ghost   DragGhost  // visual under the cursor (valid once started)
	origin  geom.Point // press point
	last    geom.Point // latest cursor position
	grab    geom.Point // origin minus the source's top-left, to anchor the ghost
	started bool       // has movement crossed the threshold?
}

func dist(a, b geom.Point) float64 {
	return math.Hypot(a.X-b.X, a.Y-b.Y)
}

// advanceDrag is called from dispatchPointer on pointer movement while a drag is
// pending or active. hit is the already-resolved widget under the cursor. It
// promotes a pending drag once movement crosses the threshold, then tracks the
// drop target and fires its enter/leave/over callbacks.
func (a *App) advanceDrag(pos geom.Point, hit Widget) {
	d := a.drag
	if !d.started {
		if dist(pos, d.origin) <= dragThreshold {
			return
		}
		cfg := dragOf(d.source)
		if cfg == nil || cfg.provide == nil {
			a.drag = nil
			return
		}
		data := cfg.provide()
		if data == nil { // source vetoed the drag
			a.drag = nil
			return
		}
		d.data = *data
		d.started = true
		b := d.source.Bounds()
		d.grab = geom.Point{X: d.origin.X - b.X, Y: d.origin.Y - b.Y}
		if cfg.ghost != nil {
			d.ghost = cfg.ghost
		} else {
			d.ghost = newSnapshotGhost(d.source)
		}
	}
	d.last = pos

	tgt := dropTargetOf(hit, d.data)
	if tgt != d.over {
		if d.over != nil {
			if c := dragOf(d.over); c != nil && c.onLeave != nil {
				c.onLeave()
			}
		}
		d.over = tgt
		if tgt != nil {
			if c := dragOf(tgt); c != nil && c.onEnter != nil {
				c.onEnter(d.data)
			}
		}
	}
	if tgt != nil {
		if c := dragOf(tgt); c != nil && c.onOver != nil {
			c.onOver(d.data, pos)
		}
	}
}

// finishDrag ends the active drag: drops onto the current target if any, fires
// the source's OnDragEnd, and releases the ghost. A pending drag that never
// started is simply discarded.
func (a *App) finishDrag(pos geom.Point) {
	d := a.drag
	a.drag = nil
	if d == nil || !d.started {
		return
	}
	accepted := false
	if d.over != nil {
		if c := dragOf(d.over); c != nil && c.onDrop != nil {
			accepted = c.onDrop(d.data, pos)
		}
	}
	if c := dragOf(d.source); c != nil && c.onDragEnd != nil {
		c.onDragEnd(accepted)
	}
	if g, ok := d.ghost.(interface{ dispose() }); ok {
		g.dispose()
	}
}

// cancelDrag aborts an in-flight drag (e.g. on Escape) without dropping.
func (a *App) cancelDrag() {
	d := a.drag
	a.drag = nil
	if d == nil {
		return
	}
	if d.over != nil {
		if c := dragOf(d.over); c != nil && c.onLeave != nil {
			c.onLeave()
		}
	}
	if d.started {
		if c := dragOf(d.source); c != nil && c.onDragEnd != nil {
			c.onDragEnd(false)
		}
	}
	if g, ok := d.ghost.(interface{ dispose() }); ok {
		g.dispose()
	}
}

// drawDrag paints the ghost under the cursor, anchored at the point it was
// grabbed. Called from App.draw above overlays and below tooltips.
func (a *App) drawDrag(c render.Canvas) {
	d := a.drag
	if d == nil || !d.started || d.ghost == nil {
		return
	}
	at := geom.Point{X: d.last.X - d.grab.X, Y: d.last.Y - d.grab.Y}
	d.ghost.DrawGhost(c, at)
}

// snapshotGhost is the default ghost: a faded snapshot of the source widget
// captured into an offscreen RenderTarget at drag start.
type snapshotGhost struct {
	target render.RenderTarget
	size   geom.Size
}

func (g *snapshotGhost) DrawGhost(c render.Canvas, at geom.Point) {
	if g.target == nil {
		return
	}
	c.DrawImage(g.target, geom.Rect{X: at.X, Y: at.Y, W: g.size.W, H: g.size.H})
}

func (g *snapshotGhost) dispose() {
	if g.target != nil {
		g.target.Dispose()
		g.target = nil
	}
}

// newSnapshotGhost renders src into an offscreen target the size of its bounds,
// then washes it with a translucent overlay so it reads as a dragged copy.
// Returns nil if the source has no drawable area.
func newSnapshotGhost(src Widget) DragGhost {
	b := src.Bounds()
	w, h := int(math.Ceil(b.W)), int(math.Ceil(b.H))
	if w <= 0 || h <= 0 {
		return nil
	}
	rt := NewRenderTarget(w, h)
	if rt == nil {
		return nil
	}
	rt.Clear(color.Transparent)
	tc := rt.Canvas()
	// Widgets draw at absolute coordinates; offset by -bounds so the source
	// lands at the target's origin.
	src.Draw(&offsetCanvas{Canvas: tc, dx: -b.X, dy: -b.Y})
	tc.FillRect(geom.Rect{W: b.W, H: b.H}, ghostFade)
	return &snapshotGhost{target: rt, size: geom.Size{W: b.W, H: b.H}}
}

// offsetCanvas wraps a render.Canvas, translating every coordinate by (dx,dy).
// It lets a widget that draws in absolute coordinates be rendered into an
// offscreen target whose origin is elsewhere.
type offsetCanvas struct {
	render.Canvas
	dx, dy float64
}

func (o *offsetCanvas) pt(p geom.Point) geom.Point {
	return geom.Point{X: p.X + o.dx, Y: p.Y + o.dy}
}

func (o *offsetCanvas) rect(r geom.Rect) geom.Rect {
	return geom.Rect{X: r.X + o.dx, Y: r.Y + o.dy, W: r.W, H: r.H}
}

func (o *offsetCanvas) PushClip(r geom.Rect) { o.Canvas.PushClip(o.rect(r)) }
func (o *offsetCanvas) FillRect(r geom.Rect, c color.Color) {
	o.Canvas.FillRect(o.rect(r), c)
}
func (o *offsetCanvas) StrokeRect(r geom.Rect, c color.Color, width float64) {
	o.Canvas.StrokeRect(o.rect(r), c, width)
}
func (o *offsetCanvas) FillRoundRect(r geom.Rect, radius float64, c color.Color) {
	o.Canvas.FillRoundRect(o.rect(r), radius, c)
}
func (o *offsetCanvas) StrokeRoundRect(r geom.Rect, radius float64, c color.Color, width float64) {
	o.Canvas.StrokeRoundRect(o.rect(r), radius, c, width)
}
func (o *offsetCanvas) DrawLine(a, b geom.Point, c color.Color, width float64) {
	o.Canvas.DrawLine(o.pt(a), o.pt(b), c, width)
}
func (o *offsetCanvas) FillCircle(center geom.Point, radius float64, c color.Color) {
	o.Canvas.FillCircle(o.pt(center), radius, c)
}
func (o *offsetCanvas) StrokeCircle(center geom.Point, radius float64, c color.Color, width float64) {
	o.Canvas.StrokeCircle(o.pt(center), radius, c, width)
}
func (o *offsetCanvas) DrawText(s string, pos geom.Point, face render.FontFace, c color.Color) {
	o.Canvas.DrawText(s, o.pt(pos), face, c)
}
func (o *offsetCanvas) DrawImage(img render.Image, dst geom.Rect) {
	o.Canvas.DrawImage(img, o.rect(dst))
}
func (o *offsetCanvas) SubCanvas(r geom.Rect) render.Canvas {
	return &offsetCanvas{Canvas: o.Canvas.SubCanvas(o.rect(r)), dx: o.dx, dy: o.dy}
}
