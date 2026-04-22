package tiktok

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type APIClient struct {
	httpClient *http.Client
}

func NewAPIClient() *APIClient {
	return &APIClient{httpClient: &http.Client{Timeout: 30 * time.Second}}
}

// BusinessAccountDetails fetches the TikTok Business profile for a given
// business_id using the supplied access token.
// GET /business/get/?business_id=...&fields=["username","display_name","profile_image"]
func (c *APIClient) BusinessAccountDetails(ctx context.Context, accessToken, businessID string) (*BusinessAccountData, error) {
	params := url.Values{}
	params.Set("business_id", businessID)
	params.Set("fields", `["username","display_name","profile_image"]`)
	endpoint := APIBase + BusinessGetEndpoint + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("tiktok business: new request: %w", err)
	}
	req.Header.Set("Access-Token", accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tiktok business: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("tiktok business: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tiktok business: status %d: %s", resp.StatusCode, string(raw))
	}
	var out BusinessAccountResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("tiktok business: unmarshal: %w", err)
	}
	if out.Code != 0 {
		return nil, fmt.Errorf("tiktok business: code %d: %s", out.Code, out.Message)
	}
	return &out.Data, nil
}

// SendMessage posts an outbound TikTok message.
// POST /business/message/send/
func (c *APIClient) SendMessage(ctx context.Context, accessToken string, body SendMessageRequest) (*SendMessageInfo, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("tiktok send: marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, APIBase+SendMessageEndpoint, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("tiktok send: new request: %w", err)
	}
	req.Header.Set("Access-Token", accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tiktok send: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("tiktok send: read body: %w", err)
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, ErrReauthRequired
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tiktok send: status %d: %s", resp.StatusCode, string(raw))
	}
	var out SendMessageResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("tiktok send: unmarshal: %w", err)
	}
	if out.Code != 0 {
		return nil, fmt.Errorf("tiktok send: code %d: %s", out.Code, out.Message)
	}
	return &out.Data.Message, nil
}
