package ui

import (
	"testing"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
)

// testGhost is a no-op DragGhost used in tests so a drag never allocates a GPU
// RenderTarget (the default snapshot ghost does; that path is exercised by the
// example, per the project's no-GPU-in-tests convention).
type testGhost struct{ drawn int }

func (g *testGhost) DrawGhost(c render.Canvas, at geom.Point) { g.drawn++ }

// heldMoveTo is a move InputState with the left button still held (mid-drag).
func heldMoveTo(pos geom.Point) render.InputState {
	return render.InputState{MousePos: pos, MouseDown: ButtonSetOf(render.MouseLeft)}
}

// dndApp builds an app with a draggable source and an accepting drop target,
// laid out side by side. The source carries a custom ghost so no GPU is touched.
func dndApp(t *testing.T) (app *App, src, tgt *Container) {
	t.Helper()
	app = NewApp()
	root := NewContainer()
	root.SetBounds(geom.Rect{X: 0, Y: 0, W: 400, H: 200})

	src = NewContainer()
	src.SetBounds(geom.Rect{X: 10, Y: 10, W: 80, H: 80})
	src.SetDragGhost(&testGhost{})

	tgt = NewContainer()
	tgt.SetBounds(geom.Rect{X: 200, Y: 10, W: 150, H: 150})

	root.Add(src)
	root.Add(tgt)
	app.SetContent(root)

	return app, src, tgt
}

var (
	srcCenter = geom.Point{X: 50, Y: 50}
	tgtCenter = geom.Point{X: 275, Y: 85}
	nearOrig  = geom.Point{X: 52, Y: 51} // < threshold from srcCenter
	pastOrig  = geom.Point{X: 70, Y: 50} // > threshold from srcCenter
)

func TestDragStartsOnlyAfterThreshold(t *testing.T) {
	app, src, _ := dndApp(t)
	provides := 0
	src.SetDragSource(func() *DragData {
		provides++
		return &DragData{Type: "x", Value: 1}
	})

	app.dispatchPointer(downAt(srcCenter))
	app.dispatchPointer(heldMoveTo(nearOrig))
	if app.drag == nil || app.drag.started {
		t.Fatalf("a sub-threshold move must not start the drag (started=%v)", app.drag != nil && app.drag.started)
	}
	if provides != 0 {
		t.Fatalf("provider must not run before the threshold, ran %d times", provides)
	}

	app.dispatchPointer(heldMoveTo(pastOrig))
	if app.drag == nil || !app.drag.started {
		t.Fatal("a past-threshold move should start the drag")
	}
	if provides != 1 {
		t.Fatalf("provider should run exactly once at start, ran %d times", provides)
	}

	// Further moves must not call the provider again.
	app.dispatchPointer(heldMoveTo(tgtCenter))
	if provides != 1 {
		t.Fatalf("provider should not run again after start, ran %d times", provides)
	}
}

func TestDropOnAcceptingTarget(t *testing.T) {
	app, src, tgt := dndApp(t)
	src.SetDragSource(func() *DragData { return &DragData{Type: "row", Value: 7} })

	var endAccepted *bool
	src.OnDragEnd(func(accepted bool) { endAccepted = &accepted })

	var dropped *DragData
	tgt.SetDropTarget(func(d DragData) bool { return d.Type == "row" })
	tgt.OnDrop(func(d DragData, pos geom.Point) bool { dropped = &d; return true })

	app.dispatchPointer(downAt(srcCenter))
	app.dispatchPointer(heldMoveTo(pastOrig))
	app.dispatchPointer(heldMoveTo(tgtCenter))
	app.dispatchPointer(upAt(tgtCenter))

	if dropped == nil {
		t.Fatal("OnDrop should fire when released over an accepting target")
	}
	if dropped.Value.(int) != 7 {
		t.Fatalf("dropped payload = %v, want 7", dropped.Value)
	}
	if endAccepted == nil || !*endAccepted {
		t.Fatalf("OnDragEnd should report accepted=true, got %v", endAccepted)
	}
	if app.drag != nil {
		t.Fatal("the session should be cleared after the drop")
	}
}

func TestDropOffTargetIsCancelled(t *testing.T) {
	app, src, tgt := dndApp(t)
	src.SetDragSource(func() *DragData { return &DragData{Type: "row"} })
	var endAccepted *bool
	src.OnDragEnd(func(accepted bool) { endAccepted = &accepted })
	dropped := false
	tgt.SetDropTarget(func(d DragData) bool { return true })
	tgt.OnDrop(func(d DragData, pos geom.Point) bool { dropped = true; return true })

	app.dispatchPointer(downAt(srcCenter))
	app.dispatchPointer(heldMoveTo(pastOrig))
	// Release over empty space (back near the source, not over the target).
	app.dispatchPointer(upAt(geom.Point{X: 120, Y: 50}))

	if dropped {
		t.Fatal("OnDrop must not fire when released off any target")
	}
	if endAccepted == nil || *endAccepted {
		t.Fatalf("OnDragEnd should report accepted=false, got %v", endAccepted)
	}
}

