package handler

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"

	"backend/internal/channel"
	twitterchan "backend/internal/channel/twitter"
	appcrypto "backend/internal/crypto"
	"backend/internal/dto"
	"backend/internal/logger"
	"backend/internal/model"
	"backend/internal/repo"
)

const (
	twitterRequestTokenSecretPrefix = "elodesk:twitter:request_token_secret:"
	twitterAccountKeyPrefix         = "elodesk:twitter:request_account:"
	twitterRequestTokenTTL          = 10 * time.Minute
)

// TwitterHandler exposes provisioning (3-legged OAuth) and the public
// Account Activity webhook endpoints. All provisioning routes are gated by
// the FEATURE_CHANNEL_TWITTER flag — when off, every route returns 403.
type TwitterHandler struct {
	twitterRepo      *repo.ChannelTwitterRepo
	inboxRepo        *repo.InboxRepo
	contactRepo      *repo.ContactRepo
	contactInboxRepo *repo.ContactInboxRepo
	conversationRepo *repo.ConversationRepo
	messageRepo      *repo.MessageRepo
	cipher           *appcrypto.Cipher
	dedup            *channel.DedupLock
	oauth            *twitterchan.OAuthClient
	consumerSecret   string
	redisClient      redis.Cmdable
	featureEnabled   bool
}

func NewTwitterHandler(
	twitterRepo *repo.ChannelTwitterRepo,
	inboxRepo *repo.InboxRepo,
	contactRepo *repo.ContactRepo,
	contactInboxRepo *repo.ContactInboxRepo,
	conversationRepo *repo.ConversationRepo,
	messageRepo *repo.MessageRepo,
	cipher *appcrypto.Cipher,
	dedup *channel.DedupLock,
	oauth *twitterchan.OAuthClient,
	consumerSecret string,
	redisClient redis.Cmdable,
	featureEnabled bool,
) *TwitterHandler {
	return &TwitterHandler{
		twitterRepo:      twitterRepo,
		inboxRepo:        inboxRepo,
		contactRepo:      contactRepo,
		contactInboxRepo: contactInboxRepo,
		conversationRepo: conversationRepo,
		messageRepo:      messageRepo,
		cipher:           cipher,
		dedup:            dedup,
		oauth:            oauth,
		consumerSecret:   consumerSecret,
		redisClient:      redisClient,
		featureEnabled:   featureEnabled,
	}
}

// Authorize handles POST /api/v1/accounts/:aid/inboxes/twitter/authorize.
// Mints an OAuth 1.0a request_token, stashes the matching token_secret +
// owning account in Redis (10m TTL), and returns the consent URL.
//
//	@Summary		Begin Twitter OAuth handshake
//	@Tags			inboxes
//	@Security		BearerAuth
//	@Produce		json
//	@Param			aid	path		int	true	"Account ID"
//	@Success		200	{object}	dto.APIResponse{data=dto.TwitterAuthorizeResp}
//	@Failure		403	{object}	dto.APIError
//	@Router			/api/v1/accounts/{aid}/inboxes/twitter/authorize [post]
func (h *TwitterHandler) Authorize(c *fiber.Ctx) error {
	if !h.featureEnabled {
		return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResp("feature_disabled", "Twitter channel is disabled"))
	}
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	rt, err := h.oauth.RequestToken(c.Context())
	if err != nil {
		logger.Warn().Str("component", "channel.twitter").Err(err).Msg("twitter request_token failed")
		return c.Status(fiber.StatusBadGateway).JSON(dto.ErrorResp("oauth_request_token_failed", "failed to obtain request token"))
	}
	if !rt.OAuthCallbackConfirmed {
		return c.Status(fiber.StatusBadGateway).JSON(dto.ErrorResp("callback_not_confirmed", "twitter rejected the callback url"))
	}

	if err := h.redisClient.Set(c.Context(), twitterRequestTokenSecretPrefix+rt.OAuthToken, rt.OAuthTokenSecret, twitterRequestTokenTTL).Err(); err != nil {
		logger.Error().Str("component", "channel.twitter").Err(err).Msg("failed to persist request token secret")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to persist request token"))
	}
	if err := h.redisClient.Set(c.Context(), twitterAccountKeyPrefix+rt.OAuthToken, strconv.FormatInt(accountID, 10), twitterRequestTokenTTL).Err(); err != nil {
		logger.Error().Str("component", "channel.twitter").Err(err).Msg("failed to persist request account")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to persist request account"))
	}

	return c.JSON(dto.SuccessResp(dto.TwitterAuthorizeResp{URL: h.oauth.AuthorizeURL(rt.OAuthToken)}))
}

