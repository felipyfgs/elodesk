package dto

import "time"

type AccountDetailResp struct {
	ID               int64          `json:"id"`
	Name             string         `json:"name"`
	Slug             string         `json:"slug"`
	Locale           string         `json:"locale"`
	Status           int            `json:"status"`
	CustomAttributes map[string]any `json:"customAttributes,omitempty"`
	Settings         map[string]any `json:"settings,omitempty"`
	CreatedAt        time.Time      `json:"createdAt"`
	UpdatedAt        time.Time      `json:"updatedAt"`
}

type UpdateAccountReq struct {
	Name     *string        `json:"name,omitempty" validate:"omitempty,min=1"`
	Locale   *string        `json:"locale,omitempty" validate:"omitempty,oneof=pt-BR en"`
	Settings map[string]any `json:"settings,omitempty"`
}
