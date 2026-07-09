package storage_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/nsmeds/livery-stable/storage"
)

// verifyClient talks to the same MinIO instance R2Store does, but
// independently of it, so tests can check on-the-wire state (existence,
// content) without relying on the code under test to report its own
// correctness.
func verifyClient(t *testing.T) *minio.Client {
	t.Helper()
	client, err := minio.New(testR2Cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(testR2Cfg.AccessKeyID, testR2Cfg.SecretAccessKey, ""),
		Secure: testR2Cfg.Secure,
		Region: testR2Cfg.Region,
	})
	if err != nil {
		t.Fatalf("minio.New: %v", err)
	}
	return client
}

func TestR2Store_PutAndDelete(t *testing.T) {
	store, err := storage.NewR2Store(context.Background(), testR2Cfg)
	if err != nil {
		t.Fatalf("NewR2Store: %v", err)
	}
	verify := verifyClient(t)
	ctx := context.Background()

	key := fmt.Sprintf("test/%s.txt", uuid.New())
	content := []byte("fake audio content")
	t.Cleanup(func() {
		_ = verify.RemoveObject(context.Background(), testR2Cfg.Bucket, key, minio.RemoveObjectOptions{})
	})

	if err := store.Put(ctx, key, bytes.NewReader(content), int64(len(content))); err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	obj, err := verify.GetObject(ctx, testR2Cfg.Bucket, key, minio.GetObjectOptions{})
	if err != nil {
		t.Fatalf("GetObject: %v", err)
	}
	got, err := io.ReadAll(obj)
	if err != nil {
		t.Fatalf("reading object: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("stored content = %q, want %q", got, content)
	}

	if err := store.Delete(ctx, key); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if _, err := verify.StatObject(ctx, testR2Cfg.Bucket, key, minio.StatObjectOptions{}); err == nil {
		t.Error("expected object to be deleted, but StatObject succeeded")
	}
}

func TestR2Store_DeleteMissingKeyIsNoop(t *testing.T) {
	store, err := storage.NewR2Store(context.Background(), testR2Cfg)
	if err != nil {
		t.Fatalf("NewR2Store: %v", err)
	}

	key := fmt.Sprintf("test/does-not-exist-%s.txt", uuid.New())
	if err := store.Delete(context.Background(), key); err != nil {
		t.Errorf("Delete() on missing key error = %v, want nil", err)
	}
}
