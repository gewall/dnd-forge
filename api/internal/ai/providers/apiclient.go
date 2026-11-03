package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// apiClient calls a JSON API that follows OpenAI's conventions (bearer
// key, {"error": {...}} bodies), as OpenAI and DeepSeek do.
type apiClient struct {
	name    string
	baseURL string
	client  *http.Client
}

func newAPIClient(name, baseURL string, timeout time.Duration) apiClient {
	return apiClient{name: name, baseURL: strings.TrimRight(baseURL, "/"), client: &http.Client{Timeout: timeout}}
}

// apiError is a non-2xx response. Providers map it to ai errors.
type apiError struct {
	Vendor  string
	Status  int
	Code    string
	Type    string
	Message string
}

func (e *apiError) Error() string {
	return fmt.Sprintf("%s: status %d: %s", e.Vendor, e.Status, e.Message)
}

// do sends body as JSON (when not nil) and decodes the response into out
// (when not nil). Non-2xx responses return *apiError.
func (c apiClient) do(ctx context.Context, method, path, apiKey string, body, out any) error {
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("%s: encode: %w", c.name, err)
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, rd)
	if err != nil {
		return fmt.Errorf("%s: %w", c.name, err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("%s: %w", c.name, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		return fmt.Errorf("%s: read: %w", c.name, err)
	}

	if resp.StatusCode >= 300 {
		var e struct {
			Error struct {
				Message string `json:"message"`
				Code    any    `json:"code"` // a string at OpenAI, sometimes a number elsewhere
				Type    string `json:"type"`
			} `json:"error"`
		}
		_ = json.Unmarshal(raw, &e)
		code := ""
		if e.Error.Code != nil {
			code = fmt.Sprint(e.Error.Code)
		}
		return &apiError{Vendor: c.name, Status: resp.StatusCode, Code: code, Type: e.Error.Type, Message: e.Error.Message}
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("%s: decode: %w", c.name, err)
	}
	return nil
}
