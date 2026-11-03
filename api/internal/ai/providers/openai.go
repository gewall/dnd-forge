// Package providers holds one ai.Vendor implementation per vendor.
package providers

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gewall/dnd-forge/api/internal/ai"
)

// OpenAI generates images with the OpenAI Images API.
type OpenAI struct {
	api apiClient
}

// NewOpenAI returns an OpenAI provider talking to baseURL
// (https://api.openai.com/v1 in production).
func NewOpenAI(baseURL string) *OpenAI {
	return &OpenAI{api: newAPIClient("openai", baseURL, 150*time.Second)}
}

func (o *OpenAI) Type() ai.ProviderType { return ai.OpenAI }

type openAIImageRequest struct {
	Model          string `json:"model"`
	Prompt         string `json:"prompt"`
	Size           string `json:"size"`
	N              int    `json:"n"`
	ResponseFormat string `json:"response_format,omitempty"`
}

type openAIImageResponse struct {
	Data []struct {
		B64JSON string `json:"b64_json"`
	} `json:"data"`
}

// maxPromptChars stays under the smallest prompt limit of the image models.
const maxPromptChars = 3800

func (o *OpenAI) Generate(ctx context.Context, in ai.GenerateInput) (ai.GenerateOutput, error) {
	prompt := in.Prompt
	if in.NegativePrompt != "" {
		// The Images API has no negative prompt; state it in words.
		prompt += "\nAvoid: " + in.NegativePrompt
	}
	if r := []rune(prompt); len(r) > maxPromptChars {
		prompt = string(r[:maxPromptChars])
	}
	req := openAIImageRequest{Model: in.Model, Prompt: prompt, Size: openAISize(in.Model, in.AspectRatio), N: 1}
	if strings.HasPrefix(in.Model, "dall-e") {
		req.ResponseFormat = "b64_json" // GPT image models always return base64
	}

	var out openAIImageResponse
	if err := o.api.do(ctx, http.MethodPost, "/images/generations", in.APIKey, req, &out); err != nil {
		return ai.GenerateOutput{}, openAIError(err)
	}
	if len(out.Data) == 0 || out.Data[0].B64JSON == "" {
		return ai.GenerateOutput{}, fmt.Errorf("openai: response has no image")
	}
	data, err := base64.StdEncoding.DecodeString(out.Data[0].B64JSON)
	if err != nil {
		return ai.GenerateOutput{}, fmt.Errorf("openai: decode image: %w", err)
	}
	return ai.GenerateOutput{Data: data, MIMEType: http.DetectContentType(data)}, nil
}

// TestConnection checks the key by listing models, which costs nothing.
func (o *OpenAI) TestConnection(ctx context.Context, apiKey string) error {
	return openAIError(o.api.do(ctx, http.MethodGet, "/models", apiKey, nil, nil))
}

// openAIError maps vendor failures to the ai errors callers act on.
func openAIError(err error) error {
	e, ok := errors.AsType[*apiError](err)
	if !ok {
		return err
	}
	switch {
	case e.Status == http.StatusUnauthorized || e.Status == http.StatusForbidden:
		return fmt.Errorf("%w: %w", ai.ErrBadKey, e)
	case e.Code == "moderation_blocked" || e.Code == "content_policy_violation" || strings.Contains(e.Message, "safety system"):
		return fmt.Errorf("%w: %w", ai.ErrRejected, e)
	case e.Code == "insufficient_quota" || e.Code == "billing_hard_limit_reached":
		return fmt.Errorf("%w: %w", ai.ErrBilling, e)
	}
	return e
}

// openAISize maps an aspect ratio to a size the model accepts.
func openAISize(model, aspect string) string {
	dalle3 := strings.HasPrefix(model, "dall-e-3")
	switch aspect {
	case ai.Portrait:
		if dalle3 {
			return "1024x1792"
		}
		return "1024x1536"
	case ai.Landscape:
		if dalle3 {
			return "1792x1024"
		}
		return "1536x1024"
	}
	return "1024x1024"
}
