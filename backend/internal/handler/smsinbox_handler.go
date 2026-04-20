package handler

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"

	"backend/internal/channel/sms"
	"backend/internal/crypto"
	"backend/internal/dto"
	"backend/internal/logger"
	"backend/internal/model"
	"backend/internal/repo"
)

type SMSInboxHandler struct {
	channelSMSRepo *repo.ChannelSMSRepo
	inboxRepo      *repo.InboxRepo
	registry       *sms.Registry
	cipher         *crypto.Cipher
	baseURL        string
}

func NewSMSInboxHandler(
	channelSMSRepo *repo.ChannelSMSRepo,
	inboxRepo *repo.InboxRepo,
	registry *sms.Registry,
	cipher *crypto.Cipher,
	baseURL string,
) *SMSInboxHandler {
	return &SMSInboxHandler{
		channelSMSRepo: channelSMSRepo,
		inboxRepo:      inboxRepo,
		registry:       registry,
		cipher:         cipher,
		baseURL:        baseURL,
	}
}

func (h *SMSInboxHandler) Provision(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	var req dto.CreateSMSInboxReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	providerConfig, err := h.buildProviderConfig(req.Provider, req.ProviderConfig)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp(err.Error(), "invalid provider config"))
	}

	prov, err := h.registry.Get(req.Provider)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("unsupported_provider", "provider not supported"))
	}

	if err := prov.ValidateCredentials(c.Context(), *providerConfig); err != nil {
		if sms.IsAuthError(err) {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("invalid_credentials", "credential validation failed"))
		}
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("invalid_credentials", err.Error()))
	}

	configJSON, err := providerConfig.Serialize()
	if err != nil {
		logger.Error().Str("component", "sms-inbox").Err(err).Msg("failed to serialize sms provider config")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to serialize config"))
	}

	configCiphertext, err := h.cipher.Encrypt(configJSON)
	if err != nil {
		logger.Error().Str("component", "sms-inbox").Err(err).Msg("failed to encrypt sms provider config")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to encrypt config"))
	}

	webhookIdentifier, err := generateWebhookIdentifier()
	if err != nil {
		logger.Error().Str("component", "sms-inbox").Err(err).Msg("failed to generate sms webhook identifier")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to generate webhook identifier"))
	}

	ch := &model.ChannelSMS{
		AccountID:                accountID,
		Provider:                 req.Provider,
		PhoneNumber:              req.PhoneNumber,
		WebhookIdentifier:        webhookIdentifier,
		ProviderConfigCiphertext: configCiphertext,
	}

	if req.ProviderConfig.Twilio != nil && req.ProviderConfig.Twilio.MessagingServiceSID != "" {
		ch.MessagingServiceSid = &req.ProviderConfig.Twilio.MessagingServiceSID
	}

	if err := h.channelSMSRepo.Create(c.Context(), ch); err != nil {
		logger.Error().Str("component", "sms-inbox").Err(err).Msg("failed to create sms channel")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to create sms channel"))
	}

	inbox := &model.Inbox{
		AccountID:   accountID,
		ChannelID:   ch.ID,
		Name:        req.Name,
		ChannelType: "Channel::Sms",
	}
	if err := h.inboxRepo.Create(c.Context(), inbox); err != nil {
		logger.Error().Str("component", "sms-inbox").Err(err).Msg("failed to create inbox for sms")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to create inbox"))
	}

	if err := h.channelSMSRepo.UpdateInboxID(c.Context(), ch.ID, inbox.ID); err != nil {
		logger.Error().Str("component", "sms-inbox").Err(err).Msg("failed to link sms channel to inbox")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to link channel to inbox"))
	}

	webhookBase := fmt.Sprintf("%s/webhooks/sms/%s/%s", h.baseURL, req.Provider, webhookIdentifier)

	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResp(dto.SMSInboxResp{
		InboxResp: inboxModelToResp(inbox),
		Channel: dto.SMSChannelResp{
			ID:                  ch.ID,
			AccountID:           ch.AccountID,
			Provider:            ch.Provider,
			PhoneNumber:         ch.PhoneNumber,
			WebhookIdentifier:   ch.WebhookIdentifier,
			MessagingServiceSid: ch.MessagingServiceSid,
			RequiresReauth:      ch.RequiresReauth,
			CreatedAt:           ch.CreatedAt.Format(time.RFC3339),
			UpdatedAt:           ch.UpdatedAt.Format(time.RFC3339),
		},
		WebhookURLs: &dto.SMSWebhookURLs{
			Primary: webhookBase,
			Status:  webhookBase + "/status",
		},
	}))
}

func (h *SMSInboxHandler) buildProviderConfig(provider string, cfg *dto.SMSProviderConfig) (*sms.ProviderConfig, error) {
	pc := &sms.ProviderConfig{}

	switch provider {
	case "twilio":
		if cfg.Twilio == nil {
			return nil, fmt.Errorf("unsupported_provider: twilio config required")
		}
		pc.Twilio = &sms.TwilioConfig{
			AccountSID:          cfg.Twilio.AccountSID,
			AuthToken:           cfg.Twilio.AuthToken,
			MessagingServiceSID: cfg.Twilio.MessagingServiceSID,
		}
	case "bandwidth":
		if cfg.Bandwidth == nil {
			return nil, fmt.Errorf("unsupported_provider: bandwidth config required")
		}
		pc.Bandwidth = &sms.BandwidthConfig{
			AccountID:     cfg.Bandwidth.AccountID,
			ApplicationID: cfg.Bandwidth.ApplicationID,
			BasicAuthUser: cfg.Bandwidth.BasicAuthUser,
			BasicAuthPass: cfg.Bandwidth.BasicAuthPass,
		}
	case "zenvia":
		if cfg.Zenvia == nil {
			return nil, fmt.Errorf("unsupported_provider: zenvia config required")
		}
		pc.Zenvia = &sms.ZenviaConfig{
			APIToken: cfg.Zenvia.APIToken,
		}
	default:
		return nil, fmt.Errorf("unsupported_provider: %s", provider)
	}

	return pc, nil
}

func generateWebhookIdentifier() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate webhook identifier: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
