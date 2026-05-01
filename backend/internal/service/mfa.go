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
	ErrMfaInvalidCode     = errors.New("invalid mfa code")
	ErrMfaNotSetup        = errors.New("mfa not set up")
	ErrMfaAlreadyEnabled  = errors.New("mfa already enabled")
	ErrMfaInvalidPassword = errors.New("invalid current password")
)

const (
	mfaTokenTTL       = 5 * time.Minute
	recoveryCodeCount = 8
)

type MfaService struct {
	userRepo      *repo.UserRepo
	recoveryRepo  *repo.MfaRecoveryCodeRepo
	refreshRepo   *repo.RefreshTokenRepo
	cipher        *crypto.Cipher
	mfaTokenStore MfaTokenStore
}

// MfaTokenStore is an interface for storing ephemeral MFA tokens.
// In production this uses Redis; for now we use an in-memory store.
type MfaTokenStore interface {
	Set(key string, userID int64, ttl time.Duration) error
	GetAndDelete(key string) (int64, bool)
}

type InMemoryMfaTokenStore struct {
	mu    sync.Mutex
	store map[string]mfaTokenEntry
}

type mfaTokenEntry struct {
	UserID    int64
	ExpiresAt time.Time
}

func NewInMemoryMfaTokenStore() *InMemoryMfaTokenStore {
	s := &InMemoryMfaTokenStore{
		store: make(map[string]mfaTokenEntry),
	}
	go s.cleanupLoop()
	return s
}

func (s *InMemoryMfaTokenStore) Set(key string, userID int64, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.store[key] = mfaTokenEntry{
		UserID:    userID,
		ExpiresAt: time.Now().Add(ttl),
	}
	return nil
}

func (s *InMemoryMfaTokenStore) GetAndDelete(key string) (int64, bool) {
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

func (s *InMemoryMfaTokenStore) cleanupLoop() {
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

func NewMfaService(
	userRepo *repo.UserRepo,
	recoveryRepo *repo.MfaRecoveryCodeRepo,
	refreshRepo *repo.RefreshTokenRepo,
	cipher *crypto.Cipher,
	tokenStore MfaTokenStore,
) *MfaService {
	return &MfaService{
		userRepo:      userRepo,
		recoveryRepo:  recoveryRepo,
		refreshRepo:   refreshRepo,
		cipher:        cipher,
		mfaTokenStore: tokenStore,
	}
}

type MfaSetupResult struct {
	OTPAuthURI string
	Secret     string
}

// Setup generates a new TOTP secret, encrypts and stores it (mfa_enabled=false),
// and returns the URI and plaintext secret for QR display.
func (s *MfaService) Setup(ctx context.Context, userID int64) (*MfaSetupResult, error) {
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

	if err := s.userRepo.UpdateMfaSecret(ctx, userID, secretCiphertext, false); err != nil {
		return nil, fmt.Errorf("failed to store mfa secret: %w", err)
	}

	return &MfaSetupResult{
		OTPAuthURI: key.URL(),
		Secret:     key.Secret(),
	}, nil
}

type MfaEnableResult struct {
	RecoveryCodes []string
}

// Enable validates the TOTP code, enables MFA, generates recovery codes,
// and returns them ONCE in plaintext.
func (s *MfaService) Enable(ctx context.Context, userID int64, code string) (*MfaEnableResult, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	if user.MfaSecretCiphertext == nil {
		return nil, ErrMfaNotSetup
	}

	secret, err := s.cipher.Decrypt(*user.MfaSecretCiphertext)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt mfa secret: %w", err)
	}

	valid := totp.Validate(code, secret)
	if !valid {
		return nil, ErrMfaInvalidCode
	}

	if err := s.userRepo.EnableMfa(ctx, userID); err != nil {
		return nil, fmt.Errorf("failed to enable mfa: %w", err)
	}

	// Generate recovery codes.
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

	return &MfaEnableResult{
		RecoveryCodes: codes,
	}, nil
}

type MfaVerifyResult struct {
	UserID       int64
	UsedRecovery bool
}

// Verify validates a TOTP code or recovery code against the stored MFA secret.
// If the mfaToken is valid, it returns the user ID for JWT generation.
func (s *MfaService) Verify(ctx context.Context, mfaToken, code string) (*MfaVerifyResult, error) {
	userID, ok := s.mfaTokenStore.GetAndDelete(mfaToken)
	if !ok {
		return nil, ErrMfaInvalidCode
	}

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	if !user.MfaEnabled || user.MfaSecretCiphertext == nil {
		return nil, ErrMfaNotSetup
	}

	secret, err := s.cipher.Decrypt(*user.MfaSecretCiphertext)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt mfa secret: %w", err)
	}

	// Try TOTP first.
	valid := totp.Validate(code, secret)
	usedRecovery := false

	if !valid {
		// Try recovery code.
		codeHash := hashRecoveryCode(code)
		recoveryCode, err := s.recoveryRepo.FindByHash(ctx, codeHash)
		if err != nil {
			return nil, ErrMfaInvalidCode
		}
		if recoveryCode.UserID != userID {
			return nil, ErrMfaInvalidCode
		}
		if err := s.recoveryRepo.Consume(ctx, recoveryCode.ID); err != nil {
			return nil, fmt.Errorf("failed to consume recovery code: %w", err)
		}
		usedRecovery = true
	}

	return &MfaVerifyResult{
		UserID:       userID,
		UsedRecovery: usedRecovery,
	}, nil
}

// Disable deactivates MFA, clears secrets and recovery codes.
func (s *MfaService) Disable(ctx context.Context, userID int64, currentPassword string) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to find user: %w", err)
	}

	match, err := argon2id.ComparePasswordAndHash(currentPassword, user.PasswordHash)
	if err != nil {
		logger.Error().Str("component", "auth").Err(err).Msg("failed to compare password for mfa disable")
		return ErrMfaInvalidPassword
	}
	if !match {
		return ErrMfaInvalidPassword
	}

	if err := s.userRepo.DisableMfa(ctx, userID); err != nil {
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

// GenerateMfaToken creates an ephemeral token for the MFA verification step.
func (s *MfaService) GenerateMfaToken(userID int64) (string, error) {
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
