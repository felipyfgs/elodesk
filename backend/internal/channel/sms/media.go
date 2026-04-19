package sms

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"

	"backend/internal/logger"
	"backend/internal/media"
	"backend/internal/model"
	"backend/internal/repo"
)

const smsHTTPTimeout = 30 * time.Second

type MediaHandler struct {
	httpClient *http.Client
	minio      *media.MinioClient
	attachRepo *repo.AttachmentRepo
}

func NewMediaHandler(minio *media.MinioClient, attachRepo *repo.AttachmentRepo) *MediaHandler {
	return &MediaHandler{
		httpClient: &http.Client{Timeout: smsHTTPTimeout},
		minio:      minio,
		attachRepo: attachRepo,
	}
}

func (h *MediaHandler) DownloadAndStore(ctx context.Context, mediaURL, accountID, inboxID, messageID string, mediaType string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, mediaURL, nil)
	if err != nil {
		return fmt.Errorf("sms media: create request: %w", err)
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("sms media: download: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("sms media: download failed with status %d", resp.StatusCode)
	}

	ext := extensionFromURL(mediaURL, mediaType)
	objectPath := fmt.Sprintf("%s/%s/%s/%s", accountID, inboxID, messageID, ext)

	_, err = h.minio.Client().PutObject(ctx, h.minio.Bucket(), objectPath, resp.Body, resp.ContentLength, minio.PutObjectOptions{
		ContentType: contentTypeFromMIME(mediaType),
	})
	if err != nil {
		return fmt.Errorf("sms media: upload to minio: %w", err)
	}

	fileType := fileTypeFromMIME(mediaType)
	attachment := &model.Attachment{
		MessageID: mustParseInt64(messageID),
		AccountID: mustParseInt64(accountID),
		FileType:  fileType,
		FileKey:   &objectPath,
		Extension: &ext,
	}

	if err := h.attachRepo.Create(ctx, attachment); err != nil {
		logger.Warn().Str("component", "sms.media").Err(err).Msg("failed to create attachment row")
		return fmt.Errorf("sms media: create attachment: %w", err)
	}

	return nil
}

func (h *MediaHandler) DownloadAndStoreAll(ctx context.Context, urls []string, types []string, accountID, inboxID, messageID string) {
	for i, u := range urls {
		mt := ""
		if i < len(types) {
			mt = types[i]
		}
		if err := h.DownloadAndStore(ctx, u, accountID, inboxID, messageID, mt); err != nil {
			logger.Warn().Str("component", "sms.media").Err(err).Str("url", u).Msg("failed to download media")
		}
	}
}

func extensionFromURL(url, mediaType string) string {
	if mediaType != "" {
		exts := map[string]string{
			"image/jpeg":      "jpg",
			"image/png":       "png",
			"image/gif":       "gif",
			"image/webp":      "webp",
			"video/mp4":       "mp4",
			"video/3gpp":      "3gp",
			"audio/aac":       "aac",
			"audio/ogg":       "ogg",
			"audio/mpeg":      "mp3",
			"application/pdf": "pdf",
		}
		if ext, ok := exts[mediaType]; ok {
			return fmt.Sprintf("%d.%s", time.Now().UnixNano(), ext)
		}
	}

	p := strings.Split(url, "/")
	if len(p) > 0 {
		filename := p[len(p)-1]
		ext := filepath.Ext(filename)
		if ext != "" {
			return fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
		}
	}

	return fmt.Sprintf("%d.unknown", time.Now().UnixNano())
}

func contentTypeFromMIME(mime string) string {
	if mime != "" {
		return mime
	}
	return "application/octet-stream"
}

func fileTypeFromMIME(mime string) model.AttachmentFileType {
	switch {
	case strings.HasPrefix(mime, "image/"):
		return model.FileTypeImage
	case strings.HasPrefix(mime, "video/"):
		return model.FileTypeVideo
	case strings.HasPrefix(mime, "audio/"):
		return model.FileTypeAudio
	default:
		return model.FileTypeFile
	}
}

func mustParseInt64(s string) int64 {
	var n int64
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int64(c-'0')
		}
	}
	return n
}
