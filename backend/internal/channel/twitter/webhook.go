package twitter

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"backend/internal/channel"
	"backend/internal/logger"
	"backend/internal/model"
	"backend/internal/repo"
)

const (
	dedupKeyPrefix = "elodesk:twitter:"
	dmTypeMessage  = "message_create"
)

// CRCChallenge computes the response token Twitter expects in reply to a
// GET /webhooks/twitter/:profile_id?crc_token=... probe.
// https://developer.twitter.com/en/docs/twitter-api/premium/account-activity-api/guides/securing-webhooks
func CRCChallenge(consumerSecret, crcToken string) string {
	if consumerSecret == "" || crcToken == "" {
		return ""
	}
	mac := hmac.New(sha256.New, []byte(consumerSecret))
	mac.Write([]byte(crcToken))
	return "sha256=" + base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// VerifySignature checks the x-twitter-webhooks-signature header against the
// raw POST body. Header value is "sha256=<base64hmac>".
func VerifySignature(consumerSecret string, body []byte, signature string) bool {
	if consumerSecret == "" || signature == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(consumerSecret))
	mac.Write(body)
	expected := "sha256=" + base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

// ProcessWebhook ingests a Twitter Account Activity payload. Only
// direct_message_events are processed; tweet_create_events and other event
// types are silently dropped per spec.
func ProcessWebhook(
	ctx context.Context,
	body []byte,
	ch *model.ChannelTwitter,
	inbox *model.Inbox,
	dedup *channel.DedupLock,
	contactRepo *repo.ContactRepo,
	contactInboxRepo *repo.ContactInboxRepo,
	conversationRepo *repo.ConversationRepo,
	messageRepo *repo.MessageRepo,
) error {
	var payload WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return fmt.Errorf("twitter webhook: unmarshal: %w", err)
	}

	if !hasSupportedEvent(body) {
		return nil
	}

	for i := range payload.DirectMessageEvents {
		evt := &payload.DirectMessageEvents[i]
		if err := processDM(ctx, evt, &payload, ch, inbox, dedup,
			contactRepo, contactInboxRepo, conversationRepo, messageRepo); err != nil {
			logger.Warn().Str("component", "channel.twitter").
				Err(err).Str("eventId", evt.ID).Msg("twitter dm processing error")
			continue
		}
	}
	return nil
}

// hasSupportedEvent peeks at the JSON envelope for direct_message_events
// using a streaming decoder to avoid false-positives from tweet text that
// happens to contain the key string.
func hasSupportedEvent(body []byte) bool {
	dec := json.NewDecoder(strings.NewReader(string(body)))
	// We only need the top-level keys.
	if token, err := dec.Token(); err != nil || token != json.Delim('{') {
		return false
	}
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return false
		}
		key, ok := tok.(string)
		if !ok {
			continue
		}
		if key == supportedEventKey {
			return true
		}
		// Skip the value so we can read the next key.
		if err := skipValue(dec); err != nil {
			return false
		}
	}
	return false
}

// skipValue fast-forwards a json.Decoder past the current value (primitive,
// object, or array) so we can continue reading sibling keys.
func skipValue(dec *json.Decoder) error {
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	switch tok {
	case json.Delim('['), json.Delim('{'):
		for dec.More() {
			if err := skipValue(dec); err != nil {
				return err
			}
		}
		_, err = dec.Token() // consume closing ] or }
		return err
	}
	return nil
}

func processDM(
	ctx context.Context,
	evt *DirectMessageEvent,
	payload *WebhookPayload,
	ch *model.ChannelTwitter,
	inbox *model.Inbox,
	dedup *channel.DedupLock,
	contactRepo *repo.ContactRepo,
	contactInboxRepo *repo.ContactInboxRepo,
	conversationRepo *repo.ConversationRepo,
	messageRepo *repo.MessageRepo,
) error {
	if evt.Type != dmTypeMessage || evt.MessageCreate == nil {
		return nil
	}

	create := evt.MessageCreate
	// Skip echoes from our own profile
	if create.SenderID == ch.ProfileID {
		return nil
	}

	if dedup != nil {
		ok, err := dedup.Acquire(ctx, dedupKeyPrefix+evt.ID)
		if err != nil {
			logger.Warn().Str("component", "channel.twitter").Err(err).Msg("dedup acquire error")
		}
		if !ok {
			return nil
		}
	}

	displayName := create.SenderID
	screenName := ""
	if u, ok := payload.Users[create.SenderID]; ok {
		if u.Name != "" {
			displayName = u.Name
		}
		screenName = u.ScreenName
	}

	ci, conv, err := ensureContactAndConversation(ctx, create.SenderID, displayName, screenName,
		ch.AccountID, inbox.ID, contactRepo, contactInboxRepo, conversationRepo)
	if err != nil {
		return err
	}
	if conv == nil {
		return nil
	}

	sourceID := evt.ID
	content := create.MessageData.Text
	senderType := "Contact"
	contactID := ci.ContactID
	dbMsg := &model.Message{
		AccountID:      ch.AccountID,
		InboxID:        inbox.ID,
		ConversationID: conv.ID,
		MessageType:    model.MessageIncoming,
		ContentType:    model.ContentTypeText,
		Content:        &content,
		SourceID:       &sourceID,
		SenderType:     &senderType,
		SenderID:       &contactID,
	}
	if _, err := messageRepo.Create(ctx, dbMsg); err != nil {
		return fmt.Errorf("create message: %w", err)
	}
	return nil
}

func ensureContactAndConversation(
	ctx context.Context,
	senderID, displayName, screenName string,
	accountID, inboxID int64,
	contactRepo *repo.ContactRepo,
	contactInboxRepo *repo.ContactInboxRepo,
	conversationRepo *repo.ConversationRepo,
) (*model.ContactInbox, *model.Conversation, error) {
	ci, err := contactInboxRepo.FindBySourceID(ctx, senderID, inboxID)
	if err != nil {
		if !repo.IsErrNotFound(err) {
			return nil, nil, fmt.Errorf("find contact inbox: %w", err)
		}
		contact := &model.Contact{
			AccountID:  accountID,
			Name:       displayName,
			Identifier: &senderID,
		}
		if screenName != "" {
			extras := fmt.Sprintf(`{"twitter_screen_name":%q}`, screenName)
			contact.AdditionalAttrs = &extras
		}
		if err := contactRepo.Create(ctx, contact); err != nil {
			return nil, nil, fmt.Errorf("create contact: %w", err)
		}
		ci = &model.ContactInbox{
			ContactID: contact.ID,
			InboxID:   inboxID,
			SourceID:  senderID,
		}
		if err := contactInboxRepo.Create(ctx, ci); err != nil {
			return nil, nil, fmt.Errorf("create contact inbox: %w", err)
		}
	}

	if c, cErr := contactRepo.FindByID(ctx, ci.ContactID, accountID); cErr == nil && c.Blocked {
		logger.Warn().Str("component", "channel.twitter").
			Int64("contact_id", c.ID).Msg("contact_blocked_inbound_dropped")
		return ci, nil, nil
	}

	conv, err := conversationRepo.EnsureOpen(ctx, accountID, inboxID, ci.ContactID)
	if err != nil {
		return nil, nil, fmt.Errorf("ensure open conversation: %w", err)
	}
	return ci, conv, nil
}
