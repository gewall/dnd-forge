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
	DeepSeek  ProviderType = "deepseek"
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
	// ErrBilling means the vendor account has no balance or quota left.
	ErrBilling = apperr.New(apperr.Unavailable, "saldo atau kuota akun provider AI habis")
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
// ErrRejected, ErrBadKey and ErrBilling (possibly wrapped) for
// non-retryable failures; any other error is treated as transient.
type ImageProvider interface {
	Vendor
	Generate(ctx context.Context, in GenerateInput) (GenerateOutput, error)
}

type TextInput struct {
	System    string // instructions; may be empty
	Prompt    string
	Model     string
	MaxTokens int    // upper bound of the reply; 0 leaves it to the vendor
	APIKey    string // already decrypted
}

// ModelLister is implemented by vendors that can list the models a key may
// use, so admins can pick one instead of typing its name.
type ModelLister interface {
	Models(ctx context.Context, apiKey string) ([]string, error)
}

// Usage is what a text request consumed, as the vendor bills it. Zero when
// the vendor does not report it.
type Usage struct {
	PromptTokens     int
	CompletionTokens int
}

// Completion is a written text and its cost.
type Completion struct {
	Text  string
	Usage Usage
}

// TextProvider writes text with one vendor, with the same error contract
// as ImageProvider.
type TextProvider interface {
	Vendor
	Complete(ctx context.Context, in TextInput) (Completion, error)
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
	return implAs[ImageProvider](r, t, Image)
}

// Text returns the text implementation for t.
func (r *Registry) Text(t ProviderType) (TextProvider, error) {
	return implAs[TextProvider](r, t, Text)
}

func implAs[P Vendor](r *Registry, t ProviderType, c Capability) (P, error) {
	var zero P
	v, err := r.Vendor(t)
	if err != nil {
		return zero, err
	}
	p, ok := v.(P)
	if !ok {
		return zero, fmt.Errorf("%w: %s/%s", ErrUnsupported, t, c)
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
	if _, ok := v.(TextProvider); ok {
		cs = append(cs, Text)
	}
	return cs
}
