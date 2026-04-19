package media

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
)

const MaxUploadSize int64 = 268435456

var ErrFileTooLarge = errors.New("file size exceeds maximum allowed size of 256MB")

type UploadService struct {
	client *MinioClient
}

func NewUploadService(client *MinioClient) *UploadService {
	return &UploadService{client: client}
}

func (s *UploadService) Upload(ctx context.Context, reader io.Reader, objectPath string, size int64, contentType string) error {
	if size > MaxUploadSize {
		return fmt.Errorf("%w: got %d bytes", ErrFileTooLarge, size)
	}

	_, err := s.client.Client().PutObject(ctx, s.client.Bucket(), objectPath, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("failed to upload object %s: %w", objectPath, err)
	}

	return nil
}

func BuildObjectPath(accountID, inboxID, messageID string, ext string) string {
	return fmt.Sprintf("%s/%s/%s.%s", accountID, inboxID, messageID, ext)
}
