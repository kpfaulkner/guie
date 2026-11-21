// Package ebitenbackend implements the render package's backend interfaces
// (Canvas, FontFace, Image and Driver) on top of EBiten. It is the only
// package in the framework that imports EBiten; everything above it depends
// solely on the render and geom packages.
package ebitenbackend

import (
	"bytes"
	"os"
	"sync"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
	"golang.org/x/image/font/gofont/goregular"
)

// fontFace is the backend's concrete render.FontFace. It wraps an EBiten text
// face and caches the framework-neutral metrics.
type fontFace struct {
	face    *text.GoTextFace
	metrics render.FontMetrics

	// The face at the scale it was last drawn at, for scaled displays. Kept so
	// drawing text does not make a new face for every call. UI goroutine only.
	scaled  *text.GoTextFace
	scaleOf float64
}

// scaledFace is the face for drawing at scale times its size. The copy shares
// the font source; it is made again only when the scale changes.
func (f *fontFace) scaledFace(scale float64) *text.GoTextFace {
	if scale == 1 {
		return f.face
	}
	if f.scaled == nil || f.scaleOf != scale {
		s := *f.face
		s.Size = f.face.Size * scale
		f.scaled, f.scaleOf = &s, scale
	}
	return f.scaled
}

// Metrics returns the vertical extents of the face.
func (f *fontFace) Metrics() render.FontMetrics { return f.metrics }

// Measure returns the rendered size of s in this face.
func (f *fontFace) Measure(s string) geom.Size {
	w, h := text.Measure(s, f.face, f.metrics.LineHeight)
	return geom.Size{W: w, H: h}
}

var (
	defaultSourceOnce sync.Once
	defaultSource     *text.GoTextFaceSource
	defaultSourceErr  error
)

// goRegularSource lazily parses the bundled Go font once and caches it.
func goRegularSource() (*text.GoTextFaceSource, error) {
	defaultSourceOnce.Do(func() {
		defaultSource, defaultSourceErr = text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	})
	return defaultSource, defaultSourceErr
}

// newFace builds a render.FontFace from a text source at the given point size.
func newFace(src *text.GoTextFaceSource, size float64) render.FontFace {
	gf := &text.GoTextFace{Source: src, Size: size}
	m := gf.Metrics()
	return &fontFace{
		face: gf,
		metrics: render.FontMetrics{
			Ascent:     m.HAscent,
			Descent:    m.HDescent,
			LineHeight: m.HAscent + m.HDescent + m.HLineGap,
		},
	}
}

// DefaultFont returns the bundled Go font at the given size as a render.FontFace.
// It panics only if the embedded font asset fails to parse, which would be a
// build-time corruption rather than a runtime condition.
func DefaultFont(size float64) render.FontFace {
	src, err := goRegularSource()
	if err != nil {
		panic("ebitenbackend: parsing bundled font: " + err.Error())
	}
	return newFace(src, size)
}

// NewFontFace parses a TrueType/OpenType font from ttf and returns a
// render.FontFace at the given size.
func NewFontFace(ttf []byte, size float64) (render.FontFace, error) {
	src, err := text.NewGoTextFaceSource(bytes.NewReader(ttf))
	if err != nil {
		return nil, err
	}
	return newFace(src, size), nil
}

// fileSources holds the font files loaded so far, parsed once each: a file is
// read and parsed the first time a face is asked of it, and every later face,
// at any size, shares that source and its glyph caches. A file that failed is
// remembered too, so it is not read again.
var fileSources struct {
	mu sync.Mutex
	m  map[string]fileSource
}

type fileSource struct {
	src *text.GoTextFaceSource
	err error
}

// LoadFontFile returns a face of the font file at path at the given size. The
// file is read and parsed once; a file changed on disk after that is not seen
// until the program restarts.
func LoadFontFile(path string, size float64) (render.FontFace, error) {
	fileSources.mu.Lock()
	defer fileSources.mu.Unlock()
	fs, ok := fileSources.m[path]
	if !ok {
		data, err := os.ReadFile(path)
		if err == nil {
			fs.src, err = text.NewGoTextFaceSource(bytes.NewReader(data))
		}
		fs.err = err
		if fileSources.m == nil {
			fileSources.m = make(map[string]fileSource)
		}
		fileSources.m[path] = fs
	}
	if fs.err != nil {
		return nil, fs.err
	}
	return newFace(fs.src, size), nil
}
