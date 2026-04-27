package realtime

// Canonical server-emitted event names, in `resource.action` form. These are
// the authoritative strings broadcast on the realtime hub; clients MUST
// subscribe using these exact values. The legacy names `message.new` and
// `conversation.new` are removed — no emitter ever used them and no client
// besides the official frontend consumes this transport.
const (
	EventMessageCreated      = "message.created"
	EventMessageUpdated      = "message.updated"
	EventMessageDeleted      = "message.deleted"
	EventConversationCreated = "conversation.created"
	EventConversationUpdated = "conversation.updated"
	EventConversationDeleted = "conversation.deleted"
	EventContactUpdated      = "contact.updated"
	EventInboxStatus         = "inbox.status"
)
