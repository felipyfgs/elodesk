package twitter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// ErrReauthRequired indicates a 401/403 from Twitter — the access token was
// revoked or the consumer credentials were rotated. Callers must record this
// against the reauth tracker and surface it via realtime.
var ErrReauthRequired = errors.New("twitter: reauth required")

// APIClient performs OAuth 1.0a-signed requests against Twitter's v2 API
// using a consumer key/secret + per-channel access token/secret pair.
type APIClient struct {
	httpClient     *http.Client
	consumerKey    string
	consumerSecret string
}

func NewAPIClient(consumerKey, consumerSecret string) *APIClient {
	return &APIClient{
		httpClient:     &http.Client{Timeout: 30 * time.Second},
		consumerKey:    consumerKey,
		consumerSecret: consumerSecret,
	}
}

// GetMe resolves the authenticated user's profile id. Used right after the
// OAuth handshake to materialize the channels_twitter row when /access_token
// did not return a user_id (Twitter v2-only flows).
func (c *APIClient) GetMe(ctx context.Context, accessToken, accessTokenSecret string) (*MeResponse, error) {
	endpoint := apiBase + usersMePath
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("twitter get me: new req: %w", err)
	}
	header, err := c.signedHeader(http.MethodGet, endpoint, accessToken, accessTokenSecret, nil, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", header)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("twitter get me: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("twitter get me: read: %w", err)
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, ErrReauthRequired
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("twitter get me: status %d: %s", resp.StatusCode, string(raw))
	}
	var out MeResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("twitter get me: unmarshal: %w", err)
	}
	return &out, nil
}

// SendDM posts an outbound direct message via
// POST /2/dm_conversations/with/:participant_id/messages.
// Returns the v2 dm_event id of the created message.
func (c *APIClient) SendDM(ctx context.Context, accessToken, accessTokenSecret, participantID, text string) (string, error) {
	endpoint := apiBase + fmt.Sprintf(dmConversationsFmt, url.PathEscape(participantID))
	body, err := json.Marshal(map[string]any{"text": text})
	if err != nil {
		return "", fmt.Errorf("twitter send dm: marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("twitter send dm: new req: %w", err)
	}
	header, err := c.signedHeader(http.MethodPost, endpoint, accessToken, accessTokenSecret, nil, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", header)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("twitter send dm: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("twitter send dm: read: %w", err)
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return "", ErrReauthRequired
	}
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("twitter send dm: status %d: %s", resp.StatusCode, string(raw))
	}

	var out struct {
		Data struct {
			DMEventID string `json:"dm_event_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("twitter send dm: unmarshal: %w", err)
	}
	return out.Data.DMEventID, nil
}

// signedHeader builds the OAuth 1.0a Authorization header used by the API
// client (no body signing — v2 endpoints accept JSON bodies that are NOT
// included in the signature base string).
func (c *APIClient) signedHeader(method, endpoint, token, tokenSecret string, extraOauth map[string]string, body url.Values) (string, error) {
	signer := &OAuthClient{consumerKey: c.consumerKey, consumerSecret: c.consumerSecret}
	return signer.buildAuthHeader(method, endpoint, token, tokenSecret, extraOauth, body)
}
