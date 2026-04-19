package dto

import (
	"backend/internal/model"
)

type CreateTeamReq struct {
	Name            string  `json:"name" validate:"required,min=1,max=255"`
	Description     *string `json:"description,omitempty"`
	AllowAutoAssign *bool   `json:"allow_auto_assign,omitempty"`
}

type UpdateTeamReq struct {
	Name            *string `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Description     *string `json:"description,omitempty"`
	AllowAutoAssign *bool   `json:"allow_auto_assign,omitempty"`
}

type AddTeamMembersReq struct {
	UserIDs []int64 `json:"user_ids" validate:"required,min=1"`
}

type RemoveTeamMembersReq struct {
	UserIDs []int64 `json:"user_ids" validate:"required,min=1"`
}

type TeamResp struct {
	ID              int64   `json:"id"`
	AccountID       int64   `json:"accountId"`
	Name            string  `json:"name"`
	Description     *string `json:"description,omitempty"`
	AllowAutoAssign bool    `json:"allowAutoAssign"`
	CreatedAt       string  `json:"createdAt"`
	UpdatedAt       string  `json:"updatedAt"`
}

func TeamToResp(t *model.Team) TeamResp {
	return TeamResp{
		ID:              t.ID,
		AccountID:       t.AccountID,
		Name:            t.Name,
		Description:     t.Description,
		AllowAutoAssign: t.AllowAutoAssign,
		CreatedAt:       t.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:       t.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func TeamsToResp(teams []model.Team) []TeamResp {
	result := make([]TeamResp, len(teams))
	for i := range teams {
		result[i] = TeamToResp(&teams[i])
	}
	return result
}
