// Package imaging decodes uploaded images, scales them down and re-encodes
// them as WebP. It is pure Go (WebP via WASM), so the API builds without cgo.
package imaging

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/jpeg" // register decoders
	_ "image/png"
	"io"

	"github.com/gen2brain/webp"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"

	"github.com/gewall/dnd-forge/api/internal/apperr"
)

var (
	ErrUnsupported = apperr.New(apperr.Invalid, "format gambar harus JPEG, PNG, atau WebP")
	ErrTooLarge    = apperr.New(apperr.Invalid, "gambar terlalu besar")
)

// maxPixels bounds the decoded size: 24 MP is ~96 MB as RGBA, which fits the
// container's 256 MB limit only because Encoder handles one image at a time.
const maxPixels = 24_000_000

// Size is the bounding box of an output image.
type Size struct {
	MaxWidth  int
	MaxHeight int
}

// Result is an encoded image.
type Result struct {
	Data        []byte
	Width       int
	Height      int
	ContentType string
}

// Encoder converts images to WebP. The zero value is not usable; use New.
type Encoder struct {
	quality int
	sem     chan struct{} // serialises work: images are large in memory
}

// New returns an Encoder with the given WebP quality (0–100).
func New(quality int) *Encoder {
	return &Encoder{quality: quality, sem: make(chan struct{}, 1)}
}

// Encode decodes data and returns one WebP per requested size, scaled down
// (never up) to fit each box while keeping the aspect ratio.
func (e *Encoder) Encode(ctx context.Context, data []byte, sizes ...Size) ([]Result, error) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, ErrUnsupported
	}
	if cfg.Width*cfg.Height > maxPixels {
		return nil, ErrTooLarge
	}

	select {
	case e.sem <- struct{}{}:
		defer func() { <-e.sem }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, ErrUnsupported
	}
	out := make([]Result, 0, len(sizes))
	for _, s := range sizes {
		img := fit(src, s)
		var buf bytes.Buffer
		if err := webp.Encode(&buf, img, webp.Options{Quality: e.quality, Method: 4}); err != nil {
			return nil, fmt.Errorf("imaging: encode webp: %w", err)
		}
		b := img.Bounds()
		out = append(out, Result{Data: buf.Bytes(), Width: b.Dx(), Height: b.Dy(), ContentType: "image/webp"})
	}
	return out, nil
}

// ReadLimited reads at most limit bytes from r and reports ErrTooLarge if
// there is more.
func ReadLimited(r io.Reader, limit int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, fmt.Errorf("imaging: read: %w", err)
	}
	if int64(len(data)) > limit {
		return nil, ErrTooLarge
	}
	return data, nil
}

func fit(src image.Image, s Size) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	scale := min(1, float64(s.MaxWidth)/float64(w), float64(s.MaxHeight)/float64(h))
	if scale >= 1 {
		return src
	}
	dst := image.NewRGBA(image.Rect(0, 0, max(1, int(float64(w)*scale)), max(1, int(float64(h)*scale))))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, b, draw.Over, nil)
	return dst
}
