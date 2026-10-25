# guie

A cross-platform GUI framework for Go: windows, layouts and a catalogue of
widgets with theming, events and animations. It renders on
[Ebiten](https://github.com/hajimehoshi/ebiten), but that is an internal
detail - application code imports only `ui`, `geom`, `render` and `theme`, never
Ebiten.

> **Status: v0.x.** The API is still settling and may change between releases;
> expect to pin a version. Development is on Windows; macOS and Linux should work
> (Ebiten is cross-platform and guie has almost no OS-specific code) but have not
> been run yet, so treat them as untested. See [Platforms](#platforms).

## What is in it

- **Layout:** `Container` with `VBox` / `HBox` / `Grid` / `Stack`, weights,
  alignment, padding, `SplitPane`.
- **Widgets:** `Label`, `Button` (text or image), `Checkbox`, `RadioButton`,
  `TextField` (optionally masked), `TextArea` (soft wrap, selection, undo/redo),
  `Slider`, `Stepper`, `ProgressBar`, `Spinner`, `List`, `Tree`, `Table`
  (sortable), `DropdownCombo`, `MenuBar` and context menus, `TabContainer`,
  `ScrollView` and a standalone `ScrollBar`, `DatePicker` / `DateField`,
  `ColourPicker`, `Image`, tooltips and toasts.
- **Custom drawing:** `CanvasWidget` for charts, paint surfaces and HUDs, drawn
  with the same `render.Canvas` primitives the built-in widgets use.
- **Dialogs:** modal dialogs, `App.ShowMessage` and `App.ShowConfirm`.
- **Input:** callbacks as the main API plus an optional event bus, focus and Tab
  order, keyboard accelerators, double-click, in-process drag-and-drop, OS file
  drops, clipboard (the OS clipboard by default), and IME for committed text.
- **Scrolling:** wheel, draggable scroll bars and gutter paging everywhere
  content overflows; horizontal scrolling is opt-in on `ScrollView`, `Table`,
  `List` and `Tree` and automatic in an unwrapped `TextArea`.
- **Look:** a dark theme and a light theme you can switch at runtime, per-widget
  colour overrides, fonts, HiDPI scaling, animated hover and focus.
- **App plumbing:** frames are drawn on demand so an idle window is nearly free,
  `App.Do` for updates from other goroutines, window size and position saved
  across runs (`ui.WithWindowStateFile`).
- **Testing:** a headless backend, `guitest`, to test a UI with no window or GPU
  (see [Testing](#testing)).

Not done yet: accessibility (screen readers), rich text, data binding, multiple
OS windows, native file dialogs and a native macOS menu bar. They are tracked in
[`design.md`](design.md).

## Documentation

- [`docs/`](docs/tutorial-basics.md) - tutorials: windows and widgets,
  containers and placement, events and overlays.
- [`examples/`](examples/README.md) - small runnable programs for each feature.
- [`design.md`](design.md) - the architecture, decisions and roadmap.
- [`INTERNALS.md`](INTERNALS.md) - how it is built, for contributors.

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


## Requirements

- Go 1.25 or newer. `go get github.com/kpfaulkner/guie`.
- Windows builds without cgo or any extra tools. macOS and Linux build with cgo, so they need a C toolchain (on macOS, the Xcode command-line tools).

### Platforms

| Platform | State |
|---|---|
| Windows | Developed and tested here. |
| macOS | Expected to work, **untested**. HiDPI and the Cmd-based shortcuts are implemented but have not been seen on real hardware. |
| Linux | Expected to work, **untested**. Ebiten needs cgo here, so a C compiler and the X11/GL development packages are required (on Debian/Ubuntu: `libgl1-mesa-dev xorg-dev libasound2-dev`). |
| Raspberry Pi | Builds natively on a Pi (arm64); a Pi 4 or 5 with the desktop Mesa driver is the realistic target. Untested. |
| Web (WASM) | The library packages build for `GOOS=js GOARCH=wasm`; running it in a browser has not been tried. |

`guitest` needs no display or GPU, so UI tests run on a headless CI machine. The
packages still link Ebiten, though, so on Linux and macOS `go test` needs the cgo
toolchain above (it does not on Windows).

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
- `h.DropFiles(map[string][]byte)` simulates dropping files from a file manager at
  the current mouse position (name → content) and steps one frame to deliver them,
  so `OnFileDrop` routing and bubbling are testable headlessly.
- `h.RequestClose()` simulates the user closing the window and reports whether the
  close was allowed — false when an `App.OnCloseRequest` handler vetoed it (the
  loop keeps running, so `Step` still works afterwards). `h.CloseHandled()` reports
  whether the app asked the backend to stop closing the window by itself.

### Asserting what was drawn

`Step()`/`Frame()` return a `*Recording` — the ordered list of drawing ops for
that frame. Query it without a real surface:

- `HasText(s)`, `TextContaining(substr)`, `Texts()`
- `Count(kind)`, `OpsOfKind(kind)`
- `FillsOfColour(c)` — rectangles filled with a colour (e.g. a selection highlight)
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

## License

Apache License 2.0. See [`LICENSE`](LICENSE).
