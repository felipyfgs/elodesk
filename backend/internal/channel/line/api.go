package line

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const lineAPIBase = "https://api.line.me"
const lineDataAPIBase = "https://api-data.line.me"

type APIClient struct {
	httpClient *http.Client
}

func NewAPIClient() *APIClient {
	return &APIClient{httpClient: &http.Client{Timeout: 30 * time.Second}}
}

func (c *APIClient) authedRequest(ctx context.Context, method, url, token string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("line api new request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

// GetBotInfo calls GET /v2/bot/info and returns the bot metadata.
func (c *APIClient) GetBotInfo(ctx context.Context, token string) (*BotInfo, error) {
	req, err := c.authedRequest(ctx, http.MethodGet, lineAPIBase+"/v2/bot/info", token, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("line getBotInfo: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("line getBotInfo: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("line getBotInfo: status %d: %s", resp.StatusCode, string(data))
	}
	var out BotInfo
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("line getBotInfo: unmarshal: %w", err)
	}
	return &out, nil
}

// GetProfile fetches a LINE user profile via GET /v2/bot/profile/:userId.
func (c *APIClient) GetProfile(ctx context.Context, token, userID string) (*UserProfile, error) {
	req, err := c.authedRequest(ctx, http.MethodGet, lineAPIBase+"/v2/bot/profile/"+userID, token, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("line getProfile: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("line getProfile: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("line getProfile: status %d: %s", resp.StatusCode, string(data))
	}
	var out UserProfile
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("line getProfile: unmarshal: %w", err)
	}
	return &out, nil
}

// Reply sends a reply message using the reply_token returned in webhook events.
// https://developers.line.biz/en/reference/messaging-api/#send-reply-message
func (c *APIClient) Reply(ctx context.Context, token string, body ReplyRequest) error {
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("line reply: marshal: %w", err)
	}
	req, err := c.authedRequest(ctx, http.MethodPost, lineAPIBase+"/v2/bot/message/reply", token, bytes.NewReader(data))
	if err != nil {
		return err
	}
	return c.executeAndExpectOK(req, "reply")
}

// Push sends a push message to a LINE user/group/room.
// https://developers.line.biz/en/reference/messaging-api/#send-push-message
func (c *APIClient) Push(ctx context.Context, token string, body PushRequest) error {
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("line push: marshal: %w", err)
	}
	req, err := c.authedRequest(ctx, http.MethodPost, lineAPIBase+"/v2/bot/message/push", token, bytes.NewReader(data))
	if err != nil {
		return err
	}
	return c.executeAndExpectOK(req, "push")
}

// GetMessageContent fetches binary content for a user-uploaded message (image/video/audio/file).
// https://developers.line.biz/en/reference/messaging-api/#get-content
func (c *APIClient) GetMessageContent(ctx context.Context, token, messageID string) ([]byte, string, error) {
	req, err := c.authedRequest(ctx, http.MethodGet, lineDataAPIBase+"/v2/bot/message/"+messageID+"/content", token, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("line getMessageContent: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("line getMessageContent: status %d: %s", resp.StatusCode, string(body))
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("line getMessageContent: read: %w", err)
	}
	return body, resp.Header.Get("Content-Type"), nil
}

func (c *APIClient) executeAndExpectOK(req *http.Request, op string) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("line %s: %w", op, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusOK {
		return nil
	}
	data, _ := io.ReadAll(resp.Body)
	var apiErr APIErrorResponse
	if err := json.Unmarshal(data, &apiErr); err == nil && apiErr.Message != "" {
		return fmt.Errorf("line %s failed: status %d: %s", op, resp.StatusCode, apiErr.Message)
	}
	return fmt.Errorf("line %s failed: status %d: %s", op, resp.StatusCode, string(data))
}
