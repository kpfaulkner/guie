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
- **TextField** — single-line text entry (uses `TextInput`/focus/caret).
- **ScrollBar** + **ScrollView** — clip + offset content; vertical & horizontal.

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
- Clipboard and IME support for text fields. *(Done for clipboard — an
  OS-backed clipboard ships in the opt-in `clipboard` package, injected via
  `ui.WithClipboard`; see `examples/clipboard`. IME: the framework-side seam,
  events, preedit rendering and tests are built — see IME below; the EBiten last
  mile is blocked upstream.)*
- Animation/transition primitives (timeline vs per-frame). *(Done — per-frame
  hook `App.OnFrame` plus `App.Animate`/`App.Tween` with easings; see
  `internals.md` §11a and `examples/animation`.)*
- Persistence of window size/position.
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

- **OS clipboard integration.** *(Done.)* The default `Clipboard` is still the
  in-process `memClipboard`, but apps can now opt into system-wide copy/paste by
  passing `ui.WithClipboard(cb)` where `cb` comes from the new `clipboard`
  package (`clipboard.New()`). It is a separate, opt-in package so the core
  `ui`/`render` packages stay dependency-free; it's backed by
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

- **CGo is required on Linux.** Unlike Windows/macOS, there is no CGo-free path:
  `CGO_ENABLED=0` fails because EBiten's Linux backend pulls in GLFW (OpenGL /
  X11) via CGo. So a C compiler and the target's dev headers are mandatory.
- **Build natively on the Pi (recommended).** Install Go for arm64, then the
  EBiten Linux build deps via apt (`libgl1-mesa-dev xorg-dev libasound2-dev
  libxcursor-dev libxi-dev libxinerama-dev libxrandr-dev libxxf86vm-dev`), then
  `go build`. CGo "just works" because the compiler and headers are local.
- **Cross-compiling is fiddly.** `GOOS=linux GOARCH=arm64 CGO_ENABLED=1` needs an
  ARM cross C toolchain (e.g. `CC="zig cc -target aarch64-linux-gnu"`) *plus* a
  sysroot of the Pi's X11/GL/ALSA headers. Doable for CI, not worth it for a
  one-off — prefer the native build.
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
16. **Accessibility hooks** — expose roles/labels/state for assistive technology.
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
| Accessibility (seam) | Add `Role` / `AccessibleName` / `AccessibleState` to `BaseWidget` + a `SemanticNode` tree-walk. **Annotate simple widgets now** (Button, Label, Checkbox, Radio, TextField); complex self-drawing widgets (List, Tree, Table) and OS bridges (UIAutomation / AT-SPI / NSAccessibility) **deferred** (like IME). Assert the semantic tree headlessly via `guitest`. |
| i18n (seam-safety) | Ensure text layout does **not** assume `x-advance == Σ per-rune widths`, so a future bidi pass can land without an API break. (Translations/locale formatting itself is additive — see below.) |
| Custom-draw widget | Public, blessed **`CanvasWidget`** — a thin `BaseWidget` wrapper taking a `func(c render.Canvas, bounds geom.Rect)` draw callback + an input callback. The escape hatch for charts, viz, diagram/drawing surfaces, game HUDs. The `render.Canvas` seam already exists. |
| Theming | Ship a **dark theme + runtime theme switch** now; verify every widget reads `theme` in `Draw` (no colours cached at construction) — the switch is a correctness check on the theming architecture being frozen. OS light/dark **auto-detect deferred** (opt-in bridge). |
| Multi-window | **Lock the `App`/`Window` API shape** so it doesn't bake in a single-window assumption (e.g. `App.MainWindow()` singleton); **implement lazily** when a real app needs it. Inherently desktop-only (no WASM). |
| Rich text | Reserve an attributed-text / **`[]TextSpan`** type and ship a basic **`RichText`** widget (multi-style spans + clickable links). Keep plain `Label` simple/fast. **Markdown deferred** (layer on top / userland). |
| Text undo | Built-in **undo/redo inside `TextField`/`TextArea`** (a correctness expectation of the widget). App-level command history stays userland. |
| Window geometry | Small helper to persist/restore window **size + position** (mostly non-OS-specific; "which monitor" restore needs backend help later). |
| OS file drops | `InputState.DroppedFiles` (`fs.FS`) + `EventFileDrop` + `OnFileDrop` — public API on the seam, the widget and the event set, so the shape must land pre-freeze. Small: EBiten already surfaces the drop. Spec in *DRAG-AND-DROP → OS file drops*. |

### OSS-release milestone (additive; around/just after release)

| Feature | Decision / scope |
|---|---|
| WASM target | **Supported + advertised.** Whole repo builds clean for `GOOS=js GOARCH=wasm` today (verified). Guard OS opt-in packages (clipboard, dialogs) behind build tags. Browser runtime smoke-test + docs-site playground to follow. |
| In-app dialogs | `MessageBox` / `Confirm` + an **in-surface file chooser** built from existing List/Tree/TextField. Pure `ui`, cross-platform incl. WASM. |
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

### Out of scope / userland

Form validation (compose a `Label` under the field) · app-level undo/command
history · markdown parsing · app preferences / recent-files storage · charts &
data-viz (build on `CanvasWidget`) · printing / PDF export · video / media
playback · responsive layout breakpoints.
