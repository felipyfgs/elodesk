package dto

import (
	"backend/internal/model"
)

type CreateCannedResponseReq struct {
	ShortCode string `json:"short_code" validate:"required"`
	Content   string `json:"content" validate:"required,max=10000"`
}

type UpdateCannedResponseReq struct {
	ShortCode *string `json:"short_code,omitempty" validate:"omitempty"`
	Content   *string `json:"content,omitempty" validate:"omitempty,max=10000"`
}

type CannedResponseResp struct {
	ID        int64  `json:"id"`
	AccountID int64  `json:"accountId"`
	ShortCode string `json:"shortCode"`
	Content   string `json:"content"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

func CannedResponseToResp(c *model.CannedResponse) CannedResponseResp {
	return CannedResponseResp{
		ID:        c.ID,
		AccountID: c.AccountID,
		ShortCode: c.ShortCode,
		Content:   c.Content,
		CreatedAt: c.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: c.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func CannedResponsesToResp(items []model.CannedResponse) []CannedResponseResp {
	result := make([]CannedResponseResp, len(items))
	for i := range items {
		result[i] = CannedResponseToResp(&items[i])
	}
	return result
}