// Callback handles GET /api/v1/accounts/twitter/oauth/callback.
// Twitter redirects the user here after consent; we exchange the verifier
// for the long-lived access token pair and create the inbox.
//
//	@Summary		Twitter OAuth callback
//	@Tags			inboxes
//	@Produce		json
//	@Param			oauth_token		query		string	false	"Request token returned by Twitter"
//	@Param			oauth_verifier	query		string	false	"OAuth verifier"
//	@Param			denied			query		string	false	"Set when user denies consent"
//	@Success		201				{object}	dto.APIResponse{data=dto.TwitterInboxResp}
//	@Failure		400				{object}	dto.APIError
//	@Router			/api/v1/accounts/twitter/oauth/callback [get]
func (h *TwitterHandler) Callback(c *fiber.Ctx) error {
	if !h.featureEnabled {
		return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResp("feature_disabled", "Twitter channel is disabled"))
	}
	if denied := c.Query("denied"); denied != "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("oauth_denied", "user declined the OAuth grant"))
	}

	requestToken := c.Query("oauth_token")
	verifier := c.Query("oauth_verifier")
	if requestToken == "" || verifier == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("bad_request", "missing oauth_token or oauth_verifier"))
	}

	storedAccount, err := h.redisClient.Get(c.Context(), twitterAccountKeyPrefix+requestToken).Result()
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("state_expired", "oauth state expired or unknown"))
	}
	accountID, err := strconv.ParseInt(storedAccount, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("state_invalid", "invalid stored account"))
	}

	requestTokenSecret, err := h.redisClient.Get(c.Context(), twitterRequestTokenSecretPrefix+requestToken).Result()
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("state_expired", "oauth token secret expired or unknown"))
	}

	access, err := h.oauth.AccessToken(c.Context(), requestToken, requestTokenSecret, verifier)
	if err != nil {
		logger.Warn().Str("component", "channel.twitter").Err(err).Msg("twitter access_token exchange failed")
		return c.Status(fiber.StatusBadGateway).JSON(dto.ErrorResp("oauth_exchange_failed", "failed to exchange access token"))
	}
	_ = h.redisClient.Del(c.Context(), twitterAccountKeyPrefix+requestToken)
	_ = h.redisClient.Del(c.Context(), twitterRequestTokenSecretPrefix+requestToken)

	profileID := access.UserID
	screenName := access.ScreenName
	if profileID == "" {
		api := twitterchan.NewAPIClient(h.oauth.ConsumerKey(), h.consumerSecret)
		me, meErr := api.GetMe(c.Context(), access.OAuthToken, access.OAuthTokenSecret)
		if meErr != nil {
			logger.Warn().Str("component", "channel.twitter").Err(meErr).Msg("twitter get me failed")
			return c.Status(fiber.StatusBadGateway).JSON(dto.ErrorResp("get_me_failed", "failed to resolve twitter profile"))
		}
		profileID = me.Data.ID
		screenName = me.Data.Username
	}
	if profileID == "" {
		return c.Status(fiber.StatusBadGateway).JSON(dto.ErrorResp("profile_id_missing", "twitter did not return a profile id"))
	}

	tokenCipher, err := h.cipher.Encrypt(access.OAuthToken)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to encrypt access token"))
	}
	secretCipher, err := h.cipher.Encrypt(access.OAuthTokenSecret)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to encrypt access token secret"))
	}

	inboxName := screenName
	if inboxName == "" {
		inboxName = "Twitter " + profileID
	}

	ch := &model.ChannelTwitter{
		AccountID:                          accountID,
		ProfileID:                          profileID,
		ScreenName:                         optionalString(screenName),
		TwitterAccessTokenCiphertext:       tokenCipher,
		TwitterAccessTokenSecretCiphertext: secretCipher,
		TweetsEnabled:                      true,
	}
	if err := h.twitterRepo.Create(c.Context(), ch); err != nil {
		logger.Error().Str("component", "channel.twitter").Err(err).Msg("failed to create twitter channel")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to create twitter channel"))
	}

	inbox := &model.Inbox{
		AccountID:   accountID,
		ChannelID:   ch.ID,
		Name:        inboxName,
		ChannelType: string(channel.KindTwitter),
	}
	if err := h.inboxRepo.Create(c.Context(), inbox); err != nil {
		logger.Error().Str("component", "channel.twitter").Err(err).Msg("failed to create twitter inbox")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to create inbox"))
	}

	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResp(dto.TwitterInboxResp{
		InboxResp: inboxModelToResp(inbox),
		Channel: dto.TwitterChannelResp{
			ID:             ch.ID,
			ProfileID:      ch.ProfileID,
			ScreenName:     ch.ScreenName,
			TweetsEnabled:  ch.TweetsEnabled,
			RequiresReauth: ch.RequiresReauth,
			CreatedAt:      ch.CreatedAt,
			UpdatedAt:      ch.UpdatedAt,
		},
	}))
}

// CRC handles GET /webhooks/twitter/:profile_id?crc_token=...
// Twitter periodically pings this endpoint and expects an HMAC-SHA256 of
// the crc_token using the consumer secret as response.
//
//	@Summary		Twitter webhook CRC challenge
//	@Tags			webhooks
//	@Produce		json
//	@Param			profile_id	path		string	true	"Twitter profile id"
//	@Param			crc_token	query		string	true	"Challenge token"
//	@Success		200			{object}	map[string]string
//	@Failure		400			{object}	dto.APIError
//	@Router			/webhooks/twitter/{profile_id} [get]
func (h *TwitterHandler) CRC(c *fiber.Ctx) error {
	crcToken := c.Query("crc_token")
	if crcToken == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("bad_request", "missing crc_token"))
	}
	response := twitterchan.CRCChallenge(h.consumerSecret, crcToken)
	return c.JSON(fiber.Map{"response_token": response})
}

