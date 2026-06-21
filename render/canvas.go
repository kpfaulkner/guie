// Package render defines the backend-neutral rendering and input abstractions
// that sit between the framework core and a concrete graphics backend.
//
// Nothing in this package imports a specific graphics library. Widgets draw
// through the Canvas interface and read input through InputState, so the
// framework never depends on the backend (currently EBiten) directly. A single
// backend package implements Canvas, FontFace, Image and Driver.
package render

import (
	"image/color"

	"github.com/kpfaulkner/guie/geom"
)

// Canvas is the drawing surface handed to widgets each frame. All coordinates
// are logical pixels in the surface's coordinate space.
//
// Clipping is a stack: PushClip intersects a rectangle with the current clip
// region; PopClip restores the previous region. Backends must honour the
// active clip for every draw call.
type Canvas interface {
	// Size returns the dimensions of the drawing surface.
	Size() geom.Size

	// PushClip intersects r with the current clip region and makes it active.
	PushClip(r geom.Rect)
	// PopClip restores the clip region in effect before the matching PushClip.
	PopClip()

	// Fill paints the entire current clip region with c.
	Fill(c color.Color)
	// FillRect fills the rectangle r with c.
	FillRect(r geom.Rect, c color.Color)
	// StrokeRect draws the outline of r with the given line width.
	StrokeRect(r geom.Rect, c color.Color, width float64)
	// FillRoundRect fills r with corners rounded to the given radius. A radius of
	// zero is equivalent to FillRect.
	FillRoundRect(r geom.Rect, radius float64, c color.Color)
	// StrokeRoundRect draws the outline of r with rounded corners. A radius of
	// zero is equivalent to StrokeRect.
	StrokeRoundRect(r geom.Rect, radius float64, c color.Color, width float64)
	// DrawLine draws a straight line from a to b with the given width.
	DrawLine(a, b geom.Point, c color.Color, width float64)
	// FillCircle fills a circle of radius radius centered at center.
	FillCircle(center geom.Point, radius float64, c color.Color)
	// StrokeCircle draws the outline of a circle with the given line width.
	StrokeCircle(center geom.Point, radius float64, c color.Color, width float64)

	// DrawText draws s with its top-left at pos using the given face and colour.
	DrawText(s string, pos geom.Point, face FontFace, c color.Color)
	// MeasureText returns the rendered size of s in the given face.
	MeasureText(s string, face FontFace) geom.Size

	// DrawImage draws img scaled to fit the destination rectangle dst.
	DrawImage(img Image, dst geom.Rect)

	// SubCanvas returns a Canvas clipped to r, sharing the same surface. Useful
	// for compositing windows and nested content.
	SubCanvas(r geom.Rect) Canvas
}

// FontMetrics describes the vertical extents of a font face, in logical pixels.
type FontMetrics struct {
	// Ascent is the distance from the baseline to the top of the tallest glyph.
	Ascent float64
	// Descent is the distance from the baseline to the bottom of the lowest glyph.
	Descent float64
	// LineHeight is the recommended distance between consecutive baselines.
	LineHeight float64
}

// FontFace is an opaque, backend-specific font handle at a particular size.
// Construct one via the backend (or the theme/font helpers); widgets read its
// metrics, measure strings for layout, and pass it to Canvas text calls.
type FontFace interface {
	// Metrics returns the vertical extents of the face.
	Metrics() FontMetrics
	// Measure returns the rendered size of s in this face. It does not require a
	// frame, so widgets may call it from MinSize during layout.
	Measure(s string) geom.Size
}

// Image is an opaque, backend-specific bitmap handle.
type Image interface {
	// Size returns the pixel dimensions of the image.
	Size() geom.Size
}

// RenderTarget is an offscreen drawing surface whose contents persist between
// frames. It is itself a render.Image, so it can be blitted onto a Canvas with
// DrawImage, and it hands out its own Canvas for drawing into it.
//
// It exists so callers can draw expensive, slowly-changing content once and
// then cheaply blit the result every frame, instead of re-issuing every draw
// call each frame. (A drawing app, for example, bakes finished strokes into a
// target and only re-draws the in-progress stroke live.)
//
// A target is sized in pixels at 1:1 (logical = physical); when blitted on a
// HiDPI surface it is scaled like any other image, so content drawn into it is
// at logical resolution. Targets are not safe for concurrent use and must be
// used only on the UI goroutine.
type RenderTarget interface {
	Image
	// Canvas returns a Canvas that draws into this target. The returned canvas
	// starts with a full-surface clip stack; existing pixels are preserved
	// (drawing is additive), so callers may accumulate content across frames.
	Canvas() Canvas
	// Clear fills the entire target with c. Use a fully transparent colour to
	// erase to nothing.
	Clear(c color.Color)
	// Dispose releases the target's backing resources. The target must not be
	// used afterwards.
	Dispose()
}
