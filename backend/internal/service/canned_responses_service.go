package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"backend/internal/model"
	"backend/internal/repo"
)

var ErrCannedShortCodeTaken = repo.ErrCannedShortCodeTaken

var shortCodeRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,31}$`)

type CannedResponsesService struct {
	repo *repo.CannedResponseRepo
}

func NewCannedResponsesService(repo *repo.CannedResponseRepo) *CannedResponsesService {
	return &CannedResponsesService{repo: repo}
}

func (s *CannedResponsesService) List(ctx context.Context, accountID int64, search string, limit int) ([]model.CannedResponse, error) {
	return s.repo.ListByAccount(ctx, accountID, search, limit)
}

func (s *CannedResponsesService) Create(ctx context.Context, accountID int64, shortCode, content string) (*model.CannedResponse, error) {
	shortCode = strings.TrimSpace(shortCode)
	if !shortCodeRegex.MatchString(shortCode) {
		return nil, fmt.Errorf("invalid short_code: must match ^[a-z0-9][a-z0-9_-]{0,31}$")
	}
	if len(content) > 10000 {
		return nil, fmt.Errorf("content must be at most 10000 characters")
	}
	m := &model.CannedResponse{
		AccountID: accountID,
		ShortCode: shortCode,
		Content:   content,
	}
	if err := s.repo.Create(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

func (s *CannedResponsesService) Update(ctx context.Context, id, accountID int64, shortCode *string, content *string) (*model.CannedResponse, error) {
	m, err := s.repo.FindByID(ctx, id, accountID)
	if err != nil {
		return nil, err
	}
	if shortCode != nil {
		sc := strings.TrimSpace(*shortCode)
		if !shortCodeRegex.MatchString(sc) {
			return nil, fmt.Errorf("invalid short_code: must match ^[a-z0-9][a-z0-9_-]{0,31}$")
		}
		m.ShortCode = sc
	}
	if content != nil {
		if len(*content) > 10000 {
			return nil, fmt.Errorf("content must be at most 10000 characters")
		}
		m.Content = *content
	}
	if err := s.repo.Update(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

func (s *CannedResponsesService) Delete(ctx context.Context, id, accountID int64) error {
	return s.repo.Delete(ctx, id, accountID)
}

func (s *CannedResponsesService) GetByID(ctx context.Context, id, accountID int64) (*model.CannedResponse, error) {
	return s.repo.FindByID(ctx, id, accountID)
}
