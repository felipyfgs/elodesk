package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"

	"backend/internal/audit"
	"backend/internal/dto"
	"backend/internal/logger"
	"backend/internal/media"
	"backend/internal/model"
	"backend/internal/phone"
	"backend/internal/repo"
)

var (
	ErrSameContactMerge    = repo.ErrSameContactMerge
	ErrInvalidAvatarObject = errors.New("invalid avatar object key")
)

type ContactService struct {
	contactRepo      *repo.ContactRepo
	contactInboxRepo *repo.ContactInboxRepo
	conversationRepo *repo.ConversationRepo
	auditLogRepo     *repo.AuditLogRepo
	auditLogger      *audit.Logger
	minio            *media.MinioClient
}

func NewContactService(
	contactRepo *repo.ContactRepo,
	contactInboxRepo *repo.ContactInboxRepo,
	conversationRepo *repo.ConversationRepo,
) *ContactService {
	return &ContactService{
		contactRepo:      contactRepo,
		contactInboxRepo: contactInboxRepo,
		conversationRepo: conversationRepo,
	}
}

// WithAudit wires the audit logger + audit log repo. Separated from the
// constructor so existing call sites keep compiling until router.go wires it.
func (s *ContactService) WithAudit(logger *audit.Logger, auditLogRepo *repo.AuditLogRepo) *ContactService {
	s.auditLogger = logger
	s.auditLogRepo = auditLogRepo
	return s
}

// WithMinio wires the MinIO client used for avatar object deletion.
func (s *ContactService) WithMinio(m *media.MinioClient) *ContactService {
	s.minio = m
	return s
}

func (s *ContactService) Create(ctx context.Context, accountID int64, contact *model.Contact) (*model.Contact, error) {
	contact.AccountID = accountID

	if contact.PhoneE164 == nil && contact.PhoneNumber != nil && *contact.PhoneNumber != "" {
		if e164, valid := phone.NormalizeE164(*contact.PhoneNumber); valid {
			v := e164
			contact.PhoneE164 = &v
		}
	}

	if contact.Identifier != nil && *contact.Identifier != "" {
		existing, err := s.contactRepo.FindByIdentifier(ctx, *contact.Identifier, fmt.Sprintf("%d", accountID))
		if err == nil {
			if contact.Name != "" {
				existing.Name = contact.Name
			}
			if contact.Email != nil {
				existing.Email = contact.Email
			}
			if contact.PhoneNumber != nil {
				existing.PhoneNumber = contact.PhoneNumber
			}
			if contact.AdditionalAttrs != nil {
				existing.AdditionalAttrs = contact.AdditionalAttrs
			}
			return s.contactRepo.Update(ctx, existing.ID, accountID, &existing.Name, existing.Email, existing.PhoneNumber)
		}
		if !repo.IsErrNotFound(err) {
			return nil, fmt.Errorf("failed to check existing contact: %w", err)
		}
	}

	if err := s.contactRepo.Create(ctx, contact); err != nil {
		return nil, err
	}
	return contact, nil
}

func (s *ContactService) FindByID(ctx context.Context, id, accountID int64) (*model.Contact, error) {
	return s.contactRepo.FindByID(ctx, id, accountID)
}

func (s *ContactService) Search(ctx context.Context, accountID int64, query string, page, perPage int) ([]model.Contact, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 25
	}
	filter := repo.ContactFilter{
		AccountID: accountID,
		Query:     query,
		Page:      page,
		PerPage:   perPage,
	}
	return s.contactRepo.Search(ctx, filter)
}

func (s *ContactService) SearchWithLabels(ctx context.Context, accountID int64, query string, labels []string, page, perPage int) ([]model.Contact, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 25
	}
	filter := repo.ContactFilter{
		AccountID: accountID,
		Query:     query,
		Labels:    labels,
		Page:      page,
		PerPage:   perPage,
	}
	return s.contactRepo.Search(ctx, filter)
}

func (s *ContactService) Update(ctx context.Context, id, accountID int64, name, email, phone *string) (*model.Contact, error) {
	updated, err := s.contactRepo.Update(ctx, id, accountID, name, email, phone)
	if err != nil {
		return nil, err
	}
	s.emitAudit(ctx, accountID, nil, "contact.updated", &updated.ID, map[string]any{
		"name":  updated.Name,
		"email": updated.Email,
	})
	return updated, nil
}

