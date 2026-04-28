package service

import (
	"backend/internal/model"
)

// IsAttachmentCompatibleWithChannel checks whether a given file type is
// supported by a specific channel type. Used by the forward service to
// pre-validate targets before creating messages, and by the frontend to
// disable incompatible inboxes in the picker.
func IsAttachmentCompatibleWithChannel(channelType string, fileType model.AttachmentFileType) bool {
	switch channelType {
	case "Channel::Whatsapp", "Channel::Telegram", "Channel::Email",
		"Channel::FacebookPage", "Channel::WebWidget", "Channel::Api":
		// Full media support
		return true

	case "Channel::Sms":
		// SMS: text only
		return false

	case "Channel::Twilio":
		// Twilio SMS: text only; Twilio WhatsApp routes via Channel::Whatsapp
		return false

	case "Channel::Instagram":
		// Instagram: image + video (no audio, no generic files)
		return fileType == model.FileTypeImage || fileType == model.FileTypeVideo

	case "Channel::Tiktok":
		// TikTok: image + video (no audio, no generic files)
		return fileType == model.FileTypeImage || fileType == model.FileTypeVideo

	case "Channel::Line":
		// Line: image + audio + video (no generic files)
		return fileType == model.FileTypeImage || fileType == model.FileTypeAudio || fileType == model.FileTypeVideo

	case "Channel::Twitter":
		// Twitter/DM: image + video (no audio, no generic files)
		return fileType == model.FileTypeImage || fileType == model.FileTypeVideo

	default:
		// Unknown channel — conservative: allow text-only (no attachments)
		return false
	}
}

// attachmentFileTypeNames maps enum values to human-readable names for error
// messages.
var attachmentFileTypeNames = map[model.AttachmentFileType]string{
	model.FileTypeImage:    "image",
	model.FileTypeAudio:    "audio",
	model.FileTypeVideo:    "video",
	model.FileTypeFile:     "file",
	model.FileTypeLocation: "location",
	model.FileTypeFallback: "file",
}

// AttachmentFileTypeName returns a human-readable name for a file type.
func AttachmentFileTypeName(ft model.AttachmentFileType) string {
	if name, ok := attachmentFileTypeNames[ft]; ok {
		return name
	}
	return "unknown"
}
