package dto

import (
	"encoding/json"
	"time"

	"backend/internal/model"
)

type CreateContactReq struct {
	Name               string          `json:"name" validate:"required"`
	Email              *string         `json:"email,omitempty"`
	Phone              *string         `json:"phone_number,omitempty"`
	Identifier         *string         `json:"identifier,omitempty"`
	SourceID           string          `json:"source_id,omitempty"`
	IdentifierHash     *string         `json:"identifier_hash,omitempty"`
	CustomAttributes   json.RawMessage `json:"custom_attributes,omitempty"`
	AdditionalAttributes json.RawMessage `json:"additional_attributes,omitempty"`
}

type UpdateContactReq struct {
	Name  *string `json:"name,omitempty"`
	Email *string `json:"email,omitempty"`
	Phone *string `json:"phone_number,omitempty"`
}

type ContactMergeReq struct {
	PrimaryContactID int64 `json:"primary_contact_id" validate:"required"`
}

type ContactBlockReq struct {
	Blocked bool `json:"blocked"`
}

type ContactAvatarReq struct {
	ObjectKey string `json:"object_key" validate:"required"`
}

type ContactResp struct {
	ID              int64      `json:"id"`
	AccountID       int64      `json:"accountId"`
	Name            string     `json:"name"`
	Email           *string    `json:"email,omitempty"`
	PhoneNumber     *string    `json:"phoneNumber,omitempty"`
	Identifier      *string    `json:"identifier,omitempty"`
	AdditionalAttrs *string    `json:"additionalAttributes,omitempty"`
	AvatarURL       *string    `json:"avatarUrl,omitempty"`
	Blocked         bool       `json:"blocked"`
	LastActivityAt  *time.Time `json:"lastActivityAt,omitempty"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

type ContactListResp struct {
	Meta    MetaResp      `json:"meta"`
	Payload []ContactResp `json:"payload"`
}

func ContactToResp(c *model.Contact) ContactResp {
	return ContactResp{
		ID:              c.ID,
		AccountID:       c.AccountID,
		Name:            c.Name,
		Email:           c.Email,
		PhoneNumber:     c.PhoneNumber,
		Identifier:      c.Identifier,
		AdditionalAttrs: c.AdditionalAttrs,
		AvatarURL:       c.AvatarURL,
		Blocked:         c.Blocked,
		LastActivityAt:  c.LastActivityAt,
		CreatedAt:       c.CreatedAt,
		UpdatedAt:       c.UpdatedAt,
	}
}

func ContactsToResp(contacts []model.Contact) []ContactResp {
	result := make([]ContactResp, len(contacts))
	for i := range contacts {
		result[i] = ContactToResp(&contacts[i])
	}
	return result
}

type ContactImportResp struct {
	Inserted  int           `json:"inserted"`
	Updated   int           `json:"updated"`
	Errors    []ImportError `json:"errors,omitempty"`
	TotalRows int           `json:"totalRows"`
}

type ImportError struct {
	Row    int    `json:"row"`
	Reason string `json:"reason"`
}

type AuditEventUserResp struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type AuditEventResp struct {
	ID        int64               `json:"id"`
	Action    string              `json:"action"`
	Metadata  json.RawMessage     `json:"metadata,omitempty"`
	User      *AuditEventUserResp `json:"user,omitempty"`
	CreatedAt time.Time           `json:"createdAt"`
}

type AuditEventListResp struct {
	Meta    MetaResp         `json:"meta"`
	Payload []AuditEventResp `json:"payload"`
}
