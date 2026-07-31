package guitest

import (
	"testing"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/ui"
)

// dropTree builds a root container holding one child panel in a known place, both
// recording what they receive. child consumes drops only when consumeChild is
// true, so the bubbling behaviour can be exercised both ways.
func dropTree(consumeChild bool) (h *Harness, gotChild, gotRoot *[]ui.FileDrop, childBounds geom.Rect) {
	h = New(400, 300)

	child := ui.NewContainer()
	child.SetBounds(geom.Rect{X: 50, Y: 40, W: 100, H: 80})

	var childDrops, rootDrops []ui.FileDrop
	child.OnFileDrop(func(d ui.FileDrop, _ geom.Point) bool {
		childDrops = append(childDrops, d)
		return consumeChild
	})

	root := ui.NewContainer()
	root.OnFileDrop(func(d ui.FileDrop, _ geom.Point) bool {
		rootDrops = append(rootDrops, d)
		return true
	})
	root.Add(child)
	h.SetContent(root)
	// Lay out and settle hover state before the drop.
	h.Step()

	return h, &childDrops, &rootDrops, child.Bounds()
}

func TestFileDropReachesWidgetUnderCursor(t *testing.T) {
	h, childDrops, rootDrops, b := dropTree(true)
	h.MoveMouse(b.X+b.W/2, b.Y+b.H/2)

	h.DropFiles(map[string][]byte{"tile.png": []byte("pixels")})

	if len(*childDrops) != 1 {
		t.Fatalf("child received %d drops, want 1", len(*childDrops))
	}
	if len(*rootDrops) != 0 {
		t.Errorf("root received %d drops, want 0 (the child consumed it)", len(*rootDrops))
	}
	d := (*childDrops)[0]
	if len(d.Names) != 1 || d.Names[0] != "tile.png" {
		t.Errorf("names = %v, want [tile.png]", d.Names)
	}
	data, err := d.ReadFile("tile.png")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "pixels" {
		t.Errorf("content = %q, want %q", data, "pixels")
	}
}

// An unconsumed drop must bubble, so a container can catch what its children
// decline.
func TestFileDropBubblesWhenNotConsumed(t *testing.T) {
	h, childDrops, rootDrops, b := dropTree(false)
	h.MoveMouse(b.X+5, b.Y+5)

	h.DropFiles(map[string][]byte{"a.nwf": {1, 2, 3}})

	if len(*childDrops) != 1 {
		t.Errorf("child received %d drops, want 1", len(*childDrops))
	}
	if len(*rootDrops) != 1 {
		t.Errorf("root received %d drops, want 1 (child declined)", len(*rootDrops))
	}
}

// A drop away from any child still belongs to the window.
func TestFileDropOffChildGoesToRoot(t *testing.T) {
	h, childDrops, rootDrops, b := dropTree(true)
	h.MoveMouse(b.X+b.W+60, b.Y+b.H+60)

	h.DropFiles(map[string][]byte{"a.nwf": {1}})

	if len(*childDrops) != 0 {
		t.Errorf("child received %d drops, want 0", len(*childDrops))
	}
	if len(*rootDrops) != 1 {
		t.Errorf("root received %d drops, want 1", len(*rootDrops))
	}
}

// Several files arrive as one event; the application decides what to do with them.
func TestFileDropCarriesEveryName(t *testing.T) {
	h, _, rootDrops, _ := dropTree(true)
	h.MoveMouse(300, 250)

	h.DropFiles(map[string][]byte{"a.nwf": {1}, "b.nwf": {2}, "c.txt": {3}})

	if len(*rootDrops) != 1 {
		t.Fatalf("root received %d drops, want 1", len(*rootDrops))
	}
	if got := len((*rootDrops)[0].Names); got != 3 {
		t.Errorf("names = %v, want 3 entries", (*rootDrops)[0].Names)
	}
}

// The drop is an edge: the next frame must not redeliver it.
func TestFileDropIsOneFrameOnly(t *testing.T) {
	h, _, rootDrops, _ := dropTree(true)
	h.MoveMouse(300, 250)

	h.DropFiles(map[string][]byte{"a.nwf": {1}})
	h.Step()
	h.Step()

	if len(*rootDrops) != 1 {
		t.Errorf("root received %d drops across three frames, want 1", len(*rootDrops))
	}
}

// The position handed to the handler is the cursor at the drop, so a widget can
// tell where inside itself the file landed.
func TestFileDropReportsPosition(t *testing.T) {
	h := New(400, 300)
	var got geom.Point
	root := ui.NewContainer()
	root.OnFileDrop(func(_ ui.FileDrop, pos geom.Point) bool {
		got = pos
		return true
	})
	h.SetContent(root)
	h.Step()

	want := geom.Point{X: 123, Y: 45}
	h.MoveMouse(want.X, want.Y)
	h.DropFiles(map[string][]byte{"a.nwf": {1}})

	if got != want {
		t.Errorf("drop position = %v, want %v", got, want)
	}
}

// A modal owns the surface: drops on the blocked background must not reach it.
func TestFileDropBlockedByModal(t *testing.T) {
	h, _, rootDrops, _ := dropTree(true)
	h.App.ShowMessage("Busy", "Not now")
	h.Step()

	// Far corner — on the scrim, not on the dialog panel.
	h.MoveMouse(5, 5)
	h.DropFiles(map[string][]byte{"a.nwf": {1}})

	if len(*rootDrops) != 0 {
		t.Errorf("root received %d drops behind a modal, want 0", len(*rootDrops))
	}
}

// The event is observable on the bus like every other event.
func TestFileDropPublishedToBus(t *testing.T) {
	h := New(400, 300)
	h.SetContent(ui.NewContainer())
	h.Step()

	seen := 0
	h.App.Events().Subscribe(ui.EventFileDrop, func(ev ui.Event) {
		if len(ev.Files.Names) == 1 && ev.Files.Names[0] == "a.nwf" {
			seen++
		}
	})
	h.MoveMouse(100, 100)
	h.DropFiles(map[string][]byte{"a.nwf": {1}})

	if seen != 1 {
		t.Errorf("bus saw %d matching EventFileDrop, want 1", seen)
	}
}
