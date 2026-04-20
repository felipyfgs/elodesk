package dto

import (
	"encoding/json"

	"backend/internal/model"
)

type CreateCustomFilterReq struct {
	Name       string          `json:"name" validate:"required"`
	FilterType string          `json:"filter_type" validate:"required,oneof=conversation contact"`
	Query      json.RawMessage `json:"query" validate:"required"`
}

type UpdateCustomFilterReq struct {
	Name       *string          `json:"name,omitempty" validate:"omitempty"`
	FilterType *string          `json:"filter_type,omitempty" validate:"omitempty,oneof=conversation contact"`
	Query      *json.RawMessage `json:"query,omitempty"`
}

type ApplyFilterReq struct {
	Query   json.RawMessage `json:"query" validate:"required"`
	Page    int             `json:"page,omitempty"`
	PerPage int             `json:"per_page,omitempty"`
}

type CustomFilterResp struct {
	ID         int64   `json:"id"`
	AccountID  int64   `json:"accountId"`
	UserID     int64   `json:"userId"`
	Name       string  `json:"name"`
	FilterType string  `json:"filterType"`
	Query      *string `json:"query"`
	CreatedAt  string  `json:"createdAt"`
	UpdatedAt  string  `json:"updatedAt"`
}

func CustomFilterToResp(f *model.CustomFilter) CustomFilterResp {
	return CustomFilterResp{
		ID:         f.ID,
		AccountID:  f.AccountID,
		UserID:     f.UserID,
		Name:       f.Name,
		FilterType: f.FilterType,
		Query:      f.Query,
		CreatedAt:  f.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:  f.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func CustomFiltersToResp(filters []model.CustomFilter) []CustomFilterResp {
	result := make([]CustomFilterResp, len(filters))
	for i := range filters {
		result[i] = CustomFilterToResp(&filters[i])
	}
	return result
}
