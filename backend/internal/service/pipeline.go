package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"backend/internal/audit"
	"backend/internal/dto"
	"backend/internal/model"
	"backend/internal/repo"
)

// Sentinel errors exposed to the handler layer for status mapping.
var (
	ErrPipelineNotFound          = repo.ErrPipelineNotFound
	ErrPipelineStageNotFound     = repo.ErrPipelineStageNotFound
	ErrPipelineCardNotFound      = repo.ErrPipelineCardNotFound
	ErrUnknownTemplate           = errors.New("unknown template_key")
	ErrTemplateOrStagesRequired  = errors.New("either template_key or stages must be provided")
	ErrStageHasCards             = errors.New("stage has cards; move them before deleting")
	ErrStageBelongsOtherPipeline = errors.New("stage belongs to a different pipeline")
	ErrCardKindLinkMismatch      = errors.New("linked_entity_type does not match pipeline card_kind")
	ErrLinkedEntityRequired      = errors.New("linked_entity_id is required for this card kind")
	ErrLinkedEntityForbidden     = errors.New("card_kind=free does not accept linked_entity")
	ErrPipelineUserNotInAccount  = errors.New("user does not belong to this account")
	ErrLabelNotInAccount         = errors.New("label does not belong to this account")
)

const positionGap = 100.0
const positionEpsilon = 0.0001

type PipelineService struct {
	pipelineRepo *repo.PipelineRepo
	stageRepo    *repo.PipelineStageRepo
	cardRepo     *repo.PipelineCardRepo
	labelRepo    *repo.LabelRepo
	accountRepo  *repo.AccountRepo
	rt           *RealtimeService
	auditLogger  *audit.Logger
}

func NewPipelineService(
	pipelineRepo *repo.PipelineRepo,
	stageRepo *repo.PipelineStageRepo,
	cardRepo *repo.PipelineCardRepo,
	labelRepo *repo.LabelRepo,
	accountRepo *repo.AccountRepo,
	rt *RealtimeService,
) *PipelineService {
	return &PipelineService{
		pipelineRepo: pipelineRepo,
		stageRepo:    stageRepo,
		cardRepo:     cardRepo,
		labelRepo:    labelRepo,
		accountRepo:  accountRepo,
		rt:           rt,
	}
}

func (s *PipelineService) WithAudit(l *audit.Logger) *PipelineService {
	s.auditLogger = l
	return s
}

// ===== Pipelines =====

// pipelineCreateConfig holds the resolved blueprint after applying template
// defaults / explicit fields, ready to be persisted.
type pipelineCreateConfig struct {
	cardKind    model.CardKind
	color       string
	icon        *string
	templateKey *string
	stages      []dto.PipelineTemplateStage
}

func resolvePipelineConfig(req dto.CreatePipelineReq) (pipelineCreateConfig, error) {
	cfg := pipelineCreateConfig{
		cardKind: model.CardKindFree,
		color:    "#1f93ff",
	}
	if hasTemplate(req) {
		if err := applyTemplate(*req.TemplateKey, &cfg); err != nil {
			return cfg, err
		}
	} else if err := applyCustom(req, &cfg); err != nil {
		return cfg, err
	}
	if req.Color != nil && *req.Color != "" {
		cfg.color = *req.Color
	}
	if req.Icon != nil && *req.Icon != "" {
		i := *req.Icon
		cfg.icon = &i
	}
	return cfg, nil
}

func hasTemplate(req dto.CreatePipelineReq) bool {
	return req.TemplateKey != nil && *req.TemplateKey != ""
}

func applyTemplate(key string, cfg *pipelineCreateConfig) error {
	t, ok := GetTemplate(key)
	if !ok {
		return ErrUnknownTemplate
	}
	k := key
	cfg.templateKey = &k
	cfg.cardKind, _ = dto.CardKindFromString(t.CardKind)
	cfg.color = t.Color
	ic := t.Icon
	cfg.icon = &ic
	cfg.stages = append(cfg.stages, t.Stages...)
	return nil
}

