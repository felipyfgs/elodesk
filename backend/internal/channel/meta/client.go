package meta

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

var (
	ErrMetaAuthFailed = errors.New("meta: authentication failed (OAuthException)")
	ErrMetaRateLimit  = errors.New("meta: rate limited")
	ErrMetaPermanent  = errors.New("meta: permanent client error")
)

const (
	clientTimeout = 10 * time.Second
	maxRetries    = 3
)

type Client struct {
	http    *http.Client
	baseURL string
}

// NewClient creates a Meta Graph API client. baseURL should be the full scheme+host,
// e.g. "https://graph.facebook.com" or "https://graph.instagram.com". The Graph
// API version is appended automatically.
func NewClient(baseURL string) *Client {
	return &Client{
		http:    &http.Client{Timeout: clientTimeout},
		baseURL: baseURL + "/" + GraphVersion,
	}
}

func (c *Client) Get(ctx context.Context, path, token string, out any) error {
	return c.do(ctx, http.MethodGet, path, token, nil, out)
}

func (c *Client) Post(ctx context.Context, path, token string, body, out any) error {
	return c.do(ctx, http.MethodPost, path, token, body, out)
}

func (c *Client) Delete(ctx context.Context, path, token string, out any) error {
	return c.do(ctx, http.MethodDelete, path, token, nil, out)
}

func (c *Client) do(ctx context.Context, method, path, token string, body, out any) error {
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(attempt) * time.Second):
			}
		}
		lastErr = c.doOnce(ctx, method, path, token, body, out)
		if lastErr == nil {
			return nil
		}
		// only retry on transient errors, not auth or permanent client errors
		if errors.Is(lastErr, ErrMetaAuthFailed) || errors.Is(lastErr, ErrMetaPermanent) {
			return lastErr
		}
	}
	return lastErr
}

func (c *Client) doOnce(ctx context.Context, method, path, token string, body, out any) error {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("meta client: marshal body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("meta client: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("meta client: request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBytes, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 500 {
		return fmt.Errorf("meta client: server error %d: %s", resp.StatusCode, respBytes)
	}
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return ErrMetaAuthFailed
	}
	if resp.StatusCode == 429 {
		return ErrMetaRateLimit
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("%w: status %d: %s", ErrMetaPermanent, resp.StatusCode, respBytes)
	}

	if out != nil {
		if err := json.Unmarshal(respBytes, out); err != nil {
			return fmt.Errorf("meta client: decode response: %w", err)
		}
	}
	return nil
}
