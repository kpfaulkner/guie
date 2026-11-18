//go:build darwin

package ebitenbackend

import (
	"bytes"
	"image"
	"image/png"
	"runtime"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

// setDockIcon sets the application's Dock icon to the largest of icons, since
// macOS windows have no icon of their own and SetWindowIcon does nothing there.
// A program inside an .app bundle already has the bundle's icon; a bare binary
// started from a shell has the generic one until this runs.
//
// It goes through the Objective-C runtime with purego, as Ebitengine does, so it
// needs no cgo. It runs on the main thread, before the game loop: Ebitengine
// locks the main goroutine to it at init, and Driver.Run is called from there.
// Any failure leaves the default icon; it is never worth stopping the app for.
func setDockIcon(icons []image.Image) {
	defer func() { recover() }()
	img := largestIcon(icons)
	if img == nil {
		return
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return
	}
	if _, err := purego.Dlopen("/System/Library/Frameworks/AppKit.framework/AppKit", purego.RTLD_GLOBAL|purego.RTLD_LAZY); err != nil {
		return
	}
	b := buf.Bytes()
	data := objc.ID(objc.GetClass("NSData")).Send(objc.RegisterName("dataWithBytes:length:"), &b[0], uint64(len(b)))
	runtime.KeepAlive(b)
	if data == 0 {
		return
	}
	nsImage := objc.ID(objc.GetClass("NSImage")).Send(objc.RegisterName("alloc")).Send(objc.RegisterName("initWithData:"), data)
	if nsImage == 0 {
		return
	}
	app := objc.ID(objc.GetClass("NSApplication")).Send(objc.RegisterName("sharedApplication"))
	app.Send(objc.RegisterName("setApplicationIconImage:"), nsImage)
}
