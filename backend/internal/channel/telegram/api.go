package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const telegramAPIBase = "https://api.telegram.org"

type APIClient struct {
	httpClient *http.Client
}

func NewAPIClient() *APIClient {
	return &APIClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *APIClient) get(ctx context.Context, token, method string) (*http.Response, error) {
	url := fmt.Sprintf("%s/bot%s/%s", telegramAPIBase, token, method)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("telegram api new request: %w", err)
	}
	return c.httpClient.Do(req)
}

func (c *APIClient) postJSON(ctx context.Context, token, method string, body any) (*http.Response, error) {
	url := fmt.Sprintf("%s/bot%s/%s", telegramAPIBase, token, method)
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("telegram api marshal: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("telegram api new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	return c.httpClient.Do(req)
}

func decodeResponse[T any](resp *http.Response) (*APIResponse[T], error) {
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("telegram api read body: %w", err)
	}
	var result APIResponse[T]
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("telegram api unmarshal: %w", err)
	}
	return &result, nil
}

func (c *APIClient) GetMe(ctx context.Context, token string) (*GetMeResult, error) {
	resp, err := c.get(ctx, token, "getMe")
	if err != nil {
		return nil, fmt.Errorf("telegram getMe: %w", err)
	}
	result, err := decodeResponse[GetMeResult](resp)
	if err != nil {
		return nil, err
	}
	if !result.OK {
		return nil, fmt.Errorf("telegram getMe failed: %s", result.Description)
	}
	return &result.Result, nil
}

func (c *APIClient) SetWebhook(ctx context.Context, token, url, secretToken string) error {
	body := SetWebhookRequest{
		URL:         url,
		SecretToken: secretToken,
	}
	resp, err := c.postJSON(ctx, token, "setWebhook", body)
	if err != nil {
		return fmt.Errorf("telegram setWebhook: %w", err)
	}
	result, err := decodeResponse[bool](resp)
	if err != nil {
		return err
	}
	if !result.OK {
		return fmt.Errorf("telegram setWebhook failed: %s", result.Description)
	}
	return nil
}

func (c *APIClient) DeleteWebhook(ctx context.Context, token string) error {
	resp, err := c.get(ctx, token, "deleteWebhook")
	if err != nil {
		return fmt.Errorf("telegram deleteWebhook: %w", err)
	}
	result, err := decodeResponse[bool](resp)
	if err != nil {
		return err
	}
	if !result.OK {
		return fmt.Errorf("telegram deleteWebhook failed: %s", result.Description)
	}
	return nil
}

func (c *APIClient) SendMessage(ctx context.Context, token string, req SendMessageRequest) (*MessageSentResult, error) {
	resp, err := c.postJSON(ctx, token, "sendMessage", req)
	if err != nil {
		return nil, fmt.Errorf("telegram sendMessage: %w", err)
	}
	result, err := decodeResponse[MessageSentResult](resp)
	if err != nil {
		return nil, err
	}
	if !result.OK {
		return nil, fmt.Errorf("telegram sendMessage failed: %s", result.Description)
	}
	return &result.Result, nil
}

func (c *APIClient) SendPhoto(ctx context.Context, token string, req SendPhotoRequest) (*MessageSentResult, error) {
	resp, err := c.postJSON(ctx, token, "sendPhoto", req)
	if err != nil {
		return nil, fmt.Errorf("telegram sendPhoto: %w", err)
	}
	result, err := decodeResponse[MessageSentResult](resp)
	if err != nil {
		return nil, err
	}
	if !result.OK {
		return nil, fmt.Errorf("telegram sendPhoto failed: %s", result.Description)
	}
	return &result.Result, nil
}

func (c *APIClient) SendDocument(ctx context.Context, token string, req SendDocumentRequest) (*MessageSentResult, error) {
	resp, err := c.postJSON(ctx, token, "sendDocument", req)
	if err != nil {
		return nil, fmt.Errorf("telegram sendDocument: %w", err)
	}
	result, err := decodeResponse[MessageSentResult](resp)
	if err != nil {
		return nil, err
	}
	if !result.OK {
		return nil, fmt.Errorf("telegram sendDocument failed: %s", result.Description)
	}
	return &result.Result, nil
}

func (c *APIClient) GetFile(ctx context.Context, token, fileID string) (*GetFileResult, error) {
	url := fmt.Sprintf("%s/bot%s/getFile?file_id=%s", telegramAPIBase, token, fileID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("telegram getFile new request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("telegram getFile: %w", err)
	}
	result, err := decodeResponse[GetFileResult](resp)
	if err != nil {
		return nil, err
	}
	if !result.OK {
		return nil, fmt.Errorf("telegram getFile failed: %s", result.Description)
	}
	return &result.Result, nil
}

func (c *APIClient) DownloadFile(ctx context.Context, token, filePath string) ([]byte, error) {
	url := fmt.Sprintf("%s/file/bot%s/%s", telegramAPIBase, token, filePath)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("telegram downloadFile new request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("telegram downloadFile: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	return io.ReadAll(resp.Body)
}
