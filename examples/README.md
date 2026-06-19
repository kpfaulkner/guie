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

Note: application code never imports EBiten. Widgets and apps talk only to the
`ui`, `geom`, `render` and `theme` packages; EBiten lives behind the backend.
