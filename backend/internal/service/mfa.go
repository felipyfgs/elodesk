package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"time"

	"backend/internal/crypto"
	"backend/internal/logger"
	"backend/internal/repo"

	"github.com/alexedwards/argon2id"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

var (
	ErrMFAInvalidCode     = errors.New("invalid mfa code")
	ErrMFANotSetup        = errors.New("mfa not set up")
	ErrMFAAlreadyEnabled  = errors.New("mfa already enabled")
	ErrMFAInvalidPassword = errors.New("invalid current password")
)

const (
	mfaTokenTTL       = 5 * time.Minute
	recoveryCodeCount = 8
)

type MFAService struct {
	userRepo      *repo.UserRepo
	recoveryRepo  *repo.MFARecoveryCodeRepo
	refreshRepo   *repo.RefreshTokenRepo
	cipher        *crypto.Cipher
	mfaTokenStore MFATokenStore
}

type MFATokenStore interface {
	Set(key string, userID int64, ttl time.Duration) error
	GetAndDelete(key string) (int64, bool)
}

type InMemoryMFATokenStore struct {
	mu    sync.Mutex
	store map[string]mfaTokenEntry
}

type mfaTokenEntry struct {
	UserID    int64
	ExpiresAt time.Time
}

func NewInMemoryMFATokenStore() *InMemoryMFATokenStore {
	s := &InMemoryMFATokenStore{
		store: make(map[string]mfaTokenEntry),
	}
	go s.cleanupLoop()
	return s
}

func (s *InMemoryMFATokenStore) Set(key string, userID int64, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.store[key] = mfaTokenEntry{
		UserID:    userID,
		ExpiresAt: time.Now().Add(ttl),
	}
	return nil
}

func (s *InMemoryMFATokenStore) GetAndDelete(key string) (int64, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.store[key]
	if !ok {
		return 0, false
	}
	delete(s.store, key)
	if time.Now().After(entry.ExpiresAt) {
		return 0, false
	}
	return entry.UserID, true
}

func (s *InMemoryMFATokenStore) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for k, v := range s.store {
			if now.After(v.ExpiresAt) {
				delete(s.store, k)
			}
		}
		s.mu.Unlock()
	}
}

func NewMFAService(
	userRepo *repo.UserRepo,
	recoveryRepo *repo.MFARecoveryCodeRepo,
	refreshRepo *repo.RefreshTokenRepo,
	cipher *crypto.Cipher,
	tokenStore MFATokenStore,
) *MFAService {
	return &MFAService{
		userRepo:      userRepo,
		recoveryRepo:  recoveryRepo,
		refreshRepo:   refreshRepo,
		cipher:        cipher,
		mfaTokenStore: tokenStore,
	}
}

type MFASetupResult struct {
	OTPAuthURI string
	Secret     string
}

func (s *MFAService) Setup(ctx context.Context, userID int64) (*MFASetupResult, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Elodesk",
		AccountName: user.Email,
		Period:      30,
		Digits:      otp.DigitsSix,
		Algorithm:   otp.AlgorithmSHA1,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate TOTP key: %w", err)
	}

	secretCiphertext, err := s.cipher.Encrypt(key.Secret())
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt mfa secret: %w", err)
	}

	if err := s.userRepo.UpdateMFASecret(ctx, userID, secretCiphertext, false); err != nil {
		return nil, fmt.Errorf("failed to store mfa secret: %w", err)
	}

	return &MFASetupResult{
		OTPAuthURI: key.URL(),
		Secret:     key.Secret(),
	}, nil
}

type MFAEnableResult struct {
	RecoveryCodes []string
}