func applyCustom(req dto.CreatePipelineReq, cfg *pipelineCreateConfig) error {
	if len(req.Stages) == 0 {
		return ErrTemplateOrStagesRequired
	}
	if req.CardKind != nil {
		ck, ok := dto.CardKindFromString(*req.CardKind)
		if !ok {
			return fmt.Errorf("invalid card_kind")
		}
		cfg.cardKind = ck
	}
	cfg.stages = append(cfg.stages, req.Stages...)
	return nil
}

// Create makes a new pipeline. Either template_key OR explicit (card_kind + stages) is required.
func (s *PipelineService) Create(ctx context.Context, accountID int64, userID *int64, req dto.CreatePipelineReq) (*model.Pipeline, []model.PipelineStage, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, nil, fmt.Errorf("name is required")
	}
	cfg, err := resolvePipelineConfig(req)
	if err != nil {
		return nil, nil, err
	}

	tx, err := s.pipelineRepo.Pool().Begin(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	p := &model.Pipeline{
		AccountID:   accountID,
		Name:        name,
		Description: req.Description,
		TemplateKey: cfg.templateKey,
		CardKind:    cfg.cardKind,
		Icon:        cfg.icon,
		Color:       cfg.color,
		CreatedBy:   userID,
	}
	if err := s.pipelineRepo.Insert(ctx, tx, p); err != nil {
		return nil, nil, err
	}

	stages := make([]model.PipelineStage, 0, len(cfg.stages))
	for i, st := range cfg.stages {
		stage := buildStageFromTemplate(p.ID, i, st)
		if err := s.stageRepo.Insert(ctx, tx, stage); err != nil {
			return nil, nil, err
		}
		stages = append(stages, *stage)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("failed to commit pipeline creation: %w", err)
	}

	s.rt.BroadcastAccountEvent(accountID, "pipeline.created", map[string]any{
		"pipeline_id":  p.ID,
		"name":         p.Name,
		"template_key": p.TemplateKey,
		"card_kind":    dto.CardKindToString(p.CardKind),
	})
	if s.auditLogger != nil {
		s.auditLogger.Log(ctx, accountID, userID, "created", "pipeline", &p.ID, map[string]any{
			"name":         p.Name,
			"template_key": p.TemplateKey,
		}, "", "")
	}
	return p, stages, nil
}

func buildStageFromTemplate(pipelineID int64, index int, st dto.PipelineTemplateStage) *model.PipelineStage {
	stage := &model.PipelineStage{
		PipelineID: pipelineID,
		Name:       st.Name,
		Position:   float64(index+1) * positionGap,
		Color:      defaultColor(st.Color, "#94a3b8"),
		IsTerminal: st.IsTerminal,
	}
	if st.TerminalKind != nil {
		if k, ok := dto.TerminalKindFromString(*st.TerminalKind); ok {
			stage.TerminalKind = &k
		}
	}
	return stage
}

func (s *PipelineService) List(ctx context.Context, accountID int64, includeArchived bool) ([]model.Pipeline, error) {
	return s.pipelineRepo.ListByAccount(ctx, accountID, includeArchived)
}

func (s *PipelineService) Get(ctx context.Context, id, accountID int64) (*model.Pipeline, []model.PipelineStage, error) {
	p, err := s.pipelineRepo.FindByID(ctx, id, accountID)
	if err != nil {
		return nil, nil, err
	}
	stages, err := s.stageRepo.ListByPipeline(ctx, p.ID)
	if err != nil {
		return nil, nil, err
	}
	return p, stages, nil
}

