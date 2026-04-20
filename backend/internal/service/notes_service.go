package service

import (
	"context"
	"errors"
	"fmt"

	"backend/internal/model"
	"backend/internal/repo"
)

var ErrNotNoteOwner = errors.New("not note owner")

type NoteService struct {
	repo *repo.NoteRepo
	rt   *RealtimeService
}

func NewNoteService(repo *repo.NoteRepo, rt *RealtimeService) *NoteService {
	return &NoteService{repo: repo, rt: rt}
}

func (s *NoteService) ListByContact(ctx context.Context, contactID, accountID int64, page, perPage int) ([]model.Note, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 25
	}
	return s.repo.ListByContact(ctx, contactID, accountID, page, perPage)
}

func (s *NoteService) Create(ctx context.Context, accountID, contactID, userID int64, content string) (*model.Note, error) {
	if len(content) > 50000 {
		return nil, fmt.Errorf("content must be at most 50000 characters")
	}
	m := &model.Note{
		AccountID: accountID,
		ContactID: contactID,
		UserID:    userID,
		Content:   content,
	}
	if err := s.repo.Create(ctx, m); err != nil {
		return nil, err
	}
	s.rt.BroadcastAccountEvent(accountID, "note.created", map[string]any{
		"note_id":    m.ID,
		"contact_id": contactID,
		"user_id":    userID,
		"account_id": accountID,
	})
	return m, nil
}

func (s *NoteService) Update(ctx context.Context, id, accountID, userID int64, role int, content string) (*model.Note, error) {
	note, err := s.repo.FindByID(ctx, id, accountID)
	if err != nil {
		return nil, err
	}
	if note.UserID != userID && role < int(model.RoleAdmin) {
		return nil, ErrNotNoteOwner
	}
	if len(content) > 50000 {
		return nil, fmt.Errorf("content must be at most 50000 characters")
	}
	note.Content = content
	if err := s.repo.Update(ctx, note); err != nil {
		return nil, err
	}
	return note, nil
}

func (s *NoteService) Delete(ctx context.Context, id, accountID, userID int64, role int) error {
	note, err := s.repo.FindByID(ctx, id, accountID)
	if err != nil {
		return err
	}
	if note.UserID != userID && role < int(model.RoleAdmin) {
		return ErrNotNoteOwner
	}
	return s.repo.Delete(ctx, id, accountID)
}