func (s *ContactService) UpdateDetails(ctx context.Context, id, accountID int64, name, email, phone *string, additionalAttrs map[string]any) (*model.Contact, error) {
	updated, err := s.contactRepo.Update(ctx, id, accountID, name, email, phone)
	if err != nil {
		return nil, err
	}
	if additionalAttrs != nil {
		merged := map[string]any{}
		if updated.AdditionalAttrs != nil {
			_ = json.Unmarshal([]byte(*updated.AdditionalAttrs), &merged)
		}
		for k, v := range additionalAttrs {
			merged[k] = v
		}
		encoded, err := json.Marshal(merged)
		if err != nil {
			return nil, fmt.Errorf("marshal additional_attributes: %w", err)
		}
		if _, err := s.contactRepo.UpdateAdditionalAttrs(ctx, id, accountID, string(encoded)); err != nil {
			return nil, err
		}
		updated, err = s.contactRepo.FindByID(ctx, id, accountID)
		if err != nil {
			return nil, err
		}
	}
	s.emitAudit(ctx, accountID, nil, "contact.updated", &updated.ID, map[string]any{
		"name":  updated.Name,
		"email": updated.Email,
	})
	return updated, nil
}

func (s *ContactService) Delete(ctx context.Context, accountID, id int64, userID *int64) error {
	contact, err := s.contactRepo.FindByID(ctx, id, accountID)
	if err != nil {
		return err
	}
	if err := s.contactRepo.Delete(ctx, id, accountID); err != nil {
		return err
	}
	s.emitAudit(ctx, accountID, userID, "contact.deleted", &id, map[string]any{
		"name":  contact.Name,
		"email": contact.Email,
	})
	return nil
}

func (s *ContactService) Merge(ctx context.Context, accountID, childID, primaryID int64, userID *int64) (*model.Contact, error) {
	child, err := s.contactRepo.FindByID(ctx, childID, accountID)
	if err != nil {
		return nil, err
	}
	primary, err := s.contactRepo.Merge(ctx, childID, primaryID, accountID)
	if err != nil {
		return nil, err
	}
	s.emitAudit(ctx, accountID, userID, "contact.merged", &primary.ID, map[string]any{
		"child_id":    childID,
		"child_name":  child.Name,
		"child_email": child.Email,
	})
	return primary, nil
}

func (s *ContactService) SetBlocked(ctx context.Context, accountID, id int64, userID *int64, blocked bool) error {
	if err := s.contactRepo.UpdateBlocked(ctx, id, accountID, blocked); err != nil {
		return err
	}
	action := "contact.blocked"
	if !blocked {
		action = "contact.unblocked"
	}
	s.emitAudit(ctx, accountID, userID, action, &id, map[string]any{"blocked": blocked})
	return nil
}

func (s *ContactService) SetAvatar(ctx context.Context, accountID, id int64, userID *int64, objectKey string) (*model.Contact, error) {
	expectedPrefix := fmt.Sprintf("%d/contacts/%d/", accountID, id)
	if !strings.HasPrefix(objectKey, expectedPrefix) {
		return nil, ErrInvalidAvatarObject
	}
	if err := s.contactRepo.UpdateAvatarURL(ctx, id, accountID, &objectKey); err != nil {
		return nil, err
	}
	updated, err := s.contactRepo.FindByID(ctx, id, accountID)
	if err != nil {
		return nil, err
	}
	s.emitAudit(ctx, accountID, userID, "contact.avatar_updated", &id, map[string]any{"object_key": objectKey})
	return updated, nil
}

func (s *ContactService) DeleteAvatar(ctx context.Context, accountID, id int64, userID *int64) error {
	current, err := s.contactRepo.FindByID(ctx, id, accountID)
	if err != nil {
		return err
	}
	if err := s.contactRepo.UpdateAvatarURL(ctx, id, accountID, nil); err != nil {
		return err
	}
	if current.AvatarURL != nil && s.minio != nil {
		if removeErr := s.minio.Client().RemoveObject(ctx, s.minio.Bucket(), *current.AvatarURL, minio.RemoveObjectOptions{}); removeErr != nil {
			logger.Warn().Str("component", "contacts").Err(removeErr).Int64("contact_id", id).Msg("failed to remove avatar object (ignored)")
		}
	}
	s.emitAudit(ctx, accountID, userID, "contact.avatar_deleted", &id, map[string]any{})
	return nil
}

