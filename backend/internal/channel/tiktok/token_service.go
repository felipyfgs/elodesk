package tiktok

import (
	"context"
	"fmt"
	"time"

	"backend/internal/channel"
	"backend/internal/channel/reauth"
	appcrypto "backend/internal/crypto"
	"backend/internal/logger"
	"backend/internal/model"
	"backend/internal/repo"
)

// refreshMargin tells how early we should proactively refresh an access token
// before it actually expires. Matches the Chatwoot reference (5 minutes).
const refreshMargin = 5 * time.Minute

// TokenService owns the logic to return a valid access_token for a TikTok
// channel, refreshing the short-term token when necessary. On permanent
// failures, it ticks the reauth tracker and flags requires_reauth on the row.
type TokenService struct {
	oauth   *OAuthClient
	repo    *repo.ChannelTiktokRepo
	cipher  *appcrypto.Cipher
	tracker *reauth.Tracker
}

func NewTokenService(oauth *OAuthClient, r *repo.ChannelTiktokRepo, cipher *appcrypto.Cipher, tracker *reauth.Tracker) *TokenService {
	return &TokenService{oauth: oauth, repo: r, cipher: cipher, tracker: tracker}
}

// AccessToken returns a non-empty access token when possible, refreshing the
// underlying credentials if the short-term access token is within the
// expiration margin and the refresh token itself has not expired.
func (s *TokenService) AccessToken(ctx context.Context, ch *model.ChannelTiktok) (string, error) {
	if ch == nil {
		return "", fmt.Errorf("tiktok token: nil channel")
	}
	accessToken, err := s.cipher.Decrypt(ch.AccessTokenCiphertext)
	if err != nil {
		return "", fmt.Errorf("tiktok token: decrypt access token: %w", err)
	}

	if time.Now().Add(refreshMargin).Before(ch.ExpiresAt) {
		return accessToken, nil
	}

	if !time.Now().Before(ch.RefreshTokenExpiresAt) {
		// refresh token itself expired; flag reauth and return whatever we have
		if !ch.RequiresReauth {
			_ = s.repo.SetRequiresReauth(ctx, ch.ID, true)
			ch.RequiresReauth = true
		}
		return accessToken, fmt.Errorf("tiktok token: refresh token expired, reauth required")
	}

	refreshToken, err := s.cipher.Decrypt(ch.RefreshTokenCiphertext)
	if err != nil {
		return "", fmt.Errorf("tiktok token: decrypt refresh token: %w", err)
	}

	newTokens, err := s.oauth.Refresh(ctx, refreshToken)
	if err != nil {
		logger.Warn().Str("component", "channel.tiktok").Err(err).Int64("channelId", ch.ID).Msg("tiktok token refresh failed")
		trackerKey := fmt.Sprintf("channel:tiktok:%d", ch.ID)
		if prompt, tErr := s.tracker.RecordErrorForKind(ctx, channel.KindTiktok, trackerKey); tErr == nil && prompt {
			_ = s.repo.SetRequiresReauth(ctx, ch.ID, true)
			ch.RequiresReauth = true
		}
		return accessToken, err
	}

	newAccessCiphertext, err := s.cipher.Encrypt(newTokens.AccessToken)
	if err != nil {
		return "", fmt.Errorf("tiktok token: encrypt access token: %w", err)
	}
	newRefreshCiphertext, err := s.cipher.Encrypt(newTokens.RefreshToken)
	if err != nil {
		return "", fmt.Errorf("tiktok token: encrypt refresh token: %w", err)
	}

	newExpiresAt := time.Now().Add(time.Duration(newTokens.ExpiresIn) * time.Second)
	newRefreshExpiresAt := time.Now().Add(time.Duration(newTokens.RefreshTokenExpiresIn) * time.Second)

	if err := s.repo.UpdateTokens(ctx, ch.ID, newAccessCiphertext, newRefreshCiphertext, newExpiresAt, newRefreshExpiresAt); err != nil {
		return "", fmt.Errorf("tiktok token: persist: %w", err)
	}
	ch.AccessTokenCiphertext = newAccessCiphertext
	ch.RefreshTokenCiphertext = newRefreshCiphertext
	ch.ExpiresAt = newExpiresAt
	ch.RefreshTokenExpiresAt = newRefreshExpiresAt
	_ = s.tracker.Reset(ctx, fmt.Sprintf("channel:tiktok:%d", ch.ID))

	return newTokens.AccessToken, nil
}
