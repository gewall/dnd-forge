// Package media stores uploaded images and serves them through stable URLs.
//
// Objects in the bucket are private. A stable URL such as
// https://api-dnd.gewall.my.id/v1/media/content/<id>.webp answers with a 302
// to a short-lived presigned URL, so pages cached by ISR never hold an
// expired link.
package media

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/gewall/dnd-forge/api/internal/apperr"
	"github.com/gewall/dnd-forge/api/internal/imaging"
)

const (
	contentPrefix = "content/"

	// ContentKeyPattern matches keys of uploaded content images; use it in
	// OpenAPI `pattern` tags of fields that reference an upload.
	ContentKeyPattern = `^content/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\.webp$`

	maxUploadBytes = 10 << 20
)

var (
	contentKeyRE = regexp.MustCompile(ContentKeyPattern)

	ErrNotFound = apperr.New(apperr.NotFound, "file tidak ditemukan")
)

// contentSize bounds content images (covers, race and class art).
var contentSize = imaging.Size{MaxWidth: 1600, MaxHeight: 1600}

// ImageEncoder converts an uploaded image to the stored format.
type ImageEncoder interface {
	Encode(ctx context.Context, data []byte, sizes ...imaging.Size) ([]imaging.Result, error)
}

// ObjectStore saves objects and presigns downloads.
type ObjectStore interface {
	Put(ctx context.Context, key string, data []byte, contentType string) error
	PresignGet(ctx context.Context, key string) (url string, ttl time.Duration, err error)
}

// URLs builds the stable public URL of stored objects.
type URLs struct {
	base string
}

// NewURLs returns URLs rooted at the API's public origin.
func NewURLs(publicBaseURL string) URLs {
	return URLs{base: strings.TrimRight(publicBaseURL, "/")}
}

// Content returns the stable URL of a content image key, or "" for no key.
func (u URLs) Content(key string) string {
	if key == "" {
		return ""
	}
	return u.base + "/v1/media/" + key
}

// Upload is a stored image.
type Upload struct {
	Key    string
	URL    string
	Width  int
	Height int
}

// Service handles uploads and media redirects.
type Service struct {
	enc   ImageEncoder
	store ObjectStore
	urls  URLs
	newID func() uuid.UUID
}

// NewService returns a Service.
func NewService(enc ImageEncoder, store ObjectStore, urls URLs) *Service {
	return &Service{enc: enc, store: store, urls: urls, newID: uuid.New}
}

// UploadContentImage converts data to WebP and stores it under content/.
func (s *Service) UploadContentImage(ctx context.Context, data []byte) (Upload, error) {
	results, err := s.enc.Encode(ctx, data, contentSize)
	if err != nil {
		return Upload{}, fmt.Errorf("media: upload: %w", err)
	}
	img := results[0]
	key := contentPrefix + s.newID().String() + ".webp"
	if err := s.store.Put(ctx, key, img.Data, img.ContentType); err != nil {
		return Upload{}, fmt.Errorf("media: upload: %w", err)
	}
	return Upload{Key: key, URL: s.urls.Content(key), Width: img.Width, Height: img.Height}, nil
}

// ContentURL returns a presigned URL for the content image named name
// (the part after content/).
func (s *Service) ContentURL(ctx context.Context, name string) (string, time.Duration, error) {
	key := contentPrefix + name
	if !contentKeyRE.MatchString(key) {
		return "", 0, ErrNotFound
	}
	url, ttl, err := s.store.PresignGet(ctx, key)
	if err != nil {
		return "", 0, fmt.Errorf("media: presign: %w", err)
	}
	return url, ttl, nil
}
