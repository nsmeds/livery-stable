package storage_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/nsmeds/livery-stable/storage"
)

const testR2Bucket = "livery-stable-test"

var testR2Cfg storage.R2Config

// TestMain ensures a local MinIO instance (started via `make compose-up`)
// has the test bucket before any R2Store test runs, mirroring how the store
// package's TestMain runs migrations against a real test database.
func TestMain(m *testing.M) {
	endpoint := os.Getenv("MINIO_ENDPOINT")
	if endpoint == "" {
		endpoint = "localhost:9000"
	}
	accessKeyID := os.Getenv("MINIO_ROOT_USER")
	if accessKeyID == "" {
		accessKeyID = "minioadmin"
	}
	secretAccessKey := os.Getenv("MINIO_ROOT_PASSWORD")
	if secretAccessKey == "" {
		secretAccessKey = "minioadmin"
	}

	testR2Cfg = storage.R2Config{
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
		Bucket:          testR2Bucket,
		Endpoint:        endpoint,
		Secure:          false,
		Region:          "us-east-1",
	}

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: false,
		Region: "us-east-1",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "storage tests: minio client: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	exists, err := client.BucketExists(ctx, testR2Bucket)
	if err != nil {
		fmt.Fprintf(os.Stderr, "storage tests: bucket exists check: %v\n", err)
		os.Exit(1)
	}
	if !exists {
		if err := client.MakeBucket(ctx, testR2Bucket, minio.MakeBucketOptions{Region: "us-east-1"}); err != nil {
			fmt.Fprintf(os.Stderr, "storage tests: make bucket: %v\n", err)
			os.Exit(1)
		}
	}

	os.Exit(m.Run())
}
