package ui

import (
	"testing"

	"github.com/kpfaulkner/uiframework/render"
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
	tc.OnTabChange(func(i int) { changed = i })

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
