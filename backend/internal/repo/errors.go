package repo

import (
	"errors"

	"github.com/jackc/pgx/v5"
)

func IsErrNotFound(err error) bool {
	return errors.Is(err, pgx.ErrNoRows) ||
		errors.Is(err, ErrUserNotFound) ||
		errors.Is(err, ErrAccountNotFound) ||
		errors.Is(err, ErrRefreshTokenNotFound) ||
		errors.Is(err, ErrInboxNotFound) ||
		errors.Is(err, ErrChannelApiNotFound) ||
		errors.Is(err, ErrContactNotFound) ||
		errors.Is(err, ErrConversationNotFound) ||
		errors.Is(err, ErrMessageNotFound) ||
		errors.Is(err, ErrAttachmentNotFound) ||
		errors.Is(err, ErrContactInboxNotFound)
}
