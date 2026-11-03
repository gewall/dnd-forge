// Package providers holds one ai.Vendor implementation per vendor.
package providers

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gewall/dnd-forge/api/internal/ai"
)

// OpenAI generates images with the OpenAI Images API.
type OpenAI struct {
	baseURL string
	client  *http.Client
}

// NewOpenAI returns an OpenAI provider talking to baseURL
// (https://api.openai.com/v1 in production).
func NewOpenAI(baseURL string) *OpenAI {
	return &OpenAI{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 150 * time.Second},
	}
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

type openAIError struct {
	Error struct {
		Message string `json:"message"`
		Code    string `json:"code"`
		Type    string `json:"type"`
	} `json:"error"`
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
	if err := o.do(ctx, http.MethodPost, "/images/generations", in.APIKey, req, &out); err != nil {
		return ai.GenerateOutput{}, err
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
	return o.do(ctx, http.MethodGet, "/models", apiKey, nil, nil)
}

func (o *OpenAI) do(ctx context.Context, method, path, apiKey string, body, out any) error {
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("openai: encode: %w", err)
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, o.baseURL+path, rd)
	if err != nil {
		return fmt.Errorf("openai: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := o.client.Do(req)
	if err != nil {
		return fmt.Errorf("openai: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		return fmt.Errorf("openai: read: %w", err)
	}

	if resp.StatusCode >= 300 {
		var e openAIError
		_ = json.Unmarshal(raw, &e)
		switch {
		case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
			return fmt.Errorf("%w: %s", ai.ErrBadKey, e.Error.Message)
		case e.Error.Code == "moderation_blocked" || e.Error.Code == "content_policy_violation" ||
			strings.Contains(e.Error.Message, "safety system"):
			return fmt.Errorf("%w: %s", ai.ErrRejected, e.Error.Message)
		}
		return fmt.Errorf("openai: status %d: %s", resp.StatusCode, e.Error.Message)
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("openai: decode: %w", err)
	}
	return nil
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
