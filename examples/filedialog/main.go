// Command filedialog demonstrates the in-toolkit file dialog. "Open..." browses
// the real filesystem with a filter dropdown; "Open several..." allows
// Ctrl/Cmd+click, Shift+click and Ctrl/Cmd+A to pick more than one file;
// "Save..." suggests a name, adds
// .txt if you leave the extension off and asks before replacing a file (it only
// reports the chosen path, nothing is written); "Choose folder..." picks a
// directory; "Open (in-memory)..." browses an io/fs.FS built in code, which is
// also how a dialog would browse an embed.FS or run under WASM. The chosen path
// is shown below the buttons.
//
// Every dialog also has a places pane (home folders and drives), a "Show hidden"
// checkbox, sortable columns, and type-ahead: type a few letters in the listing
// to jump to a name. The window lists which keys work where; the multiple
// selection keys (Ctrl+click, Shift+click, Space and so on) apply to "Open
// several..." only.
//
// Run with: go run ./examples/filedialog
package main

import (
	"fmt"
	"log"
	"strings"
	"testing/fstest"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/ui"
)

func main() {
	app := ui.NewApp(
		ui.WithTitle("guie - file dialog"),
		ui.WithSize(820, 480),
	)

	root := ui.NewContainer()
	root.SetLayout(ui.VBox(14))
	root.SetPadding(geom.UniformInsets(20))
	root.Add(ui.NewLabel("File dialog"))

	result := ui.NewLabel("Chosen: (nothing yet)")
	report := func(paths []string, ok bool) {
		if !ok {
			result.SetText("Chosen: (cancelled)")
			return
		}
		result.SetText(fmt.Sprintf("Chosen (%d): %s", len(paths), strings.Join(paths, ", ")))
	}

	row := ui.NewContainer()
	row.SetLayout(ui.HBox(10))

	open := ui.NewButton("Open...")
	open.OnClick(func() {
		app.ShowFileDialog(ui.FileOpen, report,
			ui.FileDialogTitle("Open a file"),
			ui.FileDialogFilters(
				ui.FileFilter{Name: "All files"},
				ui.FileFilter{Name: "Go source", Patterns: []string{"*.go"}},
				ui.FileFilter{Name: "Images", Patterns: []string{"*.png", "*.jpg", "*.jpeg", "*.gif"}},
				ui.FileFilter{Name: "Text", Patterns: []string{"*.txt", "*.md"}},
			),
		)
	})
	row.Add(open)

	several := ui.NewButton("Open several...")
	several.OnClick(func() {
		app.ShowFileDialog(ui.FileOpen, report,
			ui.FileDialogTitle("Open files"),
			ui.FileDialogMultiple(),
		)
	})
	row.Add(several)

	save := ui.NewButton("Save...")
	save.OnClick(func() {
		app.ShowFileDialog(ui.FileSave, report,
			ui.FileDialogName("untitled"),
			ui.FileDialogDefaultExt("txt"),
			ui.FileDialogFilters(
				ui.FileFilter{Name: "Text", Patterns: []string{"*.txt", "*.md"}},
				ui.FileFilter{Name: "All files"},
			),
		)
	})
	row.Add(save)

	folder := ui.NewButton("Choose folder...")
	folder.OnClick(func() {
		app.ShowFileDialog(ui.FileSelectDir, report)
	})
	row.Add(folder)

	mem := ui.NewButton("Open (in-memory)...")
	mem.OnClick(func() {
		fsys := fstest.MapFS{
			"readme.md":         {Data: []byte("# hello")},
			"notes.txt":         {Data: []byte("some notes")},
			"photos/beach.png":  {Data: make([]byte, 2048)},
			"photos/hills.jpg":  {Data: make([]byte, 4096)},
			"src/main.go":       {Data: []byte("package main")},
			"src/internal/x.go": {Data: []byte("package x")},
		}
		app.ShowFileDialog(ui.FileOpen, report,
			ui.FileDialogFS(ui.FSFileSystem(fsys)),
			ui.FileDialogTitle("Open from memory"),
		)
	})
	row.Add(mem)

	root.Add(row)

	// Which keys work where. The multiple-selection keys apply only to "Open
	// several...": the other dialogs choose one thing, so they do not use them.
	muted := ui.LabelColour(app.Theme().Palette.TextMuted)
	root.Add(ui.NewLabel(strings.Join([]string{
		"In every dialog:",
		"  Type letters in the list to jump to a name. Backspace or Alt+Up goes to the parent folder.",
		"  Click a column header to sort. Enter opens a folder or chooses a file.",
		"",
		"In \"Open several...\" only (Cmd instead of Ctrl on macOS):",
		"  Ctrl+click toggles a row. Shift+click selects a range. Ctrl+A selects everything.",
		"  Ctrl+Up/Down moves the outlined row without selecting; Space (or Ctrl+Space) toggles it.",
		"  Ctrl+Shift+Up/Down adds a range. Enter or Open takes everything selected.",
	}, "\n"), muted))
	root.Add(result)

	app.SetContent(root)
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
