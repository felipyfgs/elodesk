package dto

import "time"

type CreateLineInboxReq struct {
	Name              string `json:"name"              validate:"required"`
	LineChannelID     string `json:"lineChannelId"     validate:"required"`
	LineChannelSecret string `json:"lineChannelSecret" validate:"required"`
	LineChannelToken  string `json:"lineChannelToken"  validate:"required"`
}

type UpdateLineInboxReq struct {
	Name              string `json:"name,omitempty"`
	LineChannelSecret string `json:"lineChannelSecret,omitempty"`
	LineChannelToken  string `json:"lineChannelToken,omitempty"`
}

type LineChannelResp struct {
	ID             int64     `json:"id"`
	LineChannelID  string    `json:"lineChannelId"`
	BotBasicID     *string   `json:"botBasicId,omitempty"`
	BotDisplayName *string   `json:"botDisplayName,omitempty"`
	RequiresReauth bool      `json:"requiresReauth"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type LineInboxResp struct {
	InboxResp
	Channel LineChannelResp `json:"channel"`
}
