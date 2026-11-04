package ai

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type imageVendor struct{ t ProviderType }

func (v imageVendor) Type() ProviderType                         { return v.t }
func (imageVendor) TestConnection(context.Context, string) error { return nil }
func (imageVendor) Generate(context.Context, GenerateInput) (GenerateOutput, error) {
	return GenerateOutput{}, nil
}

type textVendor struct{}

func (textVendor) Type() ProviderType                           { return DeepSeek }
func (textVendor) TestConnection(context.Context, string) error { return nil }
func (textVendor) Complete(context.Context, TextInput) (Completion, error) {
	return Completion{}, nil
}

// bareVendor implements no capability.
type bareVendor struct{}

func (bareVendor) Type() ProviderType                           { return Google }
func (bareVendor) TestConnection(context.Context, string) error { return nil }

func TestRegistry(t *testing.T) {
	r := NewRegistry(imageVendor{OpenAI}, bareVendor{}, imageVendor{Fake}, textVendor{})

	if !r.Supports(OpenAI, Image) || r.Supports(OpenAI, Text) || r.Supports(Google, Image) || r.Supports(Stability, Image) {
		t.Error("Supports does not follow the implemented interfaces")
	}
	if _, err := r.Image(OpenAI); err != nil {
		t.Errorf("Image(openai) err = %v", err)
	}
	if !r.Supports(DeepSeek, Text) || r.Supports(DeepSeek, Image) {
		t.Error("text vendor capabilities")
	}
	if _, err := r.Text(DeepSeek); err != nil {
		t.Errorf("Text(deepseek) err = %v", err)
	}
	if _, err := r.Text(OpenAI); !errors.Is(err, ErrUnsupported) {
		t.Errorf("Text(openai) err = %v, want ErrUnsupported", err)
	}
	if _, err := r.Image(Google); !errors.Is(err, ErrUnsupported) {
		t.Errorf("Image(google) err = %v, want ErrUnsupported", err)
	}
	if _, err := r.Vendor(Stability); !errors.Is(err, ErrUnknownProvider) {
		t.Errorf("Vendor(stability) err = %v, want ErrUnknownProvider", err)
	}

	want := []TypeInfo{
		{Type: DeepSeek, Capabilities: []Capability{Text}},
		{Type: Fake, Capabilities: []Capability{Image}},
		{Type: Google},
		{Type: OpenAI, Capabilities: []Capability{Image}},
	}
	if got := r.Types(); !reflect.DeepEqual(got, want) {
		t.Errorf("Types = %+v, want %+v", got, want)
	}
}
