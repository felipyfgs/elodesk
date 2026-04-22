package tiktok

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type OAuthClient struct {
	httpClient   *http.Client
	clientKey    string
	clientSecret string
	redirectURL  string
}

func NewOAuthClient(clientKey, clientSecret, redirectURL string) *OAuthClient {
	return &OAuthClient{
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		clientKey:    clientKey,
		clientSecret: clientSecret,
		redirectURL:  redirectURL,
	}
}

// AuthorizeURL builds the user-facing consent URL.
// https://developers.tiktok.com/doc/oauth-user-access-token-management
func (c *OAuthClient) AuthorizeURL(state string) string {
	params := url.Values{}
	params.Set("response_type", "code")
	params.Set("client_key", c.clientKey)
	params.Set("redirect_uri", c.redirectURL)
	params.Set("scope", strings.Join(RequiredScopes, ","))
	params.Set("state", state)
	return AuthHost + AuthorizePath + "?" + params.Encode()
}

// ExchangeCode redeems an auth code for access + refresh tokens and business id.
// POST https://business-api.tiktok.com/open_api/v1.3/tt_user/oauth2/token/
func (c *OAuthClient) ExchangeCode(ctx context.Context, authCode string) (*TokenData, error) {
	body := map[string]string{
		"client_id":     c.clientKey,
		"client_secret": c.clientSecret,
		"grant_type":    "authorization_code",
		"auth_code":     authCode,
		"redirect_uri":  c.redirectURL,
	}
	return c.postTokenEndpoint(ctx, APIBase+TokenEndpoint, body)
}

// Refresh exchanges a refresh_token for a new access/refresh pair.
func (c *OAuthClient) Refresh(ctx context.Context, refreshToken string) (*TokenData, error) {
	body := map[string]string{
		"client_id":     c.clientKey,
		"client_secret": c.clientSecret,
		"grant_type":    "refresh_token",
		"refresh_token": refreshToken,
	}
	return c.postTokenEndpoint(ctx, APIBase+RefreshEndpoint, body)
}

func (c *OAuthClient) postTokenEndpoint(ctx context.Context, endpoint string, body map[string]string) (*TokenData, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("tiktok oauth: marshal: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("tiktok oauth: new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tiktok oauth: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("tiktok oauth: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tiktok oauth: status %d: %s", resp.StatusCode, string(raw))
	}
	var tr TokenResponse
	if err := json.Unmarshal(raw, &tr); err != nil {
		return nil, fmt.Errorf("tiktok oauth: unmarshal: %w", err)
	}
	if tr.Code != 0 {
		return nil, fmt.Errorf("tiktok oauth: code %d: %s", tr.Code, tr.Message)
	}
	return &tr.Data, nil
}

// ScopesGranted returns true when every required scope is present in the
// comma-separated scope string from TikTok.
func ScopesGranted(granted string) bool {
	have := make(map[string]struct{}, 16)
	for _, s := range strings.Split(granted, ",") {
		have[strings.TrimSpace(s)] = struct{}{}
	}
	for _, required := range RequiredScopes {
		if _, ok := have[required]; !ok {
			return false
		}
	}
	return true
}
