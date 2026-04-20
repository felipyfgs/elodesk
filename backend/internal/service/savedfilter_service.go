package service

import (
	"context"
	"encoding/json"
	"fmt"

	"backend/internal/model"
	"backend/internal/repo"
)

var (
	ErrMaxFiltersReached     = fmt.Errorf("max_filters_reached")
	ErrNestedOperators       = fmt.Errorf("nested_operators_not_supported")
	ErrInvalidFilterOperator = fmt.Errorf("invalid_filter_operator")
	ErrInvalidAttributeKey   = fmt.Errorf("invalid_attribute_key")
)

type SavedFilterService struct {
	filterRepo  *repo.CustomFilterRepo
	defRepo     *repo.CustomAttributeDefinitionRepo
	contactRepo *repo.ContactRepo
	convRepo    *repo.ConversationRepo
}

func NewSavedFilterService(filterRepo *repo.CustomFilterRepo, defRepo *repo.CustomAttributeDefinitionRepo, contactRepo *repo.ContactRepo, convRepo *repo.ConversationRepo) *SavedFilterService {
	return &SavedFilterService{filterRepo: filterRepo, defRepo: defRepo, contactRepo: contactRepo, convRepo: convRepo}
}

func (s *SavedFilterService) List(ctx context.Context, accountID, userID int64, filterType string) ([]model.CustomFilter, error) {
	return s.filterRepo.ListByUser(ctx, accountID, userID, filterType)
}

func (s *SavedFilterService) Create(ctx context.Context, accountID, userID int64, name, filterType string, query json.RawMessage) (*model.CustomFilter, error) {
	count, err := s.filterRepo.CountByUser(ctx, accountID, userID)
	if err != nil {
		return nil, err
	}
	if count >= 1000 {
		return nil, ErrMaxFiltersReached
	}

	if err := validateFilterQuery(query); err != nil {
		return nil, err
	}

	q := string(query)
	m := &model.CustomFilter{
		AccountID:  accountID,
		UserID:     userID,
		Name:       name,
		FilterType: filterType,
		Query:      &q,
	}
	if err := s.filterRepo.Create(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

func (s *SavedFilterService) Update(ctx context.Context, id, accountID, userID int64, name *string, filterType *string, query *json.RawMessage) (*model.CustomFilter, error) {
	m, err := s.filterRepo.FindByID(ctx, id, accountID, userID)
	if err != nil {
		return nil, err
	}
	if name != nil {
		m.Name = *name
	}
	if filterType != nil {
		m.FilterType = *filterType
	}
	if query != nil {
		if err := validateFilterQuery(*query); err != nil {
			return nil, err
		}
		q := string(*query)
		m.Query = &q
	}
	if err := s.filterRepo.Update(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

func (s *SavedFilterService) Delete(ctx context.Context, id, accountID, userID int64) error {
	return s.filterRepo.Delete(ctx, id, accountID, userID)
}

func validateFilterQuery(raw json.RawMessage) error {
	var q struct {
		Operator   string `json:"operator"`
		Conditions []struct {
			AttributeKey   string          `json:"attribute_key"`
			FilterOperator string          `json:"filter_operator"`
			Value          json.RawMessage `json:"value"`
		} `json:"conditions"`
	}
	if err := json.Unmarshal(raw, &q); err != nil {
		return fmt.Errorf("invalid query JSON")
	}
	if q.Operator != "AND" && q.Operator != "OR" {
		return fmt.Errorf("operator must be AND or OR")
	}
	if len(q.Conditions) > 20 {
		return fmt.Errorf("max 20 conditions allowed")
	}
	for _, cond := range q.Conditions {
		if cond.AttributeKey == "" {
			return fmt.Errorf("attribute_key is required")
		}
		if cond.FilterOperator == "" {
			return fmt.Errorf("filter_operator is required")
		}
	}
	return nil
}