func (s *ContactService) ListEvents(ctx context.Context, accountID, contactID int64, page, pageSize int) ([]dto.AuditEventResp, int, error) {
	if _, err := s.contactRepo.FindByID(ctx, contactID, accountID); err != nil {
		return nil, 0, err
	}
	if s.auditLogRepo == nil {
		return []dto.AuditEventResp{}, 0, nil
	}
	rows, total, err := s.auditLogRepo.ListByEntity(ctx, accountID, "contact", contactID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	out := make([]dto.AuditEventResp, 0, len(rows))
	for _, r := range rows {
		ev := dto.AuditEventResp{
			ID:        r.ID,
			Action:    r.Action,
			CreatedAt: r.CreatedAt,
		}
		if r.Metadata != nil {
			ev.Metadata = []byte(*r.Metadata)
		}
		if r.UserID != nil {
			name := ""
			if r.UserName != nil {
				name = *r.UserName
			}
			ev.User = &dto.AuditEventUserResp{ID: *r.UserID, Name: name}
		}
		out = append(out, ev)
	}
	return out, total, nil
}

func (s *ContactService) FindConversations(ctx context.Context, contactID, accountID int64) ([]model.Conversation, error) {
	contactIDCopy := contactID
	filter := repo.ConversationFilter{
		AccountID: accountID,
		ContactID: &contactIDCopy,
		Page:      1,
		PerPage:   1000,
	}
	convos, _, err := s.conversationRepo.ListByAccount(ctx, filter)
	return convos, err
}

func (s *ContactService) FindBySourceID(ctx context.Context, sourceID string, inboxID, accountID int64) (*model.Contact, error) {
	ci, err := s.contactInboxRepo.FindBySourceID(ctx, sourceID, inboxID)
	if err != nil {
		return nil, err
	}
	return s.contactRepo.FindByID(ctx, ci.ContactID, accountID)
}

func (s *ContactService) EnsureContactInbox(ctx context.Context, contactID, inboxID int64, sourceID string) error {
	existing, err := s.contactInboxRepo.FindByContactAndInbox(ctx, contactID, inboxID)
	if err != nil {
		return err
	}
	if existing != nil {
		return nil
	}
	ci := &model.ContactInbox{
		ContactID: contactID,
		InboxID:   inboxID,
		SourceID:  sourceID,
	}
	return s.contactInboxRepo.Create(ctx, ci)
}

func (s *ContactService) ImportBatch(ctx context.Context, accountID int64, batch []repo.ImportContact) (repo.ImportResult, error) {
	return s.contactRepo.ImportBatch(ctx, accountID, batch)
}

func (s *ContactService) emitAudit(ctx context.Context, accountID int64, userID *int64, action string, entityID *int64, metadata any) {
	if s.auditLogger == nil {
		return
	}
	s.auditLogger.Log(ctx, accountID, userID, action, "contact", entityID, metadata, "", "")
}

type ContactIdentifyParams struct {
	Identifier         *string
	Email              *string
	PhoneNumber        *string
	CustomAttributes   map[string]any
	AdditionalAttributes map[string]any
	AvatarURL          *string
}

func (s *ContactService) Identify(ctx context.Context, accountID int64, target *model.Contact, params ContactIdentifyParams) (*model.Contact, error) {
	result := target

	type matchKey struct {
		field   string
		value   string
		contact *model.Contact
	}

	var matches []matchKey

	if params.Identifier != nil && *params.Identifier != "" {
		existing, err := s.contactRepo.FindByIdentifier(ctx, *params.Identifier, fmt.Sprintf("%d", accountID))
		if err == nil && existing.ID != target.ID {
			matches = append(matches, matchKey{field: "identifier", value: *params.Identifier, contact: existing})
		}
	}
	if params.Email != nil && *params.Email != "" {
		existing, err := s.contactRepo.FindByEmail(ctx, *params.Email, accountID)
		if err == nil && existing.ID != target.ID {
			matches = append(matches, matchKey{field: "email", value: *params.Email, contact: existing})
		}
	}
	if params.PhoneNumber != nil && *params.PhoneNumber != "" {
		existing, err := s.contactRepo.FindByPhone(ctx, *params.PhoneNumber, accountID)
		if err == nil && existing.ID != target.ID {
			matches = append(matches, matchKey{field: "phone_number", value: *params.PhoneNumber, contact: existing})
		}
	}

	for _, m := range matches {
		if m.contact.Identifier != nil && *m.contact.Identifier != "" {
			if m.field != "identifier" {
				continue
			}
		}
		if m.field == "email" {
			if m.contact.Identifier != nil && *m.contact.Identifier != "" && target.Identifier != nil && *target.Identifier != "" {
				if *m.contact.Identifier != *target.Identifier {
					continue
				}
			}
		}
		if m.field == "phone_number" {
			if m.contact.Email != nil && target.Email != nil && *m.contact.Email != *target.Email {
				continue
			}
		}

		merged, err := s.Merge(ctx, accountID, target.ID, m.contact.ID, nil)
		if err != nil {
			return nil, fmt.Errorf("merge contact: %w", err)
		}
		result = merged
		target = merged
	}

	updates := make(map[string]*string)
	if params.Email != nil {
		updates["email"] = params.Email
	}
	if params.PhoneNumber != nil {
		updates["phone_number"] = params.PhoneNumber
	}
	if params.AvatarURL != nil {
		updates["avatar_url"] = params.AvatarURL
		if result.AvatarURL == nil || *result.AvatarURL != *params.AvatarURL {
			// URL changed — also recompute hash. Provider (Wzap) cache-buster URLs
			// already imply content change; we hash the URL itself. v2: HEAD/GET
			// the URL to hash the actual bytes.
			hash := avatarHashFromURL(*params.AvatarURL)
			if err := s.contactRepo.UpdateAvatar(ctx, result.ID, accountID, params.AvatarURL, &hash); err != nil {
				return nil, err
			}
			result.AvatarURL = params.AvatarURL
			result.AvatarHash = &hash
		}
	}

	if len(updates) > 0 {
		updated, err := s.contactRepo.Update(ctx, result.ID, accountID, nil, updates["email"], updates["phone_number"])
		if err != nil {
			return nil, err
		}
		result = updated
	}

	if params.CustomAttributes != nil {
		if result.AdditionalAttrs != nil {
			existing := map[string]any{}
			_ = json.Unmarshal([]byte(*result.AdditionalAttrs), &existing)
			for k, v := range params.CustomAttributes {
				existing[k] = v
			}
			encoded, _ := json.Marshal(existing)
			attrsStr := string(encoded)
			if _, err := s.contactRepo.UpdateAdditionalAttrs(ctx, result.ID, accountID, attrsStr); err == nil {
				result, _ = s.contactRepo.FindByID(ctx, result.ID, accountID)
			}
		}
	}

	return result, nil
}

type ContactCreateAttrs struct {
	Name                 string
	Email                *string
	PhoneNumber          *string
	Identifier           *string
	AvatarURL            *string
	// AdditionalAttrs is the raw JSON object as a string (matches the persisted
	// shape on contacts.additional_attributes). When the caller passes a
	// non-empty value, it is set on the contact at creation time and merged on
	// updates of existing contacts.
	AdditionalAttrs *string
}

func (s *ContactService) CreateOrReuseContactInbox(ctx context.Context, inbox *model.Inbox, attrs ContactCreateAttrs, sourceID string, hmacVerified bool) (*model.ContactInbox, error) {
	if sourceID != "" {
		existing, err := s.contactInboxRepo.FindBySourceID(ctx, sourceID, inbox.ID)
		if err == nil {
			s.applyAvatarUpdate(ctx, existing.ContactID, inbox.AccountID, attrs.AvatarURL)
			return existing, nil
		}
	}

	var contact *model.Contact

	if attrs.Identifier != nil && *attrs.Identifier != "" {
		existing, err := s.contactRepo.FindByIdentifier(ctx, *attrs.Identifier, fmt.Sprintf("%d", inbox.AccountID))
		if err == nil {
			contact = existing
		}
	}
	if contact == nil && attrs.Email != nil && *attrs.Email != "" {
		existing, err := s.contactRepo.FindByEmail(ctx, *attrs.Email, inbox.AccountID)
		if err == nil {
			contact = existing
		}
	}
	if contact == nil && attrs.PhoneNumber != nil && *attrs.PhoneNumber != "" {
		existing, err := s.contactRepo.FindByPhone(ctx, *attrs.PhoneNumber, inbox.AccountID)
		if err == nil {
			contact = existing
		}
	}

	if contact == nil {
		contact = &model.Contact{
			AccountID:       inbox.AccountID,
			Name:            attrs.Name,
			Email:           attrs.Email,
			PhoneNumber:     attrs.PhoneNumber,
			Identifier:      attrs.Identifier,
			AdditionalAttrs: attrs.AdditionalAttrs,
		}
		if err := s.contactRepo.Create(ctx, contact); err != nil {
			return nil, fmt.Errorf("create contact: %w", err)
		}
	} else if attrs.AdditionalAttrs != nil && *attrs.AdditionalAttrs != "" {
		// Merge incoming JSONB into the existing additional_attributes so callers
		// can attach metadata (e.g. is_group, test_run_id) without erasing prior keys.
		merged := mergeAdditionalAttrs(contact.AdditionalAttrs, *attrs.AdditionalAttrs)
		if merged != "" && (contact.AdditionalAttrs == nil || *contact.AdditionalAttrs != merged) {
			if _, err := s.contactRepo.UpdateAdditionalAttrs(ctx, contact.ID, inbox.AccountID, merged); err == nil {
				contact.AdditionalAttrs = &merged
			}
		}
	}

	s.applyAvatarUpdate(ctx, contact.ID, inbox.AccountID, attrs.AvatarURL)

	// Dedupe before minting a fresh source_id: a contact can already have a ci
	// for this inbox (e.g. matched via email/phone above). Without this guard
	// we'd accumulate one contact_inbox per visit, which is exactly the
	// regression seen on production for WhatsApp + widget flows.
	if existing, err := s.contactInboxRepo.FindByContactAndInbox(ctx, contact.ID, inbox.ID); err != nil {
		return nil, fmt.Errorf("find contact inbox: %w", err)
	} else if existing != nil {
		return existing, nil
	}

	if sourceID == "" {
		sourceID = uuid.NewString()
	}

	ci := &model.ContactInbox{
		ContactID:    contact.ID,
		InboxID:      inbox.ID,
		SourceID:     sourceID,
		HmacVerified: hmacVerified,
	}
	if err := s.contactInboxRepo.Create(ctx, ci); err != nil {
		return nil, fmt.Errorf("create contact inbox: %w", err)
	}

	return ci, nil
}

// applyAvatarUpdate is a best-effort write of avatar_url + avatar_hash.
// Channels (e.g. Wzap) post the upstream avatar on every contact upsert; we
// only write when the URL actually changed. avatar_hash is a SHA-256 of the
// avatar_url string — providers ship cache-buster URLs (WhatsApp embeds a
// timestamp), so URL inequality already implies content change. Hashing the
// fetched bytes is deferred to v2 (would add a HEAD/GET round-trip per upsert).
// Failures are logged and swallowed so a transient DB hiccup does not break
// message ingestion.
func (s *ContactService) applyAvatarUpdate(ctx context.Context, contactID, accountID int64, avatarURL *string) {
	if avatarURL == nil || *avatarURL == "" {
		return
	}
	current, err := s.contactRepo.FindByID(ctx, contactID, accountID)
	if err != nil {
		logger.Warn().Str("component", "contacts").Err(err).Int64("contact_id", contactID).Msg("avatar refresh: failed to load contact")
		return
	}
	if current.AvatarURL != nil && *current.AvatarURL == *avatarURL {
		return
	}
	hash := avatarHashFromURL(*avatarURL)
	if err := s.contactRepo.UpdateAvatar(ctx, contactID, accountID, avatarURL, &hash); err != nil {
		logger.Warn().Str("component", "contacts").Err(err).Int64("contact_id", contactID).Msg("avatar refresh: failed to update avatar")
	}
}

// avatarHashFromURL computes SHA-256(avatar_url) as a hex string. Used as a
// stable, cheap content-cache invalidation key — see applyAvatarUpdate.
func avatarHashFromURL(url string) string {
	sum := sha256.Sum256([]byte(url))
	return hex.EncodeToString(sum[:])
}

// mergeAdditionalAttrs merges a new JSON object into an existing one (both
// stored as raw JSON strings on contacts.additional_attributes). Keys in the
// incoming object overwrite existing keys; everything else is preserved.
// Returns "" if the incoming JSON is invalid (caller skips the update).
func mergeAdditionalAttrs(existing *string, incoming string) string {
	out := map[string]any{}
	if existing != nil && *existing != "" {
		_ = json.Unmarshal([]byte(*existing), &out)
	}
	add := map[string]any{}
	if err := json.Unmarshal([]byte(incoming), &add); err != nil {
		return ""
	}
	for k, v := range add {
		out[k] = v
	}
	encoded, err := json.Marshal(out)
	if err != nil {
		return ""
	}
	return string(encoded)
}
