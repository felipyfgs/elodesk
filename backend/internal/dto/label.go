package dto

import (
	"backend/internal/model"
)

type CreateLabelReq struct {
	Title         string  `json:"title" validate:"required,min=1,max=255"`
	Color         string  `json:"color" validate:"omitempty,hexcolor"`
	Description   *string `json:"description,omitempty"`
	ShowOnSidebar *bool   `json:"show_on_sidebar,omitempty"`
}

type UpdateLabelReq struct {
	Title         *string `json:"title,omitempty" validate:"omitempty,min=1,max=255"`
	Color         *string `json:"color,omitempty" validate:"omitempty,hexcolor"`
	Description   *string `json:"description,omitempty"`
	ShowOnSidebar *bool   `json:"show_on_sidebar,omitempty"`
}

type ApplyLabelReq struct {
	LabelID int64 `json:"label_id" validate:"required"`
}

type LabelResp struct {
	ID            int64   `json:"id"`
	AccountID     int64   `json:"accountId"`
	Title         string  `json:"title"`
	Color         string  `json:"color"`
	Description   *string `json:"description,omitempty"`
	ShowOnSidebar bool    `json:"showOnSidebar"`
	CreatedAt     string  `json:"createdAt"`
	UpdatedAt     string  `json:"updatedAt"`
}

func LabelToResp(l *model.Label) LabelResp {
	return LabelResp{
		ID:            l.ID,
		AccountID:     l.AccountID,
		Title:         l.Title,
		Color:         l.Color,
		Description:   l.Description,
		ShowOnSidebar: l.ShowOnSidebar,
		CreatedAt:     l.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:     l.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func LabelsToResp(labels []model.Label) []LabelResp {
	result := make([]LabelResp, len(labels))
	for i := range labels {
		result[i] = LabelToResp(&labels[i])
	}
	return result
}
