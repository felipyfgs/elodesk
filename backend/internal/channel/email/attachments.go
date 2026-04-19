package email

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"backend/internal/logger"
	"backend/internal/media"
	"backend/internal/model"
	"backend/internal/repo"
)

const maxAttachmentSize = media.MaxUploadSize

// AttachmentHandler streams email attachments to MinIO and creates DB rows.
type AttachmentHandler struct {
	upload         *media.UploadService
	attachmentRepo *repo.AttachmentRepo
}

func NewAttachmentHandler(upload *media.UploadService, attachmentRepo *repo.AttachmentRepo) *AttachmentHandler {
	return &AttachmentHandler{upload: upload, attachmentRepo: attachmentRepo}
}

// ProcessAttachments iterates env.Attachments, uploads each to MinIO, and
// inserts an attachment row. Attachments > 256 MB are logged and skipped —
// the message itself is not aborted.
func (h *AttachmentHandler) ProcessAttachments(ctx context.Context, msg *model.Message, attachments []Attachment) error {
	for _, a := range attachments {
		if int64(len(a.Data)) > maxAttachmentSize {
			logger.Warn().
				Str("component", "email-attachments").
				Int64("messageID", msg.ID).
				Str("filename", a.Filename).
				Int("size", len(a.Data)).
				Msg("attachment exceeds 256 MB limit, skipping")
			continue
		}
		if err := h.uploadOne(ctx, msg, a); err != nil {
			logger.Warn().Str("component", "email-attachments").Err(err).Str("filename", a.Filename).Int64("messageID", msg.ID).Msg("attachment upload failed")
		}
	}
	return nil
}

func (h *AttachmentHandler) uploadOne(ctx context.Context, msg *model.Message, a Attachment) error {
	ext := strings.TrimPrefix(filepath.Ext(a.Filename), ".")
	if ext == "" {
		ext = extensionFromMIME(a.ContentType)
	}

	objectPath := fmt.Sprintf("%d/%d/%d/%s", msg.AccountID, msg.InboxID, msg.ID, a.Filename)
	if err := h.upload.Upload(ctx, bytes.NewReader(a.Data), objectPath, int64(len(a.Data)), a.ContentType); err != nil {
		return fmt.Errorf("upload attachment: %w", err)
	}

	fileType := fileTypeFromMIME(a.ContentType)
	fileKey := objectPath
	metaJSON := buildAttachmentMeta(a)

	row := &model.Attachment{
		MessageID: msg.ID,
		AccountID: msg.AccountID,
		FileType:  fileType,
		FileKey:   &fileKey,
		Extension: &ext,
		Meta:      &metaJSON,
	}
	return h.attachmentRepo.Create(ctx, row)
}

func fileTypeFromMIME(ct string) model.AttachmentFileType {
	ct = strings.ToLower(ct)
	switch {
	case strings.HasPrefix(ct, "image/"):
		return model.FileTypeImage
	case strings.HasPrefix(ct, "audio/"):
		return model.FileTypeAudio
	case strings.HasPrefix(ct, "video/"):
		return model.FileTypeVideo
	default:
		return model.FileTypeFile
	}
}

func extensionFromMIME(ct string) string {
	parts := strings.Split(ct, "/")
	if len(parts) == 2 {
		return parts[1]
	}
	return "bin"
}

func buildAttachmentMeta(a Attachment) string {
	inline := "false"
	if a.Inline {
		inline = "true"
	}
	return fmt.Sprintf(`{"inline":%s}`, inline)
}
