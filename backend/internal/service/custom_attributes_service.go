package service

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"backend/internal/model"
	"backend/internal/repo"
)

var (
	ErrAttributeKeyReserved  = fmt.Errorf("attribute_key_reserved")
	ErrListValuesRequired    = fmt.Errorf("list_values_required")
	ErrUnknownAttributeKey   = fmt.Errorf("unknown_attribute_key")
	ErrValueNotInList        = fmt.Errorf("value_not_in_list")
	ErrInvalidAttributeValue = fmt.Errorf("invalid_attribute_value")
)

var attrKeyRegex = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)

var standardContactAttributes = map[string]bool{
	"id": true, "name": true, "email": true, "phone_number": true,
	"identifier": true, "created_at": true, "updated_at": true,
	"blocked": true, "last_activity_at": true, "additional_attributes": true, "account_id": true,
}

var standardConversationAttributes = map[string]bool{
	"id": true, "status": true, "assignee_id": true, "team_id": true,
	"contact_id": true, "contact_inboxes_id": true, "inbox_id": true,
	"display_id": true, "uuid": true, "created_at": true, "updated_at": true,
	"last_activity_at": true, "priority": true, "additional_attributes": true, "account_id": true,
}

var validDisplayTypes = map[string]bool{
	"text": true, "number": true, "currency": true, "percent": true,
	"link": true, "date": true, "list": true, "checkbox": true,
}

type CustomAttributesService struct {
	defRepo     *repo.CustomAttributeDefinitionRepo
	contactRepo *repo.ContactRepo
	convRepo    *repo.ConversationRepo
}

func NewCustomAttributesService(defRepo *repo.CustomAttributeDefinitionRepo, contactRepo *repo.ContactRepo, convRepo *repo.ConversationRepo) *CustomAttributesService {
	return &CustomAttributesService{defRepo: defRepo, contactRepo: contactRepo, convRepo: convRepo}
}

func (s *CustomAttributesService) ListDefinitions(ctx context.Context, accountID int64, attributeModel string) ([]model.CustomAttributeDefinition, error) {
	return s.defRepo.ListByAccount(ctx, accountID, attributeModel)
}

func (s *CustomAttributesService) CreateDefinition(ctx context.Context, accountID int64, m *model.CustomAttributeDefinition) (*model.CustomAttributeDefinition, error) {
	m.AttributeKey = strings.TrimSpace(m.AttributeKey)
	if !attrKeyRegex.MatchString(m.AttributeKey) {
		return nil, fmt.Errorf("invalid attribute_key format")
	}
	if !validDisplayTypes[m.AttributeDisplayType] {
		return nil, fmt.Errorf("invalid attribute_display_type")
	}
	if m.AttributeModel != "contact" && m.AttributeModel != "conversation" {
		return nil, fmt.Errorf("attribute_model must be 'contact' or 'conversation'")
	}
	reserved := standardContactAttributes
	if m.AttributeModel == "conversation" {
		reserved = standardConversationAttributes
	}
	if reserved[m.AttributeKey] {
		return nil, ErrAttributeKeyReserved
	}
	if m.AttributeDisplayType == "list" {
		av := ""
		if m.AttributeValues != nil {
			av = *m.AttributeValues
		}
		if av == "" || av == "null" || av == "[]" {
			return nil, ErrListValuesRequired
		}
	}
	if err := s.defRepo.Create(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

func (s *CustomAttributesService) UpdateDefinition(ctx context.Context, id, accountID int64, m *model.CustomAttributeDefinition) (*model.CustomAttributeDefinition, error) {
	existing, err := s.defRepo.FindByID(ctx, id, accountID)
	if err != nil {
		return nil, err
	}
	if m.AttributeKey != "" {
		existing.AttributeKey = strings.TrimSpace(m.AttributeKey)
		if !attrKeyRegex.MatchString(existing.AttributeKey) {
			return nil, fmt.Errorf("invalid attribute_key format")
		}
	}
	if m.AttributeDisplayName != "" {
		existing.AttributeDisplayName = m.AttributeDisplayName
	}
	if m.AttributeDisplayType != "" {
		if !validDisplayTypes[m.AttributeDisplayType] {
			return nil, fmt.Errorf("invalid attribute_display_type")
		}
		existing.AttributeDisplayType = m.AttributeDisplayType
	}
	if m.AttributeModel != "" {
		existing.AttributeModel = m.AttributeModel
	}
	if m.AttributeValues != nil {
		existing.AttributeValues = m.AttributeValues
	}
	if m.AttributeDescription != nil {
		existing.AttributeDescription = m.AttributeDescription
	}
	if m.RegexPattern != nil {
		existing.RegexPattern = m.RegexPattern
	}
	if m.DefaultValue != nil {
		existing.DefaultValue = m.DefaultValue
	}
	if err := s.defRepo.Update(ctx, existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *CustomAttributesService) DeleteDefinition(ctx context.Context, id, accountID int64) error {
	return s.defRepo.Delete(ctx, id, accountID)
}

func (s *CustomAttributesService) GetDefinitionByID(ctx context.Context, id, accountID int64) (*model.CustomAttributeDefinition, error) {
	return s.defRepo.FindByID(ctx, id, accountID)
}

func (s *CustomAttributesService) SetContactAttributes(ctx context.Context, contactID, accountID int64, values map[string]any) (*string, error) {
	return s.setAttributes(ctx, "contact", contactID, accountID, values)
}

func (s *CustomAttributesService) RemoveContactAttributes(ctx context.Context, contactID, accountID int64, keys []string) (*string, error) {
	return s.removeAttributes(ctx, "contact", contactID, accountID, keys)
}

func (s *CustomAttributesService) SetConversationAttributes(ctx context.Context, conversationID, accountID int64, values map[string]any) (*string, error) {
	return s.setAttributes(ctx, "conversation", conversationID, accountID, values)
}

func (s *CustomAttributesService) RemoveConversationAttributes(ctx context.Context, conversationID, accountID int64, keys []string) (*string, error) {
	return s.removeAttributes(ctx, "conversation", conversationID, accountID, keys)
}

func (s *CustomAttributesService) setAttributes(ctx context.Context, targetType string, targetID, accountID int64, values map[string]any) (*string, error) {
	for key := range values {
		def, err := s.defRepo.FindByKeyAndModel(ctx, accountID, key, targetType)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrUnknownAttributeKey, key)
		}
		if err := validateAttributeValue(values[key], def); err != nil {
			return nil, err
		}
	}

	jsonbStr, err := json.Marshal(values)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal values: %w", err)
	}

	var tableName string
	if targetType == "contact" {
		tableName = "contacts"
	} else {
		tableName = "conversations"
	}
	_ = tableName

	contactRepo := s.contactRepo
	var result string
	if targetType == "contact" {
		c, err := contactRepo.FindByID(ctx, targetID, accountID)
		if err != nil {
			return nil, err
		}
		if c.AdditionalAttrs != nil {
			merged := map[string]any{}
			if err := json.Unmarshal([]byte(*c.AdditionalAttrs), &merged); err == nil {
				for k, v := range values {
					merged[k] = v
				}
				jsonbStr, _ = json.Marshal(merged)
			}
		}
		updated, err := contactRepo.UpdateAdditionalAttrs(ctx, targetID, accountID, string(jsonbStr))
		if err != nil {
			return nil, err
		}
		result = *updated
	} else {
		convRepo := s.convRepo
		conv, err := convRepo.FindByID(ctx, targetID, accountID)
		if err != nil {
			return nil, err
		}
		if conv.AdditionalAttrs != nil {
			merged := map[string]any{}
			if err := json.Unmarshal([]byte(*conv.AdditionalAttrs), &merged); err == nil {
				for k, v := range values {
					merged[k] = v
				}
				jsonbStr, _ = json.Marshal(merged)
			}
		}
		updated, err := convRepo.UpdateAdditionalAttrs(ctx, targetID, accountID, string(jsonbStr))
		if err != nil {
			return nil, err
		}
		result = *updated
	}

	return &result, nil
}

