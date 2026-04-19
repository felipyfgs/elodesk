package model

import (
	"time"
)

type User struct {
	ID           int64     `json:"id"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type Account struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Role int

const (
	RoleAgent Role = 0
	RoleAdmin Role = 1
	RoleOwner Role = 2
)

type AccountUser struct {
	ID        int64     `json:"id"`
	AccountID int64     `json:"accountId"`
	UserID    int64     `json:"userId"`
	Role      Role      `json:"role"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type RefreshToken struct {
	ID        int64      `json:"id"`
	UserID    int64      `json:"userId"`
	TokenHash string     `json:"-"`
	FamilyID  string     `json:"familyId"`
	RevokedAt *time.Time `json:"revokedAt,omitempty"`
	ExpiresAt time.Time  `json:"expiresAt"`
	CreatedAt time.Time  `json:"createdAt"`
}

type Inbox struct {
	ID          int64     `json:"id"`
	AccountID   int64     `json:"accountId"`
	ChannelID   int64     `json:"channelId"`
	Name        string    `json:"name"`
	ChannelType string    `json:"channelType"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type ChannelApi struct {
	ID                 int64     `json:"id"`
	AccountID          int64     `json:"accountId"`
	WebhookURL         string    `json:"webhookUrl,omitempty"`
	Identifier         string    `json:"identifier"`
	HmacToken          string    `json:"hmacToken,omitempty"`
	HmacMandatory      bool      `json:"hmacMandatory"`
	Secret             string    `json:"secret,omitempty"`
	ApiToken           string    `json:"apiToken,omitempty"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

type Contact struct {
	ID                 int64      `json:"id"`
	AccountID          int64      `json:"accountId"`
	Name               string     `json:"name"`
	Email              *string    `json:"email,omitempty"`
	PhoneNumber        *string    `json:"phoneNumber,omitempty"`
	Identifier         *string    `json:"identifier,omitempty"`
	AdditionalAttrs    *string    `json:"additionalAttributes,omitempty"`
	LastActivityAt     *time.Time `json:"lastActivityAt,omitempty"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
}

type ContactInbox struct {
	ID           int64     `json:"id"`
	ContactID    int64     `json:"contactId"`
	InboxID      int64     `json:"inboxId"`
	SourceID     string    `json:"sourceId"`
	HmacVerified bool      `json:"hmacVerified"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type ConversationStatus int

const (
	ConversationOpen     ConversationStatus = 0
	ConversationResolved ConversationStatus = 1
	ConversationPending  ConversationStatus = 2
	ConversationSnoozed  ConversationStatus = 3
)

type Conversation struct {
	ID               int64              `json:"id"`
	AccountID        int64              `json:"accountId"`
	InboxID          int64              `json:"inboxId"`
	Status           ConversationStatus `json:"status"`
	AssigneeID       *int64             `json:"assigneeId,omitempty"`
	ContactID        int64              `json:"contactId"`
	ContactInboxID   *int64             `json:"contactInboxId,omitempty"`
	DisplayID        int64              `json:"displayId"`
	UUID             string             `json:"uuid"`
	LastActivityAt   time.Time          `json:"lastActivityAt"`
	AdditionalAttrs  *string            `json:"additionalAttributes,omitempty"`
	CreatedAt        time.Time          `json:"createdAt"`
	UpdatedAt        time.Time          `json:"updatedAt"`
}

type MessageType int

const (
	MessageIncoming MessageType = 0
	MessageOutgoing MessageType = 1
	MessageActivity MessageType = 2
	MessageTemplate MessageType = 3
)

type MessageContentType int

const (
	ContentTypeText         MessageContentType = 0
	ContentTypeInputText    MessageContentType = 1
	ContentTypeInputEmail   MessageContentType = 3
	ContentTypeCards        MessageContentType = 5
	ContentTypeArticle      MessageContentType = 7
	ContentTypeIncomingEmail MessageContentType = 8
	ContentTypeSticker      MessageContentType = 11
)

type MessageStatus int

const (
	MessageSent     MessageStatus = 0
	MessageDelivered MessageStatus = 1
	MessageRead     MessageStatus = 2
	MessageFailed   MessageStatus = 3
)

type Message struct {
	ID               int64             `json:"id"`
	AccountID        int64             `json:"accountId"`
	InboxID          int64             `json:"inboxId"`
	ConversationID   int64             `json:"conversationId"`
	MessageType      MessageType       `json:"messageType"`
	ContentType      MessageContentType `json:"contentType"`
	Content          *string           `json:"content,omitempty"`
	SourceID         *string           `json:"sourceId,omitempty"`
	Private          bool              `json:"private"`
	Status           MessageStatus     `json:"status"`
	ContentAttrs     *string           `json:"contentAttributes,omitempty"`
	SenderType       *string           `json:"senderType,omitempty"`
	SenderID         *int64            `json:"senderId,omitempty"`
	ExternalSourceIDs *string          `json:"externalSourceIds,omitempty"`
	CreatedAt        time.Time         `json:"createdAt"`
	UpdatedAt        time.Time         `json:"updatedAt"`
	DeletedAt        *time.Time        `json:"deletedAt,omitempty"`
}

type AttachmentFileType int

const (
	FileTypeImage       AttachmentFileType = 0
	FileTypeAudio       AttachmentFileType = 1
	FileTypeVideo       AttachmentFileType = 2
	FileTypeFile        AttachmentFileType = 3
	FileTypeLocation    AttachmentFileType = 4
	FileTypeFallback    AttachmentFileType = 5
)

type Attachment struct {
	ID         int64             `json:"id"`
	MessageID  int64             `json:"messageId"`
	AccountID  int64             `json:"accountId"`
	FileType   AttachmentFileType `json:"fileType"`
	ExternalURL *string          `json:"externalUrl,omitempty"`
	FileKey    *string           `json:"fileKey,omitempty"`
	Extension  *string           `json:"extension,omitempty"`
	Meta       *string           `json:"meta,omitempty"`
	CreatedAt  time.Time         `json:"createdAt"`
	UpdatedAt  time.Time         `json:"updatedAt"`
}
