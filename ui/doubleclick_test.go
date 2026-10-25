package ui

import (
	"testing"
	"time"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
)

// dcClick performs a full left click (press then release) through the App.
func dcClick(a *App, pos geom.Point) {
	a.dispatchPointer(downAt(pos))
	a.dispatchPointer(upAt(pos))
}

// dcApp returns an App with a button whose double-clicks are counted, and a
// controllable clock.
func dcApp(t *testing.T) (a *App, b *Button, doubles *int, clicks *int, clock *time.Time) {
	t.Helper()
	n, c := 0, 0
	a, b = appWithButton(func() { c++ })
	b.OnDoubleClick(func(geom.Point) bool { n++; return true })
	now := time.Unix(1000, 0)
	a.now = func() time.Time { return now }
	return a, b, &n, &c, &now
}

var inButton = geom.Point{X: 20, Y: 20}

func TestDoubleClickFiresOnSecondQuickClick(t *testing.T) {
	a, _, doubles, clicks, now := dcApp(t)
	dcClick(a, inButton)
	if *doubles != 0 {
		t.Fatal("a single click must not double-click")
	}
	*now = now.Add(100 * time.Millisecond)
	dcClick(a, inButton)
	if *doubles != 1 {
		t.Fatalf("doubles = %d, want 1", *doubles)
	}
	if *clicks != 2 {
		t.Errorf("clicks = %d, want 2: each click still delivers EventClick", *clicks)
	}
}

func TestDoubleClickTooSlow(t *testing.T) {
	a, _, doubles, _, now := dcApp(t)
	dcClick(a, inButton)
	*now = now.Add(doubleClickInterval + time.Millisecond)
	dcClick(a, inButton)
	if *doubles != 0 {
		t.Error("clicks further apart than the interval must not double-click")
	}
	// The slow click becomes the first of a new pair.
	*now = now.Add(50 * time.Millisecond)
	dcClick(a, inButton)
	if *doubles != 1 {
		t.Errorf("doubles = %d, want 1 after a quick follow-up", *doubles)
	}
}

func TestDoubleClickTooFarApart(t *testing.T) {
	a, _, doubles, _, now := dcApp(t)
	dcClick(a, inButton)
	*now = now.Add(50 * time.Millisecond)
	dcClick(a, geom.Point{X: inButton.X + doubleClickSlop + 3, Y: inButton.Y})
	if *doubles != 0 {
		t.Error("clicks too far apart must not double-click")
	}
}

func TestDoubleClickTripleIsNotTwoDoubles(t *testing.T) {
	a, _, doubles, _, now := dcApp(t)
	for i := 0; i < 3; i++ {
		dcClick(a, inButton)
		*now = now.Add(50 * time.Millisecond)
	}
	if *doubles != 1 {
		t.Errorf("doubles = %d, want 1: the third click starts a new sequence", *doubles)
	}
	dcClick(a, inButton)
	if *doubles != 2 {
		t.Errorf("doubles = %d, want 2 after the fourth click", *doubles)
	}
}

func TestDoubleClickNeedsSameWidget(t *testing.T) {
	a := NewApp()
	root := NewContainer()
	root.SetBounds(geom.Rect{W: 200, H: 200})
	b1, b2 := NewButton("a"), NewButton("b")
	b1.SetBounds(geom.Rect{X: 10, Y: 10, W: 50, H: 30})
	b2.SetBounds(geom.Rect{X: 100, Y: 10, W: 50, H: 30})
	root.Add(b1)
	root.Add(b2)
	a.SetContent(root)
	n := 0
	h := func(geom.Point) bool { n++; return true }
	b1.OnDoubleClick(h)
	b2.OnDoubleClick(h)
	now := time.Unix(1000, 0)
	a.now = func() time.Time { return now }

	dcClick(a, geom.Point{X: 20, Y: 20})
	dcClick(a, geom.Point{X: 110, Y: 20})
	if n != 0 {
		t.Error("clicks on different widgets must not double-click")
	}
}

func TestDoubleClickBubblesUnlessConsumed(t *testing.T) {
	a, b, doubles, _, now := dcApp(t)
	parent := 0
	a.root.(*Container).OnDoubleClick(func(geom.Point) bool { parent++; return true })

	b.OnDoubleClick(func(geom.Point) bool { *doubles++; return false }) // declines
	dcClick(a, inButton)
	*now = now.Add(50 * time.Millisecond)
	dcClick(a, inButton)
	if *doubles != 1 || parent != 1 {
		t.Errorf("declined: child=%d parent=%d, want 1 and 1", *doubles, parent)
	}

	*now = now.Add(time.Second)
	b.OnDoubleClick(func(geom.Point) bool { *doubles++; return true }) // consumes
	dcClick(a, inButton)
	*now = now.Add(50 * time.Millisecond)
	dcClick(a, inButton)
	if parent != 1 {
		t.Errorf("parent = %d, want 1: a consumed double-click must not bubble", parent)
	}
}

func TestDoubleClickReachesBusAndHandleEvent(t *testing.T) {
	a, _, _, _, now := dcApp(t)
	got := 0
	a.Events().Subscribe(EventDoubleClick, func(Event) { got++ })
	dcClick(a, inButton)
	*now = now.Add(50 * time.Millisecond)
	dcClick(a, inButton)
	if got != 1 {
		t.Errorf("bus got %d EventDoubleClick, want 1", got)
	}

	ev := Event{Type: EventDoubleClick, Button: render.MouseLeft}
	if NewButton("x").HandleEvent(&ev) {
		t.Error("a Button has no double-click behaviour and should not consume it")
	}
}
