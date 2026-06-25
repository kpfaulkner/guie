# Tutorial: windows, containers and widgets

This tutorial walks through the three things every guie program is built from:

1. an **App** — the window and the main loop,
2. **Containers** with a **layout** — the boxes that arrange things,
3. **Widgets** — the labels, buttons and inputs the user sees and touches.

By the end you'll have a small but complete window with a couple of nested
containers and a handful of interactive widgets.

> Application code imports only `ui`, `geom`, `render` and `theme` — never
> Ebiten. The whole public API in this tutorial lives in the `ui` package, with
> a few value types (insets, alignment) coming from `geom`.

---

## 1. The smallest possible window

Every program creates an `App`, gives it some content, and calls `Run`. `Run`
blocks until the window is closed, so it's the last thing `main` does.

```go
// Run with: go run .
package main

import (
	"log"

	"github.com/kpfaulkner/guie/ui"
)

func main() {
	app := ui.NewApp(
		ui.WithTitle("guie — tutorial"),
		ui.WithSize(480, 360),
	)

	app.SetContent(ui.NewLabel("Hello, guie!"))

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
```

`ui.NewApp` takes a variadic list of options. The common ones:

| Option | Effect |
| --- | --- |
| `ui.WithTitle(string)` | window title bar text |
| `ui.WithSize(w, h int)` | initial window size, in logical pixels |
| `ui.WithResizable(bool)` | allow the user to resize (default `true`) |
| `ui.WithBackground(color.Color)` | override the theme's clear colour |

`app.SetContent(w)` installs a single widget as the **root** of the window. The
root is automatically resized to fill the surface. A bare label works, but a
real UI needs more than one widget — which is where containers come in.

---

## 2. Containers and layouts

A `Container` is a widget that holds other widgets. On its own a container does
not position its children; you give it a **layout** to do that. The most common
layouts are:

- **`ui.VBox(spacing)`** — stack children top to bottom.
- **`ui.HBox(spacing)`** — lay children left to right.
- **`ui.NewGrid(columns, spacing)`** — a grid of equal columns.
- **`ui.NewStack()`** — overlay children in the same space (handy for centering).

A container also has **padding** (inner space around its children) and an
optional **background** colour.

```go
root := ui.NewContainer()
root.SetLayout(ui.VBox(12))                  // 12px gap between rows
root.SetPadding(geom.UniformInsets(16))      // 16px breathing room on all sides

root.Add(ui.NewLabel("First row"))
root.Add(ui.NewLabel("Second row"))

app.SetContent(root)
```

`geom.UniformInsets(16)` is shorthand for `geom.Insets{Top: 16, Right: 16,
Bottom: 16, Left: 16}`; set the fields individually when you want different
amounts per side.

### Per-child layout options

`Add` takes optional per-child parameters after the widget:

- **`ui.Weight(n)`** — the child's share of leftover space along the layout's
  main axis. `Weight(0)` (the default) means "just use your natural size";
  higher weights take proportionally more of what's left over.
- **`ui.Align(a)`** — how the child sits in the space the layout gives it on the
  *cross* axis: `geom.AlignStart`, `AlignCenter`, `AlignEnd`, or `AlignStretch`
  (the default — fill the space).
- **`ui.Span(cols, rows)`** — for grids only: how many cells the child covers.

```go
// A toolbar pinned to the top, then a content area that eats the rest.
root.Add(toolbar)                 // natural height
root.Add(content, ui.Weight(1))   // grabs all remaining vertical space
```

### Nesting containers

This is the key idea: a container *is* a widget, so containers go inside other
containers. A vertical column whose rows are themselves horizontal rows is just
a `VBox` containing `HBox` children.

```go
row := ui.NewContainer()
row.SetLayout(ui.HBox(8))
row.Add(ui.NewLabel("Name:"))
row.Add(ui.NewTextField(ui.Placeholder("type here…")), ui.Weight(1))

root.Add(row) // the whole row becomes one item in the outer VBox
```

---

## 3. Adding widgets

Widgets are created with `ui.NewXxx(...)` constructors and wired up with
`OnXxx` callbacks. A few of the everyday ones:

