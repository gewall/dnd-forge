package providers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gewall/dnd-forge/api/internal/ai"
)

// DeepSeek writes text with the DeepSeek chat completions API, which follows
// OpenAI's format.
type DeepSeek struct {
	api apiClient
}

// NewDeepSeek returns a DeepSeek provider talking to baseURL
// (https://api.deepseek.com in production).
func NewDeepSeek(baseURL string) *DeepSeek {
	return &DeepSeek{api: newAPIClient("deepseek", baseURL, 150*time.Second)}
}

func (d *DeepSeek) Type() ai.ProviderType { return ai.DeepSeek }

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature float64       `json:"temperature"`
	Stream      bool          `json:"stream"`
}

type chatResponse struct {
	Choices []struct {
		Message      chatMessage `json:"message"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
}

// creativeTemperature follows DeepSeek's advice for creative writing.
const creativeTemperature = 1.5

func (d *DeepSeek) Complete(ctx context.Context, in ai.TextInput) (string, error) {
	req := chatRequest{Model: in.Model, MaxTokens: in.MaxTokens, Temperature: creativeTemperature}
	if in.System != "" {
		req.Messages = append(req.Messages, chatMessage{Role: "system", Content: in.System})
	}
	req.Messages = append(req.Messages, chatMessage{Role: "user", Content: in.Prompt})

	var out chatResponse
	if err := d.api.do(ctx, http.MethodPost, "/chat/completions", in.APIKey, req, &out); err != nil {
		return "", deepSeekError(err)
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("deepseek: response has no choices")
	}
	c := out.Choices[0]
	if c.FinishReason == "content_filter" {
		return "", fmt.Errorf("%w: finish_reason content_filter", ai.ErrRejected)
	}
	text := strings.TrimSpace(c.Message.Content)
	if text == "" {
		return "", fmt.Errorf("deepseek: empty completion (finish_reason %q)", c.FinishReason)
	}
	return text, nil
}

// TestConnection checks the key by listing models, which costs nothing.
func (d *DeepSeek) TestConnection(ctx context.Context, apiKey string) error {
	return deepSeekError(d.api.do(ctx, http.MethodGet, "/models", apiKey, nil, nil))
}

// deepSeekError maps vendor failures to the ai errors callers act on.
func deepSeekError(err error) error {
	e, ok := errors.AsType[*apiError](err)
	if !ok {
		return err
	}
	switch {
	case e.Status == http.StatusUnauthorized || e.Status == http.StatusForbidden:
		return fmt.Errorf("%w: %w", ai.ErrBadKey, e)
	case e.Status == http.StatusPaymentRequired:
		return fmt.Errorf("%w: %w", ai.ErrBilling, e)
	case strings.Contains(strings.ToLower(e.Message), "content exists risk"):
		return fmt.Errorf("%w: %w", ai.ErrRejected, e)
	}
	return e
}
