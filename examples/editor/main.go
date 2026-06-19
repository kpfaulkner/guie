// Command editor is a small general-purpose text editor built with uiframework.
// It has a menu bar (File / Edit / Help) and a flat icon toolbar. File ▸
// Open/Save/Quit, Edit ▸ Find/Replace, Help ▸ About. Open and the editing
// dialogs are custom modal panels (a label + text field(s) + OK/Cancel) shown
// via App.ShowModal; Find uses TextArea's selection API to highlight and scroll
// to a match. Toolbar icons are generated in memory and loaded via
// ui.LoadImageBytes.
//
// Run with: go run ./examples/editor
package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
	"strings"

	"github.com/kpfaulkner/uiframework/geom"
	"github.com/kpfaulkner/uiframework/render"
	"github.com/kpfaulkner/uiframework/ui"
)

// --- tiny icon rasterizer (so the example needs no asset files) ---

const iconSize = 20

func makeIcon(draw func(set func(x, y int))) render.Image {
	img := image.NewRGBA(image.Rect(0, 0, iconSize, iconSize))
	col := color.RGBA{R: 0xea, G: 0xea, B: 0xf2, A: 0xff}
	set := func(x, y int) {
		if x >= 0 && x < iconSize && y >= 0 && y < iconSize {
			img.SetRGBA(x, y, col)
		}
	}
	draw(set)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		log.Fatal(err)
	}
	im, err := ui.LoadImageBytes(buf.Bytes())
	if err != nil {
		log.Fatal(err)
	}
	return im
}

func iconLine(set func(int, int), x0, y0, x1, y1 int) {
	dx := abs(x1 - x0)
	dy := -abs(y1 - y0)
	sx, sy := 1, 1
	if x0 > x1 {
		sx = -1
	}
	if y0 > y1 {
		sy = -1
	}
	err := dx + dy
	for {
		set(x0, y0)
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func iconRect(set func(int, int), x0, y0, x1, y1 int) {
	iconLine(set, x0, y0, x1, y0)
	iconLine(set, x1, y0, x1, y1)
	iconLine(set, x1, y1, x0, y1)
	iconLine(set, x0, y1, x0, y0)
}

func iconCircle(set func(int, int), cx, cy, r int) {
	x, y, err := r, 0, 0
	for x >= y {
		for _, p := range [][2]int{{x, y}, {y, x}, {-x, y}, {-y, x}, {-x, -y}, {-y, -x}, {x, -y}, {y, -x}} {
			set(cx+p[0], cy+p[1])
		}
		y++
		if err <= 0 {
			err += 2*y + 1
		}
		if err > 0 {
			x--
			err -= 2*x + 1
		}
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func openIcon() render.Image { // a folder
	return makeIcon(func(set func(int, int)) {
		iconRect(set, 3, 8, 17, 16)
		iconLine(set, 3, 8, 3, 6)
		iconLine(set, 3, 6, 8, 6)
		iconLine(set, 8, 6, 10, 8)
	})
}

func saveIcon() render.Image { // a floppy disk
	return makeIcon(func(set func(int, int)) {
		iconRect(set, 3, 3, 16, 16)
		iconRect(set, 6, 3, 12, 7)
		iconRect(set, 5, 11, 14, 16)
	})
}

func findIcon() render.Image { // a magnifier
	return makeIcon(func(set func(int, int)) {
		iconCircle(set, 8, 8, 5)
		iconLine(set, 12, 12, 17, 17)
		iconLine(set, 13, 12, 17, 16)
	})
}

// --- dialogs ---

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
		ui.WithFontSize(18),
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

	// Actions, shared by the menu and the toolbar.
	openFile := func() {
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
	}
	saveFile := func() {
		if currentPath == "" {
			inputDialog(app, "Save as", "path to file", "", saveTo)
		} else {
			saveTo(currentPath)
		}
	}
	findText := func() {
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
	}
	replaceText := func() {
		replaceDialog(app, func(from, to string) {
			if from == "" {
				return
			}
			n := strings.Count(area.Text(), from)
			area.SetText(strings.ReplaceAll(area.Text(), from, to))
			say(fmt.Sprintf("Replaced %d occurrence(s)", n))
			area.RequestFocus()
		})
	}

	bar := ui.NewMenuBar()
	bar.AddMenu("File",
		ui.NewMenuItem("Open", openFile),
		ui.NewMenuItem("Save", saveFile),
		ui.NewMenuItem("Quit", func() { app.Quit() }),
	)
	bar.AddMenu("Edit",
		ui.NewMenuItem("Find", findText),
		ui.NewMenuItem("Replace", replaceText),
	)
	bar.AddMenu("Help",
		ui.NewMenuItem("About", func() {
			app.ShowMessage("About", "A small text editor built with uiframework.")
		}),
	)

	// Flat icon toolbar under the menu bar.
	toolBtn := func(img render.Image, tip string, fn func()) *ui.Button {
		b := ui.NewButton("", ui.ButtonImage(img), ui.ButtonFlat())
		b.SetTooltip(tip)
		b.OnClick(fn)
		return b
	}
	toolbar := ui.NewContainer()
	toolbar.SetBackground(app.Theme().Palette.Surface)
	toolbar.SetLayout(ui.HBox(4))
	toolbar.SetPadding(geom.UniformInsets(5))
	toolbar.Add(toolBtn(openIcon(), "Open", openFile))
	toolbar.Add(toolBtn(saveIcon(), "Save", saveFile))
	toolbar.Add(toolBtn(findIcon(), "Find", findText))

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
	root.Add(toolbar)
	root.Add(editorWrap, ui.Weight(1))
	root.Add(statusBar)

	app.SetContent(root)
	area.RequestFocus()

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
