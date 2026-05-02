package handler

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"

	"backend/internal/dto"
	"backend/internal/logger"
	"backend/internal/media"
	"backend/internal/repo"
)

const presignedTTL = 15 * time.Minute

type MediaResolveFunc func(ctx context.Context, attachmentID int64, accountID int64) (string, error)

type UploadHandler struct {
	minio          *media.MinioClient
	attachmentRepo *repo.AttachmentRepo
	mediaResolver  MediaResolveFunc
	tokenSecret    []byte
}

func NewUploadHandler(minio *media.MinioClient, attachmentRepo *repo.AttachmentRepo) *UploadHandler {
	return &UploadHandler{minio: minio, attachmentRepo: attachmentRepo}
}

func (h *UploadHandler) SetMediaResolver(fn MediaResolveFunc) {
	h.mediaResolver = fn
}

func (h *UploadHandler) SetAttachmentTokenSecret(secret []byte) {
	h.tokenSecret = secret
}

func (h *UploadHandler) SignedUploadURL(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	objectPath, err := scopedObjectPath(c, accountID)
	if err != nil {
		return err
	}

	logger.Info().Str("component", "uploads").
		Int64("accountId", accountID).
		Str("bucket", h.minio.Bucket()).
		Str("objectPath", objectPath).
		Msg("generating presigned put URL")

	presignedURL, err := h.minio.PresignClient().PresignedPutObject(c.Context(), h.minio.Bucket(), objectPath, presignedTTL)
	if err != nil {
		logger.Error().Str("component", "uploads").Err(err).
			Int64("accountId", accountID).
			Str("objectPath", objectPath).
			Msg("failed to generate presigned put URL")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to generate presigned URL"))
	}

	logger.Info().Str("component", "uploads").
		Int64("accountId", accountID).
		Str("url", presignedURL.String()).
		Msg("presigned put URL generated")

	return c.JSON(dto.SuccessResp(fiber.Map{"upload_url": presignedURL.String()}))
}

func (h *UploadHandler) ProxyUpload(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	fh, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "file is required"))
	}

	file, err := fh.Open()
	if err != nil {
		logger.Error().Str("component", "uploads").Err(err).Msg("failed to open uploaded file")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to read upload"))
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			logger.Warn().Str("component", "uploads").Err(closeErr).Msg("close uploaded file")
		}
	}()

	safeName := sanitizeFileName(fh.Filename)
	objectPath := fmt.Sprintf("%d/uploads/%s-%s", accountID, uuid.New().String(), safeName)

	contentType := fh.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	ctx, cancel := context.WithTimeout(c.Context(), 2*time.Minute)
	defer cancel()

	_, err = h.minio.Client().PutObject(ctx, h.minio.Bucket(), objectPath, file, fh.Size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		logger.Error().Str("component", "uploads").Err(err).
			Int64("accountId", accountID).
			Str("objectPath", objectPath).
			Msg("failed to upload to minio")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to upload file"))
	}

	logger.Info().Str("component", "uploads").
		Int64("accountId", accountID).
		Str("objectPath", objectPath).
		Int64("size", fh.Size).
		Str("contentType", contentType).
		Msg("proxy upload complete")

	return c.JSON(dto.SuccessResp(fiber.Map{
		"path":      objectPath,
		"file_type": contentType,
		"file_name": fh.Filename,
		"size":      fh.Size,
	}))
}

func sanitizeFileName(name string) string {
	replacer := strings.NewReplacer(" ", "_", "/", "_", "\\", "_")
	s := replacer.Replace(name)
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '_', r == '-':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	out := b.String()
	if out == "" {
		return "file"
	}
	return out
}

