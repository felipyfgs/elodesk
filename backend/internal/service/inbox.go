package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"

	"backend/internal/crypto"
	"backend/internal/model"
	"backend/internal/repo"

	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrInvalidAgentReplyTimeWindow = errors.New("agent_reply_time_window must be greater than 0")
var ErrInvalidBusinessHours = errors.New("invalid business hours")

type InboxCredentials struct {
	Inbox      *model.Inbox
	ChannelAPI *model.ChannelAPI
	APIToken   string
	HMACToken  string
	Secret     string
}

type InboxService struct {
	pool                   *pgxpool.Pool
	inboxRepo              *repo.InboxRepo
	channelAPIRepo         *repo.ChannelAPIRepo
	inboxAgentRepo         *repo.InboxAgentRepo
	inboxBusinessHoursRepo *repo.InboxBusinessHoursRepo
	cipher                 *crypto.Cipher
}

func NewInboxService(pool *pgxpool.Pool, inboxRepo *repo.InboxRepo, channelAPIRepo *repo.ChannelAPIRepo, inboxAgentRepo *repo.InboxAgentRepo, inboxBusinessHoursRepo *repo.InboxBusinessHoursRepo, cipher *crypto.Cipher) *InboxService {
	return &InboxService{
		pool:                   pool,
		inboxRepo:              inboxRepo,
		channelAPIRepo:         channelAPIRepo,
		inboxAgentRepo:         inboxAgentRepo,
		inboxBusinessHoursRepo: inboxBusinessHoursRepo,
		cipher:                 cipher,
	}
}

type ProvisionAPIInput struct {
	Name                 string
	WebhookURL           string
	HMACMandatory        bool
	AdditionalAttributes map[string]any
}

func (s *InboxService) Provision(ctx context.Context, accountID int64, name string) (*InboxCredentials, error) {
	return s.ProvisionAPI(ctx, accountID, ProvisionAPIInput{Name: name})
}

func (s *InboxService) ProvisionAPI(ctx context.Context, accountID int64, in ProvisionAPIInput) (*InboxCredentials, error) {
	if err := validateAgentReplyTimeWindow(in.AdditionalAttributes); err != nil {
		return nil, err
	}

	identifier, err := generateRandomToken(48)
	if err != nil {
		return nil, err
	}
	apiTokenPlaintext, err := generateRandomToken(48)
	if err != nil {
		return nil, err
	}
	hmacTokenPlaintext, err := generateRandomToken(48)
	if err != nil {
		return nil, err
	}
	secretPlaintext, err := generateRandomToken(48)
	if err != nil {
		return nil, err
	}

	hmacCiphertext, err := s.cipher.Encrypt(hmacTokenPlaintext)
	if err != nil {
		return nil, fmt.Errorf("encrypt hmac token: %w", err)
	}

	channelAPI := &model.ChannelAPI{
		AccountID:            accountID,
		WebhookURL:           in.WebhookURL,
		Identifier:           identifier,
		HMACToken:            hmacCiphertext,
		APITokenHash:         crypto.HashLookup(apiTokenPlaintext),
		HMACMandatory:        in.HMACMandatory,
		Secret:               secretPlaintext,
		AdditionalAttributes: in.AdditionalAttributes,
	}

	if err := s.channelAPIRepo.Create(ctx, channelAPI); err != nil {
		return nil, err
	}

	inbox := &model.Inbox{
		AccountID:   accountID,
		ChannelID:   channelAPI.ID,
		Name:        in.Name,
		ChannelType: "Channel::Api",
	}
	if err := s.inboxRepo.Create(ctx, inbox); err != nil {
		return nil, err
	}

	return &InboxCredentials{
		Inbox:      inbox,
		ChannelAPI: channelAPI,
		APIToken:   apiTokenPlaintext,
		HMACToken:  hmacTokenPlaintext,
		Secret:     secretPlaintext,
	}, nil
}

type UpdateAPIInput struct {
	WebhookURL           string
	HMACMandatory        bool
	AdditionalAttributes map[string]any
}