func (s *PipelineService) Update(ctx context.Context, id, accountID int64, userID *int64, req dto.UpdatePipelineReq) (*model.Pipeline, error) {
	p, err := s.pipelineRepo.FindByID(ctx, id, accountID)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		t := strings.TrimSpace(*req.Name)
		if t == "" {
			return nil, fmt.Errorf("name cannot be empty")
		}
		p.Name = t
	}
	if req.Description != nil {
		p.Description = req.Description
	}
	if req.Icon != nil {
		p.Icon = req.Icon
	}
	if req.Color != nil && *req.Color != "" {
		p.Color = *req.Color
	}
	if err := s.pipelineRepo.Update(ctx, p); err != nil {
		return nil, err
	}
	s.rt.BroadcastAccountEvent(accountID, "pipeline.updated", map[string]any{
		"pipeline_id": p.ID,
		"name":        p.Name,
	})
	if s.auditLogger != nil {
		s.auditLogger.Log(ctx, accountID, userID, "updated", "pipeline", &p.ID, map[string]any{"name": p.Name}, "", "")
	}
	return p, nil
}

func (s *PipelineService) Archive(ctx context.Context, id, accountID int64, userID *int64) error {
	if err := s.pipelineRepo.Archive(ctx, id, accountID); err != nil {
		return err
	}
	s.rt.BroadcastAccountEvent(accountID, "pipeline.archived", map[string]any{"pipeline_id": id})
	if s.auditLogger != nil {
		s.auditLogger.Log(ctx, accountID, userID, "archived", "pipeline", &id, nil, "", "")
	}
	return nil
}

// ===== Stages =====

func (s *PipelineService) CreateStage(ctx context.Context, pipelineID, accountID int64, userID *int64, req dto.CreateStageReq) (*model.PipelineStage, error) {
	if _, err := s.pipelineRepo.FindByID(ctx, pipelineID, accountID); err != nil {
		return nil, err
	}
	maxPos, err := s.stageRepo.MaxPosition(ctx, pipelineID)
	if err != nil {
		return nil, err
	}
	stage := &model.PipelineStage{
		PipelineID: pipelineID,
		Name:       strings.TrimSpace(req.Name),
		Position:   maxPos + positionGap,
		Color:      "#94a3b8",
	}
	if req.Color != nil && *req.Color != "" {
		stage.Color = *req.Color
	}
	if req.IsTerminal != nil {
		stage.IsTerminal = *req.IsTerminal
	}
	if req.TerminalKind != nil {
		if k, ok := dto.TerminalKindFromString(*req.TerminalKind); ok {
			stage.TerminalKind = &k
		}
	}
	if err := s.stageRepo.Insert(ctx, nil, stage); err != nil {
		return nil, err
	}
	s.rt.BroadcastAccountEvent(accountID, "stage.created", map[string]any{
		"pipeline_id": pipelineID,
		"stage_id":    stage.ID,
		"name":        stage.Name,
		"position":    stage.Position,
	})
	if s.auditLogger != nil {
		s.auditLogger.Log(ctx, accountID, userID, "created", "pipeline_stage", &stage.ID, map[string]any{
			"pipeline_id": pipelineID, "name": stage.Name,
		}, "", "")
	}
	return stage, nil
}

func (s *PipelineService) UpdateStage(ctx context.Context, pipelineID, stageID, accountID int64, userID *int64, req dto.UpdateStageReq) (*model.PipelineStage, []model.PipelineStage, error) {
	if _, err := s.pipelineRepo.FindByID(ctx, pipelineID, accountID); err != nil {
		return nil, nil, err
	}
	stage, err := s.stageRepo.FindByID(ctx, stageID)
	if err != nil {
		return nil, nil, err
	}
	if stage.PipelineID != pipelineID {
		return nil, nil, ErrPipelineStageNotFound
	}
	if req.Name != nil {
		stage.Name = strings.TrimSpace(*req.Name)
	}
	if req.Color != nil && *req.Color != "" {
		stage.Color = *req.Color
	}
	if req.Position != nil {
		stage.Position = *req.Position
	}
	if req.IsTerminal != nil {
		stage.IsTerminal = *req.IsTerminal
	}
	if req.TerminalKind != nil {
		if *req.TerminalKind == "" {
			stage.TerminalKind = nil
		} else if k, ok := dto.TerminalKindFromString(*req.TerminalKind); ok {
			stage.TerminalKind = &k
		}
	}
	if err := s.stageRepo.Update(ctx, stage); err != nil {
		return nil, nil, err
	}

	rebalanced, _ := s.maybeRebalanceStages(ctx, pipelineID)

	s.rt.BroadcastAccountEvent(accountID, "stage.updated", map[string]any{
		"pipeline_id": pipelineID,
		"stage_id":    stage.ID,
		"name":        stage.Name,
		"position":    stage.Position,
	})
	for i := range rebalanced {
		st := rebalanced[i]
		s.rt.BroadcastAccountEvent(accountID, "stage.updated", map[string]any{
			"pipeline_id": pipelineID,
			"stage_id":    st.ID,
			"name":        st.Name,
			"position":    st.Position,
		})
	}
	if s.auditLogger != nil {
		s.auditLogger.Log(ctx, accountID, userID, "updated", "pipeline_stage", &stage.ID, map[string]any{
			"pipeline_id": pipelineID,
		}, "", "")
	}
	return stage, rebalanced, nil
}

