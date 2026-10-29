package media

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/gewall/dnd-forge/api/internal/imaging"
)

type fakeEncoder struct{ err error }

func (f fakeEncoder) Encode(_ context.Context, _ []byte, sizes ...imaging.Size) ([]imaging.Result, error) {
	if f.err != nil {
		return nil, f.err
	}
	return []imaging.Result{{Data: []byte("webp"), Width: sizes[0].MaxWidth, Height: 10, ContentType: "image/webp"}}, nil
}

type fakeObjects struct {
	puts map[string][]byte
}

func (f *fakeObjects) Put(_ context.Context, key string, data []byte, _ string) error {
	f.puts[key] = data
	return nil
}

func (f *fakeObjects) PresignGet(_ context.Context, key string) (string, time.Duration, error) {
	return "https://s3.example.com/bucket/" + key + "?X-Amz-Signature=abc", time.Hour, nil
}

func TestUploadContentImage(t *testing.T) {
	objs := &fakeObjects{puts: map[string][]byte{}}
	svc := NewService(fakeEncoder{}, objs, NewURLs("https://api.example.com/"))

	up, err := svc.UploadContentImage(context.Background(), []byte("png"))
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(ContentKeyPattern).MatchString(up.Key) {
		t.Errorf("key %q does not match ContentKeyPattern", up.Key)
	}
	if up.URL != "https://api.example.com/v1/media/"+up.Key {
		t.Errorf("URL = %q", up.URL)
	}
	if string(objs.puts[up.Key]) != "webp" {
		t.Error("encoded image not stored under the key")
	}

	bad := NewService(fakeEncoder{err: imaging.ErrUnsupported}, objs, NewURLs(""))
	if _, err := bad.UploadContentImage(context.Background(), nil); !errors.Is(err, imaging.ErrUnsupported) {
		t.Errorf("err = %v, want ErrUnsupported", err)
	}
}

func TestContentURL(t *testing.T) {
	svc := NewService(fakeEncoder{}, &fakeObjects{}, NewURLs("https://api.example.com"))
	ctx := context.Background()

	url, ttl, err := svc.ContentURL(ctx, "4f1c7a52-0d8e-4c6b-9a43-0f7d2b1e9c11.webp")
	if err != nil || ttl != time.Hour || url == "" {
		t.Fatalf("ContentURL = %q, %v, %v", url, ttl, err)
	}
	for _, name := range []string{"../users.webp", "x.webp", "4f1c7a52-0d8e-4c6b-9a43-0f7d2b1e9c11.png", ""} {
		if _, _, err := svc.ContentURL(ctx, name); !errors.Is(err, ErrNotFound) {
			t.Errorf("ContentURL(%q) err = %v, want ErrNotFound", name, err)
		}
	}
}

func TestURLsContent(t *testing.T) {
	u := NewURLs("https://api.example.com/")
	if got := u.Content(""); got != "" {
		t.Errorf("empty key -> %q, want empty", got)
	}
	if got := u.Content("content/a.webp"); got != "https://api.example.com/v1/media/content/a.webp" {
		t.Errorf("got %q", got)
	}
}
