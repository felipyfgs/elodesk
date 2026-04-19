package handler

import (
	"fmt"
	"net/url"
	"time"

	"github.com/gofiber/fiber/v2"

	"backend/internal/dto"
	"backend/internal/media"
)

const presignedTTL = 15 * time.Minute

type UploadHandler struct {
	minio *media.MinioClient
}

func NewUploadHandler(minio *media.MinioClient) *UploadHandler {
	return &UploadHandler{minio: minio}
}

func (h *UploadHandler) SignedUploadURL(c *fiber.Ctx) error {
	objectPath := c.Query("path")
	if objectPath == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "path query parameter is required"))
	}

	presignedURL, err := h.minio.Client().PresignedPutObject(c.Context(), h.minio.Bucket(), objectPath, presignedTTL)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to generate presigned URL"))
	}

	return c.JSON(dto.SuccessResp(fiber.Map{
		"upload_url": presignedURL.String(),
	}))
}

func (h *UploadHandler) SignedDownloadURL(c *fiber.Ctx) error {
	accountID := c.Params("aid")
	attachmentID := c.Params("id")

	objectPath := fmt.Sprintf("attachments/%s/%s", accountID, attachmentID)

	reqParams := url.Values{}

	presignedURL, err := h.minio.Client().PresignedGetObject(c.Context(), h.minio.Bucket(), objectPath, presignedTTL, reqParams)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to generate download URL"))
	}

	return c.JSON(dto.SuccessResp(fiber.Map{
		"download_url": presignedURL.String(),
	}))
}
