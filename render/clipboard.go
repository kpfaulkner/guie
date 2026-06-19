package render

// Clipboard provides plain-text clipboard access for text widgets. The
// framework ships a simple in-process implementation; applications that want
// real OS clipboard integration can supply their own via ui.WithClipboard
// (e.g. wrapping a platform clipboard library), keeping the core dependency-free
// and cross-platform.
type Clipboard interface {
	// ReadText returns the current clipboard text (empty if none).
	ReadText() string
	// WriteText replaces the clipboard text.
	WriteText(s string)
}
