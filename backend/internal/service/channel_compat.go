package service

import (
	"backend/internal/model"
)

func IsAttachmentCompatibleWithChannel(channelType string, fileType model.AttachmentFileType) bool {
	switch channelType {
	case "Channel::Whatsapp", "Channel::Telegram", "Channel::Email",
		"Channel::FacebookPage", "Channel::WebWidget", "Channel::Api":
		return true

	case "Channel::Sms":
		return false

	case "Channel::Twilio":
		return false

	case "Channel::Instagram":
		return fileType == model.FileTypeImage || fileType == model.FileTypeVideo

	case "Channel::Tiktok":
		return fileType == model.FileTypeImage || fileType == model.FileTypeVideo

	case "Channel::Line":
		return fileType == model.FileTypeImage || fileType == model.FileTypeAudio || fileType == model.FileTypeVideo

	case "Channel::Twitter":
		return fileType == model.FileTypeImage || fileType == model.FileTypeVideo

	default:
		return false
	}
}

var attachmentFileTypeNames = map[model.AttachmentFileType]string{
	model.FileTypeImage:    "image",
	model.FileTypeAudio:    "audio",
	model.FileTypeVideo:    "video",
	model.FileTypeFile:     "file",
	model.FileTypeLocation: "location",
	model.FileTypeFallback: "file",
}

func AttachmentFileTypeName(ft model.AttachmentFileType) string {
	if name, ok := attachmentFileTypeNames[ft]; ok {
		return name
	}
	return "unknown"
}