func (s *PipelineService) DeleteStage(ctx context.Context, pipelineID, stageID, accountID int64, userID *int64) error {
	if _, err := s.pipelineRepo.FindByID(ctx, pipelineID, accountID); err != nil {
		return err
	}
	stage, err := s.stageRepo.FindByID(ctx, stageID)
	if err != nil {
		return err
	}
	if stage.PipelineID != pipelineID {
		return ErrPipelineStageNotFound
	}
	count, err := s.cardRepo.CountByStage(ctx, stageID)
	if err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("%w (count=%d)", ErrStageHasCards, count)
	}
	if err := s.stageRepo.Delete(ctx, stageID); err != nil {
		return err
	}
	s.rt.BroadcastAccountEvent(accountID, "stage.deleted", map[string]any{
		"pipeline_id": pipelineID,
		"stage_id":    stageID,
	})
	if s.auditLogger != nil {
		s.auditLogger.Log(ctx, accountID, userID, "deleted", "pipeline_stage", &stageID, map[string]any{
			"pipeline_id": pipelineID,
		}, "", "")
	}
	return nil
}

// ===== Cards =====

func (s *PipelineService) ListCards(ctx context.Context, pipelineID, accountID int64) ([]repo.CardWithRelations, error) {
	if _, err := s.pipelineRepo.FindByID(ctx, pipelineID, accountID); err != nil {
		return nil, err
	}
	return s.cardRepo.ListByPipelineWithRelations(ctx, pipelineID)
}

func (s *PipelineService) GetCard(ctx context.Context, cardID, accountID int64) (*repo.CardWithRelations, error) {
	return s.cardRepo.FindByIDWithRelations(ctx, cardID, accountID)
}

func (s *PipelineService) CreateCard(ctx context.Context, pipelineID, accountID int64, userID *int64, req dto.CreateCardReq) (*repo.CardWithRelations, error) {
	pipeline, err := s.pipelineRepo.FindByID(ctx, pipelineID, accountID)
	if err != nil {
		return nil, err
	}
	stage, err := s.stageRepo.FindByID(ctx, req.StageID)
	if err != nil {
		return nil, err
	}
	if stage.PipelineID != pipelineID {
		return nil, ErrStageBelongsOtherPipeline
	}
	if err := validateLink(pipeline.CardKind, req.LinkedEntityType, req.LinkedEntityID); err != nil {
		return nil, err
	}

	maxPos, err := s.cardRepo.MaxPositionInStage(ctx, stage.ID)
	if err != nil {
		return nil, err
	}

	card := &model.PipelineCard{
		PipelineID:    pipelineID,
		StageID:       stage.ID,
		Position:      maxPos + positionGap,
		Title:         strings.TrimSpace(req.Title),
		Description:   req.Description,
		ValueCents:    req.ValueCents,
		ValueCurrency: req.ValueCurrency,
		CreatedBy:     userID,
	}
	if req.DueDate != nil && *req.DueDate != "" {
		t, err := dto.ParseDueDate(*req.DueDate)
		if err != nil {
			return nil, fmt.Errorf("invalid due_date: %w", err)
		}
		card.DueDate = t
	}
	if attrs, err := dto.MarshalCustomAttrs(req.CustomAttrs); err == nil {
		card.CustomAttrs = attrs
	}
	if req.LinkedEntityType != nil && req.LinkedEntityID != nil {
		t, _ := dto.LinkedEntityTypeFromString(*req.LinkedEntityType)
		card.LinkedEntityType = &t
		card.LinkedEntityID = req.LinkedEntityID
	}
	if err := s.cardRepo.Insert(ctx, card); err != nil {
		return nil, err
	}
	rel, err := s.cardRepo.FindByIDWithRelations(ctx, card.ID, accountID)
	if err != nil {
		return nil, err
	}
	s.rt.BroadcastAccountEvent(accountID, "card.created", cardEventPayload(rel))
	if s.auditLogger != nil {
		s.auditLogger.Log(ctx, accountID, userID, "created", "pipeline_card", &card.ID, map[string]any{
			"pipeline_id": pipelineID, "stage_id": stage.ID, "title": card.Title,
		}, "", "")
	}
	return rel, nil
}

