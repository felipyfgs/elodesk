package email

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var googleScopes = []string{
	"https://www.googleapis.com/auth/gmail.readonly",
	"https://www.googleapis.com/auth/gmail.send",
}

func googleOAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_OAUTH_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("GOOGLE_OAUTH_REDIRECT_URI"),
		Scopes:       googleScopes,
		Endpoint:     google.Endpoint,
	}
}

// GoogleAuthURL returns the consent-screen URL for a given state token.
func GoogleAuthURL(state string) string {
	return googleOAuthConfig().AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
}

// GoogleExchangeCode exchanges an authorization code for access+refresh tokens.
func GoogleExchangeCode(ctx context.Context, code string) (*OAuthTokens, error) {
	cfg := googleOAuthConfig()
	tok, err := cfg.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("google exchange: %w", err)
	}
	return &OAuthTokens{
		AccessToken:  tok.AccessToken,
		RefreshToken: tok.RefreshToken,
		ExpiresOn:    tok.Expiry,
	}, nil
}

// GoogleRefreshToken exchanges a refresh_token for a new access_token.
func GoogleRefreshToken(ctx context.Context, refreshToken string) (*OAuthTokens, error) {
	cfg := googleOAuthConfig()
	src := cfg.TokenSource(ctx, &oauth2.Token{RefreshToken: refreshToken})
	tok, err := src.Token()
	if err != nil {
		return nil, fmt.Errorf("google refresh: %w", err)
	}
	return &OAuthTokens{
		AccessToken:  tok.AccessToken,
		RefreshToken: tok.RefreshToken,
		ExpiresOn:    tok.Expiry,
	}, nil
}

// OAuthTokens is the in-memory representation of provider OAuth tokens.
type OAuthTokens struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresOn    time.Time `json:"expires_on"`
}

// NeedsRefresh returns true when the access token expires within 5 minutes.
func (t *OAuthTokens) NeedsRefresh() bool {
	return time.Until(t.ExpiresOn) < 5*time.Minute
}

// MarshalJSON serialises OAuthTokens for storage in provider_config.
func MarshalTokens(t *OAuthTokens) (string, error) {
	b, err := json.Marshal(t)
	if err != nil {
		return "", fmt.Errorf("marshal tokens: %w", err)
	}
	return string(b), nil
}

// UnmarshalTokens deserialises the plaintext provider_config JSON.
func UnmarshalTokens(raw string) (*OAuthTokens, error) {
	var t OAuthTokens
	if err := json.Unmarshal([]byte(raw), &t); err != nil {
		return nil, fmt.Errorf("unmarshal tokens: %w", err)
	}
	return &t, nil
}