| Widget | Constructor | Notable methods |
| --- | --- | --- |
| Label | `ui.NewLabel("text")` | `SetText`, `LabelColour(...)` option |
| Button | `ui.NewButton("text")` | `OnClick(func())`, `SetEnabled`, `SetText` |
| Text field | `ui.NewTextField(...)` | `OnChange(func(string))`, `Text()`, `Placeholder(...)` option |
| Checkbox | `ui.NewCheckbox("label")` | `OnChange(func(bool))` |
| Slider | `ui.NewSlider(...)` | `OnChange(func(float64))`, `Value()`, `SliderValue(v)` option |

Callbacks run on the UI goroutine, so it's safe to touch other widgets from
inside them (e.g. set a label's text when a button is clicked).

---

## 4. Putting it together

Here's a complete program with a window, two nested containers, and a few
interactive widgets. A text field and a slider feed a label, and a button
clears them.

```go
// Run with: go run .
package main

import (
	"fmt"
	"log"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/ui"
)

func main() {
	app := ui.NewApp(
		ui.WithTitle("guie — basics"),
		ui.WithSize(480, 320),
	)

	// The label we'll keep up to date as the user interacts.
	status := ui.NewLabel("Say something…")

	name := ui.NewTextField(ui.Placeholder("your name"))
	volume := ui.NewSlider(ui.SliderValue(0.5))

	update := func() {
		status.SetText(fmt.Sprintf("Hi %s — volume %.0f%%",
			name.Text(), volume.Value()*100))
	}
	name.OnChange(func(string) { update() })
	volume.OnChange(func(float64) { update() })

	// --- A labelled row: "Name:" beside the text field. ---
	nameRow := ui.NewContainer()
	nameRow.SetLayout(ui.HBox(8))
	nameRow.Add(ui.NewLabel("Name:"))
	nameRow.Add(name, ui.Weight(1)) // field stretches to fill the row

	// --- A labelled row for the slider. ---
	volRow := ui.NewContainer()
	volRow.SetLayout(ui.HBox(8))
	volRow.Add(ui.NewLabel("Volume:"))
	volRow.Add(volume, ui.Weight(1))

	// --- A button that clears everything. ---
	clear := ui.NewButton("Clear")
	clear.OnClick(func() {
		name.SetText("")
		volume.SetValue(0.5)
		update()
	})

	// --- The root column that stacks everything vertically. ---
	root := ui.NewContainer()
	root.SetLayout(ui.VBox(12))
	root.SetPadding(geom.UniformInsets(16))
	root.Add(ui.NewLabel("guie basics: containers + widgets"))
	root.Add(nameRow)
	root.Add(volRow)
	root.Add(clear, ui.Align(geom.AlignStart)) // don't stretch the button
	root.Add(status, ui.Weight(1))             // status fills the leftover space

	app.SetContent(root)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
```

What's happening, top to bottom:

- The **root** is a `VBox`, so its items are stacked as rows with 12px gaps and
  16px of padding around the lot.
- `nameRow` and `volRow` are **nested `HBox` containers**. Inside each, the
  label takes its natural width and the input gets `Weight(1)` so it stretches
  to fill the rest of the row.
- The **button** is added with `ui.Align(geom.AlignStart)` so it hugs the left
  edge instead of stretching across the window.
- The final **status label** has `Weight(1)`, so it soaks up all the vertical
  space left after the fixed-height rows — try resizing the window and watch it
  grow.

---

## 5. Where to go next

- **Other layouts** — try swapping the root's `VBox` for `ui.NewGrid(2, 8)` and
  see how the items flow into a 2-column grid; use `ui.Span(2, 1)` to make one
  item span both columns.
- **Centering** — wrap a single widget in a container with `ui.NewStack()` and
  add it with `ui.Align(geom.AlignCenter)`.
- **More widgets** — there are checkboxes, radio groups, dropdowns, lists,
  trees, tables, tabs, dialogs and more. Browse the runnable programs under
  [`examples/`](../examples) — each one is a focused demo you can run with
  `go run ./examples/<name>`.
- **Styling** — `ui.WithTheme(...)` and per-widget options (e.g.
  `ui.LabelColour(...)`) control colours; `SetBackground`, `SetBorder` and
  `SetCornerRadius` on a container let you draw panels.

Good next reads are the [containers and widget placement tutorial](tutorial-containers-and-placement.md)
for the layout engine in depth, and the [events and stacked widgets tutorial](tutorial-events-and-overlays.md)
for input handling, dialogs and overlays — or jump straight to
[`examples/layouts`](../examples/layouts) and [`examples/widgets`](../examples/widgets).
