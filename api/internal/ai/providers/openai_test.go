package providers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gewall/dnd-forge/api/internal/ai"
)

func TestOpenAIGenerate(t *testing.T) {
	png := []byte("\x89PNG\r\n\x1a\n-fake-")
	var got openAIImageRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/images/generations" || r.Header.Get("Authorization") != "Bearer sk-test" {
			t.Errorf("unexpected request %s %s auth=%q", r.Method, r.URL.Path, r.Header.Get("Authorization"))
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]string{{"b64_json": base64.StdEncoding.EncodeToString(png)}}})
	}))
	defer srv.Close()

	out, err := NewOpenAI(srv.URL).Generate(context.Background(), ai.GenerateInput{
		Prompt: "an elf", NegativePrompt: "text", Model: "gpt-image-1", AspectRatio: ai.Portrait, APIKey: "sk-test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if string(out.Data) != string(png) || out.MIMEType != "image/png" {
		t.Errorf("output = %q %s", out.Data, out.MIMEType)
	}
	if got.Size != "1024x1536" || got.ResponseFormat != "" || got.Prompt != "an elf\nAvoid: text" {
		t.Errorf("request = %+v", got)
	}
}

func TestOpenAIErrors(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		want   error
	}{
		{"bad key", 401, `{"error":{"message":"Incorrect API key"}}`, ai.ErrBadKey},
		{"moderation", 400, `{"error":{"message":"blocked","code":"moderation_blocked"}}`, ai.ErrRejected},
		{"content policy", 400, `{"error":{"message":"rejected by safety system","code":"content_policy_violation"}}`, ai.ErrRejected},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()
			_, err := NewOpenAI(srv.URL).Generate(context.Background(), ai.GenerateInput{Model: "gpt-image-1", APIKey: "k"})
			if !errors.Is(err, tt.want) {
				t.Errorf("err = %v, want %v", err, tt.want)
			}
		})
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()
	_, err := NewOpenAI(srv.URL).Generate(context.Background(), ai.GenerateInput{Model: "gpt-image-1", APIKey: "k"})
	if err == nil || errors.Is(err, ai.ErrRejected) || errors.Is(err, ai.ErrBadKey) {
		t.Errorf("429 should be a transient error, got %v", err)
	}
}

func TestOpenAISizes(t *testing.T) {
	tests := []struct{ model, aspect, want string }{
		{"gpt-image-1", ai.Square, "1024x1024"},
		{"gpt-image-1", ai.Landscape, "1536x1024"},
		{"dall-e-3", ai.Portrait, "1024x1792"},
		{"dall-e-3", "", "1024x1024"},
	}
	for _, tt := range tests {
		if got := openAISize(tt.model, tt.aspect); got != tt.want {
			t.Errorf("openAISize(%s, %s) = %s, want %s", tt.model, tt.aspect, got, tt.want)
		}
	}
}
