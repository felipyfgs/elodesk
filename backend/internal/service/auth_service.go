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
	"backend/internal/model"
	"backend/internal/repo"

	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrRefreshTokenReused = errors.New("refresh token reuse detected")
)

type AuthService struct {
	userRepo         *repo.UserRepo
	accountRepo      *repo.AccountRepo
	refreshTokenRepo *repo.RefreshTokenRepo
	mfaService       *MfaService
	jwtSecret        string
	accessTTL        time.Duration
	refreshTTL       time.Duration
}

func NewAuthService(
	userRepo *repo.UserRepo,
	accountRepo *repo.AccountRepo,
	refreshTokenRepo *repo.RefreshTokenRepo,
	mfaService *MfaService,
	jwtSecret string,
	accessTTL, refreshTTL time.Duration,
) *AuthService {
	return &AuthService{
		userRepo:         userRepo,
		accountRepo:      accountRepo,
		refreshTokenRepo: refreshTokenRepo,
		mfaService:       mfaService,
		jwtSecret:        jwtSecret,
		accessTTL:        accessTTL,
		refreshTTL:       refreshTTL,
	}
}

type RegisterResult struct {
	User         *model.User
	Account      *model.Account
	AccessToken  string
	RefreshToken string
}

func (s *AuthService) Register(ctx context.Context, email, password, name, accountName string) (*RegisterResult, error) {
	hash, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	if accountName == "" {
		accountName = name + "'s Account"
	}

	user := &model.User{
		Email:        email,
		Name:         name,
		PasswordHash: hash,
	}

	account := &model.Account{
		Name: accountName,
		Slug: generateSlug(email),
	}

	tx, err := s.userRepo.Pool().Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := s.userRepo.CreateTx(ctx, tx, user); err != nil {
		if errors.Is(err, repo.ErrUserEmailExists) {
			return nil, err
		}
		return nil, err
	}

	if err := s.accountRepo.CreateTx(ctx, tx, account); err != nil {
		return nil, err
	}

	if _, err := s.accountRepo.AddUserTx(ctx, tx, account.ID, user.ID, model.RoleOwner); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit registration: %w", err)
	}

	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.generateRefreshToken(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &RegisterResult{
		User:         user,
		Account:      account,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

type LoginResult struct {
	User         *model.User
	Account      *model.Account
	AccessToken  string
	RefreshToken string
	MfaToken     string // non-empty when MFA is required
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*LoginResult, error) {
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repo.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	match, err := argon2id.ComparePasswordAndHash(password, user.PasswordHash)
	if err != nil {
		logger.Error().Str("component", "auth").Err(err).Msg("failed to compare password hash")
		return nil, ErrInvalidCredentials
	}
	if !match {
		return nil, ErrInvalidCredentials
	}

	// If MFA is enabled, return an MFA token instead of JWT pair.
	if user.MfaEnabled {
		mfaToken, err := s.mfaService.GenerateMfaToken(user.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to generate mfa token: %w", err)
		}
		return &LoginResult{User: user, MfaToken: mfaToken}, nil
	}

	account, err := s.accountRepo.FindPrimaryByUserID(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve primary account: %w", err)
	}

	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.generateRefreshToken(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &LoginResult{
		User:         user,
		Account:      account,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *AuthService) Refresh(ctx context.Context, rawToken string) (string, string, error) {
	tokenHash := hashToken(rawToken)

	stored, err := s.refreshTokenRepo.FindByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, repo.ErrRefreshTokenNotFound) {
			return "", "", ErrInvalidCredentials
		}
		return "", "", fmt.Errorf("failed to find refresh token: %w", err)
	}

	if stored.RevokedAt != nil {
		logger.Warn().Str("component", "auth").Str("userId", fmt.Sprintf("%d", stored.UserID)).
			Str("familyId", stored.FamilyID).Msg("refresh token reuse detected, revoking family")
		if err := s.refreshTokenRepo.RevokeByFamily(ctx, stored.UserID, stored.FamilyID); err != nil {
			logger.Error().Str("component", "auth").Err(err).Msg("failed to revoke token family")
		}
		return "", "", ErrRefreshTokenReused
	}

	if time.Now().UTC().After(stored.ExpiresAt) {
		return "", "", ErrInvalidCredentials
	}

	// Abort rotation if the old token can't be revoked — otherwise both old
	// and new tokens would be valid simultaneously, defeating replay detection.
	if err := s.refreshTokenRepo.Revoke(ctx, stored.ID); err != nil {
		logger.Error().Str("component", "auth").Err(err).Msg("refresh: abort rotation, revoke of previous token failed")
		return "", "", fmt.Errorf("refresh rotation aborted: %w", err)
	}

	user, err := s.userRepo.FindByID(ctx, stored.UserID)
	if err != nil {
		return "", "", fmt.Errorf("failed to find user: %w", err)
	}

	newAccess, err := s.generateAccessToken(user)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	newRefresh, err := s.generateRefreshTokenWithFamily(ctx, stored.UserID, stored.FamilyID)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return newAccess, newRefresh, nil
}

func (s *AuthService) Logout(ctx context.Context, userID int64, rawToken string, allDevices bool) error {
	if allDevices {
		return s.refreshTokenRepo.RevokeAllByUserID(ctx, userID)
	}

	tokenHash := hashToken(rawToken)
	stored, err := s.refreshTokenRepo.FindByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, repo.ErrRefreshTokenNotFound) {
			return nil
		}
		return fmt.Errorf("failed to find refresh token: %w", err)
	}
	return s.refreshTokenRepo.Revoke(ctx, stored.ID)
}

