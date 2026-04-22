package handler

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"

	"backend/internal/channel"
	tiktokchan "backend/internal/channel/tiktok"
	appcrypto "backend/internal/crypto"
	"backend/internal/dto"
	"backend/internal/logger"
	"backend/internal/model"
	"backend/internal/repo"
)

const (
	tiktokStateKeyPrefix = "elodesk:tiktok:oauth_state:"
	tiktokStateTTL       = 10 * time.Minute
)

type TiktokHandler struct {
	tiktokRepo       *repo.ChannelTiktokRepo
	inboxRepo        *repo.InboxRepo
	contactRepo      *repo.ContactRepo
	contactInboxRepo *repo.ContactInboxRepo
	conversationRepo *repo.ConversationRepo
	messageRepo      *repo.MessageRepo
	cipher           *appcrypto.Cipher
	dedup            *channel.DedupLock
	asynqClient      *asynq.Client
	oauth            *tiktokchan.OAuthClient
	appSecret        string
	redisClient      redis.Cmdable
	featureEnabled   bool
}

func NewTiktokHandler(
	tiktokRepo *repo.ChannelTiktokRepo,
	inboxRepo *repo.InboxRepo,
	contactRepo *repo.ContactRepo,
	contactInboxRepo *repo.ContactInboxRepo,
	conversationRepo *repo.ConversationRepo,
	messageRepo *repo.MessageRepo,
	cipher *appcrypto.Cipher,
	dedup *channel.DedupLock,
	asynqClient *asynq.Client,
	oauth *tiktokchan.OAuthClient,
	appSecret string,
	redisClient redis.Cmdable,
	featureEnabled bool,
) *TiktokHandler {
	return &TiktokHandler{
		tiktokRepo:       tiktokRepo,
		inboxRepo:        inboxRepo,
		contactRepo:      contactRepo,
		contactInboxRepo: contactInboxRepo,
		conversationRepo: conversationRepo,
		messageRepo:      messageRepo,
		cipher:           cipher,
		dedup:            dedup,
		asynqClient:      asynqClient,
		oauth:            oauth,
		appSecret:        appSecret,
		redisClient:      redisClient,
		featureEnabled:   featureEnabled,
	}
}

// Authorize handles POST /api/v1/accounts/:aid/inboxes/tiktok/authorize and
// returns a redirect URL for the TikTok consent screen.
func (h *TiktokHandler) Authorize(c *fiber.Ctx) error {
	if !h.featureEnabled {
		return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResp("feature_disabled", "TikTok channel is disabled"))
	}
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	nonce, err := randomHex(32)
	if err != nil {
		logger.Error().Str("component", "channel.tiktok").Err(err).Msg("failed to generate oauth state")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to generate state"))
	}
	state := fmt.Sprintf("%d:%s", accountID, nonce)
	if err := h.redisClient.Set(c.Context(), tiktokStateKeyPrefix+nonce, strconv.FormatInt(accountID, 10), tiktokStateTTL).Err(); err != nil {
		logger.Error().Str("component", "channel.tiktok").Err(err).Msg("failed to persist oauth state")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to persist state"))
	}

	return c.JSON(dto.SuccessResp(dto.TiktokAuthorizeResp{URL: h.oauth.AuthorizeURL(state)}))
}

// Callback handles GET /api/v1/accounts/tiktok/oauth/callback.
func (h *TiktokHandler) Callback(c *fiber.Ctx) error {
	if !h.featureEnabled {
		return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResp("feature_disabled", "TikTok channel is disabled"))
	}
	if errParam := c.Query("error"); errParam != "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("oauth_error", errParam))
	}
	code := c.Query("code")
	state := c.Query("state")
	if code == "" || state == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("bad_request", "missing code or state"))
	}

	accountID, nonce, err := parseTiktokState(state)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("bad_request", "invalid state"))
	}

	storedAccountStr, err := h.redisClient.Get(c.Context(), tiktokStateKeyPrefix+nonce).Result()
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("state_expired", "oauth state expired or unknown"))
	}
	storedAccountID, err := strconv.ParseInt(storedAccountStr, 10, 64)
	if err != nil || storedAccountID != accountID {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("state_mismatch", "oauth state mismatch"))
	}
	_ = h.redisClient.Del(c.Context(), tiktokStateKeyPrefix+nonce)

	ctx := c.Context()
	tokens, err := h.oauth.ExchangeCode(ctx, code)
	if err != nil {
		logger.Warn().Str("component", "channel.tiktok").Err(err).Msg("tiktok oauth exchange failed")
		return c.Status(fiber.StatusBadGateway).JSON(dto.ErrorResp("oauth_exchange_failed", "failed to exchange auth code"))
	}
	if !tiktokchan.ScopesGranted(tokens.Scope) {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("ungranted_scopes", "user did not grant all required scopes"))
	}

	api := tiktokchan.NewAPIClient()
	businessInfo, err := api.BusinessAccountDetails(ctx, tokens.AccessToken, tokens.OpenID)
	if err != nil {
		logger.Warn().Str("component", "channel.tiktok").Err(err).Msg("tiktok business details fetch failed")
		businessInfo = &tiktokchan.BusinessAccountData{}
	}

	accessCipher, err := h.cipher.Encrypt(tokens.AccessToken)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to encrypt access token"))
	}
	refreshCipher, err := h.cipher.Encrypt(tokens.RefreshToken)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to encrypt refresh token"))
	}

	displayName := businessInfo.DisplayName
	username := businessInfo.Username
	inboxName := displayName
	if inboxName == "" {
		inboxName = username
	}
	if inboxName == "" {
		inboxName = "TikTok " + tokens.OpenID
	}

	now := time.Now()
	ch := &model.ChannelTiktok{
		AccountID:              accountID,
		BusinessID:             tokens.OpenID,
		AccessTokenCiphertext:  accessCipher,
		RefreshTokenCiphertext: refreshCipher,
		ExpiresAt:              now.Add(time.Duration(tokens.ExpiresIn) * time.Second),
		RefreshTokenExpiresAt:  now.Add(time.Duration(tokens.RefreshTokenExpiresIn) * time.Second),
		DisplayName:            optionalString(displayName),
		Username:               optionalString(username),
	}
	if err := h.tiktokRepo.Create(ctx, ch); err != nil {
		logger.Error().Str("component", "channel.tiktok").Err(err).Msg("failed to create tiktok channel")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to create tiktok channel"))
	}

	inbox := &model.Inbox{
		AccountID:   accountID,
		ChannelID:   ch.ID,
		Name:        inboxName,
		ChannelType: string(channel.KindTiktok),
	}
	if err := h.inboxRepo.Create(ctx, inbox); err != nil {
		logger.Error().Str("component", "channel.tiktok").Err(err).Msg("failed to create tiktok inbox")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to create inbox"))
	}

	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResp(dto.TiktokInboxResp{
		InboxResp: inboxModelToResp(inbox),
		Channel: dto.TiktokChannelResp{
			ID:                    ch.ID,
			BusinessID:            ch.BusinessID,
			DisplayName:           ch.DisplayName,
			Username:              ch.Username,
			ExpiresAt:             ch.ExpiresAt,
			RefreshTokenExpiresAt: ch.RefreshTokenExpiresAt,
			RequiresReauth:        ch.RequiresReauth,
			CreatedAt:             ch.CreatedAt,
			UpdatedAt:             ch.UpdatedAt,
		},
	}))
}

