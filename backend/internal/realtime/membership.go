package realtime

import "context"

// MembershipChecker decides whether the authenticated user may join a given
// room. Cross-tenant room IDs MUST return false without leaking which ones
// exist. Implementations should be cheap (indexed lookups in the tenant path).
type MembershipChecker interface {
	UserInAccount(ctx context.Context, userID, accountID int64) bool
	InboxAccount(ctx context.Context, inboxID int64) (accountID int64, ok bool)
	ConversationAccount(ctx context.Context, conversationID int64) (accountID int64, ok bool)
}