// IssueTokenPair generates a JWT pair for a given user ID. Used after MFA verification.
func (s *AuthService) IssueTokenPair(ctx context.Context, userID int64) (*LoginResult, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	account, err := s.accountRepo.FindPrimaryByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve primary account: %w", err)
	}

	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.generateRefreshToken(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &LoginResult{
		User:         user,
		Account:      account,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *AuthService) ValidateAccessToken(tokenStr string) (*repo.AuthUser, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.jwtSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidCredentials
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrInvalidCredentials
	}

	id, ok := claims["sub"].(float64)
	if !ok {
		return nil, ErrInvalidCredentials
	}

	email, _ := claims["email"].(string)
	name, _ := claims["name"].(string)

	return &repo.AuthUser{
		ID:    int64(id),
		Email: email,
		Name:  name,
	}, nil
}

func (s *AuthService) generateAccessToken(user *model.User) (string, error) {
	claims := jwt.MapClaims{
		"sub":   user.ID,
		"email": user.Email,
		"name":  user.Name,
		"exp":   time.Now().Add(s.accessTTL).Unix(),
		"iat":   time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

func (s *AuthService) generateRefreshToken(ctx context.Context, userID int64) (string, error) {
	familyID := uuid.New().String()
	return s.generateRefreshTokenWithFamily(ctx, userID, familyID)
}

func (s *AuthService) generateRefreshTokenWithFamily(ctx context.Context, userID int64, familyID string) (string, error) {
	rawBytes := make([]byte, 48)
	if _, err := rand.Read(rawBytes); err != nil {
		return "", fmt.Errorf("failed to generate refresh token: %w", err)
	}
	rawToken := base64.RawURLEncoding.EncodeToString(rawBytes)
	tokenHash := hashToken(rawToken)

	refreshToken := &model.RefreshToken{
		UserID:    userID,
		TokenHash: tokenHash,
		FamilyID:  familyID,
		ExpiresAt: time.Now().Add(s.refreshTTL),
	}

	if err := s.refreshTokenRepo.Create(ctx, refreshToken); err != nil {
		return "", fmt.Errorf("failed to store refresh token: %w", err)
	}

	return rawToken, nil
}

func hashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

func generateSlug(email string) string {
	b := make([]byte, 3)
	_, _ = rand.Read(b)
	return email + "-" + base64.RawURLEncoding.EncodeToString(b)
}