func (s *InboxService) GetChannelAPIEditable(ctx context.Context, inboxID, accountID int64) (*model.ChannelAPI, error) {
	ch, err := s.channelAPIRepo.FindByInboxID(ctx, inboxID)
	if err != nil {
		return nil, err
	}
	if ch.AccountID != accountID {
		return nil, repo.ErrChannelAPINotFound
	}
	return ch, nil
}

func (s *InboxService) UpdateChannelAPIEditable(ctx context.Context, inboxID, accountID int64, in UpdateAPIInput) (*model.ChannelAPI, error) {
	if err := validateAgentReplyTimeWindow(in.AdditionalAttributes); err != nil {
		return nil, err
	}

	ch, err := s.channelAPIRepo.FindByInboxID(ctx, inboxID)
	if err != nil {
		return nil, err
	}
	if ch.AccountID != accountID {
		return nil, repo.ErrChannelAPINotFound
	}

	ch.WebhookURL = in.WebhookURL
	ch.HMACMandatory = in.HMACMandatory
	ch.AdditionalAttributes = in.AdditionalAttributes

	if err := s.channelAPIRepo.UpdateEditable(ctx, ch); err != nil {
		return nil, err
	}
	return ch, nil
}

func (s *InboxService) RotateAPIToken(ctx context.Context, inboxID, accountID int64) (*model.ChannelAPI, string, string, error) {
	ch, err := s.channelAPIRepo.FindByInboxID(ctx, inboxID)
	if err != nil {
		return nil, "", "", err
	}
	if ch.AccountID != accountID {
		return nil, "", "", repo.ErrChannelAPINotFound
	}

	newIdentifier, err := generateRandomToken(48)
	if err != nil {
		return nil, "", "", err
	}
	newAPIToken, err := generateRandomToken(48)
	if err != nil {
		return nil, "", "", err
	}
	newSecret, err := generateRandomToken(48)
	if err != nil {
		return nil, "", "", err
	}

	ch.Identifier = newIdentifier
	ch.APITokenHash = crypto.HashLookup(newAPIToken)
	ch.Secret = newSecret

	if err := s.channelAPIRepo.RotateToken(ctx, ch); err != nil {
		return nil, "", "", err
	}
	return ch, newAPIToken, newSecret, nil
}

func validateAgentReplyTimeWindow(attrs map[string]any) error {
	v, ok := attrs["agent_reply_time_window"]
	if !ok {
		return nil
	}
	switch n := v.(type) {
	case float64:
		if n <= 0 {
			return ErrInvalidAgentReplyTimeWindow
		}
	case int:
		if n <= 0 {
			return ErrInvalidAgentReplyTimeWindow
		}
	case int64:
		if n <= 0 {
			return ErrInvalidAgentReplyTimeWindow
		}
	default:
		return ErrInvalidAgentReplyTimeWindow
	}
	return nil
}

func (s *InboxService) ListByAccount(ctx context.Context, accountID int64) ([]model.Inbox, error) {
	return s.inboxRepo.ListByAccount(ctx, accountID)
}

var channelTypeTable = map[string]string{
	"Channel::Api":          "channels_api",
	"Channel::Whatsapp":     "channels_whatsapp",
	"Channel::Sms":          "channels_sms",
	"Channel::Instagram":    "channels_instagram",
	"Channel::FacebookPage": "channels_facebook_page",
	"Channel::WebWidget":    "channels_web_widget",
	"Channel::Telegram":     "channels_telegram",
	"Channel::Line":         "channels_line",
	"Channel::Tiktok":       "channels_tiktok",
	"Channel::Twilio":       "channels_twilio",
	"Channel::Twitter":      "channels_twitter",
	"Channel::Email":        "channels_email",
}

func (s *InboxService) DeleteInbox(ctx context.Context, inboxID, accountID int64) error {
	inbox, err := s.inboxRepo.FindByID(ctx, inboxID, accountID)
	if err != nil {
		return err
	}

	table, ok := channelTypeTable[inbox.ChannelType]
	if !ok {
		return fmt.Errorf("unknown channel type: %s", inbox.ChannelType)
	}

	if _, err := s.pool.Exec(ctx, fmt.Sprintf("DELETE FROM %s WHERE id = $1", table), inbox.ChannelID); err != nil {
		return fmt.Errorf("delete channel: %w", err)
	}

	return s.inboxRepo.Delete(ctx, inboxID, accountID)
}

