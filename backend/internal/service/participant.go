package service

import (
	"context"
	"fmt"

	"backend/internal/repo"
)

type ParticipantService struct {
	repo *repo.ParticipantRepo
}

func NewParticipantService(r *repo.ParticipantRepo) *ParticipantService {
	return &ParticipantService{repo: r}
}

func (s *ParticipantService) List(ctx context.Context, accountID, convID int64) ([]repo.ParticipantWithContact, error) {
	out, err := s.repo.List(ctx, accountID, convID)
	if err != nil {
		return nil, fmt.Errorf("list participants: %w", err)
	}
	if out == nil {
		out = []repo.ParticipantWithContact{}
	}
	return out, nil
}

func (s *ParticipantService) SyncMembers(ctx context.Context, accountID, convID int64, members []repo.Member) error {
	if err := s.repo.SyncMembers(ctx, accountID, convID, members); err != nil {
		return fmt.Errorf("sync members: %w", err)
	}
	return nil
}
