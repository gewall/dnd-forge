package imaging

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"

	"github.com/gen2brain/webp"
)

func pngOf(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := range w {
		img.Set(x, x%h, color.RGBA{R: 200, G: 40, B: 40, A: 255})
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestEncodeScalesDownToWebP(t *testing.T) {
	enc := New(80)
	res, err := enc.Encode(context.Background(), pngOf(t, 400, 200),
		Size{MaxWidth: 100, MaxHeight: 100},
		Size{MaxWidth: 1000, MaxHeight: 1000},
	)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	tests := []struct{ w, h int }{{100, 50}, {400, 200}} // keeps ratio, never upscales
	for i, want := range tests {
		got := res[i]
		if got.Width != want.w || got.Height != want.h || got.ContentType != "image/webp" {
			t.Errorf("result %d = %dx%d %s, want %dx%d image/webp", i, got.Width, got.Height, got.ContentType, want.w, want.h)
		}
		cfg, err := webp.DecodeConfig(bytes.NewReader(got.Data))
		if err != nil || cfg.Width != want.w || cfg.Height != want.h {
			t.Errorf("result %d is not a %dx%d WebP: %+v, %v", i, want.w, want.h, cfg, err)
		}
	}
}

func TestEncodeRejects(t *testing.T) {
	enc := New(80)
	if _, err := enc.Encode(context.Background(), []byte("bukan gambar"), Size{100, 100}); !errors.Is(err, ErrUnsupported) {
		t.Errorf("garbage: err = %v, want ErrUnsupported", err)
	}
	// 5000x5000 = 25 MP, over the limit. PNG of a blank image stays small.
	if _, err := enc.Encode(context.Background(), pngOf(t, 5000, 5000), Size{100, 100}); !errors.Is(err, ErrTooLarge) {
		t.Errorf("huge: err = %v, want ErrTooLarge", err)
	}
}

func TestReadLimited(t *testing.T) {
	if _, err := ReadLimited(strings.NewReader("12345"), 4); !errors.Is(err, ErrTooLarge) {
		t.Errorf("over limit: err = %v, want ErrTooLarge", err)
	}
	if b, err := ReadLimited(strings.NewReader("1234"), 4); err != nil || string(b) != "1234" {
		t.Errorf("at limit: %q, %v", b, err)
	}
}
