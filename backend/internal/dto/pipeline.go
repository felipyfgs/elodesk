package dto

import (
	"time"

	"backend/internal/model"
)

const (
	isoFormat                = "2006-01-02T15:04:05Z07:00"
	linkedTypeContact        = "contact"
	linkedTypeConversation   = "conversation"
	cardKindLabelFree        = "free"
	cardKindLabelContact     = "contact"
	cardKindLabelConv        = "conversation"
	terminalKindLabelWon     = "won"
	terminalKindLabelLost    = "lost"
	terminalKindLabelResolve = "resolved"
)

// ===== Templates =====

type PipelineTemplate struct {
	Key         string                  `json:"key"`
	Name        string                  `json:"name"`
	Description string                  `json:"description"`
	Icon        string                  `json:"icon"`
	Color       string                  `json:"color"`
	CardKind    string                  `json:"card_kind"`
	Stages      []PipelineTemplateStage `json:"stages"`
}

type PipelineTemplateStage struct {
	Name         string  `json:"name"`
	Color        string  `json:"color"`
	IsTerminal   bool    `json:"is_terminal,omitempty"`
	TerminalKind *string `json:"terminal_kind,omitempty"`
}

// ===== Pipelines =====

type CreatePipelineReq struct {
	Name        string                  `json:"name" validate:"required,min=1,max=255"`
	Description *string                 `json:"description,omitempty"`
	TemplateKey *string                 `json:"template_key,omitempty"`
	CardKind    *string                 `json:"card_kind,omitempty" validate:"omitempty,oneof=free contact conversation"`
	Icon        *string                 `json:"icon,omitempty"`
	Color       *string                 `json:"color,omitempty" validate:"omitempty,hexcolor"`
	Stages      []PipelineTemplateStage `json:"stages,omitempty"`
}

type UpdatePipelineReq struct {
	Name        *string `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Description *string `json:"description,omitempty"`
	Icon        *string `json:"icon,omitempty"`
	Color       *string `json:"color,omitempty" validate:"omitempty,hexcolor"`
}

type PipelineResp struct {
	ID          int64        `json:"id"`
	AccountID   int64        `json:"account_id"`
	Name        string       `json:"name"`
	Description *string      `json:"description,omitempty"`
	TemplateKey *string      `json:"template_key,omitempty"`
	CardKind    string       `json:"card_kind"`
	Icon        *string      `json:"icon,omitempty"`
	Color       string       `json:"color"`
	ArchivedAt  *string      `json:"archived_at,omitempty"`
	CreatedBy   *int64       `json:"created_by,omitempty"`
	CreatedAt   string       `json:"created_at"`
	UpdatedAt   string       `json:"updated_at"`
	Stages      []StageResp  `json:"stages,omitempty"`
}

func PipelineToResp(p *model.Pipeline, stages []model.PipelineStage) PipelineResp {
	resp := PipelineResp{
		ID:          p.ID,
		AccountID:   p.AccountID,
		Name:        p.Name,
		Description: p.Description,
		TemplateKey: p.TemplateKey,
		CardKind:    CardKindToString(p.CardKind),
		Icon:        p.Icon,
		Color:       p.Color,
		CreatedBy:   p.CreatedBy,
		CreatedAt:   p.CreatedAt.Format(isoFormat),
		UpdatedAt:   p.UpdatedAt.Format(isoFormat),
	}
	if p.ArchivedAt != nil {
		s := p.ArchivedAt.Format(isoFormat)
		resp.ArchivedAt = &s
	}
	if stages != nil {
		resp.Stages = StagesToResp(stages)
	}
	return resp
}

func PipelinesToResp(items []model.Pipeline) []PipelineResp {
	out := make([]PipelineResp, len(items))
	for i := range items {
		out[i] = PipelineToResp(&items[i], nil)
	}
	return out
}

// ===== Stages =====

type CreateStageReq struct {
	Name         string  `json:"name" validate:"required,min=1,max=255"`
	Color        *string `json:"color,omitempty" validate:"omitempty,hexcolor"`
	IsTerminal   *bool   `json:"is_terminal,omitempty"`
	TerminalKind *string `json:"terminal_kind,omitempty" validate:"omitempty,oneof=won lost resolved"`
}

type UpdateStageReq struct {
	Name         *string  `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Color        *string  `json:"color,omitempty" validate:"omitempty,hexcolor"`
	Position     *float64 `json:"position,omitempty"`
	IsTerminal   *bool    `json:"is_terminal,omitempty"`
	TerminalKind *string  `json:"terminal_kind,omitempty" validate:"omitempty,oneof=won lost resolved"`
}

