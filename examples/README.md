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

Note: application code never imports EBiten. Widgets and apps talk only to the
`ui`, `geom`, `render` and `theme` packages; EBiten lives behind the backend.
