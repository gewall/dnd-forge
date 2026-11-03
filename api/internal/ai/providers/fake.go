package providers

import (
	"bytes"
	"context"
	"hash/fnv"
	"image"
	"image/color"
	"image/png"
	"strings"
	"time"

	"github.com/gewall/dnd-forge/api/internal/ai"
)

// Fake draws a gradient whose colours derive from the prompt and writes a
// fixed story. It is for development and tests: no network, no cost.
// Prompts containing "#reject" are rejected, to exercise the moderation
// path.
type Fake struct {
	Delay time.Duration
}

func (Fake) Type() ai.ProviderType { return ai.Fake }

func (f Fake) Generate(ctx context.Context, in ai.GenerateInput) (ai.GenerateOutput, error) {
	if strings.Contains(in.Prompt, "#reject") {
		return ai.GenerateOutput{}, ai.ErrRejected
	}
	select {
	case <-time.After(f.Delay):
	case <-ctx.Done():
		return ai.GenerateOutput{}, ctx.Err()
	}

	w, h := 512, 512
	switch in.AspectRatio {
	case ai.Portrait:
		w, h = 384, 512
	case ai.Landscape:
		w, h = 512, 384
	}
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(in.Prompt))
	seed := hash.Sum32()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.Set(x, y, color.RGBA{
				R: channel(int(seed) + x),
				G: channel(int(seed>>8) + y),
				B: channel(int(seed>>16) + x + y),
				A: 255,
			})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return ai.GenerateOutput{}, err
	}
	return ai.GenerateOutput{Data: buf.Bytes(), MIMEType: "image/png"}, nil
}

// Complete returns a short fixed story that quotes the prompt's first line.
func (f Fake) Complete(ctx context.Context, in ai.TextInput) (string, error) {
	if strings.Contains(in.Prompt, "#reject") {
		return "", ai.ErrRejected
	}
	select {
	case <-time.After(f.Delay):
	case <-ctx.Done():
		return "", ctx.Err()
	}
	first, _, _ := strings.Cut(strings.TrimSpace(in.Prompt), "\n")
	return "Lore palsu dari provider fake.\n\nPermintaan: " + first, nil
}

func (Fake) TestConnection(context.Context, string) error { return nil }

// channel wraps n into a colour channel value.
func channel(n int) uint8 {
	return uint8(n % 256) //nolint:gosec // G115: n is never negative, so % 256 keeps it in range
}
