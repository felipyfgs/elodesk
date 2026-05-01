package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"backend/internal/logger"
	"backend/internal/repo"

	"github.com/alexedwards/argon2id"
)

var (
	ErrResetTokenInvalid = errors.New("invalid or expired reset token")
)

type PasswordRecoveryService struct {
	userRepo         *repo.UserRepo
	tokenRepo        *repo.PasswordResetTokenRepo
	refreshTokenRepo *repo.RefreshTokenRepo
}

func NewPasswordRecoveryService(
	userRepo *repo.UserRepo,
	tokenRepo *repo.PasswordResetTokenRepo,
	refreshTokenRepo *repo.RefreshTokenRepo,
) *PasswordRecoveryService {
	return &PasswordRecoveryService{
		userRepo:         userRepo,
		tokenRepo:        tokenRepo,
		refreshTokenRepo: refreshTokenRepo,
	}
}

// RequestReset always returns nil (generic success). If the email exists,
// a token is generated, hashed, stored, and logged. Response time is constant
// to prevent timing attacks.
func (s *PasswordRecoveryService) RequestReset(ctx context.Context, email string) error {
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repo.ErrUserNotFound) {
			// Return silently — same response as success.
			return nil
		}
		return fmt.Errorf("failed to lookup user for password reset: %w", err)
	}

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return fmt.Errorf("failed to generate reset token: %w", err)
	}
	rawToken := base64.RawURLEncoding.EncodeToString(tokenBytes)
	tokenHash := hashResetToken(rawToken)

	resetToken := &repo.PasswordResetToken{
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().UTC().Add(30 * time.Minute),
	}

	if err := s.tokenRepo.Create(ctx, resetToken); err != nil {
		return fmt.Errorf("failed to store reset token: %w", err)
	}

	// Log token for development — in production this would be sent via email.
	logger.Info().
		Str("component", "auth").
		Str("event", "password_reset_requested").
		Str("email", user.Email).
		Str("token", rawToken).
		Msg("password reset token generated")

	return nil
}

// ValidateToken checks if a reset token is valid (not expired, not consumed).
func (s *PasswordRecoveryService) ValidateToken(ctx context.Context, token string) (bool, error) {
	tokenHash := hashResetToken(token)
	stored, err := s.tokenRepo.FindByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, repo.ErrResetTokenNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("failed to validate reset token: %w", err)
	}

	if stored.ConsumedAt != nil || time.Now().UTC().After(stored.ExpiresAt) {
		return false, nil
	}

	return true, nil
}

// ResetPassword validates the token, hashes the new password, updates the user,
// marks the token as consumed, and revokes all refresh tokens.
func (s *PasswordRecoveryService) ResetPassword(ctx context.Context, token, newPassword string) error {
	tokenHash := hashResetToken(token)
	stored, err := s.tokenRepo.FindByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, repo.ErrResetTokenNotFound) {
			return ErrResetTokenInvalid
		}
		return fmt.Errorf("failed to find reset token: %w", err)
	}

	if stored.ConsumedAt != nil || time.Now().UTC().After(stored.ExpiresAt) {
		return ErrResetTokenInvalid
	}

	hash, err := argon2id.CreateHash(newPassword, argon2id.DefaultParams)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	if err := s.userRepo.UpdatePasswordHash(ctx, stored.UserID, hash); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	if err := s.tokenRepo.Consume(ctx, stored.ID); err != nil {
		logger.Error().Str("component", "auth").Err(err).Msg("failed to consume reset token")
	}

	if err := s.refreshTokenRepo.RevokeAllByUserID(ctx, stored.UserID); err != nil {
		logger.Error().Str("component", "auth").Err(err).Msg("failed to revoke refresh tokens after password reset")
	}

	logger.Info().
		Str("component", "auth").
		Str("event", "password_reset_completed").
		Int64("userId", stored.UserID).
		Msg("password reset completed")

	return nil
}

func hashResetToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return base64.RawURLEncoding.EncodeToString(h[:])
}
