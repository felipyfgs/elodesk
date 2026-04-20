package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"backend/internal/model"
	"backend/internal/repo"
)

var ErrLabelTitleTaken = repo.ErrLabelTitleTaken
var ErrInvalidLabelColor = errors.New("invalid hex color")

var hexColorRegex = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

type LabelService struct {
	labelRepo *repo.LabelRepo
	rt        *RealtimeService
}

func NewLabelService(labelRepo *repo.LabelRepo, rt *RealtimeService) *LabelService {
	return &LabelService{labelRepo: labelRepo, rt: rt}
}

func (s *LabelService) List(ctx context.Context, accountID int64) ([]model.Label, error) {
	return s.labelRepo.ListByAccount(ctx, accountID)
}

func (s *LabelService) Create(ctx context.Context, accountID int64, title string, color string, description *string, showOnSidebar bool) (*model.Label, error) {
	title = strings.ToLower(strings.TrimSpace(title))
	if title == "" {
		return nil, fmt.Errorf("label title is required")
	}
	if !hexColorRegex.MatchString(color) {
		return nil, ErrInvalidLabelColor
	}
	exists, err := s.labelRepo.ExistsByTitle(ctx, title, accountID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrLabelTitleTaken
	}
	m := &model.Label{
		AccountID:     accountID,
		Title:         title,
		Color:         color,
		Description:   description,
		ShowOnSidebar: showOnSidebar,
	}
	if err := s.labelRepo.Create(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

func (s *LabelService) Update(ctx context.Context, id, accountID int64, title *string, color *string, description *string, showOnSidebar *bool) (*model.Label, error) {
	m, err := s.labelRepo.FindByID(ctx, id, accountID)
	if err != nil {
		return nil, err
	}
	if title != nil {
		m.Title = strings.ToLower(strings.TrimSpace(*title))
		if m.Title == "" {
			return nil, fmt.Errorf("label title is required")
		}
	}
	if color != nil {
		if !hexColorRegex.MatchString(*color) {
			return nil, ErrInvalidLabelColor
		}
		m.Color = *color
	}
	if description != nil {
		m.Description = description
	}
	if showOnSidebar != nil {
		m.ShowOnSidebar = *showOnSidebar
	}
	if err := s.labelRepo.Update(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

func (s *LabelService) Delete(ctx context.Context, id, accountID int64) error {
	if err := s.labelRepo.Delete(ctx, id, accountID); err != nil {
		return err
	}
	s.rt.BroadcastAccountEvent(accountID, "label.deleted", map[string]any{
		"label_id":   id,
		"account_id": accountID,
	})
	return nil
}

func (s *LabelService) ApplyLabel(ctx context.Context, accountID, labelID int64, taggableType string, taggableID int64) error {
	if _, err := s.labelRepo.FindByID(ctx, labelID, accountID); err != nil {
		return err
	}
	if err := s.labelRepo.ApplyLabel(ctx, accountID, labelID, taggableType, taggableID); err != nil {
		return err
	}
	payload := map[string]any{
		"label_id":      labelID,
		"taggable_type": taggableType,
		"taggable_id":   taggableID,
		"account_id":    accountID,
	}
	if taggableType == "conversation" {
		payload["conversation_id"] = taggableID
	} else {
		payload["contact_id"] = taggableID
	}
	s.rt.BroadcastAccountEvent(accountID, "label.added", payload)
	return nil
}

func (s *LabelService) RemoveLabel(ctx context.Context, accountID, labelID int64, taggableType string, taggableID int64) error {
	if err := s.labelRepo.RemoveLabel(ctx, accountID, labelID, taggableType, taggableID); err != nil {
		return err
	}
	payload := map[string]any{
		"label_id":      labelID,
		"taggable_type": taggableType,
		"taggable_id":   taggableID,
		"account_id":    accountID,
	}
	if taggableType == "conversation" {
		payload["conversation_id"] = taggableID
	} else {
		payload["contact_id"] = taggableID
	}
	s.rt.BroadcastAccountEvent(accountID, "label.removed", payload)
	return nil
}

func (s *LabelService) ListByTaggable(ctx context.Context, accountID int64, taggableType string, taggableID int64) ([]model.Label, error) {
	return s.labelRepo.ListByTaggable(ctx, accountID, taggableType, taggableID)
}
