package handler

import (
	"fmt"

	"github.com/gofiber/fiber/v2"

	emailch "backend/internal/channel/email"
	"backend/internal/crypto"
	"backend/internal/dto"
	"backend/internal/logger"
	"backend/internal/model"
	"backend/internal/repo"
	"backend/internal/service"
)

// OAuthCallbackHandler handles OAuth provider callbacks for email channel creation.
type OAuthCallbackHandler struct {
	channelEmailRepo *repo.ChannelEmailRepo
	inboxRepo        *repo.InboxRepo
	inboxSvc         *service.InboxService
	cipher           *crypto.Cipher
	frontendURL      string
}

func NewOAuthCallbackHandler(
	channelEmailRepo *repo.ChannelEmailRepo,
	inboxRepo *repo.InboxRepo,
	inboxSvc *service.InboxService,
	cipher *crypto.Cipher,
	frontendURL string,
) *OAuthCallbackHandler {
	return &OAuthCallbackHandler{
		channelEmailRepo: channelEmailRepo,
		inboxRepo:        inboxRepo,
		inboxSvc:         inboxSvc,
		cipher:           cipher,
		frontendURL:      frontendURL,
	}
}

// GoogleCallback handles GET /oauth/google/callback?code=...&state=...
//
//	@Summary     Google OAuth callback for email inbox
//	@Description Exchanges the OAuth code for tokens, creates the email channel, redirects to frontend
//	@Tags        oauth
//	@Param       code  query string true "Authorization code"
//	@Param       state query string true "CSRF state token"
//	@Router      /oauth/google/callback [get]
func (h *OAuthCallbackHandler) GoogleCallback(c *fiber.Ctx) error {
	return h.handleCallback(c, "google")
}

// MicrosoftCallback handles GET /oauth/microsoft/callback?code=...&state=...
//
//	@Summary     Microsoft OAuth callback for email inbox
//	@Description Exchanges the OAuth code for tokens, creates the email channel, redirects to frontend
//	@Tags        oauth
//	@Param       code  query string true "Authorization code"
//	@Param       state query string true "CSRF state token"
//	@Router      /oauth/microsoft/callback [get]
func (h *OAuthCallbackHandler) MicrosoftCallback(c *fiber.Ctx) error {
	return h.handleCallback(c, "microsoft")
}

func (h *OAuthCallbackHandler) handleCallback(c *fiber.Ctx, provider string) error {
	code := c.Query("code")
	state := c.Query("state")
	if code == "" || state == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "missing code or state"))
	}

	pending, ok := emailch.GlobalOAuthPending.Get(state)
	if !ok {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid or expired state"))
	}

	ctx := c.Context()

	var tokens *emailch.OAuthTokens
	var err error
	switch provider {
	case "google":
		tokens, err = emailch.GoogleExchangeCode(ctx, code)
	case "microsoft":
		tokens, err = emailch.MicrosoftExchangeCode(ctx, code)
	default:
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "unknown provider"))
	}
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(dto.ErrorResp("OAuth Error", err.Error()))
	}

	plainJSON, err := emailch.MarshalTokens(tokens)
	if err != nil {
		logger.Error().Str("component", "oauth").Err(err).Msg("failed to marshal OAuth tokens")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "marshal tokens"))
	}
	configCiphertext, err := h.cipher.Encrypt(plainJSON)
	if err != nil {
		logger.Error().Str("component", "oauth").Err(err).Msg("failed to encrypt OAuth tokens")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "encrypt tokens"))
	}

	ch := &model.ChannelEmail{
		AccountID:          pending.AccountID,
		Email:              fmt.Sprintf("%s@oauth.elodesk.io", provider),
		Name:               pending.InboxName,
		Provider:           provider,
		ProviderConfig:     &configCiphertext,
		VerifiedForSending: true,
		ImapEnabled:        true,
	}
	if err := h.channelEmailRepo.Create(ctx, ch); err != nil {
		logger.Error().Str("component", "oauth").Err(err).Msg("failed to create email channel")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "create channel"))
	}

	inbox := &model.Inbox{
		AccountID:   pending.AccountID,
		ChannelID:   ch.ID,
		Name:        pending.InboxName,
		ChannelType: "Channel::Email",
	}
	if err := h.inboxRepo.Create(ctx, inbox); err != nil {
		logger.Error().Str("component", "oauth").Err(err).Msg("failed to create inbox")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "create inbox"))
	}

	redirectURL := fmt.Sprintf("%s/app/accounts/%d/settings/inboxes/%d", h.frontendURL, pending.AccountID, inbox.ID)
	return c.Redirect(redirectURL, fiber.StatusFound)
}