func (s *CustomAttributesService) removeAttributes(ctx context.Context, targetType string, targetID, accountID int64, keys []string) (*string, error) {
	var existingAttrs string
	if targetType == "contact" {
		c, err := s.contactRepo.FindByID(ctx, targetID, accountID)
		if err != nil {
			return nil, err
		}
		if c.AdditionalAttrs == nil {
			return c.AdditionalAttrs, nil
		}
		existingAttrs = *c.AdditionalAttrs
	} else {
		conv, err := s.convRepo.FindByID(ctx, targetID, accountID)
		if err != nil {
			return nil, err
		}
		if conv.AdditionalAttrs == nil {
			return conv.AdditionalAttrs, nil
		}
		existingAttrs = *conv.AdditionalAttrs
	}

	merged := map[string]any{}
	if err := json.Unmarshal([]byte(existingAttrs), &merged); err != nil {
		return nil, fmt.Errorf("failed to parse additional_attributes: %w", err)
	}
	for _, k := range keys {
		delete(merged, k)
	}

	jsonbStr, err := json.Marshal(merged)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal values: %w", err)
	}
	str := string(jsonbStr)

	if targetType == "contact" {
		_, err = s.contactRepo.UpdateAdditionalAttrs(ctx, targetID, accountID, str)
	} else {
		_, err = s.convRepo.UpdateAdditionalAttrs(ctx, targetID, accountID, str)
	}
	if err != nil {
		return nil, err
	}
	return &str, nil
}

func validateAttributeValue(value any, def *model.CustomAttributeDefinition) error {
	switch def.AttributeDisplayType {
	case "text":
		if _, ok := value.(string); !ok {
			return ErrInvalidAttributeValue
		}
		if def.RegexPattern != nil && *def.RegexPattern != "" {
			re, err := regexp.Compile(*def.RegexPattern)
			if err != nil {
				return fmt.Errorf("invalid regex_pattern in definition")
			}
			if !re.MatchString(value.(string)) {
				return ErrInvalidAttributeValue
			}
		}
	case "number", "currency", "percent":
		switch value.(type) {
		case float64, int, int64:
		default:
			return ErrInvalidAttributeValue
		}
	case "link":
		s, ok := value.(string)
		if !ok || !strings.HasPrefix(s, "http://") && !strings.HasPrefix(s, "https://") {
			return ErrInvalidAttributeValue
		}
	case "date":
		if _, ok := value.(string); !ok {
			return ErrInvalidAttributeValue
		}
	case "list":
		s, ok := value.(string)
		if !ok {
			return ErrInvalidAttributeValue
		}
		if def.AttributeValues == nil {
			return ErrValueNotInList
		}
		var allowed []string
		if err := json.Unmarshal([]byte(*def.AttributeValues), &allowed); err != nil {
			return ErrValueNotInList
		}
		found := false
		for _, a := range allowed {
			if a == s {
				found = true
				break
			}
		}
		if !found {
			return ErrValueNotInList
		}
	case "checkbox":
		if _, ok := value.(bool); !ok {
			return ErrInvalidAttributeValue
		}
	}
	return nil
}
