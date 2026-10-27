# DESIGN

## GOAL

Make a Golang library that can be used to make cross-platform applications.
This is a user interface library/framework that provides things such as
windows, widgets, events, scrolling, menus, lists, dropdown combos etc.

As a graphical basis it uses the EBiten (`github.com/hajimehoshi/ebiten/v2`)
library for rendering, windowing and input.

**Core principle: EBiten is an internal implementation detail.** A developer
using this framework should never need to know EBiten exists. They do not
import it, do not implement `ebiten.Game`, and do not see `*ebiten.Image` or
`ebiten` input types in any public API. The framework owns the main loop and
exposes its own abstractions for drawing surfaces, colours, input and events. If
we ever swap the rendering backend, application code should not change.

Features that should be available:

- windows
- buttons
- scroll bars
- menus
- lists
- checkboxes
- radio buttons
- events
- themes
- control of window and widget colours
- control of fonts

---

## DESIGN DECISIONS (locked)

| Area          | Decision                                                                 |
|---------------|--------------------------------------------------------------------------|
| Layout        | Layout managers (HBox / VBox / Grid / Flex). Widgets resize with parent. |
| Architecture  | Retained widget tree — persistent, stateful widget objects.              |
| API style     | Struct + methods, constructed via functional options.                    |
| Theming       | Simple named colour palette + a default font, to start.                   |
| Events        | Callbacks as the primary API; optional event bus for global listening.   |
| Main loop     | Framework owns the loop. EBiten fully abstracted away.                    |
| Fonts         | Bundle a default TTF so apps work out of the box; allow custom fonts.    |
| Build order   | Finish this design, then implement a core vertical slice first.          |

---

## ARCHITECTURE

### Layered structure

```
+-----------------------------------------------------------+
|  Application code (no ebiten imports)                     |
+-----------------------------------------------------------+
|  Public API:  App, Window, Widgets, Layouts, Events, Theme|
+-----------------------------------------------------------+
|  Core:  widget tree, layout engine, event dispatch, focus |
+-----------------------------------------------------------+
|  Backend abstraction:  Canvas, Input, Clock, FontFace     |  <- interfaces
+-----------------------------------------------------------+
|  EBiten backend (internal)                                |  <- only place that imports ebiten
+-----------------------------------------------------------+
```

The **backend abstraction** layer is the seam that keeps EBiten out of the
public API. The core and public API talk only to interfaces (`Canvas`, `Input`,
etc.); a single internal package implements those interfaces against EBiten.

### Suggested package layout

```
guie/
  ui/            # public API: App, Window, widgets, layouts, options
  core/          # widget tree, layout engine, dispatch, focus mgmt (internal)
  render/        # Canvas + drawing primitive interfaces (backend-agnostic)
  internal/ebiten/ # the only package that imports ebiten/v2 (unimportable by users)
  geom/          # Point, Size, Rect, Insets, alignment enums
  theme/         # Palette, Theme, default theme, fonts
  assets/        # bundled default font (go:embed)
```

### Main loop (owned by the framework)

The framework runs the loop internally. Conceptually each frame:

1. **Input poll** — backend collects mouse/keyboard/touch into a backend-neutral
   `InputState`.
2. **Event dispatch** — translate input into UI events (click, hover, key,
   scroll) and route them through the widget tree (hit-testing, focus).
3. **Update** — widgets update animation/state; app callbacks fire.
4. **Layout** — if the tree is dirty, re-run layout from the root.
5. **Draw** — paint the tree onto the `Canvas` back-to-front.

Public entry point is roughly:

```go
app := ui.NewApp(ui.WithTitle("My App"), ui.WithSize(800, 600))
win := app.MainWindow()
win.SetContent(/* root widget */)
app.Run()   // blocks; starts the loop. No ebiten visible anywhere.
```

---

## BACKEND ABSTRACTION

The render seam. Public widgets receive a `Canvas` in their draw method, never
an `*ebiten.Image`.

```go
// render package (no ebiten import)
type Canvas interface {
    Size() geom.Size
    Clip(r geom.Rect)            // push a clip rect (for scroll views, windows)
    ClearClip()
    FillRect(r geom.Rect, c color.Color)
    StrokeRect(r geom.Rect, c color.Color, width float64)
    DrawLine(a, b geom.Point, c color.Color, width float64)
    DrawText(s string, pos geom.Point, face FontFace, c color.Color)
    MeasureText(s string, face FontFace) geom.Size
    DrawImage(img Image, dst geom.Rect)
    SubCanvas(r geom.Rect) Canvas // for compositing windows/widgets
}

type FontFace interface {
    Metrics() (ascent, descent, lineHeight float64)
}

type Image interface { Size() geom.Size }
```

Input is likewise neutral:

```go
type InputState struct {
    MousePos      geom.Point
    MouseButtons  ButtonSet      // pressed this frame
    MousePressed  ButtonSet      // just went down
    MouseReleased ButtonSet      // just went up
    WheelDelta    geom.Point
    KeysDown      []Key
    KeysPressed   []Key
    Runes         []rune         // text input
    Modifiers     ModifierSet    // shift/ctrl/alt/meta
}
```

The EBiten backend translates `ebiten` input/keys into these types and
implements `Canvas` over `*ebiten.Image`. Nothing else imports `ebiten`.

---

## WIDGET MODEL

### The Widget interface

```go
type Widget interface {
    // identity / tree
    Parent() Widget
    Children() []Widget

    // layout
    MinSize() geom.Size            // intrinsic minimum
    Bounds() geom.Rect             // assigned by parent during layout
    SetBounds(geom.Rect)
    Layout()                       // position children within Bounds

    // paint
    Draw(c render.Canvas)

    // input / events
    HandleEvent(ev Event) bool     // returns true if consumed

    // state
    Visible() bool
    Enabled() bool
}
```

Most widgets embed a `BaseWidget` that provides bounds storage, visibility,
enabled state, parent/child bookkeeping, and default event handling — so a
concrete widget overrides only what it needs.

### Construction (struct + functional options)

```go
btn := ui.NewButton(
    ui.WithText("Save"),
    ui.OnClick(func() { save() }),
    ui.Disabled(false),
)
btn.SetText("Saved")   // methods mutate the retained widget at runtime
```

`NewButton` returns a concrete `*Button` (so methods are discoverable). Options
are `func(*Button)` values; shared options (size, padding, visibility, colours)
are generic helpers that apply to `BaseWidget`.

### Retained tree

Widgets are persistent objects. The framework holds the tree between frames,
re-laying-out only when something is marked dirty (`Invalidate()`), and
redrawing each frame (a later optimization can add dirty-region redraw).

---

## LAYOUT SYSTEM

Two-phase, top-down layout over the retained tree:

1. **Measure** — each widget reports `MinSize()` (bottom-up).
2. **Arrange** — each container assigns `Bounds` to children via its layout
   manager (top-down), then calls each child's `Layout()`.

Containers and layout managers:

- `Box` with direction `Horizontal`/`Vertical` — sequential, with spacing,
  per-child stretch weight, and alignment (start/center/end/stretch).
- `Grid` — rows × columns, per-cell span, column/row weights.
- `Flex` — weighted distribution of free space along one axis.
- `Stack` / `Overlay` — children share the same area (z-ordered) for things
  like dialogs and dropdown popups.
- `Padding` / `Insets` wrapper for margins.

Sizing model per child: a minimum size plus optional stretch weight; containers
distribute leftover space by weight. Layout re-runs on window resize or when a
widget calls `Invalidate()`.

---

## EVENT SYSTEM

Two complementary mechanisms (both requested):

### 1. Callbacks (primary)

Per-widget handlers registered at construction or via setters:

```go
ui.OnClick(func())
ui.OnChange(func(newValue))
ui.OnHover(func(entered bool))
ui.OnFocus(func(focused bool))
ui.OnKey(func(ev KeyEvent) bool)
```

### 2. Event bus (optional, global)

For cross-cutting / global listening without wiring every widget:

```go
app.Events().Subscribe(EventClick, func(ev Event) { ... })
```

Backed by a channel/subscriber model internally; callbacks remain the default
path so simple apps never touch it.

### Dispatch rules

- **Hit-testing**: pointer events go to the top-most widget under the cursor,
  bubbling up to ancestors until consumed (`HandleEvent` returns true).
- **Focus**: one focused widget receives keyboard events; Tab/Shift-Tab move
  focus; clicking a focusable widget focuses it.
- **Capture**: a widget (e.g. a scrollbar thumb or a button being pressed) can
  capture the pointer so it keeps receiving move/release events until release.

Event types (initial): `Click`, `DoubleClick`, `PointerDown/Up/Move`,
`PointerEnter/Leave`, `Wheel`, `KeyDown/KeyUp`, `TextInput`, `FocusGained/Lost`,
`Resize`.

---

## THEMING & COLOUR CONTROL

Start simple: a named colour palette plus a default font, with per-widget
overrides so any window or widget colour can be controlled individually.

```go
type Palette struct {
    Background   color.Color
    Surface      color.Color
    Primary      color.Color
    OnPrimary    color.Color
    Text         color.Color
    TextMuted    color.Color
    Border       color.Color
    Accent       color.Color
    Disabled     color.Color
}

type Theme struct {
    Palette      Palette
    Font         render.FontFace
    FontSize     float64
    Spacing      float64   // default gap between widgets
    Padding      geom.Insets
    CornerRadius float64
}
```

- A `DefaultTheme()` ships with the library (light palette + bundled font).
- Theme lives on the `App`; widgets read it during `Draw`.
- **Window and widget colour control**: every widget accepts colour overrides
  via options/setters (`ui.WithBackground(c)`, `ui.WithTextColour(c)`,
  `ui.WithBorderColour(c)`); a window has its own background colour. Unset
  colours fall back to the theme palette.
- Room to grow into multiple themes / dark mode later without API churn.

---

## FONTS & FONT CONTROL

- Bundle a default TTF (the Go font, `golang.org/x/image/font/gofont`, or an
  embedded TTF via `go:embed`) so apps render text with zero setup.
- Expose font loading: `ui.LoadFont(path)` / `ui.LoadFontBytes([]byte)` returning
  a `render.FontFace`.
- **Font control**: font family and size are settable globally via the theme and
  per-widget via options (`ui.WithFont(face)`, `ui.WithFontSize(n)`).
- Text measurement goes through `Canvas.MeasureText` so layout can size labels,
  buttons and list rows correctly.

---

## WIDGET CATALOG

Grouped by build priority.

### Core (built first — the vertical slice)

- **Window** — top-level surface; for now a single OS window with content,
  optional title bar, settable background colour. (Multiple/child windows later.)
- **Container / Box / Grid / Flex** — layout primitives (see Layout System).
- **Label** — static text.
- **Button** — text (and later icon); `OnClick`, pressed/hover/disabled states.
- **Events + focus + theme** wired through these.

### Second wave

- **Checkbox** — boolean toggle, `OnChange(bool)`.
- **RadioButton** / **RadioGroup** — mutually exclusive selection.
- **TextField** — single-line text entry (uses `TextInput`/focus/caret). Optionally
  **masked** for credential entry — see *MASKED TEXT INPUT*. *(Masking done —
  `INTERNALS.md` §14.4.)*
- **ScrollBar** + **ScrollView** - clip + offset content. *(Done: a shared `ScrollBar` drives every scrolling widget and is also a standalone widget, INTERNALS 13g. Horizontal scrolling: opt-in (`SetHorizontalScroll`) on `ScrollView`, `Table`, `List` and `Tree`; automatic on `TextArea` without wrap.)*

### Third wave

- **List** — virtualized vertical list of selectable rows.
- **Tree** — hierarchical list of expand/collapse nodes (`TreeNode`), keyboard
  navigable. *(Done — see `internals.md` §13a and `examples/tree`.)*
- **DropdownCombo** — collapsed selector that opens a popup list (uses Overlay).
- **Menu / MenuBar** — popup menus, items, separators, submenus.
- **Slider**, **ProgressBar** — convenience widgets.
- **Stepper** — numeric input with up/down buttons, arrow keys and wheel
  (min/max/step/decimals). *(Done — `internals.md` §13c, `examples/widgets2`.)*