func (s *InboxService) FindByID(ctx context.Context, id, accountID int64) (*model.Inbox, error) {
	return s.inboxRepo.FindByID(ctx, id, accountID)
}

func (s *InboxService) DecryptHMACToken(ciphertext string) (string, error) {
	return s.cipher.Decrypt(ciphertext)
}

func generateRandomToken(numBytes int) (string, error) {
	b := make([]byte, numBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func (s *InboxService) ListInboxAgents(ctx context.Context, inboxID, accountID int64) ([]model.InboxAgent, error) {
	return s.inboxAgentRepo.ListByInbox(ctx, inboxID, accountID)
}

func (s *InboxService) SetInboxAgents(ctx context.Context, inboxID, accountID int64, userIDs []int64) error {
	return s.inboxAgentRepo.SetByInbox(ctx, inboxID, accountID, userIDs)
}

func (s *InboxService) UpdateName(ctx context.Context, id, accountID int64, name string) error {
	return s.inboxRepo.UpdateName(ctx, id, accountID, name)
}

func (s *InboxService) GetBusinessHours(ctx context.Context, inboxID, accountID int64) (*model.InboxBusinessHours, error) {
	inbox, err := s.inboxRepo.FindByID(ctx, inboxID, accountID)
	if err != nil {
		return nil, err
	}

	hours, err := s.inboxBusinessHoursRepo.FindByInbox(ctx, inboxID, accountID)
	if err != nil {
		if repo.IsErrNotFound(err) {
			return &model.InboxBusinessHours{
				AccountID: accountID,
				InboxID:   inbox.ID,
				Timezone:  "America/Sao_Paulo",
				Schedule:  DefaultBusinessHoursSchedule(),
			}, nil
		}
		return nil, err
	}
	return hours, nil
}

func (s *InboxService) UpdateBusinessHours(ctx context.Context, inboxID, accountID int64, timezone string, schedule map[string]model.BusinessHoursSlot) (*model.InboxBusinessHours, error) {
	inbox, err := s.inboxRepo.FindByID(ctx, inboxID, accountID)
	if err != nil {
		return nil, err
	}
	if timezone == "" {
		return nil, ErrInvalidBusinessHours
	}
	normalized, err := NormalizeBusinessHoursSchedule(schedule)
	if err != nil {
		return nil, err
	}

	hours := &model.InboxBusinessHours{
		AccountID: accountID,
		InboxID:   inbox.ID,
		Timezone:  timezone,
		Schedule:  normalized,
	}
	if err := s.inboxBusinessHoursRepo.Upsert(ctx, hours); err != nil {
		return nil, err
	}
	return hours, nil
}

func DefaultBusinessHoursSchedule() map[string]model.BusinessHoursSlot {
	days := []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"}
	schedule := make(map[string]model.BusinessHoursSlot, len(days))
	for _, day := range days {
		enabled := day != "saturday" && day != "sunday"
		schedule[day] = model.BusinessHoursSlot{
			Enabled:     enabled,
			OpenHour:    9,
			OpenMinute:  0,
			CloseHour:   18,
			CloseMinute: 0,
		}
	}
	return schedule
}

func NormalizeBusinessHoursSchedule(schedule map[string]model.BusinessHoursSlot) (map[string]model.BusinessHoursSlot, error) {
	if schedule == nil {
		return nil, ErrInvalidBusinessHours
	}
	defaults := DefaultBusinessHoursSchedule()
	for day, slot := range defaults {
		saved, ok := schedule[day]
		if !ok {
			schedule[day] = slot
			continue
		}
		if saved.OpenHour < 0 || saved.OpenHour > 23 || saved.CloseHour < 0 || saved.CloseHour > 23 ||
			saved.OpenMinute < 0 || saved.OpenMinute > 59 || saved.CloseMinute < 0 || saved.CloseMinute > 59 {
			return nil, ErrInvalidBusinessHours
		}
		schedule[day] = saved
	}
	return schedule, nil
}
