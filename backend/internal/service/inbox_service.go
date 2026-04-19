package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"backend/internal/crypto"
	"backend/internal/model"
	"backend/internal/repo"
)

// InboxCredentials carries plaintext secrets returned ONCE at inbox creation.
// Plaintexts are never persisted (api_token is stored as SHA-256 hash,
// hmac_token as AES-GCM ciphertext) and never returned again.
type InboxCredentials struct {
	Inbox      *model.Inbox
	ChannelApi *model.ChannelApi
	ApiToken   string
	HmacToken  string
}

type InboxService struct {
	inboxRepo      *repo.InboxRepo
	channelApiRepo *repo.ChannelApiRepo
	cipher         *crypto.Cipher
}

func NewInboxService(inboxRepo *repo.InboxRepo, channelApiRepo *repo.ChannelApiRepo, cipher *crypto.Cipher) *InboxService {
	return &InboxService{
		inboxRepo:      inboxRepo,
		channelApiRepo: channelApiRepo,
		cipher:         cipher,
	}
}

func (s *InboxService) Provision(ctx context.Context, accountID int64, name string) (*InboxCredentials, error) {
	identifier := generateRandomToken(48)
	apiTokenPlaintext := generateRandomToken(48)
	hmacTokenPlaintext := generateRandomToken(48)

	hmacCiphertext, err := s.cipher.Encrypt(hmacTokenPlaintext)
	if err != nil {
		return nil, fmt.Errorf("encrypt hmac token: %w", err)
	}

	channelApi := &model.ChannelApi{
		AccountID:     accountID,
		Identifier:    identifier,
		HmacToken:     hmacCiphertext,
		ApiTokenHash:  crypto.HashLookup(apiTokenPlaintext),
		HmacMandatory: false,
	}

	if err := s.channelApiRepo.Create(ctx, channelApi); err != nil {
		return nil, err
	}

	inbox := &model.Inbox{
		AccountID:   accountID,
		ChannelID:   channelApi.ID,
		Name:        name,
		ChannelType: "Channel::Api",
	}
	if err := s.inboxRepo.Create(ctx, inbox); err != nil {
		return nil, err
	}

	return &InboxCredentials{
		Inbox:      inbox,
		ChannelApi: channelApi,
		ApiToken:   apiTokenPlaintext,
		HmacToken:  hmacTokenPlaintext,
	}, nil
}

func (s *InboxService) ListByAccount(ctx context.Context, accountID int64) ([]model.Inbox, error) {
	return s.inboxRepo.ListByAccount(ctx, accountID)
}

func (s *InboxService) GetByID(ctx context.Context, id, accountID int64) (*model.Inbox, error) {
	return s.inboxRepo.FindByID(ctx, id, accountID)
}

// DecryptHmacToken returns the plaintext HMAC key from the stored ciphertext.
// Callers must not log or leak the result; it is a per-channel signing secret.
func (s *InboxService) DecryptHmacToken(ciphertext string) (string, error) {
	return s.cipher.Decrypt(ciphertext)
}

func generateRandomToken(numBytes int) string {
	b := make([]byte, numBytes)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