- **Spinner** — indeterminate busy/activity indicator. *(Done — `internals.md`
  §13c, `examples/widgets2`.)*
- **DatePicker** — inline month calendar over `time.Time`, with a **DateField**
  popup variant (collapsed control + dropdown calendar). *(Done — `internals.md`
  §13d, `examples/datepicker`.)*
- **ColourPicker** — HSV+alpha picker (swatch + hue/saturation/value/alpha
  sliders, transparency shown over a checkerboard). *(Done — `internals.md`
  §13e, `examples/colourpicker`.)*
- **Toast** — transient, auto-dismissing notifications raised via
  `App.ShowToast` (info/success/warning/error). *(Done — see `internals.md`
  §13b and `examples/tree`.)*

### Later / nice-to-have

- Tabs, tooltips, modal dialogs, splitters, tables, drag-and-drop, multi-window,
  dirty-region redraw optimization, accessibility hooks.

---

## OPEN QUESTIONS / TBD

- High-DPI / scaling strategy (logical vs physical pixels). *(Done — device
  scale factor wired through the loop + canvas; see MACOS/CROSS-PLATFORM POLISH.)*
- Clipboard and IME support for text fields. *(Done for clipboard — the OS
  clipboard is the **default**, resolved lazily with an in-process fallback;
  `ui.WithClipboard` overrides it. See `examples/clipboard`. IME: the framework-side seam,
  events, preedit rendering and tests are built — see IME below; the EBiten last
  mile is blocked upstream.)*
- Animation/transition primitives (timeline vs per-frame). *(Done — per-frame
  hook `App.OnFrame` plus `App.Animate`/`App.Tween` with easings; see
  `internals.md` §11a and `examples/animation`.)*
- Persistence of window size/position. *(Done - `ui.WithWindowStateFile`, `WithWindowGeometry`, `App.WindowGeometry`; see INTERNALS 8.7 and `examples/windowstate`. Which-monitor restore is still open: EBiten reports position relative to the current monitor only.)*
- Testing approach (headless backend implementing `Canvas`/`Input` for unit
  tests of layout and event dispatch without a real window). *(Done — the
  `guitest` package provides a GPU-free `render.Driver`/`Canvas`/`FontFace`
  plus a `Harness` that steps an `App` frame by frame, synthesizes input and
  records drawing ops for assertions. Injected via `ui.WithDriver`; see
  `internals.md` §17.1.)*

---

## MACOS / CROSS-PLATFORM POLISH (TBD)

The framework runs on macOS as-is (EBiten is cross-platform; guie has no
OS-specific code beyond one `runtime.GOOS` check in the backend). Two
foundational pieces are already done: the **primary shortcut modifier**
(`ModPrimary` — ⌘ on macOS, Ctrl elsewhere; used by `TextField`/`TextArea`) and
**HiDPI/Retina scaling** (device scale factor wired through the loop + canvas, so
rendering is crisp while widgets stay logical). Remaining niceties to make it
feel native, for a later date:

- **OS clipboard integration.** *(Done, and now the default.)* `ui.NewApp`
  installs an OS-backed clipboard, resolved lazily on the first copy or paste and
  falling back to the in-process `memClipboard` if the platform clipboard is
  unavailable. `ui.WithClipboard` still overrides it. This **reverses the earlier
  opt-in decision**: with the in-process clipboard as the default, Ctrl+V from a
  browser silently did nothing unless an app knew to wire the `clipboard` package
  up, which reads as a broken widget rather than a design choice. The cost, taken
  deliberately, is that `ui` now depends on the `clipboard` package. It's backed by
  `golang.design/x/clipboard` (CGo-free on Windows; uses the same CGo/X11
  toolchain EBiten already needs on macOS/Linux — no external `xclip` binary).
  See `examples/clipboard`.
- **Native top menu bar.** macOS apps put menus in the global menu bar with the
  app name, not in an in-surface `MenuBar` widget. A native menu bridge would be
  backend-specific and sizeable; the in-surface menu works everywhere in the
  meantime.
- **Standard macOS shortcuts beyond clipboard.** Word-jump is ⌥←/⌥→ (Option),
  line start/end is ⌘←/⌘→, and ⌘↑/⌘↓ go to document start/end. Today navigation
  assumes the Windows/Linux convention. Generalise text navigation to honour the
  platform's modifier map (extend the `ModPrimary` idea to a small per-platform
  shortcut table).
- **HiDPI mid-run scale changes.** Dragging a window between monitors of
  different DPI is picked up on the next frame, but cached scaled glyph state
  isn't pre-warmed (minor; see `internals.md` §16).
- **Window controls / chrome conventions.** Traffic-light buttons, title bar
  behaviour, full-screen handling — currently whatever EBiten provides; revisit
  if a more native window experience is wanted.
- **Retina visual verification.** The HiDPI path is the standard EBiten recipe
  and passes build/test, but crispness should be eyeballed on real Retina
  hardware (`go run ./examples/showcase`) — can't be checked headlessly.
- **Trackpad / gesture input.** Smooth/inertial two-finger scroll and pinch are
  not surfaced beyond `WheelDelta`; native-feeling scrolling could be added.

---

## ARM / RASPBERRY PI (TBD)

**guie itself needs zero code changes for ARM** — everything platform-specific
lives behind the EBiten seam, and the one `runtime.GOOS` check correctly yields
`ModPrimary = Ctrl` on Linux. Getting it onto a Pi is entirely an "EBiten builds
and runs here" exercise, not a framework one. Findings (verified by cross-compile
attempts from a Windows dev box):

- **CGo is not required on Linux (EBiten 2.10+).** guie is on EBiten v2.10.5,
  whose macOS and Linux/BSD backends no longer use CGo. No C compiler or dev
  headers are needed to build; the X11/GL/ALSA runtime libraries are still
  needed on the target.
- **Cross-compiling is straightforward.** `GOOS=linux GOARCH=arm64
  CGO_ENABLED=0 go build ./...` from a Windows dev box builds cleanly (checked
  for linux/amd64, linux/arm64, darwin and windows). Building natively on the Pi
  also works, with just Go for arm64 installed. Running on the Pi is untested.
- **Architecture.** Use 64-bit Raspberry Pi OS → `GOARCH=arm64`. 32-bit is
  `GOARCH=arm GOARM=7`; Pi Zero/1 are ARMv6 (`GOARM=6`) and realistically too
  slow to bother with.
- **Runtime needs a GPU + display.** EBiten requires OpenGL (~2.1 / GLES). A
  **Pi 4/5** on Raspberry Pi OS Desktop with the modern Mesa V3D driver and X11
  should run it; older Pis with the legacy Broadcom driver are dicey. Headless
  won't work without `Xvfb` + software GL (llvmpipe), which is slow.
- **Performance is the real risk, and it links to task 15.** guie does a full
  redraw every frame at 60 TPS (dirty-region redraw, plan item 15, is not done).
  Fine for simple UIs on a Pi 4/5; sluggish on weaker hardware or busy screens.
  The Pi is the use case that would actually justify implementing task 15; a
  cheap interim lever is lowering the tick rate.
- **Touch input.** The official Pi touchscreen usually presents as mouse events
  through the OS, so basic tap/drag works as-is. True multitouch/gestures aren't
  wired — guie's input layer only polls mouse, not EBiten touches. A later
  nicety, not a blocker.
- **Visual verification.** Can't be checked headlessly — rendering and
  performance should be eyeballed on real Pi hardware.

---

## DRAG-AND-DROP

In-process drag-and-drop: pick up data/a widget from a **drag source**, drag it
over the tree, and release it on a **drop target** that accepts it. Covers
list/table row reordering, dragging items between panels, dragging tabs and
palette→canvas drops.

Cross-application **OS file drops** are a separate, much smaller mechanism —
EBiten surfaces the drop but nothing else about the drag — and are specified in
*OS file drops* below. The two share the vocabulary and nothing else; neither is
built on the other.

### Mechanism — layered on pointer capture

The dispatcher already captures `pressTarget` on press and routes every later
`PointerMove`/`PointerUp` to it until release. DnD is a **state machine on top of
that capture**: a captured press that moves past a small threshold *becomes* a
drag. No new capture path, no change to hit-testing.

```
press on source ──move>threshold──▶ DRAGGING ──release over accepting target──▶ DROP
     │                                  │                                          │
   (also fires PointerDown,          (App intercepts moves: enter/over/leave   source.OnDragEnd(accepted)
    widget can stay clickable)        to targets, draws ghost; suppresses      target.OnDrop(data,pos)
                                      normal move to source)                   click on source suppressed
                                          └────release off target / Esc────▶ CANCEL → source.OnDragEnd(false)
```

### API — mirrors SetContextMenu / the OnX convention

Configuration is stored on `BaseWidget` and read by the dispatcher via
accessors, exactly like `SetContextMenu`/`ContextMenu()`. Behaviour uses `OnX`
callbacks.

```go
// DragData is the in-process payload. Type is a free-form tag drop targets
// match on ("row", "file", "tab"); Value is the payload.
type DragData struct { Type string; Value any }

// Source — any widget becomes draggable:
func (b *BaseWidget) SetDragSource(provide func() *DragData) // nil return → no drag
func (b *BaseWidget) SetDragGhost(g DragGhost)               // optional custom ghost
func (b *BaseWidget) OnDragEnd(func(accepted bool))

// Target — any widget can accept drops:
func (b *BaseWidget) SetDropTarget(accept func(DragData) bool)
func (b *BaseWidget) OnDragEnter(func(d DragData))
func (b *BaseWidget) OnDragLeave(func())
func (b *BaseWidget) OnDragOver(func(d DragData, pos geom.Point))
func (b *BaseWidget) OnDrop(func(d DragData, pos geom.Point) bool) // true = accepted
```

`provide` runs once when the threshold is crossed (so the source snapshots its
current state); returning `nil` vetoes the drag. `OnDrop`/`OnDragOver` carry
`pos` so self-drawing multi-row widgets (`List`, `Table`) can map the cursor to a
row, just as they already do for clicks. Drop-target resolution walks the hit
chain for the nearest widget whose `accept(data)` is true — the same shape as
`contextTarget`.

### Ghost rendering

The dragged "ghost" is painted above the root and overlays, below the tooltip
(one line in `App.draw`). Default ghost is a translucent snapshot of the source
drawn at the cursor via the existing `render.RenderTarget` (`ui.NewRenderTarget`,
§3.3) — no new rendering capability needed. Overridable with a custom ghost
widget, or disabled so targets show their own insert indicator via `OnDragOver`.

### Design decisions (locked)

| Area | Decision |
|---|---|
| Transport | Reuse `pressTarget` pointer capture; drag is a state machine on top. |
| Config API | `SetDragSource`/`SetDropTarget` stored on `BaseWidget`, read by dispatcher (mirrors `SetContextMenu`). |
| Behaviour API | `OnDrop`/`OnDragEnter/Leave/Over`/`OnDragEnd` (the `OnX` convention). |
| Payload | `DragData{Type string, Value any}`, in-process; targets match on `Type`. |
| Threshold | 4 logical px before a press becomes a drag, so clicks still work. |
| Click suppression | A release that ends a drag does not derive a `Click`. |
| Buttons | Left button only initiates a drag. |
| Ghost | Default = translucent `RenderTarget` snapshot of source; overridable/none. |
| Cancel | Escape, or release off any accepting target. |
| Bus | Drag lifecycle is observable; per-widget callbacks remain primary. |
| Scope | In-process payloads only. OS file drops are a separate mechanism (below), not a `DragData` source. |

Deferred for v1: edge auto-scroll over a `ScrollView`, copy-vs-move drop effects,
multi-button drags.

---

### OS file drops

