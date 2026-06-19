// Command editor is a small general-purpose text editor built with uiframework.
// It has a menu bar (File / Edit / Help): File ▸ Open/Save/Quit, Edit ▸
// Find/Replace, Help ▸ About. Open and the editing dialogs are custom modal
// panels (a label + text field(s) + OK/Cancel) shown via App.ShowModal; Find
// uses TextArea's selection API to highlight and scroll to a match.
//
// Run with: go run ./examples/editor
package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/kpfaulkner/uiframework/geom"
	"github.com/kpfaulkner/uiframework/ui"
)

// dialogPanel builds a styled modal panel (surface, border, rounded) with a
// title and a vertical layout for its contents.
func dialogPanel(app *ui.App, title string) *ui.Container {
	th := app.Theme()
	p := ui.NewContainer()
	p.SetBackground(th.Palette.Surface)
	p.SetBorder(th.Palette.Border, 1)
	p.SetCornerRadius(th.CornerRadius)
	p.SetLayout(ui.VBox(12))
	p.SetPadding(geom.UniformInsets(16))
	p.Add(ui.NewLabel(title))
	return p
}

// inputDialog shows a modal with one text field and OK/Cancel. OK (or Enter)
// calls onOK with the field's text.
func inputDialog(app *ui.App, title, placeholder, initial string, onOK func(string)) {
	panel := dialogPanel(app, title)
	field := ui.NewTextField(ui.Placeholder(placeholder))
	field.SetText(initial)
	panel.Add(field)

	var popup *ui.Popup
	submit := func() {
		v := field.Text()
		app.Close(popup)
		onOK(v)
	}
	field.OnSubmit(func(string) { submit() })

	cancel := ui.NewButton("Cancel")
	cancel.OnClick(func() { app.Close(popup) })
	ok := ui.NewButton("OK")
	ok.OnClick(submit)
	row := ui.NewContainer()
	row.SetLayout(ui.HBox(10))
	row.Add(cancel)
	row.Add(ok)
	panel.Add(row, ui.Align(geom.AlignEnd))

	popup = app.ShowModal(panel)
	field.RequestFocus()
}

// replaceDialog shows a modal with "find" and "replace with" fields.
func replaceDialog(app *ui.App, onReplace func(from, to string)) {
	panel := dialogPanel(app, "Replace")
	fromF := ui.NewTextField(ui.Placeholder("find"))
	toF := ui.NewTextField(ui.Placeholder("replace with"))
	panel.Add(fromF)
	panel.Add(toF)

	var popup *ui.Popup
	cancel := ui.NewButton("Cancel")
	cancel.OnClick(func() { app.Close(popup) })
	ok := ui.NewButton("Replace")
	ok.OnClick(func() {
		app.Close(popup)
		onReplace(fromF.Text(), toF.Text())
	})
	row := ui.NewContainer()
	row.SetLayout(ui.HBox(10))
	row.Add(cancel)
	row.Add(ok)
	panel.Add(row, ui.Align(geom.AlignEnd))

	popup = app.ShowModal(panel)
	fromF.RequestFocus()
}

func main() {
	app := ui.NewApp(
		ui.WithTitle("uiframework — editor"),
		ui.WithSize(820, 600),
	)

	area := ui.NewTextArea(
		ui.TextAreaWrap(),
		ui.TextAreaPlaceholder("Open a file (File ▸ Open) or start typing..."),
	)
	status := ui.NewLabel("Ready.")
	say := func(s string) { status.SetText(s) }

	currentPath := ""
	lastFind := ""

	saveTo := func(path string) {
		if err := os.WriteFile(path, []byte(area.Text()), 0o644); err != nil {
			app.ShowMessage("Save failed", err.Error())
			return
		}
		currentPath = path
		say("Saved " + path)
	}

	bar := ui.NewMenuBar()
	bar.AddMenu("File",
		ui.NewMenuItem("Open", func() {
			inputDialog(app, "Open file", "path to file", currentPath, func(path string) {
				data, err := os.ReadFile(path)
				if err != nil {
					app.ShowMessage("Open failed", err.Error())
					return
				}
				area.SetText(string(data))
				currentPath = path
				say("Opened " + path)
				area.RequestFocus()
			})
		}),
		ui.NewMenuItem("Save", func() {
			if currentPath == "" {
				inputDialog(app, "Save as", "path to file", "", saveTo)
			} else {
				saveTo(currentPath)
			}
		}),
		ui.NewMenuItem("Quit", func() { app.Quit() }),
	)
	bar.AddMenu("Edit",
		ui.NewMenuItem("Find", func() {
			inputDialog(app, "Find", "text to find", lastFind, func(q string) {
				lastFind = q
				_, ok := area.Find(q, area.CaretOffset())
				if !ok {
					_, ok = area.Find(q, 0) // wrap around
				}
				if ok {
					say(fmt.Sprintf("Found %q", q))
				} else {
					say(fmt.Sprintf("%q not found", q))
				}
				area.RequestFocus()
			})
		}),
		ui.NewMenuItem("Replace", func() {
			replaceDialog(app, func(from, to string) {
				if from == "" {
					return
				}
				n := strings.Count(area.Text(), from)
				area.SetText(strings.ReplaceAll(area.Text(), from, to))
				say(fmt.Sprintf("Replaced %d occurrence(s)", n))
				area.RequestFocus()
			})
		}),
	)
	bar.AddMenu("Help",
		ui.NewMenuItem("About", func() {
			app.ShowMessage("About", "A small text editor built with uiframework.")
		}),
	)

	// Editor area with a little margin around it.
	editorWrap := ui.NewContainer()
	editorWrap.SetLayout(ui.NewStack())
	editorWrap.SetPadding(geom.UniformInsets(8))
	editorWrap.Add(area)

	statusBar := ui.NewContainer()
	statusBar.SetLayout(ui.VBox(0))
	statusBar.SetPadding(geom.UniformInsets(6))
	statusBar.Add(status)

	root := ui.NewContainer()
	root.SetLayout(ui.VBox(0))
	root.Add(bar)
	root.Add(editorWrap, ui.Weight(1))
	root.Add(statusBar)

	app.SetContent(root)
	area.RequestFocus()

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