// Receive handles POST /webhooks/twitter/:profile_id (Account Activity events).
//
//	@Summary		Twitter webhook delivery
//	@Tags			webhooks
//	@Accept			json
//	@Produce		json
//	@Param			profile_id	path		string	true	"Twitter profile id"
//	@Success		200			{object}	dto.APIResponse
//	@Failure		401			{object}	dto.APIError
//	@Router			/webhooks/twitter/{profile_id} [post]
func (h *TwitterHandler) Receive(c *fiber.Ctx) error {
	profileID := c.Params("profile_id")

	body := c.Body()
	signature := c.Get("x-twitter-webhooks-signature")
	if !twitterchan.VerifySignature(h.consumerSecret, body, signature) {
		logger.Warn().Str("component", "channel.twitter").Str("profileId", profileID).Msg("invalid signature")
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "invalid signature"))
	}

	ch, err := h.twitterRepo.FindByProfileID(c.Context(), profileID)
	if err != nil {
		return c.SendStatus(fiber.StatusOK)
	}
	inbox, err := h.inboxRepo.FindByChannelID(c.Context(), ch.ID)
	if err != nil {
		return c.SendStatus(fiber.StatusOK)
	}

	if err := twitterchan.ProcessWebhook(c.Context(), body, ch, inbox, h.dedup,
		h.contactRepo, h.contactInboxRepo, h.conversationRepo, h.messageRepo); err != nil {
		logger.Warn().Str("component", "channel.twitter").Err(err).Msg("twitter process webhook error")
	}
	return c.SendStatus(fiber.StatusOK)
}

// Delete handles DELETE /api/v1/accounts/:aid/inboxes/:id/twitter.
//
//	@Summary		Delete Twitter inbox
//	@Tags			inboxes
//	@Security		BearerAuth
//	@Param			aid	path		int	true	"Account ID"
//	@Param			id	path		int	true	"Inbox ID"
//	@Success		200	{object}	dto.APIResponse
//	@Failure		404	{object}	dto.APIError
//	@Router			/api/v1/accounts/{aid}/inboxes/{id}/twitter [delete]
// GetByInboxID handles GET /api/v1/accounts/:aid/inboxes/:id/twitter.
func (h *TwitterHandler) GetByInboxID(c *fiber.Ctx) error {
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
	if inbox.ChannelType != string(channel.KindTwitter) {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "inbox is not a twitter channel"))
	}

	ch, err := h.twitterRepo.FindByID(c.Context(), inbox.ChannelID, accountID)
	if err != nil {
		return handleNotFound(c, err)
	}

	return c.JSON(dto.SuccessResp(dto.TwitterInboxResp{
		InboxResp: inboxModelToResp(inbox),
		Channel: dto.TwitterChannelResp{
			ID:             ch.ID,
			ProfileID:      ch.ProfileID,
			ScreenName:     ch.ScreenName,
			TweetsEnabled:  ch.TweetsEnabled,
			RequiresReauth: ch.RequiresReauth,
			CreatedAt:      ch.CreatedAt,
			UpdatedAt:      ch.UpdatedAt,
		},
	}))
}

// Update handles PUT /api/v1/accounts/:aid/inboxes/:id/twitter.
func (h *TwitterHandler) Update(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}
	inboxID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid inbox id"))
	}

	var req dto.UpdateTwitterInboxReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	inbox, err := h.inboxRepo.FindByID(c.Context(), inboxID, accountID)
	if err != nil {
		return handleNotFound(c, err)
	}
	if inbox.ChannelType != string(channel.KindTwitter) {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "inbox is not a twitter channel"))
	}

	if req.Name != "" {
		if err := h.inboxRepo.UpdateName(c.Context(), inboxID, accountID, req.Name); err != nil {
			return handleNotFound(c, err)
		}
	}

	if req.TweetsEnabled != nil {
		if err := h.twitterRepo.SetTweetsEnabled(c.Context(), inbox.ChannelID, *req.TweetsEnabled); err != nil {
			return handleNotFound(c, err)
		}
	}

	return h.GetByInboxID(c)
}

func (h *TwitterHandler) Delete(c *fiber.Ctx) error {
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
	if inbox.ChannelType != string(channel.KindTwitter) {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "inbox is not a twitter channel"))
	}
	if err := h.twitterRepo.Delete(c.Context(), inbox.ChannelID); err != nil {
		logger.Error().Str("component", "channel.twitter").Err(err).Msg("failed to delete twitter channel")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to delete twitter channel"))
	}
	return c.JSON(dto.SuccessResp(nil))
}