Dropping files from the desktop (Explorer / Finder / a browser's file list) onto
the window. Requested by a real app — the NWF viewer, whose stated design is
"select an NWF file **(or drag and drop)**" — which is the trigger this was
waiting on.

*(Implemented — see `INTERNALS.md` §11c. The spec below is the design as locked.)*

**This is not the in-process engine with a different payload.** The OS hands over
*one thing*: the fact that a drop happened, and the files. There is no source
widget, no ghost, no threshold, no Escape-cancel, and — critically — **no
enter/over/leave stream**, because GLFW reports only the completed drop, not the
drag that preceded it. Modelling it as a `DragData` source would advertise
callbacks (`OnDragEnter`, `OnDragOver`) that could never fire.

#### Mechanism

```
Explorer drag ─release─▶ GLFW drop callback (paths)
        └─▶ EBiten: inputstate.DroppedFiles = VirtualFS(paths)   [cleared after one tick]
              └─▶ Driver.pollInput → render.InputState.DroppedFiles (fs.FS, nil = none)
                    └─▶ App.update: hit-test at the cursor → EventFileDrop, bubbling
```

Like `WindowBeingClosed` (see the window-close hook), `DroppedFiles` is an
**edge**: EBiten clears it after the tick that reads it, so the framework must
consume it the frame it arrives and must not treat it as a level.

#### Payload: `fs.FS`, not paths

The payload is an `io/fs.FS` plus the entry names, **not** a `[]string` of file
paths. Three reasons, in increasing order of importance:

1. **EBiten does not expose paths.** `ebiten.DroppedFiles()` returns a virtual FS
   whose root entries are `filepath.Base(path)`; the absolute paths GLFW handed
   over are kept private inside it. Paths are simply not available to ask for.
2. **`render` may not leak backend types** (core principle). `io/fs` is stdlib, so
   the seam stays backend-neutral for free.
3. **WASM.** In a browser there is no filesystem path at all, only file contents.
   An `fs.FS` payload is what lets browser drops work *at all*, and the roadmap
   advertises WASM as supported. A path-shaped API would be desktop-only and would
   have to be broken later.

The cost is real and must be stated: **a dropped file has no reloadable path.** An
app can read its bytes and its base name, and nothing more — no reload-from-disk,
no save-in-place, no "recent files" entry that survives a restart. Apps that need
those must keep the native file dialog for that job.

#### Position and routing

The drop is routed by **hit-testing the cursor position** at the tick the drop
arrives, then dispatching `EventFileDrop` to the deepest widget under it and
**bubbling** through the ancestors like every other event — so a leaf can claim a
drop and the window as a whole can catch everything else, with no extra
mechanism.

This depends on the cursor position being current at the drop tick, which is not
obvious: the OS owns the mouse during a drag, and GLFW's drop callback carries no
position of its own. It holds because EBiten **polls** `window.GetCursorPos()`
from the OS every tick (`internal/ui/input_glfw.go`) rather than relying on the
cursor-position callback, and the drop callback fires on *release* — by which
point the drag is over and the polled position is the release point.

**Confirmed on Windows** by dropping files onto individual tiles in the NWF
viewer: they land on the tile under the cursor. macOS and X11 are unverified — if
the position turns out stale there, the fallback is window-level drops on that
platform, dispatching to the root, and `OnFileDrop`'s position argument would have
to go rather than be left lying.

#### API — mirrors the `OnX` convention

```go
// render (seam)
type InputState struct {
    // ...
    // DroppedFiles holds files the user dropped on the window this frame, or nil
    // if none were. It is an edge: it is set for exactly the frame the drop
    // arrives. Entry names are base names; the FS is the only way to the content.
    DroppedFiles fs.FS
}

// ui
type FileDrop struct {
    FS    fs.FS    // the dropped files
    Names []string // root entry names, read once by the App
}

// BaseWidget, stored in the existing lazy dragConfig
func (b *BaseWidget) OnFileDrop(fn func(d FileDrop, pos geom.Point) bool)
```

`Event` gains a `Files FileDrop` field (as `EventWheel` has `Wheel` and
`EventComposition` has `Comp`), and `EventFileDrop` joins the `EventType` set.
The handler returns true to consume the drop and stop it bubbling. The App reads
the root entries once so that N widgets in the bubble chain do not each walk the
FS.

Names are passed through **unfiltered**: GLFW can hand over a directory, and the
framework does not know whether an app wants it. Apps that only accept files call
`fs.Stat`. Filtering by extension is likewise the app's business.

#### Testing

`guitest` grows `Harness.DropFiles(files map[string][]byte)`, which builds an
`fstest.MapFS` and delivers it on the next `Step` at the harness's current mouse
position — so routing, bubbling and consumption are all assertable headlessly.
The EBiten half (the GLFW callback) stays a manual check, exactly as the
window-close hook does.

#### Design decisions (locked)

| Area | Decision |
|---|---|
| Relationship to in-process DnD | Separate mechanism. Shares no state, no session, no `DragData`. |
| Lifecycle | **Drop-only.** No enter/over/leave/cancel — GLFW reports nothing before the drop. |
| Seam | `InputState.DroppedFiles fs.FS`, an edge (one frame), nil when no drop. |
| Payload | `FileDrop{FS fs.FS, Names []string}` — content, not paths (WASM-safe; EBiten exposes no paths). |
| Consequence | A dropped file has **no reloadable path**; reload/save-in-place need the native dialog. |
| Routing | Hit-test at the cursor, dispatch to the deepest widget, bubble to the root. |
| Position | From the tick's polled cursor position. Verified on Windows with real drags; macOS/X11 unverified (fallback there: window-level only). |
| Config API | `OnFileDrop(func(FileDrop, geom.Point) bool)` on `BaseWidget`, in the existing lazy `dragConfig`. |
| Directories & filtering | Names passed through unfiltered; the app decides (`fs.Stat`, extension checks). |
| Multi-file | All names delivered in one event; the app chooses what to do with more than one. |
| Bus | Published like every other event. |
| Testing | `guitest.Harness.DropFiles` headlessly; the GLFW callback manually. |

---

## IME (INPUT METHOD EDITORS)

IME has three separable concerns; the framework-side machinery for all three is
built, but only the first is deliverable on the current backend:

1. **Committed text** — the IME's final output (e.g. 日本語). *Already works:* the
   OS commits runes, EBiten surfaces them via `AppendInputChars`, the backend
   puts them in `InputState.Runes`, and the text widgets insert them on
   `EventText`.
2. **Preedit / composition** — the uncommitted string shown inline (underlined)
   while composing. The seam, event and widget rendering exist; **EBiten v2.9.9
   exposes no preedit API**, so no composition is fed on that backend today.
3. **Candidate-window placement** — telling the OS where the caret is. The seam
   exists (`SetIMERect`); **EBiten exposes no positioning API**, so it is a no-op
   there today.

### Seam (`render`)

```go
// Per-frame preedit; zero value (empty Text) = not composing. Backends that
// support IME fill it; others leave it zero and only committed Runes flow.
type Composition struct { Text string; Caret int; SelLo, SelHi int }

type InputState struct { /* …; */ Composition Composition }

// Optional capability a Driver may implement; the framework type-asserts for it.
type IMEController interface {
    SetIMEEnabled(on bool)   // toggled on focus/blur of an editable widget
    SetIMERect(r geom.Rect)  // caret rect (abs logical px) for the candidate window
}
```

### Events & widgets

- New `EventComposition` (carries `Event.Comp`) is delivered to the focused
  widget like `EventText`; committed text stays on `EventText` (independent
  channel). An empty `Comp.Text` ends/clears the preedit.
- `TextField`/`TextArea` hold `preedit`/`preeditCaret`, drawn **inline at the
  caret, underlined**, never mixed into the committed runes; starting a
  composition replaces any selection. They expose `imeCaretRect()`; the App
  reports it via `SetIMERect` each frame while focused and toggles
  `SetIMEEnabled` on focus change.
- **Key gating:** while a composition is active the App does not forward
  KeyDown/KeyUp to the focused widget (the IME owns those keys). Committed text
  still arrives as `EventText`.

### Degradation & testing

- No `IMEController`/`Composition` from the backend → committed-text-only
  (today's behavior), no crashes. When EBiten gains preedit (or another backend
  is used) it simply starts populating `Composition`/honoring `SetIMERect`; the
  already-wired widgets light up with **no API churn above the backend**.
- The `guitest` headless backend can simulate composition fully (`Compose`,
  `CommitText`, `CancelComposition`), so the entire preedit→commit lifecycle is
  deterministically tested even though EBiten can't drive it through a window.

See `internals.md` §14.3.

---

## MASKED TEXT INPUT (PASSWORD FIELDS)

*(Implemented — see `INTERNALS.md` §14.4. The spec below is the design as
locked.)*

`TextField` can hide what it holds: an API key, a password, a token. The field
still edits, selects, scrolls and pastes exactly as it does today - only what
reaches the screen changes.

Driven by a real app: vectorworks-watcher asks the user to paste a
Lighting-Paperwork API key into a window that may be on a projector, in a screen
share or in a support screenshot. This is a widget-library feature rather than an
application one, because the alternative - the app drawing its own field - means
reimplementing selection, scrolling, IME and clipboard.

### API - deliberately two symbols

```go
func Masked() TextFieldOption           // construct masked
func (t *TextField) SetMasked(v bool)   // toggle at runtime (a "Show" button)
```

No getter. `Placeholder` has no accessor either, and `SetFont` has no `Font()` -
a caller with a reveal toggle already knows which way it set the field.
Pre-freeze, the fewer exported symbols the better; a getter is additive if
something ever needs one.

Mask glyph: the constant `maskRune = '\u2022'` (BULLET). Not an option, for the
same reason - `MaskRune(r rune)` can be added later without breaking anything,
whereas an option shipped now is frozen.

The default face carries it: measured through `text.GoTextFace` over
`goregular.TTF` (the face `internal/ebiten/font.go` loads), U+2022 advances
5.609px at size 16, against 12.0px for an unmapped rune falling back to the
replacement glyph. A caller supplying a face without it gets that replacement
glyph across the whole field, and **no test will catch it** - the headless font
measures every rune identically, so the substitution is invisible to `guitest`.
That is the one failure mode this decision accepts, and it is why `MaskRune`
exists as a named additive escape hatch rather than being ruled out.

### Mechanism - one accessor, and one contract

The naive implementation substitutes the text in `Draw`. That is wrong, and
visibly so: the same rune slice is measured in four other places, so a
display-only substitution leaves the caret, the click-to-position mapping, the
selection highlight and the horizontal scroll all computing against the real
text while the user looks at the mask. Every one of them lands in the wrong
place the moment the mask glyph's advance differs from the real character's -
which, in a proportional font, is almost always.

So masking is one unexported accessor that every text-measuring and
text-drawing path goes through:

```go
// display returns what the field shows: the real runes, or one mask rune per
// real rune when masked.
func (t *TextField) display() []rune
```

**The contract, and the whole reason this is a display-only change: `display()`
returns exactly `len(t.runes)` runes.** One mask glyph per real rune, never a
fixed-width placeholder and never a single collapsed glyph. Because the count is
preserved, every existing caret index, selection bound and scroll offset stays
valid with no index translation anywhere. A future change that breaks the 1:1
rule breaks the caret, silently.

Call sites to route through it:

| Path | What it does today | Why it must change |
|---|---|---|
| `Draw` | `DrawText(string(t.runes), ...)` | draws the secret |
| `Draw`, selection highlight | `f.Measure(string(t.runes[:lo]))`, `[:hi]` | highlight lands on the wrong span |
| `caretVisualWidth` | `f.Measure(string(t.runes[:t.caret]))` | caret drawn at the wrong x |
| `caretIndexAt` | `caretIndexForX(t.runes, ...)` | a click puts the caret in the wrong place |
| `updateScroll` | via `caretVisualWidth` | covered by the above, but named so it is not missed |
| `imeCaretRect` | via `caretVisualWidth` | candidate window points at the wrong spot |

### The preedit leaks, and is the easiest path to miss

`Draw`'s composing branch draws `pre + t.preedit + post` in one call and
measures the underline from the real text. A masked field driven through an IME
therefore renders the in-progress composition in the clear - the one part of the
content that was never in `t.runes` and so is not covered by `display()`.

**Decision: mask the preedit too**, one mask rune per preedit rune, with the
underline still drawn beneath it. The alternative - refusing composition while
masked - is a worse surprise: a field that silently will not accept typed input
looks broken, whereas dots appearing as you compose looks like every other
password field. `caretVisualWidth`'s preedit measurement follows the same rule.

Pasting an API key never triggers this. Someone typing a password on a machine
with an IME active does, which is exactly the case a widget library has to get
right on the user's behalf.

### Clipboard: copy and cut are blocked while masked

`copySelection` and `cutSelection` do nothing while `masked` is true. Paste is
untouched - putting a secret in is the point.

There is deliberately no separate "revealed" state inside the widget. Revealing
*is* `SetMasked(false)`, so an app with a Show toggle gets readback for free
through the same rule, and the widget keeps one condition rather than two
behaviours. That matters because blocking copy outright would make a stored key
unrecoverable from the UI, which is at odds with the reason the toggle exists.

`Text()` and `OnChange` always carry the real text: masking is presentation, not
storage, and an app that could not read back what the user typed would be unable
to save it.

The placeholder is drawn unmasked. It is not the secret, and a row of dots where
a prompt should be tells the user nothing.

### Interactions with roadmap items - name them now

- **Accessibility (seam)**, now deferred (see ROADMAP -> Deferred): when it lands it lists `TextField` among the
  widgets to annotate immediately. A masked field must not expose its content
  through `AccessibleName`/`AccessibleState` - the semantic tree is a second
  rendering path, and masking it in `Draw` alone would leak it there.
- **Text undo**, Pre-freeze: undo history holds the real text. That is correct
  and not a leak (undo never renders anything the field would not), and is
  written down so nobody later "fixes" it into storing masked text and corrupts
  the buffer.
- **TextArea** stays unmasked, and this does not generalise to it. A multi-line
  secret is not a real thing, and the wrapping/line-measuring paths would double
  the surface for no case anyone has.

### Whitespace, for callers

A masked field makes leading and trailing whitespace invisible, and `paste()`
already flattens newlines to spaces. A caller taking a credential should trim
what it reads before using or storing it - otherwise a key copied out of a web
page or an email fails authentication later with nothing on screen to explain
it.

### Tests (`guitest`)

The headless font measures a fixed advance per rune (`guitest/backend.go`:
`Measure` counts runes), which sets what these tests can and cannot prove. They
verify **index bookkeeping** - that every path uses `display()` and that counts
line up. They cannot prove **proportional geometry**, because headlessly the
mask glyph and the real character are the same width by construction. The 1:1
rune-count contract is what carries that half, which is why it is stated as a
contract rather than left as an implementation detail.

- A masked field draws neither the text nor any substring of it
  (`Recording.HasText`, `TextContaining`), and draws `len(text)` mask runes.
- `Text()` returns the real text while masked; `OnChange` reports the real text.
- `SetMasked(false)` reveals, `SetMasked(true)` re-masks, and a redraw follows.
- Clicking at a given x lands on the same caret index masked and unmasked.
- Selection highlight spans the same rect masked and unmasked.
- Copy and cut are no-ops while masked and work once revealed; paste works in
  both states.
- A composition (`Harness.Compose`) draws no preedit text while masked, and the
  committed text is still the real text.
- The placeholder is drawn unmasked on an empty, unfocused, masked field.

---

## TEXT UNDO / REDO (TextField, TextArea)

*(Implemented - `ui/undo.go`, `ui/undo_test.go`; INTERNALS 14.5. Roadmap row: "Text undo", pre-freeze.)*

Built-in undo and redo inside `TextField` and `TextArea`: Ctrl+Z / Ctrl+Shift+Z
(Cmd on macOS) and a small method set so an app's Edit menu can drive it. This
is a correctness expectation of the widgets, not a feature: a text box that
loses ten minutes of typing to a stray Ctrl+X, with no way back, reads as
broken. App-level command history (undo of non-text actions) stays userland.

Today neither widget has any history. Every mutation edits the rune storage in
place (`insert`, `deleteBack`, `deleteForward`, `insertString`, `cutSelection`,
`deleteSelection`, `TextArea.insertNewline`) and `SetText` replaces it
wholesale, so there is nothing to rewind.

### API - deliberately small

```go
// Both TextField and TextArea:
func (t *TextField) Undo() bool      // false if nothing to undo
func (t *TextField) Redo() bool
func (t *TextField) CanUndo() bool   // for enabling Edit-menu items
func (t *TextField) CanRedo() bool
func (t *TextField) ClearHistory()
```

- **Keys:** `ModPrimary+Z` undo; `ModPrimary+Shift+Z` redo; `ModPrimary+Y` redo
  (Windows/Linux habit; harmless elsewhere). Handled in `handleKey` beside
  copy/cut/paste, so the platform modifier map is the existing one.
- **No option for depth or disabling**, pre-freeze: a package constant
  (`undoDepth = 500` steps). `UndoLimit(n)` is additive if someone needs it;
  a frozen option is not removable. Same reasoning as `MaskRune` (masked input).
- Undo/redo **fire `OnChange`** with the new text, like any other change, and
  keep the caret in view.
- A disabled widget, or one mid-IME-composition, ignores the keys (the App
  already withholds key events from the focused widget while composing).

### What an edit is

History records **edits, not snapshots.** A snapshot per step is O(text) each, so
a TextArea holding a file costs the whole file per keystroke; an edit costs only
what changed.

```go
// TextField: positions are rune indexes into runes.
type fieldEdit struct {
    at                     int
    removed, inserted      []rune
    before, after          fieldSel   // {caret, anchor}
    kind                   editKind
}

// TextArea: positions are (row, col); text may contain '\n'.
type areaEdit struct {
    at                     pos        // {row, col} where removed/inserted start
    removed, inserted      []rune
    before, after          areaSel    // {caretRow, caretCol, anchorRow, anchorCol}
    kind                   editKind
}
```

One shared generic container holds the stacks and the coalescing policy:

```go
type undoStack[E any] struct { undo, redo []E; ... }
```

Undo of an edit removes `inserted` at `at`, re-inserts `removed`, and restores
`before` (caret **and** selection, so undoing a cut re-selects the text). Redo
is the reverse and restores `after`. TextArea addresses edits by `(row, col)`
rather than rune offset on purpose: `posToOffset`/`offsetToPos` walk the lines,
which would make every keystroke in a large document O(lines).

### One mutation funnel

Today a "replace the selection with a paste" is `deleteSelection` followed by an
insert - two mutations that would record as two undo steps. Instead every
editing path is rewritten onto one primitive per widget:

```go
func (t *TextField) replace(lo, hi int, with []rune, kind editKind)
func (t *TextArea)  replace(from, to pos, with []rune, kind editKind)
```

`replace` captures `removed`, applies the change, sets the caret, records the
edit, and fires `OnChange` once. `insert`, `deleteBack`, `deleteForward`,
`insertString`, `cutSelection` and `insertNewline` become thin callers. This is
the bulk of the work and it is also a simplification: change notification and
history live in one place instead of at seven call sites. Undo/redo apply edits
through the same low-level splice but do **not** record themselves.

### Coalescing - when several edits are one step

Per-keystroke undo is useless; per-paragraph is dangerous. Edits merge into the
top step when all of these hold:

- both are the same **mergeable kind**: typed rune, backspace, or forward delete.
  Paste, cut, selection replacement, `Enter` and IME commits are `kindOther` and
  never merge, so each is its own step;
- the step is still **open** (not sealed, below);
- they are **contiguous**: a typed rune lands at the end of the previous insert;
  a backspace removes the rune immediately before the previous deletion; a
  forward delete removes at the same position;
- typing does not cross a **word boundary**: a whitespace rune typed after a
  non-whitespace one starts a new step. So `hello world` undoes as `" world"`
  then `"hello"`, which matches what users expect without a timer.

A step is **sealed** (no further merging) by anything that is not an edit:
caret or selection change (arrows, Home/End, mouse, select-all), losing focus,
undo/redo, and `SetText`. No wall-clock rule: a timer would make history depend
on typing speed and make headless tests non-deterministic.

Coalescing needs no cap pre-freeze. One consequence is accepted and noted: an IME
that commits multi-rune strings through repeated `EventText` runes with no
whitespace (CJK) coalesces into long steps. Acceptable; revisit with a per-step
rune cap if it proves annoying.

A new edit **clears the redo stack**. History is capped at `undoDepth` steps,
dropping the oldest.

### SetText and programmatic edits

`SetText` **clears history.** It is how apps load content (open a file, reset a
form), and undoing back into the previous document is wrong. The consequence,
stated plainly: a programmatic change through `SetText` is **not** undoable, so
`examples/editor`'s replace-all, which called `SetText(strings.ReplaceAll(...))`,
lost the user's history until it moved to `ReplaceRange` (see Open points).

### Interactions

- **Masked `TextField`:** history holds the real text. That is correct, not a leak:
  undo never renders anything the field would not (already recorded in MASKED
  TEXT INPUT, so nobody "fixes" it into storing mask runes and corrupts the
  buffer). Undo is allowed while masked.
- **Clipboard:** `copy` is not an edit. `cut` is one `kindOther` step; undoing it
  restores the text and the selection but does not touch the clipboard.
- **IME preedit:** uncommitted composition is not in the text, so it is never in
  history; only the commit is, as `kindOther`.
- **TextArea wrap:** wrapping is display only; edits are in logical (row, col)
  space and unaffected.
- **`CanvasWidget` and other custom widgets:** unaffected; they own their state.

### Design decisions (locked)

| Area | Decision |
|---|---|
| Representation | Edits (position + removed + inserted + before/after selection), not snapshots. |
| Plumbing | One `replace` primitive per widget; all mutations go through it, so history and `OnChange` live in one place. |
| TextArea addressing | `(row, col)`, not rune offset, to keep keystrokes O(1) in large documents. |
| Public API | `Undo`/`Redo`/`CanUndo`/`CanRedo`/`ClearHistory` on both widgets. No options. |
| Keys | `ModPrimary+Z`, `ModPrimary+Shift+Z`, `ModPrimary+Y`. |
| Coalescing | Contiguous typed runes / backspaces / forward deletes, split at word boundaries; no timer. |
| Never merges | Paste, cut, selection replacement, Enter, IME commit. |
| Sealing | Caret/selection change, focus loss, undo/redo, `SetText`. |
| `SetText` | Clears history; not undoable. |
| Depth | `undoDepth = 500` constant, oldest dropped; redo cleared by any new edit. |
| Selection | Undo restores the pre-edit caret and selection; redo the post-edit. |
| Masked | Real text in history; undo allowed. |
| Out of scope | App-level command history (userland, per ROADMAP). |

### Tests (`guitest` / `ui`)

- Type `abc`: one Undo empties the field; Redo restores it with the caret at the
  end. `OnChange` fires with the right text each time.
- Type `hello world`: Undo gives `hello`, a second gives empty.
- Backspace run coalesces; a forward-delete run coalesces; moving the caret
  between them splits the steps.
- Caret move (arrow, click, Home) between typed runs splits them.
- Paste over a selection is **one** step: Undo restores the original text and
  the original selection. Cut: Undo restores text and selection.
- A new edit after Undo clears Redo (`CanRedo()` false).
- `SetText` clears both stacks.
- Depth: 501 distinct steps leave 500 undoable, the oldest gone.
- Masked field: type, Undo, Redo all work and the display stays masked
  (`Recording` shows no real text at any point).
- TextArea: Enter, multi-line paste, multi-line delete and selection across lines
  each undo exactly; Undo of a line-joining Backspace re-splits the line; caret
  and selection restored (row, col).
- TextArea in a large document: a typing run's cost does not grow with line
  count (assert `replace` makes no whole-document pass).
- Disabled widget and active IME composition ignore Ctrl+Z.
- Focus loss seals: type, blur, refocus, type: two steps.

### Open points

- **Undoable programmatic edits (done).** `ReplaceRange(start, end, s)` on both
  widgets replaces a rune range (offsets into `Text()`, clamped, order
  independent) as one `kindOther` step, leaving the caret after `s`, firing
  `OnChange`. `examples/editor` replace-all uses it; `SetText` still clears
  history. A whole-document replace stores the old and new text once each.
- **Selection-only steps.** Select-all and caret moves are not undoable. Matches
  most toolkits; noted because some editors do record them.
- **Memory bound.** Depth is by step count, not bytes. Fifty 5 MB pastes would be
  retained. Add a byte budget if a real app pastes large blobs.

---

## CANVASWIDGET (CUSTOM DRAW)

*(Implemented - `ui/canvaswidget.go`, tests in `guitest/canvaswidget_test.go`. Roadmap row: "Custom-draw widget", pre-freeze.)*

A public widget whose appearance and input handling are supplied as callbacks,
so application code can draw charts, plots, diagram/paint surfaces, game HUDs
and visualisers without defining a widget type.

### Why it is not already possible

It is. `examples/canvas` embeds `ui.BaseWidget`, overrides `Draw` and `MinSize`,
and works. So `CanvasWidget` adds no capability. What it adds:

- **No type, no boilerplate.** One-off surfaces become a constructor call with
  closures over app state, not a struct plus `NewBase()` plus method set.
- **A blessed, documented contract.** Today the rules (absolute coordinates,
  when `Draw` runs, how to get keyboard input) are discoverable only by reading
  `App`. A named widget is where they get written down and frozen.
- **Correct defaults.** Clipping to bounds, focus handling, hover tracking and
  redraw-on-demand done once, rather than re-derived (and gotten wrong) per app.

It stays a thin wrapper. Anything wanting real structure (child widgets, own
layout) should still embed `BaseWidget`, and the doc comment says so.

### API - mirrors NewButton / ButtonOption

```go
type CanvasWidget struct { BaseWidget /* ... */ }

type CanvasOption func(*CanvasWidget)

func NewCanvasWidget(draw func(c render.Canvas, bounds geom.Rect), opts ...CanvasOption) *CanvasWidget

// options
func CanvasMinSize(w, h float64) CanvasOption // MinSize for layout; default 0x0
func CanvasOnEvent(fn func(ev *Event) bool) CanvasOption
func CanvasFocusable() CanvasOption           // join Tab order, receive keys/text

// methods
func (w *CanvasWidget) SetDraw(draw func(render.Canvas, geom.Rect))
func (w *CanvasWidget) SetOnEvent(fn func(*Event) bool)
func (w *CanvasWidget) Hovered() bool
func (w *CanvasWidget) Focused() bool
func (w *CanvasWidget) ToLocal(p geom.Point) geom.Point // absolute -> bounds origin
```

`NewCanvasWidget` takes `draw` positionally because a canvas without one draws
nothing; options are for the genuinely optional parts. A nil `draw` is allowed
(draws nothing) so a widget can be built before its renderer exists.

`CanvasWidget` inherits the `BaseWidget` surface for free: `SetTooltip`,
`SetContextMenu`, `OnFileDrop`, `SetDragSource`/`SetDropTarget`, visibility,
enabled, colour roles. No wrapper methods for any of them.

### Drawing contract

- `bounds` is the widget's **absolute** rectangle, exactly what `Bounds()` returns
  and what every other widget's `Draw` uses. `c` is the same canvas the rest of
  the tree draws on, so coordinates are absolute too. Absolute (rather than
  translated to a 0,0 origin) is the established convention; `ToLocal` is there
  for callers who want to think locally.
- The framework `PushClip(bounds)` before calling `draw` and `PopClip()` after, so
  a callback cannot bleed over neighbours however it computes its geometry.
- `draw` runs **only when a frame is presented**. Presenting is on demand
  (INTERNALS 8.5), so a canvas that changes over time must ask for frames:
  - state changed from an event callback: nothing to do, input already schedules
    a frame;
  - changed from another goroutine: `App.Do` / `App.Invalidate`;
  - continuously animated: `App.Animate` or `App.OnFrame`, which hold the app at
    continuous presenting while they run.
  `w.Invalidate()` only requests a re-layout, not a frame, so it is not the way to repaint; use `App.Invalidate`.
- `draw` must not mutate the widget tree. It reads theme colours via
  `w.ColourOf(role)` so a canvas follows `App.SetTheme`; closures that cache
  colours at construction defeat the runtime theme switch, so the doc comment
  and example use `ColourOf`.
- Text goes through `c.DrawText` / `c.MeasureText` with a `FontFace` the caller
  supplies; the widget does not choose one.
- Hit-testing and `Layout()` are inherited: a childless widget, hit anywhere in
  its bounds.

### Input contract

A single `OnEvent` callback receives **every** event routed to the widget,
rather than one callback per event type:

```go
ui.CanvasOnEvent(func(ev *ui.Event) bool {
    switch ev.Type {
    case ui.EventPointerDown: ...
    case ui.EventPointerMove: ...
    case ui.EventWheel:       ...
    case ui.EventKeyDown:     ...
    }
    return true // consumed: stop bubbling
})
```

Reasons: a drawing surface wants pointer down/move/up, wheel, keys and text
together and switches on them anyway; one callback keeps `Event` the single
vocabulary (no parallel canvas-specific event type to freeze); and new event
types (touch, gesture) then arrive with no API change. The return value follows
`HandleEvent`: true consumes, false lets the event bubble to ancestors.

- **Pointer capture is automatic.** The App already routes `PointerMove` and
  `PointerUp` to the widget that took the `PointerDown` until release, even when
  the cursor leaves its bounds. A canvas gets drag/pan/draw for free; nothing to
  opt into. Events carry **absolute** `Event.Pos`; use `ToLocal`.
- **Hover** (`PointerEnter`/`Leave`) is tracked internally for `Hovered()` and
  then passed to `OnEvent` like everything else.
- **Keyboard and text** only arrive if `CanvasFocusable()` is set (the widget
  returns true from `Focusable()`, so it joins Tab traversal and takes focus on
  click). Default is not focusable: a pure display (chart, gauge) should not
  steal Tab stops. `FocusGained`/`FocusLost` are delivered and tracked for
  `Focused()`.
- **Focus ring:** drawn by the widget **only when focusable**, via the shared
  `strokeFocusable` helper (`ui/focus.go`), after `draw` returns, so keyboard
  focus looks the same as every other control. No opt-out pre-freeze; a game HUD
  that does not want a ring should not be focusable. Additive later if wrong.
- **Disabled:** `SetEnabled(false)` stops event delivery as for any widget; the
  callback can read `Enabled()` to dim its drawing.
- Returning `false` for `PointerDown` behaves as for any non-consuming widget
  (no special case).

### Sizing

`CanvasMinSize` feeds `MinSize()`. Default is zero, so callers normally give the
widget a stretch weight in its `Box` so it takes the remaining space. The
callback learns its real size each frame from `bounds`, so resize needs no
handling. There is deliberately no `OnResize`: `Layout()` is the only place that
would fire it and a self-drawing widget can derive everything from `bounds`.

### Offscreen / cached rendering

Redrawing a heavy canvas (thousands of points) each presented frame is wasteful
when only the cursor moved. The seam already has `render.RenderTarget`
(`ui.NewRenderTarget`, INTERNALS 3.3), so an app can render into a target and
`DrawImage` it from `draw`. **No caching is built into `CanvasWidget` v1**:
the invalidation policy (when is the cache stale?) is app knowledge and the
widget cannot guess it. The doc comment shows the pattern. A built-in cache is
additive if a real app wants one.

### Accessibility

Once the a11y seam lands, `CanvasWidget` gets a graphic/custom `Role` and an
`AccessibleName` option. A canvas is opaque to assistive technology, so the name
is all it can offer; the example sets one as a habit.

### Design decisions (locked)

| Area | Decision |
|---|---|
| Shape | One concrete widget, `NewCanvasWidget(draw, ...CanvasOption)`. Not an interface, not generic. |
| Not a new capability | Embedding `BaseWidget` still works; it is the answer for anything with children or layout. |
| Draw signature | `func(c render.Canvas, bounds geom.Rect)`, absolute coordinates, as the roadmap froze it. |
| Clipping | Framework clips to bounds around the callback. |
| Input | One `OnEvent(*Event) bool` for all event types; true consumes. |
| Capture | Automatic, inherited from the dispatcher; no opt-in. |
| Event coordinates | Absolute `Event.Pos`; `ToLocal` for local. |
| Keyboard | Off by default; `CanvasFocusable()` opts in (Tab stop + keys + text). |
| Focus ring | Drawn by the widget when focusable, via `strokeFocusable`. |
| Redraw | On demand like everything else; time-varying canvases use `Animate`/`OnFrame`, or `App.Invalidate` from outside an event. Documented, not hidden. |
| Theme | Callback reads colours via `ColourOf`, so `SetTheme` works. |
| Background fill | No option; `draw` can `FillRect` first. One fewer frozen symbol. |
| Caching | None built in; `RenderTarget` pattern documented. |
| Resize | No callback; derive from `bounds`. |
| Everything else | Tooltip, context menu, drag/drop, file drop inherited from `BaseWidget`. |

### Tests (`guitest`)

- The draw callback runs once per presented frame, receives `Bounds()`, and its
  ops are clipped to them (`Recording` clip stack).
- `SetDraw` swaps the renderer; a nil draw draws nothing and does not panic.
- Press, move outside bounds, release: `OnEvent` sees all three (capture) with
  absolute positions; `ToLocal` maps them.
- Not focusable by default: Tab skips it and `KeyDown` never arrives. With
  `CanvasFocusable`: Tab reaches it, keys and runes arrive, `Focused()` tracks,
  and a focus ring is drawn.
- `OnEvent` returning false bubbles to the parent; true does not.
- A disabled widget receives no events.
- Hover toggles `Hovered()` and the event still reaches `OnEvent`.
- Captured state changed then `App.Invalidate()` redraws on the next frame with no
  other input (the on-demand pacing path).
- Theme switch: a callback using `ColourOf` picks up the new palette.

### Open points

- Export `ToLocal`/`Hovered`/`Focused`, or leave them to closures that hold the
  widget? Lean: export. They are tiny, and otherwise every caller re-derives
  hover from events.
- Name: keep `CanvasWidget` (roadmap); plain `Canvas` would read as
  `render.Canvas` at the call site.

---

## IMPLEMENTATION PLAN

1. **Skeleton & backend seam** — `geom` types, `render.Canvas`/`Input`
   interfaces, EBiten backend implementing them, `App.Run()` owning the loop and
   clearing the screen.
2. **Widget tree & base** — `Widget` interface, `BaseWidget`, root container,
   dirty/invalidate, draw traversal.
3. **Layout engine** — `Box` (H/V) first, then `Grid`/`Flex`; resize handling.
4. **Core widgets** — `Label`, `Button`; theme + default font; click/hover/focus.
5. **Event system** — dispatch, hit-testing, focus traversal, callbacks, then
   the optional event bus.
6. **Expand widgets** — second and third waves per the catalog.
7. **Add Modal windows** - allow modal dialogs
8. **Add text area widget** - add widget that has multi line text support
9. **Tabs** — a tabbed container that switches between panes via a tab strip.
10. **Tooltips** — hover-delayed floating hints anchored to a widget.
11. **Splitters** — draggable dividers that resize adjacent panes (H/V split panes).
12. **Tables** — a grid of rows/columns with headers and selectable rows.
13. **Drag-and-drop** — pick up, drag and drop widgets/data between targets.
    *(Designed — see DRAG-AND-DROP below; in-process engine layered on pointer
    capture.)*
14. **Multi-window** — more than one top-level OS window, each with its own tree.
15. **Dirty-region redraw** — only repaint changed regions instead of every frame.
16. **Accessibility hooks** — expose roles/labels/state for assistive technology. *(Deferred - see ROADMAP -> Deferred -> Accessibility.)*
17. **Text editing enhancements** — soft word-wrap and selection/clipboard for
    `TextField` and `TextArea`.
18. **Grid cell-spanning** — per-cell column/row spans in the `Grid` layout.
19. **High-DPI scaling** — logical vs physical pixels, device scale factor.
20. **Button has images instead of text** - allow image buttons (as well as text buttons)
21. **Display Images** - Can images be displayed in canvases? Or something else?

Build a tiny example app alongside each step to exercise the new pieces.

---

## ROADMAP (post-grilling, 2026-07)

Direction (locked): **public open-source release first, then build real
applications on it.** The catalog is broad and mostly done; the remaining work is
depth (perf, a11y, native feel), consistency, and presentation — not more
widgets. This roadmap captures a grilling pass over "what else should guie have,
compared to mature toolkits (Fyne, Qt, GTK, Flutter, egui, WPF, Slint)."

### Release strategy & the freeze gate

| Area | Decision |
|---|---|
| Versioning | Tag **v0.x** (semver "may break"), not v1 — honest about instability. |
| API freeze | Do a consistency audit of every exported symbol in `ui`/`geom`/`render`/`theme`, then freeze — but only **after building a real app** to shake out warts (dogfooding is the freeze gate, not a checklist). |
| Pre-freeze rule | Anything that adds/reshapes **public API surface** must land (at least in *shape*) before freeze. Purely internal work can land after. |
| Launch bar | `LICENSE` (done) + `CONTRIBUTING` + CI (`.github/workflows`) + a strong README + push to GitHub. Docs site / playground / gallery come **later**. |

### Pre-freeze — API-surface work (must land, or lock shape, before v0.x)

These reshape public API and are hard/churny to retrofit after freeze.

| Feature | Decision / scope |
|---|---|
| Data binding | Thin **optional** `Observable[T]` layered on existing callbacks (callbacks stay primary, mirrors the event-bus pattern). **Keep it dead simple** — no reactive rewrite. |
| i18n (seam-safety) | Ensure text layout does **not** assume `x-advance == Σ per-rune widths`, so a future bidi pass can land without an API break. (Translations/locale formatting itself is additive — see below.) |
| Custom-draw widget | Public, blessed **`CanvasWidget`** — a thin `BaseWidget` wrapper taking a `func(c render.Canvas, bounds geom.Rect)` draw callback + an input callback. The escape hatch for charts, viz, diagram/drawing surfaces, game HUDs. The `render.Canvas` seam already exists. |
| Theming | Ship a **dark theme + runtime theme switch** now; verify every widget reads `theme` in `Draw` (no colours cached at construction) — the switch is a correctness check on the theming architecture being frozen. OS light/dark **auto-detect deferred** (opt-in bridge). |
| Multi-window | **Lock the `App`/`Window` API shape** so it doesn't bake in a single-window assumption (e.g. `App.MainWindow()` singleton); **implement lazily** when a real app needs it. Inherently desktop-only (no WASM). |
| Rich text | Reserve an attributed-text / **`[]TextSpan`** type and ship a basic **`RichText`** widget (multi-style spans + clickable links). Keep plain `Label` simple/fast. **Markdown deferred** (layer on top / userland). |
| Text undo | Built-in **undo/redo inside `TextField`/`TextArea`** (a correctness expectation of the widget). App-level command history stays userland. |
| Window geometry | *(Done - INTERNALS 8.7.)* Small helper to persist/restore window **size + position** (mostly non-OS-specific; "which monitor" restore needs backend help later). |
| Masked text input | `Masked()` option + `SetMasked` on `TextField`, so a credential can be entered without being displayed. Public API, so pre-freeze. Spec in *MASKED TEXT INPUT*. *(Done — `INTERNALS.md` §14.4.)* |
| OS file drops | `InputState.DroppedFiles` (`fs.FS`) + `EventFileDrop` + `OnFileDrop` — public API on the seam, the widget and the event set, so the shape must land pre-freeze. Small: EBiten already surfaces the drop. Spec in *DRAG-AND-DROP → OS file drops*. |

### OSS-release milestone (additive; around/just after release)

| Feature | Decision / scope |
|---|---|
| WASM target | **Supported + advertised.** Whole repo builds clean for `GOOS=js GOARCH=wasm` today (verified), the `clipboard` package included — a runtime failure there degrades to the in-process fallback. Guard remaining OS opt-in packages (dialogs) behind build tags. Browser runtime smoke-test + docs-site playground to follow. |
| In-app dialogs | `ShowMessage` and `ShowConfirm` (done) + an **in-surface file chooser** (not built) built from existing List/Tree/TextField. Pure `ui`, cross-platform incl. WASM. |
| Translations / locale | App-level translation strings + locale-aware date/number formatting. Additive display layer; no widget rewrite. |
| Redraw-when-dirty | Skip the whole frame draw when the tree isn't dirty, using the existing `Invalidate` signal (needs `ebiten.SetScreenClearedEveryFrame(false)`). Kills idle CPU/battery cheaply. |
| Launch collateral | `CONTRIBUTING`, CI workflow, README polish, GitHub. |

### Real-app-hardening milestone (later, mostly internal)

| Feature | Decision / scope |
|---|---|
| True dirty-region redraw | Repaint only changed rects (the hard version of the perf work; item 15). Justified by weak hardware / Raspberry Pi. |
| Native file/save dialogs | Opt-in `dialog/native` package (Win32 / NSOpenPanel / GTK), mirroring the `clipboard` package. |
| A11y OS bridges | UIAutomation / AT-SPI / NSAccessibility bridges feeding the semantic tree. |
| Complex-widget a11y | Semantic annotation of List / Tree / Table (per-row/node roles, selection, expansion). |
| OS light/dark detect | Opt-in bridge to follow the system theme. |

### Deferred (revisit when a real app demands it)

RTL / bidi text rendering (seam kept safe) · full multi-window implementation ·
OS notifications · system tray / minimize-to-tray · trackpad/gesture &
multitouch input · **dragging files *out* of the window** (the reverse direction —
EBiten surfaces no drag source, so it needs a real OS bridge, unlike inbound file
drops which are now pre-freeze work).

### Accessibility (deferred as a whole - decided, not yet scheduled)

The whole a11y effort is deferred: the semantic seam, the simple-widget
annotations and the OS bridges. It was previously pre-freeze (seam + simple
widgets now, bridges later); that is withdrawn. Nothing exists in code yet.
Because the seam is additive (new `BaseWidget` methods, a new optional backend
interface), landing it after freeze does not break callers, with one exception
to settle first - see the name clash below. Revisit before a real app has
accessibility requirements, or before a v1 freeze.

Scope when picked up (in order):

1. Seam: role/name/state on `BaseWidget`, a `SemanticNode` + `App` tree walk,
   `guitest` assertions (`Harness.Semantics()`). Annotate Button, Label,
   Checkbox, Radio, TextField, `CanvasWidget` (graphic role + name option).
2. Complex widgets: List / Tree / Table / Menu / Tabs via an optional
   `SemanticChildren()` hook (they are self-drawn and expose few real children).
3. OS bridges (UIAutomation / AT-SPI / NSAccessibility) behind an optional
   `render` interface, like `IMEController`.

Open design points to decide then:

- **Name clash:** `Role*` constants are already the colour roles (`ColourRole`:
  `RoleSurface`, `RolePrimary`...). The semantic type needs a distinct name
  (e.g. `SemanticRole`), or the frozen API conflates the two. This is the one
  thing worth deciding before freeze.
- Role set, a state struct (checked/selected/expanded/disabled/focused/value/
  min-max-now), and name vs value vs description (a field's value is not its name).
- Derived defaults (Button derives its name from its text) vs overrides.
- A labelled-by relation (`Label` naming a `TextField`), live regions (`Toast`),
  hidden widgets dropping out of the tree, focused node, screen bounds.
- **Actions** (invoke, focus, expand/collapse, set value, scroll): a bridge needs
  them to drive widgets; reserve the shape in the node type, hard to retrofit.
- A change notification (via the event bus), since on-demand presenting gives a
  bridge no frame signal to poll.
- Masked `TextField` must not leak its content into the tree (see MASKED TEXT
  INPUT); needs a test when the tree exists.
- Keyboard-parity audit: mouse-only paths (a `CanvasWidget` with no key handling).
- Related, not blocking: high-contrast theme and reduced-motion for the fades.

### Out of scope / userland

Form validation (compose a `Label` under the field) · app-level undo/command
history · markdown parsing · app preferences / recent-files storage · charts &
data-viz (build on `CanvasWidget`) · printing / PDF export · video / media
playback · responsive layout breakpoints.

---

## STAGE 2 - UI / VISUAL POLISH

Follows the roadmap above. Goal: richer look and feel without changing the
widget model. Starting point is the current theme: a 9-role dark palette plus
Font, FontSize and CornerRadius, with per-widget colour overrides
(`SetColour(role, c)`) and `App.SetTheme` for runtime switching.

### 2.1 Themes (low effort, high visible payoff)

| Item | Scope |
|---|---|
| Light theme | `theme.LightPalette()` / `theme.Light()` alongside `DefaultPalette()`. |
| High-contrast theme | *(Done: `theme.HighContrast()`, contrast-tested in `theme`; see 2.2.)* Accessibility-oriented palette; verify focus and selection stay visible. |
| Brand/custom themes | *(Done: `examples/theming` builds Ocean and Forest by hand.)* Document building a `theme.Palette` by hand. |
| Theme switcher | *(Done: `examples/theming`; the palette audit is the correctness check.)* Runtime light/dark/high-contrast toggle via `App.SetTheme`. Doubles as a correctness check that no widget caches colours at construction. |
| OS light/dark detect | Opt-in bridge to follow the system theme (already deferred in roadmap). |

### 2.2 Richer Theme model

*(Designed. Supersedes the short list that stood here.)*
*(Steps 1 to 5 and 7 built; 6 (the audit) is running and grows with the widgets. Notes on each follow.)*

*(Step 7 built: the theming example. Of the build order, only step 6 needs more work, and it is a test that grows with the widgets.)*

**Step 7, as built:** `examples/theming` is rewritten around the new model, in a 980 x 700
window with the controls on the left and a live preview card on the right.

- **Theme picker:** Dark, Light, **High contrast** (a full preset: its borders, focus
  ring, state tints, no shadows and no fades come with it) and the two custom
  palettes, Ocean and Forest, on the default look. Corner radius and accent pickers
  are kept.
- **Density and Spacing sliders** (0.7 to 1.5, and 2 to 20) drive `Theme.Density`
  and `Theme.Spacing`; every row and column in the example is built with
  `ui.ThemeGap`, so dragging Spacing visibly moves them and Density scales both the
  padding inside widgets and the gaps.
- **Reduce motion** checkbox sets `Motion.Reduced` (High contrast has it on already).
- **Try it:** a toast of each kind (fills from the status colours), and a confirm
  dialog (the backdrop is the theme's scrim).
- The preview card shows a button, a flat button, text field, checkbox, slider,
  stepper, dropdown, a list and a table with a selected row, striping and a caption,
  so the hover, pressed, selection and focus states can all be seen. Tab walks it.
- The status line reports what is applied (theme, border and focus widths, motion).
- Both columns sit in a `ScrollView` (horizontal scrolling on), because at density 1.5 and
  spacing 20 the controls are wider and taller than the window. This was found by
  laying the example out headlessly at the extremes: before the scroll views the
  card ran 34 px past the bottom of the window and the controls pushed the preview
  off the right edge. At the defaults nothing scrolls, and at every setting the
  root fits the window.
- A Container's own background and border are explicit values, so `applyTheme`
  re-applies them from the app's completed theme (including `BorderWidth`).

It compiles and runs, but **has not been looked at**: whether the focus halo reads
well, how the filled-button hover compares with the old one, and how High contrast
looks all need eyes on a real window (`go run ./examples/theming`).

*(Step 5 built: motion from the theme. Steps 6-7 are not; 6 is already running.)*

**Step 5, as built:**

- The widgets' own state fades (hover, press and the like, through `fadeTo`) and the
  tab underline's slide read `Theme.Motion`. The length is `Motion.Duration` (0.12 s
  by default; the tab slide is 1.5 x that), through `BaseWidget.motionSeconds()`;
  the old `transitionDuration` constant is gone.
- **`Motion.Reduced` makes those transitions jump to their end state in the frame
  they are triggered**, by the same path a widget takes before it is mounted. Nothing
  is left animating, so an app that is idle stays idle (`needsFrames` is false after
  a reduced-motion hover; true after a normal one, which the tests check as a pair).
- **What reduced motion does not touch**, on purpose: motion the application starts
  with `App.Animate` / `App.Tween` / `App.OnFrame`, toasts fading, and the spinner.
  Those are application behaviour or content (a spinner is the "busy" signal), and
  an app that wants them off can read `Theme.Motion.Reduced` itself. A fade already
  in flight when the theme changes simply finishes (it is a tenth of a second).
- A theme or widget with no motion settings falls back to 0.12 s and "not reduced".
- Guarded by mutation, each caught by a test: ignoring `Reduced` in `fadeTo`, ignoring
  it in the tab slide, and hard-wiring the duration. The tests cover the fade length
  ordering (0.03 < 0.12 < 0.6 s), the default being about 7 frames, snap-on-enter,
  snap-on-leave and snap-on-press, following `SetTheme` in both directions, the tab
  slide length, and `App.Animate` still running under a reduced theme.
- Nothing reads the operating system's reduced-motion setting; that is the opt-in
  bridge listed under "not in the model".

*(Step 4 built: `ThemeGap`. Steps 5-7 are not.)*

**Step 4, as built:**

- `const ui.ThemeGap = -1`, accepted anywhere a spacing is: `ui.VBox(ui.ThemeGap)`,
  `ui.HBox(ui.ThemeGap)`, `ui.NewGrid(cols, ui.ThemeGap)`. **Any negative spacing
  means the same**; a negative gap used to mean overlapping children, which nobody
  wanted. An explicit number, `0` included, is exactly that number: `VBox(10)` is 10
  at every density and under every theme, because the caller asked for 10.
- `Box` and `Grid` read their spacing through `gap()`, which returns `Spacing`, or
  the theme's gap when `Spacing` is negative. The theme's gap is **`Theme.Spacing x
  Density`** (default 8 x 1). The `Container` that owns the layout sets it through
  the unexported `spacingSetter` interface, in `resolveGap`, immediately before
  every `Measure` and `Arrange`. Nothing is resolved at construction, so a
  `SetTheme` (new spacing or new density) takes effect on the next layout pass
  with no rebuilding, and a theme written before `Spacing` existed gives 8.
- It reaches `MinSize` as well as layout, so a container's size includes its
  themed gaps (tested for both axes and at density 2).
- Grid track sizes use the same resolved gap: the last column and row end exactly at
  the container edge. A first version of the test only measured the gaps and did not
  notice a grid whose track sums used the raw `-1`; it now also checks the edges, and
  fails on that mutation.
- **What does not follow the theme:** the built-in dialogs and popups still use
  their own explicit gaps (`VBox(12)`, `HBox(10)`, padding 16), and a Container's
  `SetPadding` is the caller's number. Only `Box`/`Grid` spacing has a theme mode.
- Unmounted containers (before `SetContent`) measure with the default theme, as
  every other widget does.

*(Step 3 built: Density reaches every widget's padding and gaps. Steps 4-7 are not.)*

**Step 3, as built:**

- `BaseWidget.px(v)` and `insets(in)` multiply by `Theme.Density`; `App.px(v)` does it
  for the things that are not widgets (toasts, tooltips); `densityOf(theme)` is the
  shared rule (a missing or non-positive density counts as 1). `density()` reads
  the theme through the tree context instead of copying it, because it is asked on
  every layout and hit test. At density 1, `v * 1.0 == v` exactly, so the default
  look is unchanged to the bit (tested, and an old-style theme still draws
  identically to `Default()`).
- **Scaled:** the padding and gaps of `Button` (and its icon gap), `Checkbox` and
  `Radio` (indicator gap), `TextField`, `TextArea`, `Stepper`, `DropdownCombo`,
  `DateField`, `DatePicker`, `ColourPicker`, `List`, `Tree` (row padding, indent,
  gap), `Table` (cell padding), `TabContainer`, `MenuBar` and its popup rows, toasts
  and tooltips. The same accessor is used by `MinSize`, `Draw` and every hit test
  (`rowAt`, `caretAt`, `chevronHit`, `headerColAt`...), so they cannot drift apart.
- **Not scaled, on purpose:** text and font size; control glyphs and fixed pieces
  (the tree's chevron column, the stepper's button column, the dropdown arrow, the
  date field's icon, slider and progress-bar heights, the scroll bar thickness);
  any number an application passes itself (`VBox(10)`, `SetPadding`, a Container's
  border); and the built-in dialogs' own panel padding. Row and field **heights
  still follow the font**, so a row is `text height + 2 x scaled padding`.
- The popup menu panel and the dialog panel now use `BorderWidth` for their outline
  too (a step-2 loose end).
- **How it is guarded.** `TestDensityScalesPaddingAndOnlyPadding` holds, per widget,
  the padding it adds at density 1, worked out by hand from the constants, and
  checks the size changes by exactly that much per unit of density on each axis.
  A merely "linear in density" check was tried first and **missed** a deliberately
  unscaled `tabVPad` (a constant is linear too), so the exact-slope table replaced
  it; it now fails on that mutation. The hit-testing tests (`chevronHit` at 0.5 and
  2, caret placement after a click, row selection by scaled row height) fail if a
  hit test is left on the unscaled value. Also checked: text sits one scaled
  padding in from the edge when drawn, toast padding scales, and a dialog stays on
  a small surface at 1.25.
- No `Compact()` / `Spacious()` presets: `WithDensity(0.8)` is the whole API.

*(Step 2 built: roles, tints, widths, the focus-ring halo, and toasts, scrim and
shadows reading the theme. Steps 3-7 are not.)*

**Step 2, as built:**

- `ColourRole` gains `RoleSuccess/Warning/Error/Info/Scrim`, appended. `RoleInfo`
  resolves to `Primary` when `Palette.Info` is nil.
- **Tints are `tintKind`s resolved against `Theme.States`** (`tintOf`), replacing the
  five float constants. The flat button's hover is `States.Hover x 4/3`, which is
  exactly the old 0.08 at the default 0.06. Every call site changed; at the default
  theme the numbers are identical.
- **`NoShadows`** was not in the model above but `HighContrast` needs it (the design
  says "no soft shadow"): a `Theme.NoShadows` bool, off by default, ANDed with
  `App.SetShadows`. It is checked in one place, `App.shadowsOn`.
- **Widths.** The outlines of `List`, `Tree`, `Table`, `ProgressBar`, the colour
  picker, tooltips and toasts use `BorderWidth`; the slider's handle ring and every
  focus ring use `FocusWidth`. Still literal (deliberately): explicit `Container`
  borders (the caller chose them), the date picker's 1 px "selected day" ring, the
  date field's calendar glyph, the colour picker's handle marker.
- **Focus ring.** `strokeFocusable` is now a `BaseWidget` method, so it reads the
  theme itself, and draws the ring plus a **halo** of the page colour (`Background`)
  just inside it, `max(1, BorderWidth)` thick. The ring's own contrast against a
  filled control is unchanged (the light theme's `Accent` still equals its
  `Primary`); the halo is what keeps it visible, because the ring is now bounded on
  both sides by something that contrasts with it. **This changes how every focused
  control looks by one thin line** just inside the ring - worth eyeballing in the
  `showcase` and `theming` examples, since it cannot be checked headlessly.
- **Toasts** take their fill from `Success/Warning/Error` (or `Info`/`Primary`), text
  from `OnPrimary`, and a border that is the fill shaded 30% toward black. The old
  borders were hand-picked darker greens, ambers and reds; the derived ones are
  within a few units per channel, so a toast looks the same at a glance but is not
  bit-identical. The warning fill is the darker `#946300` noted in step 1.
- **Scrim** comes from `Palette.Scrim` (default unchanged: black at 0xA0).
- **Filled-button states** went through the tint mechanism without needing the
  fallback: hover is `tint(Primary, Hover x 1.5)` and pressed is `Primary` shaded
  toward black by `Pressed x 1.43`. Pressed is **numerically identical** to the old
  `darken(0.8)` at the default tints (checked). Hover is a little less blue-bright
  than the old `lighten(1.15)` (per channel roughly +5, -4, -17 on the dark
  `Primary`), because a tint toward white is uniform where scaling is not.
- **The palette audit** (`guitest/theme_audit_test.go`) is in and covers 27 widget
  states (every widget, plus hover, press, focus, selection, open popups, tooltips,
  toasts of each kind and a modal). It was checked by sabotage: hard-coding the
  success toast colour, or the halo colour, makes it fail naming the colours. Not
  covered: the colour picker (its hue strips are content, not chrome) and anything
  an application draws itself in a `CanvasWidget`.
- Also tested: halo and ring widths and halo-inside-ring for Default and
  HighContrast; border width; toast and scrim colours from the theme; shadows off
  for `HighContrast` and still off by `SetShadows(false)`; hover strength following
  `States.Hover`; and **an old-style `Theme` literal drawing bit-identically to
  `Default()`** for a button, list, text area and table.

**Step 1, as built, and where it differs from the design above:**

- `Theme.WithDefaults()` is exported (`ui` has to call it) and is what makes "zero
  means default" true: `NewApp`/`WithTheme` and `App.SetTheme` run every theme
  through it, so `app.Theme()` is always complete. A theme written before the new
  fields existed resolves to exactly `theme.Default()` (tested). `SetTheme` keeps
  the current font and size when the new theme has none, but every other zero field
  takes the *default*, not the previous value.
- The built-in palettes carry the new colours themselves (so `Palette ==
  DefaultPalette()` still holds); `WithDefaults` fills them for a palette that
  predates them. **`Palette.Info` stays nil by default and means "use Primary"**,
  because info toasts follow the palette today; only Success, Warning, Error and
  Scrim have defaults.
- **One default colour changed:** the warning fill, from `#B77A00` to `#946300`.
  The old amber gave white text about 3.6:1, and the dark theme's off-white
  `OnPrimary` needs a darker fill again to reach 4.5:1. This is a visible but small
  change to warning toasts. It takes effect when toasts read the palette (step 2).
- `Toggled()` keeps the user's `Density`, `Spacing` and `Motion` but takes borders,
  focus width and state tints from the preset it switches to, so leaving high
  contrast does not drag its 3 px ring along. Tested.
- `HighContrast()` clears 7:1 for text and 3:1 for the focus ring on both the page
  and a filled button.

**Findings from the contrast tests that the design did not anticipate:**

- **The focus ring is invisible on a filled button in the light theme** (its
  `Accent` is its `Primary`, 1.0:1) and weak in the dark theme (1.8:1). The ring is
  drawn in `Accent` over the button's own fill, so this is a real bug in the
  shipped themes, not a test artefact. It is **not** fixed by repainting the
  presets (that would change every app's look); it is fixed in step 2, by drawing
  the ring with a thin contrasting halo (`Background` against `Accent`) so it reads
  on any fill. `TestFocusRingStandsApartFromAFilledButton` holds high contrast to
  3:1 now and records the two known gaps; when the halo lands the test should turn
  into a drawn-pixels check for all three.
- The dark and light presets otherwise already meet the targets: text 4.5:1, muted
  text 3:1, accent on the page 3:1, status fills with white text 4.5:1.


Today a `Theme` is a 9-colour `Palette`, a font and size, and one `CornerRadius`.
Everything else a widget needs to decide how it looks is a constant in its own
file, so a theme can recolour the UI but cannot change how it feels, and a few
things cannot be themed at all. This section says what the model gains, what stays
as it is, and how it is built without breaking anyone.

#### What is wrong today (found by reading the code)

- **State shading is scattered.** Hover and pressed shades are five tint constants
  in `ui/colour.go` (`tintHover` 0.06, `tintPress` 0.14, `tintHeader` 0.05,
  `tintStrong` 0.09, `tintFlat` 0.08) plus a separate `lighten(1.15)` /
  `darken(0.8)` pair in `Button`. A theme cannot change any of them, so a
  high-contrast theme cannot make hover more obvious.
- **Hard-coded colours.** Toast fills are three literal colours per kind
  (`ui/toast.go`), the modal scrim is a literal (`ui/popup.go`), the shadow is a
  literal alpha (`ui/shadow.go`). None follow `SetTheme`, so a toast looks the same
  in light and dark and cannot be branded.
- **Metrics are per-widget constants.** `buttonPadding`, `textFieldPadding`,
  `listRowPad`, `tableCellPad`, `tabHPad`, `menuRowPad`, the row heights and gaps
  are about twenty `var`/`const` values. There is no way to ask for a compact or a
  spacious UI, and `Theme.Spacing` / `Theme.Padding`, planned in the original
  design, were never added.
- **Border and focus width are literals** (`1` in every `StrokeRoundRect`, `2` in
  `focusRingWidth`).
- **Motion is a constant** (`transitionDuration` 0.12 s). There is no way to turn
  animation down for someone who asks the OS for reduced motion.
- **No high-contrast theme**, so focus and selection visibility on the existing two
  themes has never been checked against a number.

What already works and stays: widgets draw through `ColourOf(role)` and read the
theme in `Draw`, never at construction, so `SetTheme` restyles everything on the
next frame; per-widget `SetColour(role, c)` overrides; `Theme.Toggled()`; the
light and dark presets.

#### Principles

1. **Additive, zero value = today.** A `Theme` literal written for the current
   struct must keep working and look the same. Every new field's zero value means
   "use the default", which is what the code does now. (Same rule as `MaskRune`
   and the other pre-freeze additions: nothing a caller already writes changes.)
2. **A knob must reach every widget or not exist.** No field is added that only
   some widgets honour. The audit test (below) is how that is enforced.
3. **States are amounts, not colours.** One hover colour cannot serve a filled
   button, a list row and a table header, because each starts from a different base.
   A theme says *how much* to shift a base colour, and the widget shifts its own.
4. **Few knobs.** One scalar for density, not per-widget padding tables. Per-widget
   detail stays in the widget; the theme sets the overall character.
5. **Checked by numbers.** Presets are tested for text and focus contrast, not
   eyeballed.

#### The model

```go
// theme package
type Palette struct {
    // existing nine, unchanged
    Background, Surface, Primary, OnPrimary, Text, TextMuted, Border, Accent, Disabled color.Color

    // new; nil means "use the built-in default for this role"
    Success, Warning, Error, Info color.Color // status fills (toasts, validation); text on them is OnPrimary
    Scrim                         color.Color // modal backdrop; default black at ~63% alpha
}

// How far a state shifts a widget's own base colour, as the fraction of the way
// toward white (on a dark base) or black (on a light one). Zero fields use the
// defaults shown, which are today's constants.
type StateTints struct {
    Hover    float64 // 0.06  hovered row, cell, tab, menu item
    Pressed  float64 // 0.14  pressed flat button
    Header   float64 // 0.05  table header band, striping (half)
    Emphasis float64 // 0.09  hovered header cell, tooltip body
}

type Theme struct {
    Palette      Palette
    Font         render.FontFace
    FontSize     float64
    CornerRadius float64 // unchanged: zero is genuinely square

    // new
    States       StateTints
    Density      float64 // multiplies every widget's internal padding and gaps; 1 default
    Spacing      float64 // the gap ui.ThemeGap resolves to, before Density; 8 default
    BorderWidth  float64 // control outlines; 1 default
    FocusWidth   float64 // the focus ring; 2 default
    Motion       Motion
}

type Motion struct {
    Duration float64 // hover/press/focus fades, seconds; 0.12 default
    Reduced  bool    // snap straight to the end state, no fades
}
```

`Density` is the one metrics knob. `theme.Compact()` is 0.8, the default 1,
`theme.Spacious()` 1.25 (helpers on `Theme`, below). It scales paddings and gaps,
**not** the font: text size is already `FontSize`, and scaling both would make the
two interact.

Where "zero means default" would be wrong, the field is left out rather than
overloaded: a theme that wants no outline sets `Palette.Border` to a transparent
colour, and square corners remain `CornerRadius: 0` because that field predates the
rule and already means what it says.

#### Presets

```go
func Default() Theme        // dark, as now
func Dark() Theme
func Light() Theme
func HighContrast() Theme   // new
func (t Theme) WithDensity(d float64) Theme
func (t Theme) WithReducedMotion() Theme
```

`HighContrast()` is a near-black background, white text, a bright accent (yellow
family, so focus and selection stay visible against both), **2 px borders and a 3 px
focus ring**, larger state tints (hover 0.18, pressed 0.30), reduced motion on,
and no soft shadow. Its job is to be unmistakable, not pretty. `Toggled()` keeps its
dark/light meaning; cycling three themes is the application's list.

`WithDensity` and `WithReducedMotion` return a copy, so presets compose:
`theme.Light().WithDensity(0.8)`.

#### How widgets read it

The new values reach widgets through two accessors beside `ColourOf`, so a widget
never touches `theme.Theme` directly:

```go
func (b *BaseWidget) ColourOf(role ColourRole) color.Color // + RoleSuccess/Warning/Error/Info/Scrim
func (b *BaseWidget) tintOf(s tintKind) float64            // hover, pressed, header, emphasis
func (b *BaseWidget) px(v float64) float64                 // v * Density, for paddings and gaps
func (b *BaseWidget) borderW() float64
func (b *BaseWidget) focusW() float64
```

- **New roles** (`RoleSuccess`, `RoleWarning`, `RoleError`, `RoleInfo`, `RoleScrim`)
  are appended to `ColourRole`, not inserted, so existing values do not move.
  `SetColour` works on them like any other, so a single toast can be recoloured.
- **Palette fallbacks** for the nil new colours live in one function in `ui`
  (`paletteColour`): Success/Warning/Error/Info are the colours the toast code uses
  now, Scrim is the current literal. So a palette that never heard of them is
  unchanged.
- **Padding.** The package-level `var buttonPadding = geom.Insets{...}` style values
  stay as the *base*; every use in `MinSize` and `Draw` goes through
  `b.px(...)`/`b.insets(base)`. This touches most widget files (about twenty padding values) but is mechanical,
  and `px` returns its argument at Density 1, so the default look is bit-identical.
- **Border and focus** replace the literal `1` in outline strokes and
  `focusRingWidth` in `strokeFocusable`.
- **Motion.** `fadeTo` reads `Motion.Duration`; with `Motion.Reduced` it snaps
  (the path it already takes before a widget is mounted). Running `App.Animate`
  animations (toast slide, spinner) are application motion and are left alone.
- **Filled-button states** (`lighten 1.15` / `darken 0.8`) become the same tint
  mechanism as every other state, using `States.Hover` and `States.Pressed` with a
  fixed multiplier so a filled button reads slightly stronger than a flat row. This
  is the one place the *default* look can shift (the tint direction follows the
  base colour's luminance, where `lighten` always went up). It is checked against
  both presets before it is accepted; if it is visibly different, the old factors
  stay as defaults of two extra `StateTints` fields.

#### Spacing between widgets

`Theme.Spacing` is what the original design promised and never delivered. Layouts
are built before any theme exists (`ui.VBox(10)` runs at construction), so the gap
cannot be read then. The design is a sentinel resolved at layout time:

```go
const ThemeGap = -1                          // ui.VBox(ui.ThemeGap), ui.HBox(ui.ThemeGap)
```

`Container` already holds the tree context, so when it lays out a `Box` or `Grid`
whose spacing is negative it passes `Spacing * Density` in (through a small
unexported `spacingSetter` interface the layout may implement). Explicit numbers
keep working and are **not** scaled by Density: a caller who wrote `VBox(10)` asked
for 10. Only `ThemeGap` follows the theme. This is opt-in, so nothing changes for
existing layouts.

#### Things deliberately not in the model

- **Per-widget padding tables**, **a type scale / second font**, **radii per
  control kind**: more knobs than a v0.x needs. `CornerRadius` stays one value.
- **Colour for hover or pressed as an explicit palette entry.** States are amounts
  (principle 3). If an application needs an exact colour for one widget it already
  has `SetColour`.
- **OS light/dark and reduced-motion detection.** Both are opt-in bridges that
  would *set* a theme or `Motion.Reduced`; they are separate work (see ROADMAP).
- **`Container` background/border overrides following `SetTheme`.** Those are
  explicit values a caller set (called out in `App.SetTheme`'s doc). Making them
  role-based (`SetBackground(RoleSurface)`) is a possible follow-up, not part of
  this.

#### Build order (each step ships on its own)

1. `theme`: the new fields, `Default()` values, `HighContrast()`, `WithDensity`,
   `WithReducedMotion`, and a `withDefaults` applied by `App.SetTheme`/`NewApp` (the
   same place the font fallback already lives). Contrast tests for the three presets.
2. `ui`: new roles and fallbacks; `tintOf`, `borderW`, `focusW`; replace the tint
   constants and the literal widths; toast, scrim and shadow through roles.
3. `ui`: `px`/`insets` through every padding use (Density).
4. `ui`: `ThemeGap` and `spacingSetter`.
5. `ui`: motion duration and reduced motion in `fadeTo`.
6. The audit test (below) runs from step 2 and gains cases as widgets are converted.
7. `examples/theming`: a picker for Dark / Light / High contrast, a density slider,
   a reduced-motion checkbox, and a toast of each kind.

#### Tests

- **Contrast (`theme`).** A WCAG contrast function in the test, and for Dark, Light
  and HighContrast: Text on Background and on Surface >= 4.5 (HighContrast >= 7),
  OnPrimary on Primary >= 4.5, Accent against Surface >= 3 (the focus ring is a
  graphical object), TextMuted on Surface >= 3 for the presets that claim it, and
  each status colour against OnPrimary >= 4.5. A preset that fails is fixed, not
  excused.
- **Zero-value compatibility.** `Theme{Palette: DefaultPalette(), FontSize: 14,
  CornerRadius: 5}` (the shape callers have today) resolves to exactly
  `Default()`'s metrics, and a recorded frame of every widget is identical to a
  frame drawn with the explicit defaults.
- **Palette audit (`guitest`).** The mechanical check for "no hard-coded colours and
  nothing cached at construction". Build every widget once; draw it under theme A
  and under theme B, whose palettes share **no** colour; collect every colour used
  in each recording; the intersection must be empty (apart from fully transparent
  colours). Anything in the intersection is a literal baked into a widget, or a
  colour cached before `SetTheme`. This replaces eyeballing 30 widgets and is what
  would have caught the toast colours.
- **State audit.** For each widget with hover/pressed/focus states, draw rest and
  state under a theme with large `States` and assert the state differs from rest by
  at least the configured amount, and equals rest when the tints are zero-width.
- **Density.** At 0.8 and 1.25, `MinSize()` of Button, TextField, List, Table,
  Tabs and Menu scales their padding portion and leaves the text portion alone; at
  1 it is bit-identical to today. `ThemeGap` follows `Spacing * Density`, and an
  explicit spacing does not.
- **Reduced motion.** With `Reduced`, a hover reaches its end state in the same
  frame and no animation is left running (so the app can go idle).
- **Presets are distinguishable.** HighContrast's border and focus widths are larger
  than Default's, and a focused widget's ring is drawn at `FocusWidth`.

#### Decisions this design asks for

1. **Density as one scalar** (recommended) vs per-widget padding in the theme.
2. **`ThemeGap` sentinel** (recommended; opt-in) vs dropping `Theme.Spacing` from
   the model altogether.
3. **Filled-button states through the shared tint** (recommended, with the
   fallback above) vs keeping `lighten/darken` as two more theme fields.
4. **HighContrast in this round** (recommended: it is the proof the knobs work)
   vs later.
5. **Zero means default for the new metrics**, with no way to ask for a 0-width
   border except a transparent border colour.

### 2.3 Canvas capabilities (`render.Canvas`)

Current API: rects, rounded rects, lines, circles, text, images, clip stack.
Missing, to be added on the seam and implemented in `internal/ebiten` and
`guitest`:

- Gradients (linear, at least).
- Opacity / alpha layer.
- Blur (e.g. for shadows or modal backdrops).
- Dashed strokes.
- Per-corner radii.

### 2.4 Animation

`ui/animation.go` exists. Use it for:

- Hover and press transitions on Button, Checkbox, Radio, Slider, Tabs.
- Focus-ring fade.
- Redraw stays on demand (see framepace work): animations must `Invalidate`
  only while running.

### 2.5 Widget polish

- Focus rings consistent across all focusable widgets.
- Table zebra striping, row hover, header styling.
- Tab styling (active indicator, hover).
- Audit every widget `Draw` for hard-coded colours; route through theme roles.

### 2.6 Already planned elsewhere, relevant to UI

`RichText` / `[]TextSpan`, public `CanvasWidget` (done), a11y (deferred),
dirty-region redraw. Tracked in ROADMAP; do not duplicate.

### 2.7 Suggested order

1. Light + high-contrast themes and switcher (2.1).
2. Extend `Theme` with state colours, spacing, border width (2.2).
3. Hover/press transitions via animation (2.4).
4. Widget polish pass (2.5).
5. Canvas additions (2.3) as the above demand them.

### 2.8 Testing

Use `guitest` recorded drawing ops to assert theme colours reach each widget
after `SetTheme`, and that state colours apply on hover/press/focus. Not yet
verified against a real window.
