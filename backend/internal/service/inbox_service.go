package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"

	"backend/internal/model"
	"backend/internal/repo"
)

type InboxService struct {
	inboxRepo      *repo.InboxRepo
	channelApiRepo *repo.ChannelApiRepo
}

func NewInboxService(inboxRepo *repo.InboxRepo, channelApiRepo *repo.ChannelApiRepo) *InboxService {
	return &InboxService{
		inboxRepo:      inboxRepo,
		channelApiRepo: channelApiRepo,
	}
}

func (s *InboxService) Provision(accountID int64, name string) (*model.Inbox, *model.ChannelApi, error) {
	identifier := generateRandomToken(48)
	hmacToken := generateRandomToken(48)
	apiToken := generateRandomToken(48)

	channelApi := &model.ChannelApi{
		AccountID:     accountID,
		Identifier:    identifier,
		HmacToken:     hmacToken,
		ApiToken:      apiToken,
		HmacMandatory: false,
	}

	ctx := context.Background()

	if err := s.channelApiRepo.Create(ctx, channelApi); err != nil {
		return nil, nil, err
	}

	inbox := &model.Inbox{
		AccountID:   accountID,
		ChannelID:   channelApi.ID,
		Name:        name,
		ChannelType: "Channel::Api",
	}

	if err := s.inboxRepo.Create(ctx, inbox); err != nil {
		return nil, nil, err
	}

	return inbox, channelApi, nil
}

func (s *InboxService) ListByAccount(accountID int64) ([]model.Inbox, error) {
	return s.inboxRepo.ListByAccount(context.Background(), accountID)
}

func (s *InboxService) GetByID(id, accountID int64) (*model.Inbox, error) {
	inbox, err := s.inboxRepo.FindByID(context.Background(), id)
	if err != nil {
		return nil, err
	}
	if inbox.AccountID != accountID {
		return nil, repo.ErrInboxNotFound
	}
	return inbox, nil
}

func generateRandomToken(numBytes int) string {
	b := make([]byte, numBytes)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
