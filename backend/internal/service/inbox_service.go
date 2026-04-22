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
)

// ErrInvalidAgentReplyTimeWindow is returned when callers pass
// additional_attributes.agent_reply_time_window <= 0. The Chatwoot contract
// requires a strictly positive integer (minutes).
var ErrInvalidAgentReplyTimeWindow = errors.New("agent_reply_time_window must be greater than 0")
var ErrInvalidBusinessHours = errors.New("invalid business hours")

// InboxCredentials carries plaintext secrets returned ONCE at inbox creation.
// Plaintexts are never persisted (api_token is stored as SHA-256 hash,
// hmac_token as AES-GCM ciphertext) and never returned again.
type InboxCredentials struct {
	Inbox      *model.Inbox
	ChannelAPI *model.ChannelAPI
	ApiToken   string
	HmacToken  string
	Secret     string
}

type InboxService struct {
	inboxRepo              *repo.InboxRepo
	channelApiRepo         *repo.ChannelAPIRepo
	inboxAgentRepo         *repo.InboxAgentRepo
	inboxBusinessHoursRepo *repo.InboxBusinessHoursRepo
	cipher                 *crypto.Cipher
}

func NewInboxService(inboxRepo *repo.InboxRepo, channelApiRepo *repo.ChannelAPIRepo, inboxAgentRepo *repo.InboxAgentRepo, inboxBusinessHoursRepo *repo.InboxBusinessHoursRepo, cipher *crypto.Cipher) *InboxService {
	return &InboxService{
		inboxRepo:              inboxRepo,
		channelApiRepo:         channelApiRepo,
		inboxAgentRepo:         inboxAgentRepo,
		inboxBusinessHoursRepo: inboxBusinessHoursRepo,
		cipher:                 cipher,
	}
}

// ProvisionAPIInput carries the editable fields accepted at creation time.
// Any unset field (zero value) means "use default": webhook_url stays NULL,
// hmac_mandatory defaults to false, additional_attributes defaults to {}.
type ProvisionAPIInput struct {
	Name                 string
	WebhookURL           string
	HmacMandatory        bool
	AdditionalAttributes map[string]any
}

func (s *InboxService) Provision(ctx context.Context, accountID int64, name string) (*InboxCredentials, error) {
	return s.ProvisionAPI(ctx, accountID, ProvisionAPIInput{Name: name})
}

// ProvisionAPI creates a Channel::Api inbox with the supplied editable
// attributes. Rejects invalid agent_reply_time_window up-front to avoid a
// half-built channel record.
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

	channelApi := &model.ChannelAPI{
		AccountID:            accountID,
		WebhookURL:           in.WebhookURL,
		Identifier:           identifier,
		HmacToken:            hmacCiphertext,
		ApiTokenHash:         crypto.HashLookup(apiTokenPlaintext),
		HmacMandatory:        in.HmacMandatory,
		Secret:               secretPlaintext,
		AdditionalAttributes: in.AdditionalAttributes,
	}

	if err := s.channelApiRepo.Create(ctx, channelApi); err != nil {
		return nil, err
	}

	inbox := &model.Inbox{
		AccountID:   accountID,
		ChannelID:   channelApi.ID,
		Name:        in.Name,
		ChannelType: "Channel::Api",
	}
	if err := s.inboxRepo.Create(ctx, inbox); err != nil {
		return nil, err
	}

	return &InboxCredentials{
		Inbox:      inbox,
		ChannelAPI: channelApi,
		ApiToken:   apiTokenPlaintext,
		HmacToken:  hmacTokenPlaintext,
		Secret:     secretPlaintext,
	}, nil
}

// UpdateAPIInput is the whitelist of fields accepted by PUT /inboxes/:id for
// Channel::Api. Anything else is ignored (belt-and-suspenders against DTO
// drift).
type UpdateAPIInput struct {
	WebhookURL           string
	HmacMandatory        bool
	AdditionalAttributes map[string]any
}

func (s *InboxService) GetChannelAPIEditable(ctx context.Context, inboxID, accountID int64) (*model.ChannelAPI, error) {
	ch, err := s.channelApiRepo.FindByInboxID(ctx, inboxID)
	if err != nil {
		return nil, err
	}
	if ch.AccountID != accountID {
		return nil, repo.ErrChannelAPINotFound
	}
	return ch, nil
}

// UpdateChannelAPIEditable updates the editable fields of a Channel::Api. The
// inbox is located from the channel row (by inboxID) so the account_id scope
// is enforced before any write.
func (s *InboxService) UpdateChannelAPIEditable(ctx context.Context, inboxID, accountID int64, in UpdateAPIInput) (*model.ChannelAPI, error) {
	if err := validateAgentReplyTimeWindow(in.AdditionalAttributes); err != nil {
		return nil, err
	}

	ch, err := s.channelApiRepo.FindByInboxID(ctx, inboxID)
	if err != nil {
		return nil, err
	}
	if ch.AccountID != accountID {
		return nil, repo.ErrChannelAPINotFound
	}

	ch.WebhookURL = in.WebhookURL
	ch.HmacMandatory = in.HmacMandatory
	ch.AdditionalAttributes = in.AdditionalAttributes

	if err := s.channelApiRepo.UpdateEditable(ctx, ch); err != nil {
		return nil, err
	}
	return ch, nil
}

// RotateAPIToken issues a fresh identifier + api_token for an existing
// Channel::Api inbox, invalidating the previous token. Returns the plaintext
// api_token once — callers MUST surface it to the integrator and drop it.
func (s *InboxService) RotateAPIToken(ctx context.Context, inboxID, accountID int64) (*model.ChannelAPI, string, string, error) {
	ch, err := s.channelApiRepo.FindByInboxID(ctx, inboxID)
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
	newApiToken, err := generateRandomToken(48)
	if err != nil {
		return nil, "", "", err
	}
	newSecret, err := generateRandomToken(48)
	if err != nil {
		return nil, "", "", err
	}

	ch.Identifier = newIdentifier
	ch.ApiTokenHash = crypto.HashLookup(newApiToken)
	ch.Secret = newSecret

	if err := s.channelApiRepo.RotateToken(ctx, ch); err != nil {
		return nil, "", "", err
	}
	return ch, newApiToken, newSecret, nil
}

// validateAgentReplyTimeWindow enforces the Chatwoot contract: when the key
// is present, its value MUST parse as a positive integer (minutes). Absence
// is fine (the field is optional).
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

func (s *InboxService) GetByID(ctx context.Context, id, accountID int64) (*model.Inbox, error) {
	return s.inboxRepo.FindByID(ctx, id, accountID)
}

// DecryptHmacToken returns the plaintext HMAC key from the stored ciphertext.
// Callers must not log or leak the result; it is a per-channel signing secret.
func (s *InboxService) DecryptHmacToken(ciphertext string) (string, error) {
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