func (h *UploadHandler) PublicAttachmentDownload(c *fiber.Ctx) error {
	if len(h.tokenSecret) == 0 {
		return c.Status(fiber.StatusServiceUnavailable).JSON(dto.ErrorResp("Service Unavailable", "attachment download disabled"))
	}

	idStr := c.Params("id")
	attachmentID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid attachment id"))
	}

	token := c.Query("token")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "token required"))
	}

	tokenAccountID, tokenAttachmentID, err := VerifyAttachmentToken(h.tokenSecret, token)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "invalid or expired token"))
	}
	if tokenAttachmentID != attachmentID {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "token does not match attachment"))
	}

	attachment, err := h.attachmentRepo.FindByID(c.Context(), attachmentID, tokenAccountID)
	if err != nil {
		return handleNotFound(c, err)
	}

	var objectPath string
	if attachment.FileKey != nil && *attachment.FileKey != "" {
		objectPath = *attachment.FileKey
	} else if h.mediaResolver != nil {
		resolved, rerr := h.mediaResolver(c.Context(), attachmentID, tokenAccountID)
		if rerr == nil && resolved != "" {
			objectPath = resolved
		}
	}
	if objectPath == "" {
		if attachment.ExternalURL != nil && *attachment.ExternalURL != "" {
			return c.Redirect(*attachment.ExternalURL, fiber.StatusFound)
		}
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", "attachment has no storage location"))
	}

	obj, err := h.minio.Client().GetObject(c.Context(), h.minio.Bucket(), objectPath, minio.GetObjectOptions{})
	if err != nil {
		logger.Error().Str("component", "uploads").Err(err).Msg("failed to get object")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to fetch file"))
	}
	stat, err := obj.Stat()
	if err != nil {
		if closeErr := obj.Close(); closeErr != nil {
			logger.Warn().Str("component", "uploads").Err(closeErr).Msg("close minio object after stat error")
		}
		logger.Warn().Str("component", "uploads").Err(err).Str("objectPath", objectPath).Msg("object not found")
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", "file not found"))
	}

	if stat.ContentType != "" {
		c.Set("Content-Type", stat.ContentType)
	}
	c.Set("Content-Length", strconv.FormatInt(stat.Size, 10))
	c.Set("Cache-Control", "public, max-age=31536000, immutable")
	if stat.ETag != "" {
		c.Set("ETag", stat.ETag)
	}
	c.Set("Last-Modified", stat.LastModified.UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT"))
	return c.SendStream(obj, int(stat.Size))
}

func (h *UploadHandler) ProxyDownload(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	objectPath, err := scopedObjectPath(c, accountID)
	if err != nil {
		return err
	}

	obj, err := h.minio.Client().GetObject(c.Context(), h.minio.Bucket(), objectPath, minio.GetObjectOptions{})
	if err != nil {
		logger.Error().Str("component", "uploads").Err(err).Msg("failed to get object")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to fetch file"))
	}

	stat, err := obj.Stat()
	if err != nil {
		if closeErr := obj.Close(); closeErr != nil {
			logger.Warn().Str("component", "uploads").Err(closeErr).Msg("close minio object after stat error")
		}
		logger.Warn().Str("component", "uploads").Err(err).Str("objectPath", objectPath).Msg("object not found")
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", "file not found"))
	}

	if stat.ContentType != "" {
		c.Set("Content-Type", stat.ContentType)
	}
	c.Set("Content-Length", strconv.FormatInt(stat.Size, 10))
	c.Set("Cache-Control", "private, max-age=3600")
	return c.SendStream(obj, int(stat.Size))
}

func (h *UploadHandler) SignedObjectDownloadURL(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	objectPath, err := scopedObjectPath(c, accountID)
	if err != nil {
		return err
	}

	presignedURL, err := h.minio.PresignClient().PresignedGetObject(c.Context(), h.minio.Bucket(), objectPath, presignedTTL, url.Values{})
	if err != nil {
		logger.Error().Str("component", "uploads").Err(err).Msg("failed to generate object download URL")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to generate download URL"))
	}

	return c.JSON(dto.SuccessResp(fiber.Map{"download_url": presignedURL.String()}))
}

func scopedObjectPath(c *fiber.Ctx, accountID int64) (string, error) {
	objectPath := c.Query("path")
	if objectPath == "" {
		return "", c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "path query parameter is required"))
	}

	expectedPrefix := strconv.FormatInt(accountID, 10) + "/"
	if !strings.HasPrefix(objectPath, expectedPrefix) {
		return "", c.Status(fiber.StatusForbidden).JSON(dto.ErrorResp("Forbidden", "path must start with your accountId"))
	}

	return objectPath, nil
}

func (h *UploadHandler) AttachmentMediaURL(c *fiber.Ctx) error {
	if len(h.tokenSecret) == 0 {
		return c.Status(fiber.StatusServiceUnavailable).JSON(
			dto.ErrorResp("Service Unavailable", "media URL signing not configured"),
		)
	}

	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	attachmentID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid attachment id"))
	}

	_, err = h.attachmentRepo.FindByID(c.Context(), attachmentID, accountID)
	if err != nil {
		return handleNotFound(c, err)
	}

	token := SignAttachmentToken(h.tokenSecret, accountID, attachmentID)

	return c.JSON(dto.SuccessResp(fiber.Map{
		"token":      token,
		"expires_at": 0,
	}))
}