// Receive handles POST /webhooks/tiktok/:business_id (signed webhook callbacks).
func (h *TiktokHandler) Receive(c *fiber.Ctx) error {
	businessID := c.Params("business_id")

	body := c.Body()
	signature := c.Get("Tiktok-Signature")
	if !tiktokchan.VerifySignature(h.appSecret, body, signature, time.Now()) {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "invalid signature"))
	}

	ch, err := h.tiktokRepo.FindByBusinessID(c.Context(), businessID)
	if err != nil {
		return c.SendStatus(fiber.StatusOK)
	}
	inbox, err := h.inboxRepo.FindByChannelID(c.Context(), ch.ID)
	if err != nil {
		return c.SendStatus(fiber.StatusOK)
	}

	if err := tiktokchan.ProcessWebhook(c.Context(), body, ch, inbox, h.dedup,
		h.contactRepo, h.contactInboxRepo, h.conversationRepo, h.messageRepo); err != nil {
		logger.Warn().Str("component", "channel.tiktok").Err(err).Msg("tiktok process webhook error")
	}
	return c.SendStatus(fiber.StatusOK)
}

// Delete handles DELETE /api/v1/accounts/:aid/inboxes/:id/tiktok.
// GetByInboxID handles GET /api/v1/accounts/:aid/inboxes/:id/tiktok.
func (h *TiktokHandler) GetByInboxID(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}
	inboxID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid inbox id"))
	}
	inbox, err := h.inboxRepo.FindByID(c.Context(), inboxID, accountID)
	if err != nil {
		return handleNotFound(c, err)
	}
	if inbox.ChannelType != string(channel.KindTiktok) {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "inbox is not a tiktok channel"))
	}

	ch, err := h.tiktokRepo.FindByID(c.Context(), inbox.ChannelID, accountID)
	if err != nil {
		return handleNotFound(c, err)
	}

	return c.JSON(dto.SuccessResp(dto.TiktokInboxResp{
		InboxResp: inboxModelToResp(inbox),
		Channel: dto.TiktokChannelResp{
			ID:                    ch.ID,
			BusinessID:            ch.BusinessID,
			DisplayName:           ch.DisplayName,
			Username:              ch.Username,
			ExpiresAt:             ch.ExpiresAt,
			RefreshTokenExpiresAt: ch.RefreshTokenExpiresAt,
			RequiresReauth:        ch.RequiresReauth,
			CreatedAt:             ch.CreatedAt,
			UpdatedAt:             ch.UpdatedAt,
		},
	}))
}

func (h *TiktokHandler) Delete(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}
	inboxID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid inbox id"))
	}
	inbox, err := h.inboxRepo.FindByID(c.Context(), inboxID, accountID)
	if err != nil {
		return handleNotFound(c, err)
	}
	if inbox.ChannelType != string(channel.KindTiktok) {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "inbox is not a tiktok channel"))
	}
	if err := h.tiktokRepo.Delete(c.Context(), inbox.ChannelID); err != nil {
		logger.Error().Str("component", "channel.tiktok").Err(err).Msg("failed to delete tiktok channel")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to delete tiktok channel"))
	}
	return c.JSON(dto.SuccessResp(nil))
}

func randomHex(bytesLen int) (string, error) {
	buf := make([]byte, bytesLen)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func parseTiktokState(state string) (int64, string, error) {
	for i := 0; i < len(state); i++ {
		if state[i] == ':' {
			accountID, err := strconv.ParseInt(state[:i], 10, 64)
			if err != nil {
				return 0, "", err
			}
			return accountID, state[i+1:], nil
		}
	}
	return 0, "", fmt.Errorf("invalid state")
}

func optionalString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// ensure base64 import retained even if not directly used elsewhere.
var _ = base64.StdEncoding
