package dto

import (
	"encoding/json"

	"backend/internal/model"
)

type CreateMacroReq struct {
	Name       string          `json:"name" validate:"required,min=1,max=255"`
	Visibility string          `json:"visibility" validate:"required,oneof=personal account"`
	Conditions json.RawMessage `json:"conditions" validate:"required"`
	Actions    json.RawMessage `json:"actions" validate:"required"`
}

type UpdateMacroReq struct {
	Name       *string         `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Visibility *string         `json:"visibility,omitempty" validate:"omitempty,oneof=personal account"`
	Conditions json.RawMessage `json:"conditions,omitempty"`
	Actions    json.RawMessage `json:"actions,omitempty"`
}

type MacroResp struct {
	ID         int64           `json:"id"`
	AccountID  int64           `json:"accountId"`
	Name       string          `json:"name"`
	Visibility string          `json:"visibility"`
	Conditions json.RawMessage `json:"conditions"`
	Actions    json.RawMessage `json:"actions"`
	CreatedBy  int64           `json:"createdBy"`
	CreatedAt  string          `json:"createdAt"`
	UpdatedAt  string          `json:"updatedAt"`
}

func MacroToResp(m *model.Macro) MacroResp {
	return MacroResp{
		ID:         m.ID,
		AccountID:  m.AccountID,
		Name:       m.Name,
		Visibility: m.Visibility,
		Conditions: json.RawMessage(m.Conditions),
		Actions:    json.RawMessage(m.Actions),
		CreatedBy:  m.CreatedBy,
		CreatedAt:  m.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:  m.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func MacrosToResp(macros []model.Macro) []MacroResp {
	result := make([]MacroResp, len(macros))
	for i := range macros {
		result[i] = MacroToResp(&macros[i])
	}
	return result
}
