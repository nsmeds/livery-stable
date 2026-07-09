package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// R2Config holds the credentials needed to reach a Cloudflare R2 bucket
// through its S3-compatible API.
type R2Config struct {
	AccountID       string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string

	// Endpoint, Secure, and Region override the values normally derived
	// from AccountID (an https R2 endpoint, region "auto"). Tests set
	// these to point at a local MinIO instance instead of a live bucket.
	Endpoint string
	Secure   bool
	Region   string
}

// R2Store stores objects in a Cloudflare R2 bucket via the S3-compatible
// API, using minio-go, which is purpose-built for S3-compatible stores
// (R2, MinIO, Spaces) rather than the full AWS SDK surface. It handles
// multipart upload transparently for large files.
type R2Store struct {
	client *minio.Client
	bucket string
}

func NewR2Store(ctx context.Context, cfg R2Config) (*R2Store, error) {
	endpoint := cfg.Endpoint
	secure := cfg.Secure
	region := cfg.Region
	if endpoint == "" {
		endpoint = fmt.Sprintf("%s.r2.cloudflarestorage.com", cfg.AccountID)
		secure = true
	}
	if region == "" {
		region = "auto" // Cloudflare R2's documented region value
	}

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: secure,
		Region: region,
	})
	if err != nil {
		return nil, fmt.Errorf("storage: creating R2 client: %w", err)
	}

	return &R2Store{client: client, bucket: cfg.Bucket}, nil
}

func (s *R2Store) Put(ctx context.Context, key string, r io.Reader, size int64) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, r, size, minio.PutObjectOptions{})
	return err
}

func (s *R2Store) Delete(ctx context.Context, key string) error {
	return s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
}
