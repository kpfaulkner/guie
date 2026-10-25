package ui

import (
	"testing"

	"github.com/kpfaulkner/guie/render"
)

func tabsApp(t *testing.T) (*App, *TabContainer, *stub, *stub) {
	t.Helper()
	app := NewApp()
	tc := NewTabContainer()
	a := newStub(10, 10)
	b := newStub(10, 10)
	tc.AddTab("One", a)
	tc.AddTab("Two", b)
	app.SetContent(tc)
	return app, tc, a, b
}

func TestTabsDefaultSelectsFirst(t *testing.T) {
	_, tc, a, _ := tabsApp(t)
	if tc.Selected() != 0 {
		t.Fatalf("default selected tab should be 0, got %d", tc.Selected())
	}
	kids := tc.Children()
	if len(kids) != 1 || kids[0] != Widget(a) {
		t.Fatalf("only the active pane should be a child")
	}
}

func TestTabsClickSwitches(t *testing.T) {
	changed := -1
	_, tc, _, b := tabsApp(t)
	tc.OnChange(func(i int) { changed = i })

	ev := Event{Type: EventClick, Pos: tc.titleRect(1).Center()}
	tc.HandleEvent(&ev)

	if tc.Selected() != 1 || changed != 1 {
		t.Fatalf("clicking tab 1 should select it: selected=%d cb=%d", tc.Selected(), changed)
	}
	if tc.Children()[0] != Widget(b) {
		t.Fatalf("active pane should now be the second content")
	}
}

func TestTabsKeyboardSwitch(t *testing.T) {
	_, tc, _, _ := tabsApp(t)
	right := Event{Type: EventKeyDown, Key: render.KeyRight}
	tc.HandleEvent(&right)
	if tc.Selected() != 1 {
		t.Fatalf("Right should select tab 1, got %d", tc.Selected())
	}
	left := Event{Type: EventKeyDown, Key: render.KeyLeft}
	tc.HandleEvent(&left)
	if tc.Selected() != 0 {
		t.Fatalf("Left should select tab 0, got %d", tc.Selected())
	}
	// Left at the first tab is a no-op (clamped).
	tc.HandleEvent(&left)
	if tc.Selected() != 0 {
		t.Fatalf("Left at first tab should stay at 0, got %d", tc.Selected())
	}
}

func TestTabsLayoutPositionsActivePane(t *testing.T) {
	app, tc, a, _ := tabsApp(t)
	app.resize(200, 150)
	app.layoutIfNeeded()

	got := a.Bounds()
	cr := tc.contentRect()
	if got != cr {
		t.Fatalf("active pane bounds %+v should equal content rect %+v", got, cr)
	}
	if cr.Y <= tc.Bounds().Y {
		t.Fatalf("content area should sit below the tab strip")
	}
}

func TestTabIndicatorSlidesThenSettles(t *testing.T) {
	app, tc, _, _ := tabsApp(t)
	app.resize(400, 300)
	frame(t, app)

	from, to := tc.span(0), tc.span(1)
	tc.Select(1)
	if got := tc.indicatorSpan(); got != from {
		t.Errorf("indicator should start where it was: got %+v want %+v", got, from)
	}
	frame(t, app)
	frame(t, app)
	if mid := tc.indicatorSpan(); mid == from || mid == to {
		t.Errorf("indicator should be mid-slide after two frames, got %+v", mid)
	}
	for range fadeFrames * 2 {
		frame(t, app)
	}
	if got := tc.indicatorSpan(); got != to {
		t.Errorf("indicator should settle on the new tab: got %+v want %+v", got, to)
	}
	if tc.indAnim != nil {
		t.Error("slide animation should be cleared once settled")
	}
}

func TestTabHoverFades(t *testing.T) {
	app, tc, _, _ := tabsApp(t)
	app.resize(400, 300)
	frame(t, app)

	tc.HandleEvent(&Event{Type: EventPointerMove, Pos: tc.titleRect(1).Center()})
	for range fadeFrames {
		frame(t, app)
	}
	if tc.hoverFades[1].level != 1 || tc.hoverFades[0].level != 0 {
		t.Errorf("hover fades: got %v / %v, want 0 / 1", tc.hoverFades[0].level, tc.hoverFades[1].level)
	}
	tc.HandleEvent(&Event{Type: EventPointerLeave})
	for range fadeFrames {
		frame(t, app)
	}
	if tc.hoverFades[1].level != 0 {
		t.Errorf("hover fade should return to 0, got %v", tc.hoverFades[1].level)
	}
}
