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

type chatThinking struct {
	Type string `json:"type"`
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature float64       `json:"temperature"`
	Thinking    chatThinking  `json:"thinking"`
	Stream      bool          `json:"stream"`
}

type chatResponse struct {
	Choices []struct {
		Message      chatMessage `json:"message"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

// creativeTemperature keeps some variety without derailing: 1.5 (DeepSeek's
// creative-writing advice for older models) made V4 drift into gibberish in
// longer non-thinking replies.
const creativeTemperature = 1.0

// Complete turns thinking off: V4 models think by default, which spends
// max_tokens on reasoning and can leave the answer empty (finish_reason
// "length"); lore needs no reasoning, and temperature only applies without it.
func (d *DeepSeek) Complete(ctx context.Context, in ai.TextInput) (ai.Completion, error) {
	req := chatRequest{
		Model: in.Model, MaxTokens: in.MaxTokens, Temperature: creativeTemperature,
		Thinking: chatThinking{Type: "disabled"},
	}
	if in.System != "" {
		req.Messages = append(req.Messages, chatMessage{Role: "system", Content: in.System})
	}
	req.Messages = append(req.Messages, chatMessage{Role: "user", Content: in.Prompt})

	var out chatResponse
	if err := d.api.do(ctx, http.MethodPost, "/chat/completions", in.APIKey, req, &out); err != nil {
		return ai.Completion{}, deepSeekError(err)
	}
	if len(out.Choices) == 0 {
		return ai.Completion{}, fmt.Errorf("deepseek: response has no choices")
	}
	c := out.Choices[0]
	if c.FinishReason == "content_filter" {
		return ai.Completion{}, fmt.Errorf("%w: finish_reason content_filter", ai.ErrRejected)
	}
	text := strings.TrimSpace(c.Message.Content)
	if text == "" {
		return ai.Completion{}, fmt.Errorf("deepseek: empty completion (finish_reason %q)", c.FinishReason)
	}
	return ai.Completion{Text: text, Usage: ai.Usage{
		PromptTokens: out.Usage.PromptTokens, CompletionTokens: out.Usage.CompletionTokens,
	}}, nil
}

// Models lists every model the key may use.
func (d *DeepSeek) Models(ctx context.Context, apiKey string) ([]string, error) {
	ids, err := d.api.models(ctx, apiKey)
	return ids, deepSeekError(err)
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
