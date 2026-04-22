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
		errors.Is(err, ErrChannelAPINotFound) ||
		errors.Is(err, ErrContactNotFound) ||
		errors.Is(err, ErrConversationNotFound) ||
		errors.Is(err, ErrMessageNotFound) ||
		errors.Is(err, ErrAttachmentNotFound) ||
		errors.Is(err, ErrContactInboxNotFound) ||
		errors.Is(err, ErrLabelNotFound) ||
		errors.Is(err, ErrTeamNotFound) ||
		errors.Is(err, ErrCannedResponseNotFound) ||
		errors.Is(err, ErrNoteNotFound) ||
		errors.Is(err, ErrCustomAttributeDefinitionNotFound) ||
		errors.Is(err, ErrCustomFilterNotFound) ||
		errors.Is(err, ErrChannelTelegramNotFound) ||
		errors.Is(err, ErrChannelWhatsAppNotFound) ||
		errors.Is(err, ErrChannelEmailNotFound) ||
		errors.Is(err, ErrChannelInstagramNotFound) ||
		errors.Is(err, ErrChannelFacebookNotFound) ||
		errors.Is(err, ErrChannelSMSNotFound) ||
		errors.Is(err, ErrChannelWebWidgetNotFound) ||
		errors.Is(err, ErrMacroNotFound) ||
		errors.Is(err, ErrSLANotFound) ||
		errors.Is(err, ErrWebhookNotFound) ||
		errors.Is(err, ErrInboxBusinessHoursNotFound) ||
		errors.Is(err, ErrChannelLineNotFound) ||
		errors.Is(err, ErrChannelTiktokNotFound) ||
		errors.Is(err, ErrChannelTwilioNotFound) ||
		errors.Is(err, ErrChannelTwitterNotFound) ||
		errors.Is(err, ErrUserAccessTokenNotFound)
}
