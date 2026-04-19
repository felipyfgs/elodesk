package media

import (
	"context"
	"fmt"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"backend/internal/logger"
)

type MinioClient struct {
	client *minio.Client
	bucket string
}

func New(endpoint, port string, useSSL bool, accessKey, secretKey, bucket string) (*MinioClient, error) {
	addr := fmt.Sprintf("%s:%s", endpoint, port)

	client, err := minio.New(addr, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	return &MinioClient{
		client: client,
		bucket: bucket,
	}, nil
}

func (m *MinioClient) EnsureBucket(ctx context.Context) error {
	exists, err := m.client.BucketExists(ctx, m.bucket)
	if err != nil {
		logger.Warn().
			Str("component", "media").
			Err(err).
			Str("bucket", m.bucket).
			Msg("failed to check bucket existence")
		return nil
	}

	if exists {
		return nil
	}

	if err := m.client.MakeBucket(ctx, m.bucket, minio.MakeBucketOptions{}); err != nil {
		logger.Warn().
			Str("component", "media").
			Err(err).
			Str("bucket", m.bucket).
			Msg("failed to create bucket")
		return nil
	}

	logger.Info().
		Str("component", "media").
		Str("bucket", m.bucket).
		Msg("created minio bucket")
	return nil
}

func (m *MinioClient) Client() *minio.Client {
	return m.client
}

func (m *MinioClient) Bucket() string {
	return m.bucket
}
