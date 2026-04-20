package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"backend/internal/model"
	"backend/internal/repo"
)

var ErrTeamNameTaken = repo.ErrTeamNameTaken
var ErrUserNotInAccount = errors.New("user not in account")

type TeamService struct {
	teamRepo    *repo.TeamRepo
	memberRepo  *repo.TeamMemberRepo
	accountRepo *repo.AccountRepo
}

func NewTeamService(teamRepo *repo.TeamRepo, memberRepo *repo.TeamMemberRepo, accountRepo *repo.AccountRepo) *TeamService {
	return &TeamService{teamRepo: teamRepo, memberRepo: memberRepo, accountRepo: accountRepo}
}

func (s *TeamService) List(ctx context.Context, accountID int64) ([]model.Team, error) {
	return s.teamRepo.ListByAccount(ctx, accountID)
}

func (s *TeamService) Create(ctx context.Context, accountID int64, name string, description *string, allowAutoAssign bool) (*model.Team, error) {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return nil, fmt.Errorf("team name is required")
	}
	m := &model.Team{
		AccountID:       accountID,
		Name:            name,
		Description:     description,
		AllowAutoAssign: allowAutoAssign,
	}
	if err := s.teamRepo.Create(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

func (s *TeamService) Update(ctx context.Context, id, accountID int64, name *string, description *string, allowAutoAssign *bool) (*model.Team, error) {
	m, err := s.teamRepo.FindByID(ctx, id, accountID)
	if err != nil {
		return nil, err
	}
	if name != nil {
		m.Name = strings.ToLower(strings.TrimSpace(*name))
		if m.Name == "" {
			return nil, fmt.Errorf("team name is required")
		}
	}
	if description != nil {
		m.Description = description
	}
	if allowAutoAssign != nil {
		m.AllowAutoAssign = *allowAutoAssign
	}
	if err := s.teamRepo.Update(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

func (s *TeamService) Delete(ctx context.Context, id, accountID int64) error {
	return s.teamRepo.Delete(ctx, id, accountID)
}

func (s *TeamService) GetByID(ctx context.Context, id, accountID int64) (*model.Team, error) {
	return s.teamRepo.FindByID(ctx, id, accountID)
}

func (s *TeamService) AddMembers(ctx context.Context, accountID, teamID int64, userIDs []int64) ([]model.TeamMember, error) {
	if _, err := s.teamRepo.FindByID(ctx, teamID, accountID); err != nil {
		return nil, err
	}
	for _, uid := range userIDs {
		exists, err := s.accountRepo.ExistsByUserAndAccount(ctx, uid, accountID)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, ErrUserNotInAccount
		}
	}
	return s.memberRepo.AddMembers(ctx, teamID, userIDs)
}

func (s *TeamService) RemoveMembers(ctx context.Context, accountID, teamID int64, userIDs []int64) error {
	if _, err := s.teamRepo.FindByID(ctx, teamID, accountID); err != nil {
		return err
	}
	return s.memberRepo.RemoveMembers(ctx, teamID, userIDs)
}

func (s *TeamService) ListMembers(ctx context.Context, accountID, teamID int64) ([]model.TeamMember, error) {
	if _, err := s.teamRepo.FindByID(ctx, teamID, accountID); err != nil {
		return nil, err
	}
	return s.memberRepo.ListByTeam(ctx, teamID)
}
