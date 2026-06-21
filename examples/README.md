# Examples

Each subdirectory is a standalone program. Run one with `go run` from the repo
root:

```sh
go run ./examples/hello      # smallest program: a centered label
go run ./examples/layouts    # VBox / HBox / Grid / Stack, weights, alignment, panels
go run ./examples/widgets    # Label + Button, click counter, runtime enable/disable
go run ./examples/events     # focus (Tab/Shift+Tab), keyboard activation, event bus
go run ./examples/canvas     # a custom widget drawn with the Canvas primitives
go run ./examples/controls   # TextField, Checkbox, RadioGroup, Slider, ProgressBar
go run ./examples/scroll     # ScrollView over a tall list (wheel + draggable thumb)
go run ./examples/showcase   # MenuBar + List + DropdownCombo with popups
go run ./examples/dialog     # modal dialogs (scrim, blocked background, Esc to close)
go run ./examples/textarea   # multi-line TextArea: newlines, caret nav, wheel scroll
go run ./examples/tabs       # TabContainer: switch panes via the tab strip
go run ./examples/table      # Table: header + weighted columns, selectable scrollable rows
go run ./examples/fonts      # adjust font size at runtime (App.SetFont) + per-widget font
go run ./examples/tooltips   # hover tooltips (SetTooltip) with a rest delay
go run ./examples/splitter   # SplitPane: draggable dividers, nested splits
go run ./examples/colors     # per-widget color overrides (SetColor / ColorOf, roles)
go run ./examples/images     # Image widget (scaled) + image buttons (ui.LoadImage)
go run ./examples/editor     # text editor: menus, open/save, find/replace, modals
go run ./examples/paint      # freehand drawing: mouse capture, multi-button, custom widget
go run ./examples/clipboard  # OS clipboard: system-wide copy/paste via ui.WithClipboard
go run ./examples/dragdrop   # drag-and-drop: move items between panels, ghost + highlight
go run ./examples/tree       # Tree (expand/collapse/keyboard) + Toast notifications
```

What each one exercises:

| Example   | Covers                                                                       |
|-----------|------------------------------------------------------------------------------|
| `hello`   | `App`, options (`WithTitle`/`WithSize`), `Container`, `Stack`, `Label`, `Run` |
| `layouts` | Nested containers, `VBox`/`HBox`/`Grid`, per-child `Weight`/`Align`, padding, themed colors |
| `widgets` | `Label.SetText`, `Button` + `OnClick`, `SetEnabled` toggling at runtime, pointer hover/press |
| `events`  | Focus traversal, Space/Enter activation, `app.Events().Subscribe` global listener |
| `canvas`  | Custom widget via `BaseWidget`, `Canvas` primitives (`FillRect`, `StrokeRect`, `DrawLine`, `DrawText`, `MeasureText`) |
| `controls`| `TextField` (typing, caret), `Checkbox`, `RadioButton`/`RadioGroup`, `Slider` → `ProgressBar` |
| `scroll`  | `ScrollView`: wheel + draggable thumb, interactive widgets inside the viewport |
| `showcase`| `MenuBar` + `Menu`/`MenuItem`, `List` (selection + scroll), `DropdownCombo`, overlay popups |
| `dialog`  | Modal dialogs via `App.ShowMessage`/`ShowModal`: scrim, blocked background, button + Esc dismissal |
| `textarea`| Multi-line `TextArea`: typing, newlines, soft word-wrap, selection, cut/copy/paste, scrolling |
| `tabs`    | `TabContainer`: tab strip, click/Left-Right to switch, panes keep their state |
| `table`   | `Table`: header row, weighted columns, selectable + scrollable body rows |
| `fonts`   | Runtime font sizing via `App.SetFont` + `ui.DefaultFont`, plus per-widget `SetFont` |
| `tooltips`| Hover tooltips via `SetTooltip`, with a rest delay and on-screen clamping |
| `splitter`| `SplitPane`/`HSplit`/`VSplit`: draggable dividers, ratio + min sizes, nested |
| `colors`  | Per-widget color overrides via `SetColor`/`ColorOf` and `ColorRole`s |
| `images`  | `Image` widget (fit modes) + image buttons via `ui.LoadImage`/`ButtonImage` |
| `editor`  | A text editor: `MenuBar`, file open/save, find/replace, custom modal dialogs, `TextArea` selection API |
| `paint`   | Freehand drawing: custom widget reading `ev.Pos`, pointer capture, left-draw/right-erase, color/brush controls |
| `clipboard`| OS-backed clipboard via the opt-in `clipboard` package + `ui.WithClipboard`: copy/paste with other apps |
| `dragdrop`| Drag-and-drop: `SetDragSource`/`SetDropTarget`, `OnDrop`/`OnDragEnter`/`OnDragLeave`, ghost + drop highlight, `Container.Remove` to re-parent |
| `tree`    | `Tree`/`TreeNode`: expand/collapse, click + keyboard nav, `OnSelect`/`OnActivate`; `App.ShowToast` with info/success/warning/error kinds |

Note: application code never imports EBiten. Widgets and apps talk only to the
`ui`, `geom`, `render` and `theme` packages; EBiten lives behind the backend.
