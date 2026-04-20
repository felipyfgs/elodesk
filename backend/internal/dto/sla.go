package dto

import "backend/internal/model"

type CreateSLAReq struct {
	Name                 string  `json:"name" validate:"required,min=1,max=255"`
	FirstResponseMinutes int     `json:"first_response_minutes" validate:"required,min=1"`
	ResolutionMinutes    int     `json:"resolution_minutes" validate:"required,min=1"`
	BusinessHoursOnly    *bool   `json:"business_hours_only,omitempty"`
	InboxIDs             []int64 `json:"inbox_ids,omitempty"`
	LabelIDs             []int64 `json:"label_ids,omitempty"`
}

type UpdateSLAReq struct {
	Name                 *string `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	FirstResponseMinutes *int    `json:"first_response_minutes,omitempty" validate:"omitempty,min=1"`
	ResolutionMinutes    *int    `json:"resolution_minutes,omitempty" validate:"omitempty,min=1"`
	BusinessHoursOnly    *bool   `json:"business_hours_only,omitempty"`
	InboxIDs             []int64 `json:"inbox_ids,omitempty"`
	LabelIDs             []int64 `json:"label_ids,omitempty"`
}

type SLAResp struct {
	ID                   int64   `json:"id"`
	AccountID            int64   `json:"accountId"`
	Name                 string  `json:"name"`
	FirstResponseMinutes int     `json:"firstResponseMinutes"`
	ResolutionMinutes    int     `json:"resolutionMinutes"`
	BusinessHoursOnly    bool    `json:"businessHoursOnly"`
	InboxIDs             []int64 `json:"inboxIds"`
	LabelIDs             []int64 `json:"labelIds"`
	CreatedAt            string  `json:"createdAt"`
	UpdatedAt            string  `json:"updatedAt"`
}

func SLAToResp(s *model.SLAPolicy, bindings []model.SLABinding) SLAResp {
	var inboxIDs, labelIDs []int64
	for _, b := range bindings {
		if b.InboxID != nil {
			inboxIDs = append(inboxIDs, *b.InboxID)
		}
		if b.LabelID != nil {
			labelIDs = append(labelIDs, *b.LabelID)
		}
	}
	if inboxIDs == nil {
		inboxIDs = []int64{}
	}
	if labelIDs == nil {
		labelIDs = []int64{}
	}
	return SLAResp{
		ID:                   s.ID,
		AccountID:            s.AccountID,
		Name:                 s.Name,
		FirstResponseMinutes: s.FirstResponseMinutes,
		ResolutionMinutes:    s.ResolutionMinutes,
		BusinessHoursOnly:    s.BusinessHoursOnly,
		InboxIDs:             inboxIDs,
		LabelIDs:             labelIDs,
		CreatedAt:            s.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:            s.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func SLAsToResp(slas []model.SLAPolicy, bindingsMap map[int64][]model.SLABinding) []SLAResp {
	result := make([]SLAResp, len(slas))
	for i := range slas {
		result[i] = SLAToResp(&slas[i], bindingsMap[slas[i].ID])
	}
	return result
}
