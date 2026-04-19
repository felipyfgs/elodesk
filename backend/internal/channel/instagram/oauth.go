package instagram

import (
	"context"
	"fmt"
	"time"

	"backend/internal/channel/meta"
	"backend/internal/channel/reauth"
	appcrypto "backend/internal/crypto"
	"backend/internal/logger"
	"backend/internal/model"
	"backend/internal/repo"
)

const (
	instagramRefreshBase = "https://graph.instagram.com"
	refreshThresholdDays = 10
)

type refreshTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

// RefreshIfNeeded checks whether the token expires within the threshold and
// refreshes proactively. On auth failure, the reauth tracker is incremented and
// requires_reauth is set on the channel record.
func RefreshIfNeeded(
	ctx context.Context,
	ch *model.ChannelInstagram,
	accessToken string,
	igRepo *repo.ChannelInstagramRepo,
	cipher *appcrypto.Cipher,
	tracker *reauth.Tracker,
) (string, error) {
	threshold := time.Now().Add(refreshThresholdDays * 24 * time.Hour)
	if ch.ExpiresAt.After(threshold) {
		return accessToken, nil
	}

	logger.Info().
		Str("component", "instagram.oauth").
		Int64("channelId", ch.ID).
		Time("expiresAt", ch.ExpiresAt).
		Msg("proactive token refresh")

	client := meta.NewClient(instagramRefreshBase)

	path := fmt.Sprintf("/refresh_access_token?grant_type=ig_refresh_token&access_token=%s", accessToken)
	var resp refreshTokenResponse
	if err := client.Get(ctx, path, accessToken, &resp); err != nil {
		logger.Warn().Str("component", "instagram.oauth").Int64("channelId", ch.ID).Err(err).Msg("refresh failed")
		key := fmt.Sprintf("channel:instagram:%d", ch.ID)
		if prompt, trackErr := tracker.RecordError(ctx, key); trackErr == nil && prompt {
			_ = igRepo.SetRequiresReauth(ctx, ch.ID, true)
		}
		return accessToken, fmt.Errorf("instagram token refresh: %w", err)
	}

	newExpiry := time.Now().Add(time.Duration(resp.ExpiresIn) * time.Second)
	if resp.ExpiresIn == 0 {
		newExpiry = time.Now().Add(60 * 24 * time.Hour)
	}

	ciphertext, err := cipher.Encrypt(resp.AccessToken)
	if err != nil {
		return accessToken, fmt.Errorf("instagram refresh: encrypt new token: %w", err)
	}

	if err := igRepo.UpdateToken(ctx, ch.ID, ciphertext, newExpiry); err != nil {
		return resp.AccessToken, fmt.Errorf("instagram refresh: persist token: %w", err)
	}

	ch.ExpiresAt = newExpiry
	ch.AccessTokenCiphertext = ciphertext
	_ = tracker.Reset(ctx, fmt.Sprintf("channel:instagram:%d", ch.ID))

	return resp.AccessToken, nil
}
