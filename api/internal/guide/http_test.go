package guide

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gewall/dnd-forge/api/internal/httpapi"
	"github.com/gewall/dnd-forge/api/internal/identity"
)

const testKey = "0123456789abcdef0123456789abcdef"

type adminVerifier struct{}

func (adminVerifier) Verify(token string) (identity.Principal, error) {
	if token != "admin" {
		return identity.Principal{}, errors.New("bad token")
	}
	return identity.Principal{UserID: admin, Role: identity.RoleAdmin}, nil
}

type noURLs struct{}

func (noURLs) Content(string) string { return "" }

func TestCreateArticleRequestContract(t *testing.T) {
	f := newFixture(t)
	srv := httpapi.New(httpapi.Config{Title: "t", Version: "0", InternalAPIKey: testKey, Verifier: adminVerifier{}})
	NewHandler(f.svc, noURLs{}).Register(srv.API)

	post := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/v1/admin/guides", strings.NewReader(body))
		req.Header.Set(httpapi.HeaderInternalKey, testKey)
		req.Header.Set("Authorization", "Bearer admin")
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		return rec
	}

	noCover := `{"categoryId":"` + f.cat.ID.String() + `","slug":"tanpa-cover","title":"x","status":"DRAFT","coverKey":""}`
	if rec := post(noCover); rec.Code != http.StatusCreated {
		t.Errorf("empty coverKey = %d, want 201: %s", rec.Code, rec.Body)
	}

	minimal := `{"categoryId":"` + f.cat.ID.String() + `","slug":"minimal","title":"Minimal","status":"DRAFT"}`
	if rec := post(minimal); rec.Code != http.StatusCreated {
		t.Errorf("minimal body = %d, want 201: %s", rec.Code, rec.Body)
	}

	tests := map[string]string{
		"missing title": `{"categoryId":"` + f.cat.ID.String() + `","slug":"x","status":"DRAFT"}`,
		"bad slug":      `{"categoryId":"` + f.cat.ID.String() + `","slug":"Bad Slug","title":"x","status":"DRAFT"}`,
		"bad status":    `{"categoryId":"` + f.cat.ID.String() + `","slug":"x","title":"x","status":"LIVE"}`,
		"bad cover key": `{"categoryId":"` + f.cat.ID.String() + `","slug":"x","title":"x","status":"DRAFT","coverKey":"../etc/passwd"}`,
	}
	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			if rec := post(body); rec.Code != http.StatusUnprocessableEntity {
				t.Errorf("= %d, want 422: %s", rec.Code, rec.Body)
			}
		})
	}
}