func (s *PipelineService) UpdateCard(ctx context.Context, cardID, accountID int64, userID *int64, req dto.UpdateCardReq) (*repo.CardWithRelations, error) {
	card, err := s.cardRepo.FindByID(ctx, cardID, accountID)
	if err != nil {
		return nil, err
	}
	if req.Title != nil {
		card.Title = strings.TrimSpace(*req.Title)
	}
	if req.Description != nil {
		card.Description = req.Description
	}
	if req.ValueCents != nil {
		card.ValueCents = req.ValueCents
	}
	if req.ValueCurrency != nil {
		card.ValueCurrency = req.ValueCurrency
	}
	if req.DueDate != nil {
		if *req.DueDate == "" {
			card.DueDate = nil
		} else {
			t, err := dto.ParseDueDate(*req.DueDate)
			if err != nil {
				return nil, fmt.Errorf("invalid due_date: %w", err)
			}
			card.DueDate = t
		}
	}
	if req.CustomAttrs != nil {
		if attrs, err := dto.MarshalCustomAttrs(req.CustomAttrs); err == nil {
			card.CustomAttrs = attrs
		}
	}
	if err := s.cardRepo.Update(ctx, card); err != nil {
		return nil, err
	}
	rel, err := s.cardRepo.FindByIDWithRelations(ctx, card.ID, accountID)
	if err != nil {
		return nil, err
	}
	s.rt.BroadcastAccountEvent(accountID, "card.updated", cardEventPayload(rel))
	if s.auditLogger != nil {
		s.auditLogger.Log(ctx, accountID, userID, "updated", "pipeline_card", &card.ID, map[string]any{
			"pipeline_id": card.PipelineID,
		}, "", "")
	}
	return rel, nil
}

func (s *PipelineService) MoveCard(ctx context.Context, cardID, accountID int64, userID *int64, req dto.MoveCardReq) (*repo.CardWithRelations, error) {
	card, err := s.cardRepo.FindByID(ctx, cardID, accountID)
	if err != nil {
		return nil, err
	}
	dest, err := s.stageRepo.FindByID(ctx, req.StageID)
	if err != nil {
		return nil, err
	}
	if dest.PipelineID != card.PipelineID {
		return nil, ErrStageBelongsOtherPipeline
	}
	fromStage := card.StageID
	moved, err := s.cardRepo.Move(ctx, cardID, dest.ID, req.Position)
	if err != nil {
		return nil, err
	}

	s.maybeRebalanceCards(ctx, accountID, dest.ID)

	rel, err := s.cardRepo.FindByIDWithRelations(ctx, moved.ID, accountID)
	if err != nil {
		return nil, err
	}
	s.rt.BroadcastAccountEvent(accountID, "card.moved", map[string]any{
		"card_id":       moved.ID,
		"pipeline_id":   moved.PipelineID,
		"from_stage_id": fromStage,
		"to_stage_id":   moved.StageID,
		"position":      moved.Position,
		"moved_by":      userID,
	})
	if s.auditLogger != nil {
		s.auditLogger.Log(ctx, accountID, userID, "moved", "pipeline_card", &moved.ID, map[string]any{
			"from_stage_id": fromStage,
			"to_stage_id":   moved.StageID,
			"position":      moved.Position,
		}, "", "")
	}
	return rel, nil
}

