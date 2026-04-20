package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"backend/internal/logger"
	"backend/internal/model"
	"backend/internal/repo"

	"github.com/alexedwards/argon2id"
)

var (
	ErrInvalidCurrentPassword = errors.New("invalid current password")
	ErrInvalidAvatarPath      = errors.New("invalid avatar path")
	ErrMissingCurrentPassword = errors.New("current_password required to change password")
)

type UserProfileService struct {
	userRepo         *repo.UserRepo
	refreshTokenRepo *repo.RefreshTokenRepo
	auditLogRepo     *repo.AuditLogRepo
}

func NewUserProfileService(userRepo *repo.UserRepo, refreshTokenRepo *repo.RefreshTokenRepo, auditLogRepo *repo.AuditLogRepo) *UserProfileService {
	return &UserProfileService{
		userRepo:         userRepo,
		refreshTokenRepo: refreshTokenRepo,
		auditLogRepo:     auditLogRepo,
	}
}

type UpdateProfileInput struct {
	Name            *string
	Email           *string
	AvatarURL       *string
	CurrentPassword *string
	NewPassword     *string
	AccountID       int64
}

func (s *UserProfileService) UpdateProfile(ctx context.Context, userID int64, in UpdateProfileInput) (*model.User, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("profile.update: %w", err)
	}

	if in.NewPassword != nil && *in.NewPassword != "" {
		if in.CurrentPassword == nil || *in.CurrentPassword == "" {
			return nil, ErrMissingCurrentPassword
		}
		match, err := argon2id.ComparePasswordAndHash(*in.CurrentPassword, user.PasswordHash)
		if err != nil {
			return nil, fmt.Errorf("profile.update: %w", err)
		}
		if !match {
			return nil, ErrInvalidCurrentPassword
		}
		hash, err := argon2id.CreateHash(*in.NewPassword, argon2id.DefaultParams)
		if err != nil {
			return nil, fmt.Errorf("profile.update: %w", err)
		}
		if err := s.userRepo.UpdatePasswordHash(ctx, userID, hash); err != nil {
			return nil, fmt.Errorf("profile.update: %w", err)
		}
		if err := s.refreshTokenRepo.RevokeAllByUserID(ctx, userID); err != nil {
			logger.Error().Str("component", "user_profile").Err(err).Int64("userId", userID).Msg("failed to revoke refresh tokens after password change")
		}
		if s.auditLogRepo != nil && in.AccountID > 0 {
			uid := userID
			if err := s.auditLogRepo.Create(ctx, in.AccountID, &uid, "user.password_changed", "user", &uid, "{}", nil, ""); err != nil {
				logger.Error().Str("component", "user_profile").Err(err).Msg("failed to write audit log")
			}
		}
	}

	if in.AvatarURL != nil {
		if *in.AvatarURL != "" {
			expectedPrefix := fmt.Sprintf("%d/avatars/%d/", in.AccountID, userID)
			if !strings.HasPrefix(*in.AvatarURL, expectedPrefix) {
				return nil, ErrInvalidAvatarPath
			}
		}
		if err := s.userRepo.UpdateAvatarURL(ctx, userID, in.AvatarURL); err != nil {
			return nil, fmt.Errorf("profile.update: %w", err)
		}
	}

	if in.Name != nil && *in.Name != "" {
		if err := s.userRepo.UpdateName(ctx, userID, *in.Name); err != nil {
			return nil, fmt.Errorf("profile.update: %w", err)
		}
	}

	if in.Email != nil && *in.Email != "" {
		if err := s.userRepo.UpdateEmail(ctx, userID, *in.Email); err != nil {
			return nil, fmt.Errorf("profile.update: %w", err)
		}
	}

	updated, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("profile.update: %w", err)
	}
	return updated, nil
}
