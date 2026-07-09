package storage_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/nsmeds/livery-stable/storage"
)

func TestFilesystemStore_PutAndDelete(t *testing.T) {
	root := t.TempDir()
	store := storage.NewFilesystemStore(root)
	ctx := context.Background()

	key := "abc/file.wav"
	content := []byte("fake audio content")

	if err := store.Put(ctx, key, bytes.NewReader(content)); err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	got, err := os.ReadFile(filepath.Join(root, key))
	if err != nil {
		t.Fatalf("reading stored file: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("stored content = %q, want %q", got, content)
	}

	if err := store.Delete(ctx, key); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, key)); !os.IsNotExist(err) {
		t.Errorf("expected file to be removed, stat err = %v", err)
	}
}

func TestFilesystemStore_DeleteMissingKeyIsNoop(t *testing.T) {
	store := storage.NewFilesystemStore(t.TempDir())
	if err := store.Delete(context.Background(), "does/not/exist.wav"); err != nil {
		t.Errorf("Delete() on missing key error = %v, want nil", err)
	}
}
