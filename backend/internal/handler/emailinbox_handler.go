package handler

import (
	"crypto/rand"
	"encoding/hex"
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

// EmailInboxHandler provisions and reads email-channel inboxes.
type EmailInboxHandler struct {
	channelEmailRepo *repo.ChannelEmailRepo
	inboxRepo        *repo.InboxRepo
	inboxSvc         *service.InboxService
	cipher           *crypto.Cipher
	frontendURL      string
}

func NewEmailInboxHandler(
	channelEmailRepo *repo.ChannelEmailRepo,
	inboxRepo *repo.InboxRepo,
	inboxSvc *service.InboxService,
	cipher *crypto.Cipher,
	frontendURL string,
) *EmailInboxHandler {
	return &EmailInboxHandler{
		channelEmailRepo: channelEmailRepo,
		inboxRepo:        inboxRepo,
		inboxSvc:         inboxSvc,
		cipher:           cipher,
		frontendURL:      frontendURL,
	}
}

// Create handles POST /api/v1/accounts/:aid/inboxes/email
//
//	@Summary     Create an email inbox
//	@Description For provider=generic creates IMAP/SMTP channel immediately.
//	             For provider=google|microsoft returns an OAuth authorize URL.
//	@Tags        inboxes
//	@Security    BearerAuth
//	@Param       aid  path int                  true "Account ID"
//	@Param       body body dto.CreateEmailInboxReq true "Request"
//	@Router      /api/v1/accounts/{aid}/inboxes/email [post]
func (h *EmailInboxHandler) Create(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	var req dto.CreateEmailInboxReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	ctx := c.Context()

	if req.Provider == "google" || req.Provider == "microsoft" {
		b := make([]byte, 32)
		_, _ = rand.Read(b)
		state := hex.EncodeToString(b)
		emailch.GlobalOAuthPending.Set(state, emailch.PendingState{
			AccountID: accountID,
			InboxName: req.Name,
			Provider:  req.Provider,
		})

		var authURL string
		if req.Provider == "google" {
			authURL = emailch.GoogleAuthURL(state)
		} else {
			authURL = emailch.MicrosoftAuthURL(state)
		}

		return c.Status(fiber.StatusOK).JSON(dto.SuccessResp(dto.OAuthRedirectResp{
			AuthorizeURL: authURL,
		}))
	}

	// provider = generic: create IMAP/SMTP channel immediately
	if req.ImapPassword == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Validation Error", "imapPassword required for generic provider"))
	}

	imapPassCiphertext, err := h.cipher.Encrypt(req.ImapPassword)
	if err != nil {
		logger.Error().Str("component", "email-inbox").Err(err).Msg("encrypt imap password failed")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "encrypt imap password"))
	}
	smtpPassCiphertext, err := h.cipher.Encrypt(req.SmtpPassword)
	if err != nil {
		logger.Error().Str("component", "email-inbox").Err(err).Msg("encrypt smtp password failed")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "encrypt smtp password"))
	}

	imapAddr := req.ImapAddress
	imapPort := req.ImapPort
	imapLogin := req.ImapLogin
	smtpAddr := req.SmtpAddress
	smtpPort := req.SmtpPort
	smtpLogin := req.SmtpLogin

	ch := &model.ChannelEmail{
		AccountID:              accountID,
		Email:                  req.Email,
		Name:                   req.Name,
		Provider:               "generic",
		ImapAddress:            &imapAddr,
		ImapPort:               &imapPort,
		ImapLogin:              &imapLogin,
		ImapPasswordCiphertext: &imapPassCiphertext,
		ImapEnableSSL:          req.ImapEnableSSL,
		ImapEnabled:            req.ImapEnabled,
		SmtpAddress:            &smtpAddr,
		SmtpPort:               &smtpPort,
		SmtpLogin:              &smtpLogin,
		SmtpPasswordCiphertext: &smtpPassCiphertext,
		SmtpEnableSSL:          req.SmtpEnableSSL,
		VerifiedForSending:     req.SmtpAddress != "" && req.SmtpPassword != "",
	}
	if err := h.channelEmailRepo.Create(ctx, ch); err != nil {
		logger.Error().Str("component", "email-inbox").Err(err).Msg("failed to create email channel")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", fmt.Sprintf("create channel: %v", err)))
	}

	inbox := &model.Inbox{
		AccountID:   accountID,
		ChannelID:   ch.ID,
		Name:        req.Name,
		ChannelType: "Channel::Email",
	}
	if err := h.inboxRepo.Create(ctx, inbox); err != nil {
		logger.Error().Str("component", "email-inbox").Err(err).Msg("failed to create inbox for email")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "create inbox"))
	}

	resp := dto.CreateEmailInboxResp{
		InboxResp:        inboxModelToResp(inbox),
		EmailChannelResp: channelEmailToResp(ch),
	}
	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResp(resp))
}

// GetEmailChannel handles GET /api/v1/accounts/:aid/inboxes/email/:id
//
//	@Summary     Get email channel details
//	@Description Returns email channel metadata — never exposes passwords or tokens.
//	@Tags        inboxes
//	@Security    BearerAuth
//	@Param       aid path int true "Account ID"
//	@Param       id  path int true "Inbox ID"
//	@Router      /api/v1/accounts/{aid}/inboxes/email/{id} [get]
func (h *EmailInboxHandler) GetEmailChannel(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid inbox id"))
	}

	ch, err := h.channelEmailRepo.FindByInboxID(c.Context(), int64(id))
	if err != nil {
		return handleNotFound(c, err)
	}
	if ch.AccountID != accountID {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", "not found"))
	}

	return c.JSON(dto.SuccessResp(channelEmailToResp(ch)))
}

func channelEmailToResp(ch *model.ChannelEmail) dto.EmailChannelResp {
	return dto.EmailChannelResp{
		Email:              ch.Email,
		Provider:           ch.Provider,
		ImapAddress:        ch.ImapAddress,
		ImapPort:           ch.ImapPort,
		ImapLogin:          ch.ImapLogin,
		ImapEnableSSL:      ch.ImapEnableSSL,
		ImapEnabled:        ch.ImapEnabled,
		SmtpAddress:        ch.SmtpAddress,
		SmtpPort:           ch.SmtpPort,
		SmtpLogin:          ch.SmtpLogin,
		SmtpEnableSSL:      ch.SmtpEnableSSL,
		VerifiedForSending: ch.VerifiedForSending,
		RequiresReauth:     ch.RequiresReauth,
		EmailCreatedAt:     ch.CreatedAt,
	}
}