func (s *PipelineService) DeleteCard(ctx context.Context, cardID, accountID int64, userID *int64) error {
	card, err := s.cardRepo.FindByID(ctx, cardID, accountID)
	if err != nil {
		return err
	}
	if err := s.cardRepo.Delete(ctx, cardID); err != nil {
		return err
	}
	s.rt.BroadcastAccountEvent(accountID, "card.deleted", map[string]any{
		"card_id":     card.ID,
		"pipeline_id": card.PipelineID,
		"stage_id":    card.StageID,
	})
	if s.auditLogger != nil {
		s.auditLogger.Log(ctx, accountID, userID, "deleted", "pipeline_card", &cardID, map[string]any{
			"pipeline_id": card.PipelineID,
		}, "", "")
	}
	return nil
}

// ===== Assignees / Labels =====

func (s *PipelineService) AssignAgent(ctx context.Context, cardID, accountID int64, userID *int64, targetUserID int64) (*repo.CardWithRelations, error) {
	card, err := s.cardRepo.FindByID(ctx, cardID, accountID)
	if err != nil {
		return nil, err
	}
	exists, err := s.accountRepo.ExistsByUserAndAccount(ctx, targetUserID, accountID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrPipelineUserNotInAccount
	}
	if err := s.cardRepo.AddAssignee(ctx, card.ID, targetUserID); err != nil {
		return nil, err
	}
	rel, err := s.cardRepo.FindByIDWithRelations(ctx, card.ID, accountID)
	if err != nil {
		return nil, err
	}
	s.rt.BroadcastAccountEvent(accountID, "card.updated", cardEventPayload(rel))
	if s.auditLogger != nil {
		s.auditLogger.Log(ctx, accountID, userID, "assignee_added", "pipeline_card", &card.ID, map[string]any{
			"user_id": targetUserID,
		}, "", "")
	}
	return rel, nil
}

func (s *PipelineService) UnassignAgent(ctx context.Context, cardID, accountID int64, userID *int64, targetUserID int64) (*repo.CardWithRelations, error) {
	card, err := s.cardRepo.FindByID(ctx, cardID, accountID)
	if err != nil {
		return nil, err
	}
	if err := s.cardRepo.RemoveAssignee(ctx, card.ID, targetUserID); err != nil {
		return nil, err
	}
	rel, err := s.cardRepo.FindByIDWithRelations(ctx, card.ID, accountID)
	if err != nil {
		return nil, err
	}
	s.rt.BroadcastAccountEvent(accountID, "card.updated", cardEventPayload(rel))
	if s.auditLogger != nil {
		s.auditLogger.Log(ctx, accountID, userID, "assignee_removed", "pipeline_card", &card.ID, map[string]any{
			"user_id": targetUserID,
		}, "", "")
	}
	return rel, nil
}

func (s *PipelineService) ApplyLabel(ctx context.Context, cardID, accountID int64, userID *int64, labelID int64) (*repo.CardWithRelations, error) {
	card, err := s.cardRepo.FindByID(ctx, cardID, accountID)
	if err != nil {
		return nil, err
	}
	if _, err := s.labelRepo.FindByID(ctx, labelID, accountID); err != nil {
		if errors.Is(err, repo.ErrLabelNotFound) {
			return nil, ErrLabelNotInAccount
		}
		return nil, err
	}
	if err := s.cardRepo.AddLabel(ctx, card.ID, labelID); err != nil {
		return nil, err
	}
	rel, err := s.cardRepo.FindByIDWithRelations(ctx, card.ID, accountID)
	if err != nil {
		return nil, err
	}
	s.rt.BroadcastAccountEvent(accountID, "card.updated", cardEventPayload(rel))
	if s.auditLogger != nil {
		s.auditLogger.Log(ctx, accountID, userID, "label_added", "pipeline_card", &card.ID, map[string]any{
			"label_id": labelID,
		}, "", "")
	}
	return rel, nil
}

