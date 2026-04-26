package dto

import (
	"encoding/json"
	"time"

	"backend/internal/model"
)

type CreateContactReq struct {
	Name                 string          `json:"name" validate:"required"`
	Email                *string         `json:"email,omitempty"`
	Phone                *string         `json:"phone_number,omitempty"`
	Identifier           *string         `json:"identifier,omitempty"`
	SourceID             string          `json:"source_id,omitempty"`
	IdentifierHash       *string         `json:"identifier_hash,omitempty"`
	AvatarURL            *string         `json:"avatar_url,omitempty"`
	CustomAttributes     json.RawMessage `json:"custom_attributes,omitempty"`
	AdditionalAttributes json.RawMessage `json:"additional_attributes,omitempty"`
}

type UpdateContactReq struct {
	Name      *string `json:"name,omitempty"`
	Email     *string `json:"email,omitempty"`
	Phone     *string `json:"phone_number,omitempty"`
	AvatarURL *string `json:"avatar_url,omitempty"`
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

// ContactResp mirrors Chatwoot's _contact.json.jbuilder so the cloned Vue
// frontend renders without adapter shims. snake_case JSON keys; timestamps
// emitted as epoch seconds (int64) — matches Chatwoot, differs from the
// RFC3339 default used by other Elodesk DTOs.
type ContactResp struct {
	ID                   int64           `json:"id"`
	AccountID            int64           `json:"account_id"`
	Name                 string          `json:"name"`
	Email                *string         `json:"email,omitempty"`
	PhoneNumber          *string         `json:"phone_number,omitempty"`
	Identifier           *string         `json:"identifier,omitempty"`
	AvailabilityStatus   string          `json:"availability_status"`
	Blocked              bool            `json:"blocked"`
	Thumbnail            string          `json:"thumbnail"`
	AvatarURL            *string         `json:"avatar_url,omitempty"`
	AdditionalAttributes json.RawMessage `json:"additional_attributes,omitempty"`
	CustomAttributes     json.RawMessage `json:"custom_attributes,omitempty"`
	LastActivityAt       *int64          `json:"last_activity_at,omitempty"`
	CreatedAt            int64           `json:"created_at"`
	UpdatedAt            int64           `json:"updated_at"`
	// ContactInboxes is populated only when caller passes them (e.g. when the
	// query param `with_contact_inboxes=true` is set on GET /contacts/:id).
	ContactInboxes []ContactInboxResp `json:"contact_inboxes,omitempty"`
}

// ContactInboxResp wraps a contact_inbox row with its inbox embedded.
// Matches Chatwoot's nested shape.
type ContactInboxResp struct {
	SourceID string        `json:"source_id"`
	Inbox    InboxSlimResp `json:"inbox"`
}

type ContactListResp struct {
	Meta    MetaResp      `json:"meta"`
	Payload []ContactResp `json:"payload"`
}

// ContactToResp builds the Chatwoot-shape contact response. AdditionalAttrs
// is opaque JSON in the database (JSONB) — passed through verbatim. Thumbnail
// mirrors avatar_url so the frontend has a single attribute to render.
func ContactToResp(c *model.Contact) ContactResp {
	resp := ContactResp{
		ID:                 c.ID,
		AccountID:          c.AccountID,
		Name:               c.Name,
		Email:              c.Email,
		PhoneNumber:        c.PhoneNumber,
		Identifier:         c.Identifier,
		AvailabilityStatus: "offline",
		Blocked:            c.Blocked,
		AvatarURL:          c.AvatarURL,
		CreatedAt:          c.CreatedAt.Unix(),
		UpdatedAt:          c.UpdatedAt.Unix(),
	}
	if c.AvatarURL != nil {
		resp.Thumbnail = *c.AvatarURL
	}
	if c.LastActivityAt != nil {
		ts := c.LastActivityAt.Unix()
		resp.LastActivityAt = &ts
	}
	if c.AdditionalAttrs != nil && *c.AdditionalAttrs != "" {
		resp.AdditionalAttributes = json.RawMessage(*c.AdditionalAttrs)
	}
	return resp
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
