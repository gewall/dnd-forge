package providers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gewall/dnd-forge/api/internal/ai"
)

func TestDeepSeekComplete(t *testing.T) {
	var got chatRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/chat/completions" || r.Header.Get("Authorization") != "Bearer sk-ds" {
			t.Errorf("unexpected request %s %s auth=%q", r.Method, r.URL.Path, r.Header.Get("Authorization"))
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"  Kisah Lia.\n"},"finish_reason":"stop"}]}`))
	}))
	defer srv.Close()

	text, err := NewDeepSeek(srv.URL+"/").Complete(context.Background(), ai.TextInput{
		System: "kamu penulis lore", Prompt: "tulis lore", Model: "deepseek-v4-flash", MaxTokens: 900, APIKey: "sk-ds",
	})
	if err != nil || text != "Kisah Lia." {
		t.Fatalf("Complete = %q, %v", text, err)
	}
	want := []chatMessage{{Role: "system", Content: "kamu penulis lore"}, {Role: "user", Content: "tulis lore"}}
	if got.Model != "deepseek-v4-flash" || got.MaxTokens != 900 || got.Stream || got.Thinking.Type != "disabled" || len(got.Messages) != 2 ||
		got.Messages[0] != want[0] || got.Messages[1] != want[1] {
		t.Errorf("request = %+v", got)
	}
}

func TestDeepSeekErrors(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		want   error // nil means a transient error
	}{
		{"bad key", 401, `{"error":{"message":"Authentication Fails","type":"authentication_error"}}`, ai.ErrBadKey},
		{"no balance", 402, `{"error":{"message":"Insufficient Balance"}}`, ai.ErrBilling},
		{"content risk", 400, `{"error":{"message":"Content Exists Risk","code":"invalid_request_error"}}`, ai.ErrRejected},
		{"filtered reply", 200, `{"choices":[{"message":{"content":""},"finish_reason":"content_filter"}]}`, ai.ErrRejected},
		{"rate limited", 429, `{"error":{"message":"Rate Limit Reached"}}`, nil},
		{"overloaded", 503, ``, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()
			_, err := NewDeepSeek(srv.URL).Complete(context.Background(), ai.TextInput{Prompt: "x", Model: "m", APIKey: "k"})
			if err == nil {
				t.Fatal("want an error")
			}
			if tt.want != nil && !errors.Is(err, tt.want) {
				t.Errorf("err = %v, want %v", err, tt.want)
			}
			if tt.want == nil && (errors.Is(err, ai.ErrRejected) || errors.Is(err, ai.ErrBadKey) || errors.Is(err, ai.ErrBilling)) {
				t.Errorf("err = %v, want a transient error", err)
			}
		})
	}
}

func TestDeepSeekTestConnection(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/models" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer good" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"object":"list","data":[]}`))
	}))
	defer srv.Close()
	ds := NewDeepSeek(srv.URL)
	if err := ds.TestConnection(context.Background(), "good"); err != nil {
		t.Errorf("good key: %v", err)
	}
	if err := ds.TestConnection(context.Background(), "bad"); !errors.Is(err, ai.ErrBadKey) {
		t.Errorf("bad key err = %v", err)
	}
}
