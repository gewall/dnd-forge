package character

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/gewall/dnd-forge/api/internal/httpapi"
	"github.com/gewall/dnd-forge/api/internal/identity"
)

const testKey = "0123456789abcdef0123456789abcdef"

var (
	alice = uuid.New()
	bob   = uuid.New()
)

type tokens struct{}

func (tokens) Verify(token string) (identity.Principal, error) {
	switch token {
	case "alice":
		return identity.Principal{UserID: alice, Role: identity.RoleUser}, nil
	case "bob":
		return identity.Principal{UserID: bob, Role: identity.RoleUser}, nil
	}
	return identity.Principal{}, errors.New("bad token")
}

func TestHTTPFlow(t *testing.T) {
	cat := newFakeCatalog()
	srv := httpapi.New(httpapi.Config{Title: "t", Version: "0", InternalAPIKey: testKey, Verifier: tokens{}})
	NewHandler(NewService(newFakeStore(), cat)).Register(srv.API)

	call := func(method, path, token, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set(httpapi.HeaderInternalKey, testKey)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		return rec
	}

	body := `{"name":"Lia","raceId":"` + cat.human.String() + `","classId":"` + cat.wizard.String() +
		`","backgroundId":"` + cat.sage.String() + `"}`

	if rec := call(http.MethodPost, "/v1/characters", "", body); rec.Code != http.StatusUnauthorized {
		t.Errorf("anonymous create = %d, want 401", rec.Code)
	}
	rec := call(http.MethodPost, "/v1/characters", "alice", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create = %d: %s", rec.Code, rec.Body)
	}
	id := between(rec.Body.String(), `"id":"`, `"`)

	tests := []struct {
		name, method, path, token, body string
		want                            int
	}{
		{"owner reads", http.MethodGet, "/v1/characters/" + id, "alice", "", http.StatusOK},
		{"stranger reads", http.MethodGet, "/v1/characters/" + id, "bob", "", http.StatusNotFound},
		{"stranger deletes", http.MethodDelete, "/v1/characters/" + id, "bob", "", http.StatusNotFound},
		{"bad id", http.MethodGet, "/v1/characters/not-a-uuid", "alice", "", http.StatusUnprocessableEntity},
		{"missing name", http.MethodPost, "/v1/characters", "alice", strings.Replace(body, `"name":"Lia",`, "", 1), http.StatusUnprocessableEntity},
		{"invalid choice", http.MethodPost, "/v1/characters", "alice", strings.Replace(body, cat.human.String(), cat.orc.String(), 1), http.StatusBadRequest},
		{"owner deletes", http.MethodDelete, "/v1/characters/" + id, "alice", "", http.StatusNoContent},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if rec := call(tt.method, tt.path, tt.token, tt.body); rec.Code != tt.want {
				t.Errorf("= %d, want %d: %s", rec.Code, tt.want, rec.Body)
			}
		})
	}
}

func between(s, start, end string) string {
	_, after, _ := strings.Cut(s, start)
	v, _, _ := strings.Cut(after, end)
	return v
}
