package dto

import (
	"encoding/json"

	"backend/internal/model"
)

type CreateCustomAttributeDefinitionReq struct {
	AttributeKey         string          `json:"attribute_key" validate:"required"`
	AttributeDisplayName string          `json:"attribute_display_name" validate:"required"`
	AttributeDisplayType string          `json:"attribute_display_type" validate:"required,oneof=text number currency percent link date list checkbox"`
	AttributeModel       string          `json:"attribute_model" validate:"required,oneof=contact conversation"`
	AttributeValues      json.RawMessage `json:"attribute_values,omitempty"`
	AttributeDescription *string         `json:"attribute_description,omitempty"`
	RegexPattern         *string         `json:"regex_pattern,omitempty"`
	DefaultValue         *string         `json:"default_value,omitempty"`
}

type UpdateCustomAttributeDefinitionReq struct {
	AttributeKey         *string         `json:"attribute_key,omitempty" validate:"omitempty"`
	AttributeDisplayName *string         `json:"attribute_display_name,omitempty" validate:"omitempty"`
	AttributeDisplayType *string         `json:"attribute_display_type,omitempty" validate:"omitempty,oneof=text number currency percent link date list checkbox"`
	AttributeModel       *string         `json:"attribute_model,omitempty" validate:"omitempty,oneof=contact conversation"`
	AttributeValues      json.RawMessage `json:"attribute_values,omitempty"`
	AttributeDescription *string         `json:"attribute_description,omitempty"`
	RegexPattern         *string         `json:"regex_pattern,omitempty"`
	DefaultValue         *string         `json:"default_value,omitempty"`
}

type SetCustomAttributesReq map[string]any

type RemoveCustomAttributesReq struct {
	Keys []string `json:"keys" validate:"required,min=1"`
}

type CustomAttributeDefinitionResp struct {
	ID                   int64   `json:"id"`
	AccountID            int64   `json:"accountId"`
	AttributeKey         string  `json:"attributeKey"`
	AttributeDisplayName string  `json:"attributeDisplayName"`
	AttributeDisplayType string  `json:"attributeDisplayType"`
	AttributeModel       string  `json:"attributeModel"`
	AttributeValues      *string `json:"attributeValues,omitempty"`
	AttributeDescription *string `json:"attributeDescription,omitempty"`
	RegexPattern         *string `json:"regexPattern,omitempty"`
	DefaultValue         *string `json:"defaultValue,omitempty"`
	CreatedAt            string  `json:"createdAt"`
	UpdatedAt            string  `json:"updatedAt"`
}

func CustomAttrDefToResp(d *model.CustomAttributeDefinition) CustomAttributeDefinitionResp {
	return CustomAttributeDefinitionResp{
		ID:                   d.ID,
		AccountID:            d.AccountID,
		AttributeKey:         d.AttributeKey,
		AttributeDisplayName: d.AttributeDisplayName,
		AttributeDisplayType: d.AttributeDisplayType,
		AttributeModel:       d.AttributeModel,
		AttributeValues:      d.AttributeValues,
		AttributeDescription: d.AttributeDescription,
		RegexPattern:         d.RegexPattern,
		DefaultValue:         d.DefaultValue,
		CreatedAt:            d.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:            d.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func CustomAttrDefsToResp(defs []model.CustomAttributeDefinition) []CustomAttributeDefinitionResp {
	result := make([]CustomAttributeDefinitionResp, len(defs))
	for i := range defs {
		result[i] = CustomAttrDefToResp(&defs[i])
	}
	return result
}
