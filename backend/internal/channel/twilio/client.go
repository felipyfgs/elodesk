package twilio

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("twilio api error: status=%d body=%s", e.StatusCode, e.Body)
}

func IsAuthError(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusUnauthorized || apiErr.StatusCode == http.StatusForbidden
	}
	return false
}

func IsRateLimited(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusTooManyRequests
	}
	return false
}

type Client struct {
	httpClient *http.Client
}

func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	return &Client{httpClient: httpClient}
}

func basicAuthUser(accountSID, apiKeySID string) string {
	if apiKeySID != "" {
		return apiKeySID
	}
	return accountSID
}

func apiBase() string {
	if APIBaseOverride != "" {
		return APIBaseOverride
	}
	return APIBase
}

func (c *Client) ValidateAccount(ctx context.Context, accountSID, apiKeySID, authToken string) error {
	endpoint := fmt.Sprintf("%s/Accounts/%s.json", apiBase(), accountSID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("build validate request: %w", err)
	}
	req.SetBasicAuth(basicAuthUser(accountSID, apiKeySID), authToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("validate request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return &APIError{StatusCode: resp.StatusCode, Body: string(body)}
	}

	var info accountInfoResponse
	if err := json.Unmarshal(body, &info); err != nil {
		return fmt.Errorf("parse account info: %w", err)
	}
	if info.SID == "" {
		return fmt.Errorf("empty account sid in response")
	}
	return nil
}

type SendOptions struct {
	AccountSID          string
	APIKeySID           string
	AuthToken           string
	From                string
	MessagingServiceSID string
	To                  string
	Body                string
	MediaURLs           []string
	ContentSID          string
	ContentVariables    string
	StatusCallback      string
}

func (c *Client) SendMessage(ctx context.Context, opts SendOptions) (*SendResponse, error) {
	if opts.AccountSID == "" {
		return nil, fmt.Errorf("twilio send: missing account sid")
	}
	if opts.From == "" && opts.MessagingServiceSID == "" {
		return nil, fmt.Errorf("twilio send: missing sender (from or messaging_service_sid)")
	}

	endpoint := fmt.Sprintf("%s/Accounts/%s/Messages.json", apiBase(), opts.AccountSID)

	form := url.Values{}
	if opts.MessagingServiceSID != "" {
		form.Set("MessagingServiceSid", opts.MessagingServiceSID)
	} else {
		form.Set("From", opts.From)
	}
	form.Set("To", opts.To)
	if opts.Body != "" {
		form.Set("Body", opts.Body)
	}
	for _, m := range opts.MediaURLs {
		form.Add("MediaUrl", m)
	}
	if opts.ContentSID != "" {
		form.Set("ContentSid", opts.ContentSID)
	}
	if opts.ContentVariables != "" {
		form.Set("ContentVariables", opts.ContentVariables)
	}
	if opts.StatusCallback != "" {
		form.Set("StatusCallback", opts.StatusCallback)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build send request: %w", err)
	}
	req.SetBasicAuth(basicAuthUser(opts.AccountSID, opts.APIKeySID), opts.AuthToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, &APIError{StatusCode: resp.StatusCode, Body: string(body)}
	}

	var out SendResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("parse send response: %w", err)
	}
	if out.SID == "" {
		return nil, fmt.Errorf("empty sid in response")
	}
	return &out, nil
}

func (c *Client) ListContentTemplates(ctx context.Context, accountSID, apiKeySID, authToken string) ([]ContentTemplate, error) {
	endpoint := ContentBase + "/Content?PageSize=50"
	var templates []ContentTemplate

	for endpoint != "" {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, fmt.Errorf("build list content request: %w", err)
		}
		req.SetBasicAuth(basicAuthUser(accountSID, apiKeySID), authToken)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("list content request: %w", err)
		}
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		if resp.StatusCode >= 400 {
			return nil, &APIError{StatusCode: resp.StatusCode, Body: string(body)}
		}

		var page contentListResponse
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, fmt.Errorf("parse content list: %w", err)
		}
		templates = append(templates, page.Contents...)
		endpoint = page.Meta.NextPageURL
	}
	return templates, nil
}
