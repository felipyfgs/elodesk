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
)

var (
	ErrInvitationNotFound       = errors.New("invitation not found")
	ErrInvitationExpired        = errors.New("invitation expired")
	ErrInvitationAlreadyUsed    = errors.New("invitation already used")
	ErrCannotDemoteLastOwner    = errors.New("cannot demote last owner")
	ErrAgentNotFound            = errors.New("agent not found")
	ErrInvitationAlreadyPending = errors.New("invitation already pending")
)

type AgentService struct {
	agentRepo      *repo.AgentRepo
	invitationRepo *repo.AgentInvitationRepo
	userRepo       *repo.UserRepo
	accountRepo    *repo.AccountRepo
	authSvc        *AuthService
}

func NewAgentService(
	agentRepo *repo.AgentRepo,
	invitationRepo *repo.AgentInvitationRepo,
	userRepo *repo.UserRepo,
	accountRepo *repo.AccountRepo,
	authSvc *AuthService,
) *AgentService {
	return &AgentService{
		agentRepo:      agentRepo,
		invitationRepo: invitationRepo,
		userRepo:       userRepo,
		accountRepo:    accountRepo,
		authSvc:        authSvc,
	}
}

func (s *AgentService) List(ctx context.Context, accountID int64) ([]repo.AgentMember, error) {
	return s.agentRepo.ListByAccount(ctx, accountID)
}

type InviteResult struct {
	InvitationID int64
	Status       string
}

func (s *AgentService) Invite(ctx context.Context, accountID int64, email string, role int, name *string, createdBy int64) (*InviteResult, error) {
	token := generateInvitationToken()
	tokenHash := hashInvitationToken(token)

	inv := &model.AgentInvitation{
		AccountID: accountID,
		Email:     email,
		Role:      model.Role(role),
		Name:      name,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(48 * time.Hour),
		CreatedBy: createdBy,
	}

	if err := s.invitationRepo.Create(ctx, inv); err != nil {
		if errors.Is(err, repo.ErrInvitationAlreadyPending) {
			return nil, ErrInvitationAlreadyPending
		}
		return nil, fmt.Errorf("agent.invite: %w", err)
	}

	logger.Info().Str("component", "agents").Str("email", email).Int64("accountId", accountID).Msg("invitation created")

	return &InviteResult{
		InvitationID: inv.ID,
		Status:       "pending",
	}, nil
}

type AcceptInvitationResult struct {
	User         *model.User
	Account      *model.Account
	AccessToken  string
	RefreshToken string
}

func (s *AgentService) AcceptInvitation(ctx context.Context, token, password string, name *string) (*AcceptInvitationResult, error) {
	tokenHash := hashInvitationToken(token)
	inv, err := s.invitationRepo.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, repo.ErrInvitationNotFound) {
			return nil, ErrInvitationNotFound
		}
		return nil, fmt.Errorf("agent.accept: %w", err)
	}

	if inv.ConsumedAt != nil {
		return nil, ErrInvitationAlreadyUsed
	}

	if time.Now().UTC().After(inv.ExpiresAt) {
		return nil, ErrInvitationExpired
	}

	hash, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		return nil, fmt.Errorf("agent.accept: %w", err)
	}

	userName := inv.Email
	if name != nil && *name != "" {
		userName = *name
	}

	user := &model.User{
		Email:        inv.Email,
		Name:         userName,
		PasswordHash: hash,
	}

	tx, err := s.userRepo.Pool().Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("agent.accept: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := s.userRepo.CreateTx(ctx, tx, user); err != nil {
		if errors.Is(err, repo.ErrUserEmailExists) {
			existing, findErr := s.userRepo.FindByEmail(ctx, inv.Email)
			if findErr != nil {
				return nil, fmt.Errorf("agent.accept: %w", findErr)
			}
			user = existing
		} else {
			return nil, fmt.Errorf("agent.accept: %w", err)
		}
	}

	isMember, err := s.agentRepo.IsMember(ctx, inv.AccountID, user.ID)
	if err != nil {
		return nil, fmt.Errorf("agent.accept: %w", err)
	}

	if !isMember {
		if _, err := s.accountRepo.AddUserTx(ctx, tx, inv.AccountID, user.ID, inv.Role); err != nil {
			return nil, fmt.Errorf("agent.accept: %w", err)
		}
	}

	if err := s.invitationRepo.MarkConsumedTx(ctx, tx, inv.ID); err != nil {
		return nil, fmt.Errorf("agent.accept: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("agent.accept: %w", err)
	}

	account, err := s.accountRepo.FindByID(ctx, inv.AccountID)
	if err != nil {
		return nil, fmt.Errorf("agent.accept: %w", err)
	}

	result, err := s.authSvc.IssueTokenPair(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("agent.accept: %w", err)
	}

	return &AcceptInvitationResult{
		User:         result.User,
		Account:      account,
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
	}, nil
}

func (s *AgentService) UpdateAgent(ctx context.Context, accountID, userID int64, role *int) error {
	if role != nil {
		if *role < int(model.RoleOwner) {
			currentRole, err := s.agentRepo.GetRole(ctx, accountID, userID)
			if err != nil {
				return fmt.Errorf("agent.update: %w", err)
			}
			if currentRole == int(model.RoleOwner) {
				ownerCount, err := s.agentRepo.CountOwners(ctx, accountID)
				if err != nil {
					return fmt.Errorf("agent.update: %w", err)
				}
				if ownerCount <= 1 {
					return ErrCannotDemoteLastOwner
				}
			}
		}

		if err := s.agentRepo.UpdateRole(ctx, accountID, userID, *role); err != nil {
			return fmt.Errorf("agent.update: %w", err)
		}
	}
	return nil
}

func (s *AgentService) RemoveAgent(ctx context.Context, accountID, userID int64) error {
	ownerCount, err := s.agentRepo.CountOwners(ctx, accountID)
	if err != nil {
		return fmt.Errorf("agent.remove: %w", err)
	}

	isMember, err := s.agentRepo.IsMember(ctx, accountID, userID)
	if err != nil {
		return fmt.Errorf("agent.remove: %w", err)
	}

	if isMember {
		role, err := s.agentRepo.GetRole(ctx, accountID, userID)
		if err != nil {
			return fmt.Errorf("agent.remove: %w", err)
		}
		if role == int(model.RoleOwner) && ownerCount <= 1 {
			return ErrCannotDemoteLastOwner
		}
	}

	if err := s.agentRepo.Remove(ctx, accountID, userID); err != nil {
		return fmt.Errorf("agent.remove: %w", err)
	}
	return nil
}

func generateInvitationToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func hashInvitationToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(h[:])
}
