package dto

import "backend/internal/model"

type InviteAgentReq struct {
	Email string  `json:"email" validate:"required,email"`
	Role  int     `json:"role" validate:"required,min=0,max=2"`
	Name  *string `json:"name,omitempty" validate:"omitempty,max=255"`
}

type AcceptInvitationReq struct {
	Password string  `json:"password" validate:"required,min=8"`
	Name     *string `json:"name,omitempty" validate:"omitempty,min=2,max=255"`
}

type UpdateAgentReq struct {
	Role   *int `json:"role,omitempty" validate:"omitempty,min=0,max=2"`
	Status *int `json:"status,omitempty" validate:"omitempty,min=0,max=2"`
}

type AgentResp struct {
	ID         int64   `json:"id"`
	UserID     int64   `json:"userId"`
	Name       string  `json:"name"`
	Email      string  `json:"email"`
	Role       int     `json:"role"`
	Status     string  `json:"status"`
	LastActive *string `json:"lastActive,omitempty"`
	CreatedAt  string  `json:"createdAt"`
}

type InvitationResp struct {
	ID     int64  `json:"id"`
	Email  string `json:"email"`
	Role   int    `json:"role"`
	Status string `json:"status"`
}

func AgentsToResp(agents []AgentResp) []AgentResp {
	return agents
}

func InvitationToResp(inv *model.AgentInvitation) InvitationResp {
	status := "pending"
	if inv.ConsumedAt != nil {
		status = "consumed"
	}
	return InvitationResp{
		ID:     inv.ID,
		Email:  inv.Email,
		Role:   int(inv.Role),
		Status: status,
	}
}
