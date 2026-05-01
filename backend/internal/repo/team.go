package repo

import (
	"context"
	"errors"
	"fmt"

	"backend/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrTeamNotFound = errors.New("team not found")
var ErrTeamNameTaken = errors.New("team name already taken")

const teamSelectColumns = "id, account_id, name, description, allow_auto_assign, created_at, updated_at"

type teamScanner interface {
	Scan(dest ...any) error
}

func scanTeam(scanner teamScanner, m *model.Team) error {
	return scanner.Scan(&m.ID, &m.AccountID, &m.Name, &m.Description, &m.AllowAutoAssign, &m.CreatedAt, &m.UpdatedAt)
}

type TeamRepo struct {
	pool *pgxpool.Pool
}

func NewTeamRepo(pool *pgxpool.Pool) *TeamRepo {
	return &TeamRepo{pool: pool}
}

func (r *TeamRepo) Create(ctx context.Context, m *model.Team) error {
	query := `INSERT INTO teams (account_id, name, description, allow_auto_assign)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at`
	err := r.pool.QueryRow(ctx, query, m.AccountID, m.Name, m.Description, m.AllowAutoAssign).
		Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("%w: %w", ErrTeamNameTaken, err)
		}
		return fmt.Errorf("failed to create team: %w", err)
	}
	return nil
}

func (r *TeamRepo) FindByID(ctx context.Context, id, accountID int64) (*model.Team, error) {
	query := `SELECT ` + teamSelectColumns + ` FROM teams WHERE id = $1 AND account_id = $2`
	row := r.pool.QueryRow(ctx, query, id, accountID)
	var m model.Team
	if err := scanTeam(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrTeamNotFound, err)
		}
		return nil, fmt.Errorf("failed to find team: %w", err)
	}
	return &m, nil
}

func (r *TeamRepo) ListByAccount(ctx context.Context, accountID int64) ([]model.Team, error) {
	query := `SELECT ` + teamSelectColumns + ` FROM teams WHERE account_id = $1 ORDER BY name ASC`
	rows, err := r.pool.Query(ctx, query, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to list teams: %w", err)
	}
	defer rows.Close()

	var teams []model.Team
	for rows.Next() {
		var m model.Team
		if err := scanTeam(rows, &m); err != nil {
			return nil, fmt.Errorf("failed to scan team: %w", err)
		}
		teams = append(teams, m)
	}
	return teams, rows.Err()
}

func (r *TeamRepo) Update(ctx context.Context, m *model.Team) error {
	query := `UPDATE teams SET name = $3, description = $4, allow_auto_assign = $5, updated_at = NOW()
		WHERE id = $1 AND account_id = $2
		RETURNING ` + teamSelectColumns
	row := r.pool.QueryRow(ctx, query, m.ID, m.AccountID, m.Name, m.Description, m.AllowAutoAssign)
	if err := scanTeam(row, m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w: %w", ErrTeamNotFound, err)
		}
		if isUniqueViolation(err) {
			return fmt.Errorf("%w: %w", ErrTeamNameTaken, err)
		}
		return fmt.Errorf("failed to update team: %w", err)
	}
	return nil
}

func (r *TeamRepo) Delete(ctx context.Context, id, accountID int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM teams WHERE id = $1 AND account_id = $2`, id, accountID)
	if err != nil {
		return fmt.Errorf("failed to delete team: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: %w", ErrTeamNotFound, pgx.ErrNoRows)
	}
	return nil
}

const teamMemberSelectColumns = "id, team_id, user_id, created_at"

func scanTeamMember(scanner interface{ Scan(dest ...any) error }, m *model.TeamMember) error {
	return scanner.Scan(&m.ID, &m.TeamID, &m.UserID, &m.CreatedAt)
}

type TeamMemberRepo struct {
	pool *pgxpool.Pool
}

func NewTeamMemberRepo(pool *pgxpool.Pool) *TeamMemberRepo {
	return &TeamMemberRepo{pool: pool}
}

func (r *TeamMemberRepo) AddMembers(ctx context.Context, teamID int64, userIDs []int64) ([]model.TeamMember, error) {
	query := `INSERT INTO team_members (team_id, user_id) VALUES ($1, $2)
		ON CONFLICT (team_id, user_id) DO NOTHING
		RETURNING ` + teamMemberSelectColumns

	var members []model.TeamMember
	for _, uid := range userIDs {
		var m model.TeamMember
		err := r.pool.QueryRow(ctx, query, teamID, uid).Scan(&m.ID, &m.TeamID, &m.UserID, &m.CreatedAt)
		if err == nil {
			members = append(members, m)
		}
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("failed to add team member: %w", err)
		}
	}
	return members, nil
}

func (r *TeamMemberRepo) RemoveMembers(ctx context.Context, teamID int64, userIDs []int64) error {
	if len(userIDs) == 0 {
		return nil
	}
	query := `DELETE FROM team_members WHERE team_id = $1 AND user_id = ANY($2)`
	_, err := r.pool.Exec(ctx, query, teamID, userIDs)
	if err != nil {
		return fmt.Errorf("failed to remove team members: %w", err)
	}
	return nil
}

func (r *TeamMemberRepo) ListByTeam(ctx context.Context, teamID int64) ([]model.TeamMember, error) {
	query := `SELECT ` + teamMemberSelectColumns + ` FROM team_members WHERE team_id = $1 ORDER BY created_at ASC`
	rows, err := r.pool.Query(ctx, query, teamID)
	if err != nil {
		return nil, fmt.Errorf("failed to list team members: %w", err)
	}
	defer rows.Close()

	var members []model.TeamMember
	for rows.Next() {
		var m model.TeamMember
		if err := scanTeamMember(rows, &m); err != nil {
			return nil, fmt.Errorf("failed to scan team member: %w", err)
		}
		members = append(members, m)
	}
	return members, rows.Err()
}