func TestDragEnterOverLeave(t *testing.T) {
	app, src, tgt := dndApp(t)
	src.SetDragSource(func() *DragData { return &DragData{Type: "row"} })
	tgt.SetDropTarget(func(d DragData) bool { return true })
	var seq []string
	tgt.OnDragEnter(func(d DragData) { seq = append(seq, "enter") })
	tgt.OnDragLeave(func() { seq = append(seq, "leave") })
	tgt.OnDragOver(func(d DragData, pos geom.Point) { seq = append(seq, "over") })

	app.dispatchPointer(downAt(srcCenter))
	app.dispatchPointer(heldMoveTo(pastOrig))                  // started, over nothing
	app.dispatchPointer(heldMoveTo(tgtCenter))                 // enter + over
	app.dispatchPointer(heldMoveTo(geom.Point{X: 300, Y: 90})) // still inside → over
	app.dispatchPointer(heldMoveTo(geom.Point{X: 120, Y: 50})) // leaves target
	app.dispatchPointer(upAt(geom.Point{X: 120, Y: 50}))

	if len(seq) < 3 || seq[0] != "enter" {
		t.Fatalf("expected enter first, got %v", seq)
	}
	// First entry is enter, last meaningful transition is leave.
	if seq[len(seq)-1] != "leave" {
		t.Fatalf("expected leave last, got %v", seq)
	}
	gotOver := false
	for _, s := range seq {
		if s == "over" {
			gotOver = true
		}
	}
	if !gotOver {
		t.Fatalf("expected at least one over event, got %v", seq)
	}
}

func TestProvideNilVetoesDrag(t *testing.T) {
	app, src, _ := dndApp(t)
	src.SetDragSource(func() *DragData { return nil })

	app.dispatchPointer(downAt(srcCenter))
	app.dispatchPointer(heldMoveTo(pastOrig))
	if app.drag != nil {
		t.Fatal("returning nil from the provider should veto (clear) the drag")
	}
}

func TestDragSuppressesClickButPlainPressDoesNot(t *testing.T) {
	app := NewApp()
	root := NewContainer()
	root.SetBounds(geom.Rect{X: 0, Y: 0, W: 400, H: 200})
	clicks := 0
	btn := NewButton("drag me")
	btn.OnClick(func() { clicks++ })
	btn.SetBounds(geom.Rect{X: 10, Y: 10, W: 80, H: 40})
	btn.SetDragGhost(&testGhost{})
	btn.SetDragSource(func() *DragData { return &DragData{Type: "x"} })
	root.Add(btn)
	app.SetContent(root)

	center := geom.Point{X: 50, Y: 30}

	// A drag (press → move past threshold → release) must NOT click.
	app.dispatchPointer(downAt(center))
	app.dispatchPointer(heldMoveTo(geom.Point{X: 70, Y: 30}))
	app.dispatchPointer(upAt(geom.Point{X: 70, Y: 30}))
	if clicks != 0 {
		t.Fatalf("a completed drag should not derive a click, got %d", clicks)
	}

	// A plain press+release with no drag should still click.
	app.dispatchPointer(downAt(center))
	app.dispatchPointer(upAt(center))
	if clicks != 1 {
		t.Fatalf("a press+release without dragging should click once, got %d", clicks)
	}
}

func TestEscapeCancelsActiveDrag(t *testing.T) {
	app, src, _ := dndApp(t)
	src.SetDragSource(func() *DragData { return &DragData{Type: "x"} })
	var endAccepted *bool
	src.OnDragEnd(func(accepted bool) { endAccepted = &accepted })

	app.dispatchPointer(downAt(srcCenter))
	app.dispatchPointer(heldMoveTo(pastOrig))
	if app.drag == nil || !app.drag.started {
		t.Fatal("drag should be active before Escape")
	}
	app.dispatchKeyboard(render.InputState{KeysPressed: []render.Key{render.KeyEscape}})
	if app.drag != nil {
		t.Fatal("Escape should cancel the drag")
	}
	if endAccepted == nil || *endAccepted {
		t.Fatalf("cancelled drag should report OnDragEnd(false), got %v", endAccepted)
	}
}
