package dto

import (
	"encoding/json"

	"backend/internal/model"
)

type CreateWebhookReq struct {
	URL           string          `json:"url" validate:"required,url,max=2048"`
	Subscriptions json.RawMessage `json:"subscriptions" validate:"required"`
}

type UpdateWebhookReq struct {
	URL           *string         `json:"url,omitempty" validate:"omitempty,url,max=2048"`
	Subscriptions json.RawMessage `json:"subscriptions,omitempty"`
	Active        *bool           `json:"is_active,omitempty"`
}

type WebhookResp struct {
	ID            int64           `json:"id"`
	AccountID     int64           `json:"account_id"`
	URL           string          `json:"url"`
	Subscriptions json.RawMessage `json:"subscriptions"`
	Active        bool            `json:"active"`
	CreatedAt     string          `json:"created_at"`
	UpdatedAt     string          `json:"updated_at"`
}

func WebhookToResp(w *model.OutboundWebhook) WebhookResp {
	return WebhookResp{
		ID:            w.ID,
		AccountID:     w.AccountID,
		URL:           w.URL,
		Subscriptions: json.RawMessage(w.Subscriptions),
		Active:        w.Active,
		CreatedAt:     w.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:     w.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func WebhooksToResp(webhooks []model.OutboundWebhook) []WebhookResp {
	result := make([]WebhookResp, len(webhooks))
	for i := range webhooks {
		result[i] = WebhookToResp(&webhooks[i])
	}
	return result
}
