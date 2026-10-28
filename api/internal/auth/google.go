package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"
)

const googleUserInfoURL = "https://openidconnect.googleapis.com/v1/userinfo"

// GoogleOAuth exchanges authorization codes with Google. The frontend runs
// the redirect and PKCE; only this server knows the client secret.
type GoogleOAuth struct {
	cfg    *oauth2.Config
	client *http.Client
}

// NewGoogleOAuth returns a GoogleOAuth for the given OAuth client.
func NewGoogleOAuth(clientID, clientSecret, redirectURL string) *GoogleOAuth {
	return &GoogleOAuth{
		cfg: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Endpoint:     endpoints.Google,
			Scopes:       []string{"openid", "email", "profile"},
		},
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (g *GoogleOAuth) Exchange(ctx context.Context, code, codeVerifier string) (GoogleProfile, error) {
	ctx = context.WithValue(ctx, oauth2.HTTPClient, g.client)

	var opts []oauth2.AuthCodeOption
	if codeVerifier != "" {
		opts = append(opts, oauth2.VerifierOption(codeVerifier))
	}
	tok, err := g.cfg.Exchange(ctx, code, opts...)
	if err != nil {
		return GoogleProfile{}, fmt.Errorf("%w: %w", ErrGoogleExchange, err)
	}

	// The access token comes straight from Google over TLS, so the userinfo
	// response is trustworthy without verifying the id_token signature.
	resp, err := g.cfg.Client(ctx, tok).Get(googleUserInfoURL)
	if err != nil {
		return GoogleProfile{}, fmt.Errorf("%w: userinfo: %w", ErrGoogleExchange, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return GoogleProfile{}, fmt.Errorf("%w: userinfo status %d", ErrGoogleExchange, resp.StatusCode)
	}

	var info struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return GoogleProfile{}, fmt.Errorf("%w: decode userinfo: %w", ErrGoogleExchange, err)
	}
	if info.Sub == "" || info.Email == "" {
		return GoogleProfile{}, fmt.Errorf("%w: incomplete userinfo", ErrGoogleExchange)
	}
	return GoogleProfile{
		Subject:       info.Sub,
		Email:         info.Email,
		EmailVerified: info.EmailVerified,
		Name:          info.Name,
	}, nil
}
