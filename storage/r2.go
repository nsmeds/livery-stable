package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// R2Config holds the credentials needed to reach a Cloudflare R2 bucket
// through its S3-compatible API.
type R2Config struct {
	AccountID       string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
}

// R2Store stores objects in a Cloudflare R2 bucket via the S3-compatible
// API, using the AWS SDK's multipart upload manager so large files are
// uploaded in chunks without any custom protocol.
//
// manager.Uploader is deprecated in favor of feature/s3/transfermanager, but
// that replacement is still pre-1.0 (v0.3.x) and its API isn't stable yet.
// Since this code can't be exercised against a live bucket until R2
// credentials are provisioned, staying on the stable-but-deprecated Uploader
// is the safer choice for now; revisit once transfermanager reaches 1.0.
type R2Store struct {
	bucket string
	client *s3.Client
	//lint:ignore SA1019 see rationale above
	uploader *manager.Uploader
}

func NewR2Store(ctx context.Context, cfg R2Config) (*R2Store, error) {
	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("auto"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("storage: loading R2 config: %w", err)
	}

	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfg.AccountID)
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
	})

	return &R2Store{
		bucket: cfg.Bucket,
		client: client,
		//lint:ignore SA1019 see rationale on R2Store
		uploader: manager.NewUploader(client),
	}, nil
}

func (s *R2Store) Put(ctx context.Context, key string, r io.Reader) error {
	//lint:ignore SA1019 see rationale on R2Store
	_, err := s.uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   r,
	})
	return err
}

func (s *R2Store) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	return err
}
