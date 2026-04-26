package service

import (
	"context"
	"fmt"

	"backend/internal/repo"
)

// ParticipantService is the thin wrapper around ParticipantRepo. It exists
// so the handler depends on a service-layer type (matching the rest of the
// backend's layered architecture) and so realtime/notification cross-cutting
// behavior can be added later without touching the handler.
type ParticipantService struct {
	repo *repo.ParticipantRepo
}

func NewParticipantService(r *repo.ParticipantRepo) *ParticipantService {
	return &ParticipantService{repo: r}
}

// List returns the conversation's participants with the contact hydrated.
// Returns an empty slice (never nil) when the conversation has no
// participants — Chatwoot's API contract for 1:1 conversations.
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

// SyncMembers reconciles the participants of a conversation against the
// supplied list. Used by Wzap when a WhatsApp group's roster changes.
func (s *ParticipantService) SyncMembers(ctx context.Context, accountID, convID int64, members []repo.Member) error {
	if err := s.repo.SyncMembers(ctx, accountID, convID, members); err != nil {
		return fmt.Errorf("sync members: %w", err)
	}
	return nil
}
