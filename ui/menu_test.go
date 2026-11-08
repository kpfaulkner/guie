package ui

import (
	"reflect"
	"testing"

	"github.com/kpfaulkner/guie/geom"
)

func TestTidyMenuDropsSeparatorsWithNothingToSeparate(t *testing.T) {
	sep := NewMenuSeparator()
	a, b := MenuItem{Label: "a"}, MenuItem{Label: "b"}
	labels := func(items []MenuItem) []string {
		out := []string{}
		for _, it := range items {
			if it.Separator {
				out = append(out, "-")
			} else {
				out = append(out, it.Label)
			}
		}
		return out
	}
	for _, tc := range []struct {
		in   []MenuItem
		want []string
	}{
		{nil, []string{}},
		{[]MenuItem{sep}, []string{}},
		{[]MenuItem{sep, sep}, []string{}},
		{[]MenuItem{a, b}, []string{"a", "b"}},
		{[]MenuItem{a, sep, b}, []string{"a", "-", "b"}},
		{[]MenuItem{sep, a, b}, []string{"a", "b"}},
		{[]MenuItem{a, b, sep}, []string{"a", "b"}},
		{[]MenuItem{a, sep, sep, sep, b}, []string{"a", "-", "b"}},
		{[]MenuItem{sep, a, sep, sep, b, sep, sep}, []string{"a", "-", "b"}},
	} {
		if got := labels(tidyMenu(tc.in)); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%v: got %v, want %v", labels(tc.in), got, tc.want)
		}
	}
}

func TestMenuPanelBuildsEachKindOfItem(t *testing.T) {
	app := NewApp()
	var ran []string
	items := []MenuItem{
		NewMenuItem("on", func() { ran = append(ran, "on") }),
		NewMenuSeparator(),
		{Label: "off", Action: func() { ran = append(ran, "off") }, Disabled: true},
	}
	panel, size := menuPanel(app.theme, nil, items, func(action func()) { action() })
	kids := panel.Children()
	if len(kids) != 3 {
		t.Fatalf("%d children", len(kids))
	}
	if !kids[0].(*menuRow).Enabled() || kids[2].(*menuRow).Enabled() {
		t.Fatal("wrong rows enabled")
	}
	if _, ok := kids[1].(*menuSeparator); !ok {
		t.Fatalf("the middle child is a %T", kids[1])
	}

	// The popup is as tall as what is in it, or rows would be squeezed.
	rowH := app.theme.Font.Measure("Ag").H + 2*menuRowPad*densityOf(app.theme)
	if want := 2*rowH + menuSeparatorH*densityOf(app.theme); size.H != want {
		t.Fatalf("height %v, want %v", size.H, want)
	}

	// A disabled row does not respond, even if it is sent a click.
	off := kids[2].(*menuRow)
	if off.HandleEvent(&Event{Type: EventClick}) || off.HandleEvent(&Event{Type: EventPointerEnter}) || off.hover {
		t.Fatal("a disabled row responded")
	}
	if len(ran) != 0 {
		t.Fatalf("ran %v", ran)
	}
	kids[0].(*menuRow).HandleEvent(&Event{Type: EventClick})
	if !reflect.DeepEqual(ran, []string{"on"}) {
		t.Fatalf("ran %v", ran)
	}
}

func TestShowContextMenuOfOnlySeparatorsShowsNothing(t *testing.T) {
	app := NewApp()
	app.resize(400, 300)
	if p := app.ShowContextMenu(geom.Point{X: 5, Y: 5}, NewMenuSeparator(), NewMenuSeparator()); p != nil || len(app.overlays) != 0 {
		t.Fatalf("popup %v, %d overlays", p, len(app.overlays))
	}
}
