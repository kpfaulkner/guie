# guie

A cross-platform GUI framework for Go: windows, layouts and a catalogue of
widgets (buttons, lists, trees, tables, text fields, menus, dialogs, drag-and-
drop, toasts, …) with theming, events and animations. It renders on
[Ebiten](https://github.com/hajimehoshi/ebiten), but that is an internal
detail — application code imports only `ui`, `geom`, `render` and `theme`, never
Ebiten.

See [`design.md`](design.md) for the architecture and decisions, and
[`internals.md`](internals.md) for the implementation guide.

## Quick start

```go
package main

import (
	"log"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/ui"
)

func main() {
	app := ui.NewApp(ui.WithTitle("Hello"), ui.WithSize(400, 200))

	quit := ui.NewButton("Quit")
	quit.OnClick(func() { app.Quit() })

	root := ui.NewContainer()
	root.SetLayout(ui.VBox(10))
	root.SetPadding(geom.UniformInsets(16))
	root.Add(ui.NewLabel("Hello, guie!"))
	root.Add(quit)

	app.SetContent(root)
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
```

Runnable examples live in [`examples/`](examples/README.md):

```sh
go run ./examples/hello       # smallest program
go run ./examples/tree        # Tree widget + Toast notifications
go run ./examples/dragdrop    # drag-and-drop between panels
```

## Testing

guie ships a **headless test backend**, the
[`guitest`](https://pkg.go.dev/github.com/kpfaulkner/guie/guitest) package, so
you can integration-test a UI with no window, GPU or display — it runs anywhere
`go test` runs, including CI. It implements the render seam (driver, canvas,
font) and a `Harness` that drives an `App` one frame at a time: synthesize
input, step the loop, and assert against widget state or the recorded drawing
operations.

### Writing a UI test

Put it in a normal `_test.go` file and run it with `go test` — there is no
separate runner:

```go
package myui_test

import (
	"testing"

	"github.com/kpfaulkner/guie/guitest"
	"github.com/kpfaulkner/guie/ui"
)

func TestSaveButtonFires(t *testing.T) {
	h := guitest.New(200, 100) // headless app, 200x100 logical pixels

	clicked := false
	btn := ui.NewButton("Save")
	btn.OnClick(func() { clicked = true })
	h.SetContent(btn) // the root fills the surface

	h.Click(100, 50) // press + release at the center (lays out, then dispatches)

	if !clicked {
		t.Fatal("button did not fire")
	}
	if rec := h.Frame(); !rec.HasText("Save") {
		t.Fatalf("label not drawn; texts = %v", rec.Texts())
	}
}
```

```sh
go test ./...            # run everything
go test ./path/to/myui   # just your package
```

### Driving the app

- `h.Step()` runs one frame (Update then Draw) with the input accumulated so far
  and returns that frame's `*Recording`.
- **Low-level input** (build a frame, then `Step`): `MoveMouse`, `PressMouse` /
  `ReleaseMouse`, `ScrollBy`, `PressKey` / `ReleaseKey`, `TypeText` / `TypeRune`,
  `SetModifiers`.
- **Gestures** (each performs its own steps): `Click`, `RightClick`, `Drag`,
  `TypeKey`.
- `h.Resize(w, h)` reports a new surface size; `h.App` exposes the app so you can
  read widget state in assertions.

### Asserting what was drawn

`Step()`/`Frame()` return a `*Recording` — the ordered list of drawing ops for
that frame. Query it without a real surface:

- `HasText(s)`, `TextContaining(substr)`, `Texts()`
- `Count(kind)`, `OpsOfKind(kind)`
- `FillsOfColor(c)` — rectangles filled with a color (e.g. a selection highlight)
- `TextAt(x, y, tol)` — text drawn near a point

The headless font (`guitest.NewFont`) has simple, **deterministic** metrics
(fixed per-rune advance and line height), so measurements and layout are
predictable and independent of the bundled font.

> **Caveat:** `ui.NewRenderTarget` still uses the real Ebiten backend (it is not
> part of the driver seam), so avoid it in headless tests. For a drag, set a
> custom `DragGhost` instead of relying on the default snapshot ghost. See
> [`guitest/harness_test.go`](guitest/harness_test.go) for worked examples.

### Running the framework's own tests

```sh
go test ./...            # unit + black-box tests across all packages
go test ./guitest/ -v    # the headless harness self-tests
```

GUI examples can't run headlessly (they open a window); they all compile and are
the manual "does it actually render" check via `go run ./examples/<name>`.