func (s *MFAService) Enable(ctx context.Context, userID int64, code string) (*MFAEnableResult, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	if user.MFASecretCiphertext == nil {
		return nil, ErrMFANotSetup
	}

	secret, err := s.cipher.Decrypt(*user.MFASecretCiphertext)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt mfa secret: %w", err)
	}

	valid := totp.Validate(code, secret)
	if !valid {
		return nil, ErrMFAInvalidCode
	}

	if err := s.userRepo.EnableMFA(ctx, userID); err != nil {
		return nil, fmt.Errorf("failed to enable mfa: %w", err)
	}

	codes, hashes, err := generateRecoveryCodes(recoveryCodeCount)
	if err != nil {
		return nil, fmt.Errorf("failed to generate recovery codes: %w", err)
	}

	if err := s.recoveryRepo.Create(ctx, userID, hashes); err != nil {
		return nil, fmt.Errorf("failed to store recovery codes: %w", err)
	}

	logger.Info().
		Str("component", "auth").
		Str("event", "user.mfa_enabled").
		Int64("userId", userID).
		Msg("mfa enabled")

	return &MFAEnableResult{
		RecoveryCodes: codes,
	}, nil
}

type MFAVerifyResult struct {
	UserID       int64
	UsedRecovery bool
}

func (s *MFAService) Verify(ctx context.Context, mfaToken, code string) (*MFAVerifyResult, error) {
	userID, ok := s.mfaTokenStore.GetAndDelete(mfaToken)
	if !ok {
		return nil, ErrMFAInvalidCode
	}

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	if !user.MFAEnabled || user.MFASecretCiphertext == nil {
		return nil, ErrMFANotSetup
	}

	secret, err := s.cipher.Decrypt(*user.MFASecretCiphertext)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt mfa secret: %w", err)
	}

	valid := totp.Validate(code, secret)
	usedRecovery := false

	if !valid {
		codeHash := hashRecoveryCode(code)
		recoveryCode, err := s.recoveryRepo.FindByHash(ctx, codeHash)
		if err != nil {
			return nil, ErrMFAInvalidCode
		}
		if recoveryCode.UserID != userID {
			return nil, ErrMFAInvalidCode
		}
		if err := s.recoveryRepo.Consume(ctx, recoveryCode.ID); err != nil {
			return nil, fmt.Errorf("failed to consume recovery code: %w", err)
		}
		usedRecovery = true
	}

	return &MFAVerifyResult{
		UserID:       userID,
		UsedRecovery: usedRecovery,
	}, nil
}

func (s *MFAService) Disable(ctx context.Context, userID int64, currentPassword string) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to find user: %w", err)
	}

	match, err := argon2id.ComparePasswordAndHash(currentPassword, user.PasswordHash)
	if err != nil {
		logger.Error().Str("component", "auth").Err(err).Msg("failed to compare password for mfa disable")
		return ErrMFAInvalidPassword
	}
	if !match {
		return ErrMFAInvalidPassword
	}

	if err := s.userRepo.DisableMFA(ctx, userID); err != nil {
		return fmt.Errorf("failed to disable mfa: %w", err)
	}

	if err := s.recoveryRepo.DeleteAllByUserID(ctx, userID); err != nil {
		logger.Error().Str("component", "auth").Err(err).Msg("failed to delete recovery codes")
	}

	logger.Info().
		Str("component", "auth").
		Str("event", "user.mfa_disabled").
		Int64("userId", userID).
		Msg("mfa disabled")

	return nil
}

func (s *MFAService) GenerateMFAToken(userID int64) (string, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("failed to generate mfa token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(tokenBytes)

	if err := s.mfaTokenStore.Set(token, userID, mfaTokenTTL); err != nil {
		return "", fmt.Errorf("failed to store mfa token: %w", err)
	}

	return token, nil
}

func generateRecoveryCodes(count int) (plaintext []string, hashes []string, err error) {
	codes := make([]string, count)
	hashList := make([]string, count)

	for i := 0; i < count; i++ {
		b := make([]byte, 8)
		if _, err := rand.Read(b); err != nil {
			return nil, nil, fmt.Errorf("failed to generate recovery code: %w", err)
		}
		code := base64.RawURLEncoding.EncodeToString(b)[:10]
		codes[i] = code
		hashList[i] = hashRecoveryCode(code)
	}

	return codes, hashList, nil
}

func hashRecoveryCode(code string) string {
	h := sha256.Sum256([]byte(code))
	return base64.RawURLEncoding.EncodeToString(h[:])
}
