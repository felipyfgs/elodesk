package handler

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

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
}

func NewUploadHandler(minio *media.MinioClient, attachmentRepo *repo.AttachmentRepo) *UploadHandler {
	return &UploadHandler{minio: minio, attachmentRepo: attachmentRepo}
}

func (h *UploadHandler) SetMediaResolver(fn MediaResolveFunc) {
	h.mediaResolver = fn
}

// SignedUploadURL generates a presigned PUT URL. The object path MUST begin
// with "{accountId}/" of the authenticated request — this prevents an agent
// from requesting a presigned URL that writes into another tenant's prefix.
func (h *UploadHandler) SignedUploadURL(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	objectPath, err := scopedObjectPath(c, accountID)
	if err != nil {
		return err
	}

	presignedURL, err := h.minio.Client().PresignedPutObject(c.Context(), h.minio.Bucket(), objectPath, presignedTTL)
	if err != nil {
		logger.Error().Str("component", "uploads").Err(err).Msg("failed to generate presigned put URL")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to generate presigned URL"))
	}

	return c.JSON(dto.SuccessResp(fiber.Map{"upload_url": presignedURL.String()}))
}

// SignedObjectDownloadURL generates a presigned GET URL for an object path
// already scoped to the authenticated account. This is used for private
// account-owned objects that are not attachment rows, such as contact avatars.
func (h *UploadHandler) SignedObjectDownloadURL(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	objectPath, err := scopedObjectPath(c, accountID)
	if err != nil {
		return err
	}

	presignedURL, err := h.minio.Client().PresignedGetObject(c.Context(), h.minio.Bucket(), objectPath, presignedTTL, url.Values{})
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

// SignedDownloadURL verifies the attachment belongs to the authenticated
// account before producing a presigned GET URL. Without this, any agent could
// download any tenant's attachment by guessing the id.
func (h *UploadHandler) SignedDownloadURL(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	attachmentID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid attachment id"))
	}

	attachment, err := h.attachmentRepo.FindByID(c.Context(), attachmentID, accountID)
	if err != nil {
		return handleNotFound(c, err)
	}

	var objectPath string
	if attachment.FileKey != nil && *attachment.FileKey != "" {
		objectPath = *attachment.FileKey
	} else if h.mediaResolver != nil {
		resolved, err := h.mediaResolver(c.Context(), attachmentID, accountID)
		if err == nil && resolved != "" {
			objectPath = resolved
		} else {
			objectPath = fmt.Sprintf("%d/%d", attachment.AccountID, attachment.ID)
		}
	} else {
		objectPath = fmt.Sprintf("%d/%d", attachment.AccountID, attachment.ID)
	}

	presignedURL, err := h.minio.Client().PresignedGetObject(c.Context(), h.minio.Bucket(), objectPath, presignedTTL, url.Values{})
	if err != nil {
		logger.Error().Str("component", "uploads").Err(err).Msg("failed to generate download URL")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to generate download URL"))
	}

	return c.JSON(dto.SuccessResp(fiber.Map{"download_url": presignedURL.String()}))
}
