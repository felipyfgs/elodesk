package realtime

import "context"

type MembershipChecker interface {
	UserInAccount(ctx context.Context, userID, accountID int64) bool
	InboxAccount(ctx context.Context, inboxID int64) (accountID int64, ok bool)
	ConversationAccount(ctx context.Context, conversationID int64) (accountID int64, ok bool)
}
