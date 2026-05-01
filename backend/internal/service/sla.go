package service

import (
	"context"
	"fmt"

	"backend/internal/model"
	"backend/internal/repo"
)

type SLAService struct {
	slaRepo *repo.SLARepo
}

func NewSLAService(slaRepo *repo.SLARepo) *SLAService {
	return &SLAService{slaRepo: slaRepo}
}

type UpsertSLAInput struct {
	Name                 string
	FirstResponseMinutes int
	ResolutionMinutes    int
	BusinessHoursOnly    bool
	InboxIDs             []int64
	LabelIDs             []int64
}

func (s *SLAService) Create(ctx context.Context, accountID int64, in UpsertSLAInput) (*model.SLAPolicy, []model.SLABinding, error) {
	m := &model.SLAPolicy{
		AccountID:            accountID,
		Name:                 in.Name,
		FirstResponseMinutes: in.FirstResponseMinutes,
		ResolutionMinutes:    in.ResolutionMinutes,
		BusinessHoursOnly:    in.BusinessHoursOnly,
	}
	if err := s.slaRepo.Create(ctx, m); err != nil {
		return nil, nil, fmt.Errorf("sla.create: %w", err)
	}
	bindings := buildBindings(m.ID, in.InboxIDs, in.LabelIDs)
	if err := s.slaRepo.SetBindings(ctx, m.ID, bindings); err != nil {
		return nil, nil, fmt.Errorf("sla.create: %w", err)
	}
	stored, err := s.slaRepo.GetBindings(ctx, []int64{m.ID})
	if err != nil {
		return nil, nil, fmt.Errorf("sla.create: %w", err)
	}
	return m, stored, nil
}

func (s *SLAService) List(ctx context.Context, accountID int64) ([]model.SLAPolicy, map[int64][]model.SLABinding, error) {
	policies, err := s.slaRepo.ListByAccount(ctx, accountID)
	if err != nil {
		return nil, nil, err
	}
	ids := make([]int64, 0, len(policies))
	for _, p := range policies {
		ids = append(ids, p.ID)
	}
	bindings, err := s.slaRepo.GetBindings(ctx, ids)
	if err != nil {
		return nil, nil, err
	}
	out := make(map[int64][]model.SLABinding)
	for _, b := range bindings {
		out[b.SlaID] = append(out[b.SlaID], b)
	}
	return policies, out, nil
}

func (s *SLAService) Get(ctx context.Context, accountID, id int64) (*model.SLAPolicy, []model.SLABinding, error) {
	m, err := s.slaRepo.FindByID(ctx, id, accountID)
	if err != nil {
		return nil, nil, err
	}
	bindings, err := s.slaRepo.GetBindings(ctx, []int64{m.ID})
	if err != nil {
		return nil, nil, err
	}
	return m, bindings, nil
}

type UpdateSLAInput struct {
	Name                 *string
	FirstResponseMinutes *int
	ResolutionMinutes    *int
	BusinessHoursOnly    *bool
	InboxIDs             *[]int64
	LabelIDs             *[]int64
}

func (s *SLAService) Update(ctx context.Context, accountID, id int64, in UpdateSLAInput) (*model.SLAPolicy, []model.SLABinding, error) {
	m, err := s.slaRepo.FindByID(ctx, id, accountID)
	if err != nil {
		return nil, nil, err
	}
	if in.Name != nil {
		m.Name = *in.Name
	}
	if in.FirstResponseMinutes != nil {
		m.FirstResponseMinutes = *in.FirstResponseMinutes
	}
	if in.ResolutionMinutes != nil {
		m.ResolutionMinutes = *in.ResolutionMinutes
	}
	if in.BusinessHoursOnly != nil {
		m.BusinessHoursOnly = *in.BusinessHoursOnly
	}
	if err := s.slaRepo.Update(ctx, m); err != nil {
		return nil, nil, fmt.Errorf("sla.update: %w", err)
	}
	if in.InboxIDs != nil || in.LabelIDs != nil {
		var inboxIDs, labelIDs []int64
		if in.InboxIDs != nil {
			inboxIDs = *in.InboxIDs
		}
		if in.LabelIDs != nil {
			labelIDs = *in.LabelIDs
		}
		bindings := buildBindings(m.ID, inboxIDs, labelIDs)
		if err := s.slaRepo.SetBindings(ctx, m.ID, bindings); err != nil {
			return nil, nil, fmt.Errorf("sla.update: %w", err)
		}
	}
	bindings, err := s.slaRepo.GetBindings(ctx, []int64{m.ID})
	if err != nil {
		return nil, nil, err
	}
	return m, bindings, nil
}

func (s *SLAService) Delete(ctx context.Context, accountID, id int64) error {
	return s.slaRepo.Delete(ctx, id, accountID)
}

func (s *SLAService) Report(ctx context.Context, accountID int64, from, to string) (*repo.SLAReport, error) {
	return s.slaRepo.Report(ctx, accountID, from, to)
}

func buildBindings(slaID int64, inboxIDs, labelIDs []int64) []model.SLABinding {
	out := make([]model.SLABinding, 0, len(inboxIDs)+len(labelIDs))
	for _, id := range inboxIDs {
		id := id
		out = append(out, model.SLABinding{SlaID: slaID, InboxID: &id})
	}
	for _, id := range labelIDs {
		id := id
		out = append(out, model.SLABinding{SlaID: slaID, LabelID: &id})
	}
	return out
}
