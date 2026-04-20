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
	IsActive      *bool           `json:"is_active,omitempty"`
}

type WebhookResp struct {
	ID            int64           `json:"id"`
	AccountID     int64           `json:"accountId"`
	URL           string          `json:"url"`
	Subscriptions json.RawMessage `json:"subscriptions"`
	IsActive      bool            `json:"isActive"`
	CreatedAt     string          `json:"createdAt"`
	UpdatedAt     string          `json:"updatedAt"`
}

func WebhookToResp(w *model.OutboundWebhook) WebhookResp {
	return WebhookResp{
		ID:            w.ID,
		AccountID:     w.AccountID,
		URL:           w.URL,
		Subscriptions: json.RawMessage(w.Subscriptions),
		IsActive:      w.IsActive,
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
