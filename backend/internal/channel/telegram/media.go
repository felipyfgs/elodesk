package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"

	"github.com/minio/minio-go/v7"

	"backend/internal/logger"
	"backend/internal/repo"
)

type MediaResolver struct {
	api            *APIClient
	minioClient    *minio.Client
	minioBucket    string
	chRepo         *repo.ChannelTelegramRepo
	attachmentRepo *repo.AttachmentRepo
	messageRepo    *repo.MessageRepo
	inboxRepo      *repo.InboxRepo
	cipher         Decryptor
}

type Decryptor interface {
	Decrypt(encoded string) (string, error)
}

func NewMediaResolver(
	api *APIClient,
	minioClient *minio.Client,
	minioBucket string,
	chRepo *repo.ChannelTelegramRepo,
	attachmentRepo *repo.AttachmentRepo,
	messageRepo *repo.MessageRepo,
	inboxRepo *repo.InboxRepo,
	cipher Decryptor,
) *MediaResolver {
	return &MediaResolver{
		api:            api,
		minioClient:    minioClient,
		minioBucket:    minioBucket,
		chRepo:         chRepo,
		attachmentRepo: attachmentRepo,
		messageRepo:    messageRepo,
		inboxRepo:      inboxRepo,
		cipher:         cipher,
	}
}

type fileAttrs struct {
	FileID   string `json:"file_id"`
	FileName string `json:"file_name,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
	FileSize int64  `json:"file_size,omitempty"`
}

func (r *MediaResolver) ResolveMedia(ctx context.Context, attachmentID int64, accountID int64) (string, error) {
	attachment, err := r.attachmentRepo.FindByID(ctx, attachmentID, accountID)
	if err != nil {
		return "", fmt.Errorf("telegram resolve media: find attachment: %w", err)
	}

	if attachment.FileKey != nil && *attachment.FileKey != "" {
		return *attachment.FileKey, nil
	}

	msg, err := r.messageRepo.FindByID(ctx, attachment.MessageID, accountID)
	if err != nil {
		return "", fmt.Errorf("telegram resolve media: find message: %w", err)
	}

	inbox, err := r.inboxRepo.FindByID(ctx, msg.InboxID, accountID)
	if err != nil {
		return "", fmt.Errorf("telegram resolve media: find inbox: %w", err)
	}

	if inbox.ChannelType != "Channel::Telegram" {
		return "", fmt.Errorf("telegram resolve media: inbox is not telegram channel")
	}

	ch, err := r.chRepo.FindByID(ctx, inbox.ChannelID, accountID)
	if err != nil {
		return "", fmt.Errorf("telegram resolve media: find channel: %w", err)
	}

	botToken, err := r.cipher.Decrypt(ch.BotTokenCiphertext)
	if err != nil {
		return "", fmt.Errorf("telegram resolve media: decrypt token: %w", err)
	}

	if msg.ContentAttrs == nil || *msg.ContentAttrs == "" {
		return "", fmt.Errorf("telegram resolve media: no content attributes")
	}

	var attrs fileAttrs
	if err := json.Unmarshal([]byte(*msg.ContentAttrs), &attrs); err != nil {
		return "", fmt.Errorf("telegram resolve media: parse content attrs: %w", err)
	}

	fileResult, err := r.api.GetFile(ctx, botToken, attrs.FileID)
	if err != nil {
		return "", fmt.Errorf("telegram resolve media: getFile: %w", err)
	}

	data, err := r.api.DownloadFile(ctx, botToken, fileResult.FilePath)
	if err != nil {
		return "", fmt.Errorf("telegram resolve media: download file: %w", err)
	}

	fileName := attrs.FileName
	if fileName == "" {
		fileName = filepath.Base(fileResult.FilePath)
	}

	objectPath := fmt.Sprintf("%d/%d/%d/%s", accountID, inbox.ID, msg.ID, fileName)

	reader := newBytesReader(data)
	contentType := attrs.MimeType
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	_, err = r.minioClient.PutObject(ctx, r.minioBucket, objectPath, reader, int64(len(data)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("telegram resolve media: upload to minio: %w", err)
	}

	fileKey := objectPath
	if err := r.attachmentRepo.UpdateFileKey(ctx, attachment.ID, fileKey); err != nil {
		logger.Warn().Str("component", "telegram.media").Err(err).Msg("failed to update file_key after minio upload")
	}

	return objectPath, nil
}

type bytesReader struct {
	data []byte
	pos  int
}

func newBytesReader(data []byte) *bytesReader {
	return &bytesReader{data: data}
}

func (r *bytesReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

var _ io.Reader = (*bytesReader)(nil)

func IsTelegramChannel(channelType string) bool {
	return channelType == "Channel::Telegram"
}

func ParseFileIDFromContentAttrs(contentAttrs *string) (string, bool) {
	if contentAttrs == nil || *contentAttrs == "" {
		return "", false
	}
	var attrs fileAttrs
	if err := json.Unmarshal([]byte(*contentAttrs), &attrs); err != nil {
		return "", false
	}
	if attrs.FileID == "" {
		return "", false
	}
	return attrs.FileID, true
}
