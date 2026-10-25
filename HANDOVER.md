# guie handover

Notes for picking the work up in a later session. Point Claude at this file ("read HANDOVER.md") to resume.
Written 2026-10-25, branch `theme`. Delete or untrack this file whenever it is stale; it is not part of the library.

## Working rules (from ~/.claude/CLAUDE.md)

- Plain ASCII only in anything written (no em/en dashes, smart quotes).
- Never `git commit`, `git add`/stage, push, reset --hard, amend, rebase, tag, `gh pr merge`. Read-only git and `git checkout -b` are fine. Leave changes unstaged, list changed files, give a suggested commit message. The owner commits everything.
- Repo is on C:, so building and running tests here is fine (the "never build under X:" rule does not apply).
- `go vet ./...` and `go test ./...` are authoritative. IDE diagnostics are often stale noise.

## What the project is

`guie`: Go GUI framework over Ebiten, module `github.com/kpfaulkner/guie`. Retained widget tree, `Widget` interface with `BaseWidget` embedding, `render.Canvas`/Driver seam, on-demand frame presenting, headless `guitest` harness. Read `design.md` (roadmap, decisions) and `INTERNALS.md` (how it works) first.

## Done in the recent sessions

- CanvasWidget (+ example, INTERNALS 13f).
- Text undo/redo for TextField/TextArea, `ReplaceRange` (editor replace-all is undoable).
- `EventDoubleClick`; double-click word select; double-click activation in List/Table/Tree.
- `App.ShowConfirm`.
- Window geometry persistence (example `windowstate`).
- Shared `ScrollBar` widget; horizontal scrolling in ScrollView, Table, TextArea (`SetWrap`), List, Tree.
- README rewrite, .gitignore, INTERNALS link; examples audited so each change is visible in one.
- Theme model, steps 1-5 and 7 (see below).
- Last change: `examples/comprehensive` has View > Theme: Dark / Light / High contrast. It rebuilds the whole UI on switch (explicit palette colours in cards). Vet and gofmt clean; ran and exited 0; NOT looked at visually.

## Theme model (design.md section 2.2 has the detail)

- Zero value = default (`WithDefaults`); old-style themes still work.
- States as tint amounts (`StateTints`), `Density` scalar, `ui.ThemeGap` (-1) resolved from `Theme.Spacing` x density, `Motion{Duration,Reduced}`, `NoShadows`, `HighContrast()`, status colours + `Scrim`, focus ring with a page-colour halo.
- Tests: `guitest/theme_audit_test.go` (two disjoint palettes; any colour drawn under both is hard-coded), `theme_density_test.go` (exact slope per widget), `theme_gap_test.go`, `theme_step2_test.go`, `ui/theme_defaults_test.go`, `ui/motion_test.go`, contrast tests in `theme/theme_test.go`.
- Only `examples/theming` switches themes freely (Dark/Light/High contrast/Ocean/Forest, density, spacing, reduce motion). `comprehensive` now also switches. Other examples use the default dark theme.

## Open items

Theme:
- Step 6 (the audit) is ongoing: it runs and grows; extend `auditCases()` when widgets are added.
- Never viewed in a real window: focus halo, HighContrast, filled-button hover shift, comprehensive's theme switch. Do a visual pass.
- Known contrast gaps recorded (not repainted) in `TestFocusRingStandsApartFromAFilledButton` for dark/light presets.
- Container background/border are explicit (not themed automatically).
- Built-in dialogs' own padding/gaps are not themed.
- Remaining literal widths: date picker selected-day ring, date field glyph, colour picker handle marker.
- No OS light/dark or reduced-motion detection. No `Compact()`/`Spacious()` presets.
- Optional: `-theme` flag for all examples.

Deferred by the owner:
- CI workflow and CONTRIBUTING (do "later on").
- Accessibility seam (kept in design.md as a later item).
- Later release items: App/Window shape, cursor shapes, canvas extras, etc. (see design.md roadmap and the "missing before v0.x" list).

## Housekeeping

- New untracked test files from the theme work (theme_audit_test.go, theme_density_test.go, theme_gap_test.go, theme_step2_test.go, motion_test.go, theme_defaults_test.go) were committed in "themes themes themes" per git log; re-check `git status` for anything new.
- Uncommitted now: `examples/comprehensive/main.go`, this file.
- Suggested commit message for the example: `comprehensive example: View menu to switch Dark / Light / High contrast`.

## Gotchas learned

- Widget `Invalidate()` only re-lays-out; `App.Invalidate` is the repaint call.
- guitest tests using `ui.WithTheme` lose the headless font: re-add `ui.WithFont(guitest.NewFont(...))` (helpers `densityOpts`, `themed`).
- Mutation-check new tests (break the code, confirm the test fails); several earlier tests passed vacuously.
- Bash heredocs/sed mangle apostrophes and `\n`; prefer Write/Edit.
- A full transcript of the earlier work is at `C:\Users\kenfa\.claude\projects\C--Users-kenfa-projects-guie\acaa7634-9b93-4c3e-a7c8-7f98275e9665.jsonl`.