func (s *PipelineService) RemoveLabel(ctx context.Context, cardID, accountID int64, userID *int64, labelID int64) (*repo.CardWithRelations, error) {
	card, err := s.cardRepo.FindByID(ctx, cardID, accountID)
	if err != nil {
		return nil, err
	}
	if err := s.cardRepo.RemoveLabel(ctx, card.ID, labelID); err != nil {
		return nil, err
	}
	rel, err := s.cardRepo.FindByIDWithRelations(ctx, card.ID, accountID)
	if err != nil {
		return nil, err
	}
	s.rt.BroadcastAccountEvent(accountID, "card.updated", cardEventPayload(rel))
	if s.auditLogger != nil {
		s.auditLogger.Log(ctx, accountID, userID, "label_removed", "pipeline_card", &card.ID, map[string]any{
			"label_id": labelID,
		}, "", "")
	}
	return rel, nil
}

// ===== Helpers =====

func (s *PipelineService) maybeRebalanceCards(ctx context.Context, accountID, stageID int64) {
	positions := s.cardPositions(ctx, stageID)
	if !needsRebalance(positions) {
		return
	}
	updated, err := s.cardRepo.Rebalance(ctx, stageID)
	if err != nil {
		return
	}
	for i := range updated {
		rel, err := s.cardRepo.FindByIDWithRelations(ctx, updated[i].ID, accountID)
		if err != nil {
			continue
		}
		s.rt.BroadcastAccountEvent(accountID, "card.updated", cardEventPayload(rel))
	}
}

func (s *PipelineService) cardPositions(ctx context.Context, stageID int64) []float64 {
	rows, err := s.pipelineRepo.Pool().Query(ctx, `SELECT position FROM pipeline_cards WHERE stage_id = $1 ORDER BY position ASC`, stageID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []float64
	for rows.Next() {
		var p float64
		if err := rows.Scan(&p); err == nil {
			out = append(out, p)
		}
	}
	return out
}

func (s *PipelineService) maybeRebalanceStages(ctx context.Context, pipelineID int64) ([]model.PipelineStage, error) {
	stages, err := s.stageRepo.ListByPipeline(ctx, pipelineID)
	if err != nil {
		return nil, err
	}
	positions := make([]float64, len(stages))
	for i := range stages {
		positions[i] = stages[i].Position
	}
	if !needsRebalance(positions) {
		return nil, nil
	}
	return s.stageRepo.Rebalance(ctx, pipelineID)
}

func needsRebalance(sortedPositions []float64) bool {
	for i := 1; i < len(sortedPositions); i++ {
		if sortedPositions[i]-sortedPositions[i-1] < positionEpsilon {
			return true
		}
	}
	return false
}

func validateLink(kind model.CardKind, linkedType *string, linkedID *int64) error {
	if kind == model.CardKindFree {
		if linkedType != nil || linkedID != nil {
			return ErrLinkedEntityForbidden
		}
		return nil
	}
	expected := "contact"
	if kind == model.CardKindConversation {
		expected = "conversation"
	}
	if linkedType == nil || linkedID == nil {
		return fmt.Errorf("%w (kind=%s)", ErrLinkedEntityRequired, expected)
	}
	if *linkedType != expected {
		return ErrCardKindLinkMismatch
	}
	return nil
}

func defaultColor(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

func cardEventPayload(rel *repo.CardWithRelations) map[string]any {
	return map[string]any{
		"card_id":      rel.Card.ID,
		"pipeline_id":  rel.Card.PipelineID,
		"stage_id":     rel.Card.StageID,
		"title":        rel.Card.Title,
		"position":     rel.Card.Position,
		"updated_at":   rel.Card.UpdatedAt.Format(time.RFC3339),
		"assignee_ids": rel.AssigneeIDs,
		"label_ids":    rel.LabelIDs,
	}
}
