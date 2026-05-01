package dto

import (
	"backend/internal/model"
)

type CreateNoteReq struct {
	Content string `json:"content" validate:"required,max=50000"`
}

type UpdateNoteReq struct {
	Content string `json:"content" validate:"required,max=50000"`
}

type NoteResp struct {
	ID        int64  `json:"id"`
	AccountID int64  `taccount_id`
	ContactID int64  `tcontact_id`
	UserID    int64  `ruser_id`
	Content   string `json:"content"`
	CreatedAt string `dcreated_at`
	UpdatedAt string `dupdated_at`
}

type NoteListResp struct {
	Meta    MetaResp   `json:"meta"`
	Payload []NoteResp `json:"payload"`
}

func NoteToResp(n *model.Note) NoteResp {
	return NoteResp{
		ID:        n.ID,
		AccountID: n.AccountID,
		ContactID: n.ContactID,
		UserID:    n.UserID,
		Content:   n.Content,
		CreatedAt: n.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: n.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func NotesToResp(notes []model.Note) []NoteResp {
	result := make([]NoteResp, len(notes))
	for i := range notes {
		result[i] = NoteToResp(&notes[i])
	}
	return result
}