type StageResp struct {
	ID           int64   `json:"id"`
	PipelineID   int64   `json:"pipeline_id"`
	Name         string  `json:"name"`
	Position     float64 `json:"position"`
	Color        string  `json:"color"`
	IsTerminal   bool    `json:"is_terminal"`
	TerminalKind *string `json:"terminal_kind,omitempty"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

func StageToResp(s *model.PipelineStage) StageResp {
	resp := StageResp{
		ID:         s.ID,
		PipelineID: s.PipelineID,
		Name:       s.Name,
		Position:   s.Position,
		Color:      s.Color,
		IsTerminal: s.IsTerminal,
		CreatedAt:  s.CreatedAt.Format(isoFormat),
		UpdatedAt:  s.UpdatedAt.Format(isoFormat),
	}
	if s.TerminalKind != nil {
		k := TerminalKindToString(*s.TerminalKind)
		resp.TerminalKind = &k
	}
	return resp
}

func StagesToResp(items []model.PipelineStage) []StageResp {
	out := make([]StageResp, len(items))
	for i := range items {
		out[i] = StageToResp(&items[i])
	}
	return out
}

// ===== Cards =====

type CreateCardReq struct {
	StageID          int64           `json:"stage_id" validate:"required"`
	Title            string          `json:"title" validate:"required,min=1,max=500"`
	Description      *string         `json:"description,omitempty"`
	ValueCents       *int64          `json:"value_cents,omitempty"`
	ValueCurrency    *string         `json:"value_currency,omitempty" validate:"omitempty,len=3"`
	DueDate          *string         `json:"due_date,omitempty" validate:"omitempty,datetime=2006-01-02"`
	CustomAttrs      map[string]any  `json:"custom_attrs,omitempty"`
	LinkedEntityType *string         `json:"linked_entity_type,omitempty" validate:"omitempty,oneof=contact conversation"`
	LinkedEntityID   *int64          `json:"linked_entity_id,omitempty"`
}

type UpdateCardReq struct {
	Title         *string        `json:"title,omitempty" validate:"omitempty,min=1,max=500"`
	Description   *string        `json:"description,omitempty"`
	ValueCents    *int64         `json:"value_cents,omitempty"`
	ValueCurrency *string        `json:"value_currency,omitempty" validate:"omitempty,len=3"`
	DueDate       *string        `json:"due_date,omitempty" validate:"omitempty,datetime=2006-01-02"`
	CustomAttrs   map[string]any `json:"custom_attrs,omitempty"`
}

type MoveCardReq struct {
	StageID  int64   `json:"stage_id" validate:"required"`
	Position float64 `json:"position"`
}

type AssignAgentReq struct {
	UserID int64 `json:"user_id" validate:"required"`
}

type ApplyCardLabelReq struct {
	LabelID int64 `json:"label_id" validate:"required"`
}

type CardResp struct {
	ID               int64          `json:"id"`
	PipelineID       int64          `json:"pipeline_id"`
	StageID          int64          `json:"stage_id"`
	Position         float64        `json:"position"`
	Title            string         `json:"title"`
	Description      *string        `json:"description,omitempty"`
	ValueCents       *int64         `json:"value_cents,omitempty"`
	ValueCurrency    *string        `json:"value_currency,omitempty"`
	DueDate          *string        `json:"due_date,omitempty"`
	CustomAttrs      map[string]any `json:"custom_attrs"`
	LinkedEntityType *string        `json:"linked_entity_type,omitempty"`
	LinkedEntityID   *int64         `json:"linked_entity_id,omitempty"`
	AssigneeUserIDs  []int64        `json:"assignee_user_ids"`
	LabelIDs         []int64        `json:"label_ids"`
	CreatedBy        *int64         `json:"created_by,omitempty"`
	CreatedAt        string         `json:"created_at"`
	UpdatedAt        string         `json:"updated_at"`
}

func CardToResp(c *model.PipelineCard, assignees []int64, labels []int64) CardResp {
	resp := CardResp{
		ID:              c.ID,
		PipelineID:      c.PipelineID,
		StageID:         c.StageID,
		Position:        c.Position,
		Title:           c.Title,
		Description:     c.Description,
		ValueCents:      c.ValueCents,
		ValueCurrency:   c.ValueCurrency,
		AssigneeUserIDs: assignees,
		LabelIDs:        labels,
		CreatedBy:       c.CreatedBy,
		CreatedAt:       c.CreatedAt.Format(isoFormat),
		UpdatedAt:       c.UpdatedAt.Format(isoFormat),
	}
	if resp.AssigneeUserIDs == nil {
		resp.AssigneeUserIDs = []int64{}
	}
	if resp.LabelIDs == nil {
		resp.LabelIDs = []int64{}
	}
	if c.DueDate != nil {
		s := c.DueDate.Format("2006-01-02")
		resp.DueDate = &s
	}
	if c.LinkedEntityType != nil {
		t := LinkedEntityTypeToString(*c.LinkedEntityType)
		resp.LinkedEntityType = &t
	}
	if c.LinkedEntityID != nil {
		resp.LinkedEntityID = c.LinkedEntityID
	}
	if c.CustomAttrs != "" {
		resp.CustomAttrs = parseCustomAttrs(c.CustomAttrs)
	}
	if resp.CustomAttrs == nil {
		resp.CustomAttrs = map[string]any{}
	}
	return resp
}

// ===== Helpers =====

func CardKindToString(k model.CardKind) string {
	switch k {
	case model.CardKindContact:
		return cardKindLabelContact
	case model.CardKindConversation:
		return cardKindLabelConv
	default:
		return cardKindLabelFree
	}
}

func CardKindFromString(s string) (model.CardKind, bool) {
	switch s {
	case cardKindLabelFree:
		return model.CardKindFree, true
	case cardKindLabelContact:
		return model.CardKindContact, true
	case cardKindLabelConv:
		return model.CardKindConversation, true
	}
	return 0, false
}

func LinkedEntityTypeToString(t model.LinkedEntityType) string {
	if t == model.LinkedEntityConversation {
		return linkedTypeConversation
	}
	return linkedTypeContact
}

func LinkedEntityTypeFromString(s string) (model.LinkedEntityType, bool) {
	switch s {
	case linkedTypeContact:
		return model.LinkedEntityContact, true
	case linkedTypeConversation:
		return model.LinkedEntityConversation, true
	}
	return 0, false
}

func TerminalKindToString(k model.TerminalKind) string {
	switch k {
	case model.TerminalKindLost:
		return terminalKindLabelLost
	case model.TerminalKindResolved:
		return terminalKindLabelResolve
	default:
		return terminalKindLabelWon
	}
}

func TerminalKindFromString(s string) (model.TerminalKind, bool) {
	switch s {
	case terminalKindLabelWon:
		return model.TerminalKindWon, true
	case terminalKindLabelLost:
		return model.TerminalKindLost, true
	case terminalKindLabelResolve:
		return model.TerminalKindResolved, true
	}
	return 0, false
}

func ParseDueDate(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
