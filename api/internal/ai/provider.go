// Package ai defines the contracts AI vendors implement. Vendors live in
// ai/providers; adding one is a new file there plus one line where the
// Registry is built.
package ai

import (
	"context"
	"fmt"
	"slices"
	"sort"

	"github.com/gewall/dnd-forge/api/internal/apperr"
)

type ProviderType string

const (
	OpenAI    ProviderType = "openai"
	Google    ProviderType = "google"
	Stability ProviderType = "stability"
	Replicate ProviderType = "replicate"
	Fake      ProviderType = "fake" // development only
)

// Capability is what a configured provider is used for. One provider per
// capability is active at a time.
type Capability string

const (
	Image Capability = "image" // portraits
	Text  Capability = "text"  // character lore
)

// Aspect ratios accepted by GenerateInput.
const (
	Square    = "1:1"
	Portrait  = "3:4"
	Landscape = "4:3"
)

var (
	// ErrRejected means the vendor refused the prompt or the image (content
	// policy). Retrying will not help.
	ErrRejected = apperr.New(apperr.Invalid, "prompt ditolak oleh moderasi provider AI")
	// ErrBadKey means the API key is invalid or lacks permission.
	ErrBadKey = apperr.New(apperr.Invalid, "API key provider AI tidak valid")
	// ErrUnknownProvider means no implementation is registered for a type.
	ErrUnknownProvider = apperr.New(apperr.Invalid, "jenis provider AI tidak didukung")
	// ErrUnsupported means a vendor does not offer a capability.
	ErrUnsupported = apperr.New(apperr.Invalid, "jenis provider ini tidak mendukung kemampuan tersebut")
)

type GenerateInput struct {
	Prompt         string
	NegativePrompt string
	Model          string
	AspectRatio    string // Square | Portrait | Landscape
	APIKey         string // already decrypted
}

type GenerateOutput struct {
	Data     []byte
	MIMEType string // image/png, image/webp, image/jpeg
}

// Vendor is what every implementation offers whatever it generates.
type Vendor interface {
	Type() ProviderType
	TestConnection(ctx context.Context, apiKey string) error
}

// ImageProvider generates images with one vendor. Implementations return
// ErrRejected and ErrBadKey (possibly wrapped) for non-retryable failures;
// any other error is treated as transient.
type ImageProvider interface {
	Vendor
	Generate(ctx context.Context, in GenerateInput) (GenerateOutput, error)
}

// Registry resolves a vendor implementation by type. A vendor's
// capabilities follow from the interfaces it implements.
type Registry struct {
	byType map[ProviderType]Vendor
}

// NewRegistry returns a Registry of vs; a later vendor of the same type
// replaces an earlier one.
func NewRegistry(vs ...Vendor) *Registry {
	r := &Registry{byType: make(map[ProviderType]Vendor, len(vs))}
	for _, v := range vs {
		r.byType[v.Type()] = v
	}
	return r
}

// Vendor returns the implementation for t.
func (r *Registry) Vendor(t ProviderType) (Vendor, error) {
	v, ok := r.byType[t]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownProvider, t)
	}
	return v, nil
}

// Image returns the image implementation for t.
func (r *Registry) Image(t ProviderType) (ImageProvider, error) {
	v, err := r.Vendor(t)
	if err != nil {
		return nil, err
	}
	p, ok := v.(ImageProvider)
	if !ok {
		return nil, fmt.Errorf("%w: %s/%s", ErrUnsupported, t, Image)
	}
	return p, nil
}

// Supports reports whether t is registered and offers c.
func (r *Registry) Supports(t ProviderType, c Capability) bool {
	v, ok := r.byType[t]
	return ok && slices.Contains(capabilities(v), c)
}

// TypeInfo describes a registered vendor type.
type TypeInfo struct {
	Type         ProviderType
	Capabilities []Capability
}

// Types lists the registered vendor types with their capabilities, sorted.
func (r *Registry) Types() []TypeInfo {
	out := make([]TypeInfo, 0, len(r.byType))
	for t, v := range r.byType {
		out = append(out, TypeInfo{Type: t, Capabilities: capabilities(v)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Type < out[j].Type })
	return out
}

func capabilities(v Vendor) []Capability {
	var cs []Capability
	if _, ok := v.(ImageProvider); ok {
		cs = append(cs, Image)
	}
	return cs
}
