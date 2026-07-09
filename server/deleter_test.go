package server_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/nsmeds/livery-stable/server"
	"github.com/nsmeds/livery-stable/storage"
)

func TestDeleter_EnqueueAndWait(t *testing.T) {
	root := t.TempDir()
	fsStore := storage.NewFilesystemStore(root)
	ctx := context.Background()

	key := "file.wav"
	if err := fsStore.Put(ctx, key, bytes.NewReader([]byte("data"))); err != nil {
		t.Fatalf("Put: %v", err)
	}

	d := server.NewDeleter(fsStore)
	d.Enqueue(key)
	d.Wait()

	if _, err := os.Stat(filepath.Join(root, key)); !os.IsNotExist(err) {
		t.Errorf("expected file to be deleted, stat err = %v", err)
	}
}

func TestDeleter_MultipleEnqueues(t *testing.T) {
	root := t.TempDir()
	fsStore := storage.NewFilesystemStore(root)
	ctx := context.Background()

	keys := []string{"a.wav", "b.wav", "c.wav"}
	for _, k := range keys {
		if err := fsStore.Put(ctx, k, bytes.NewReader([]byte("data"))); err != nil {
			t.Fatalf("Put: %v", err)
		}
	}

	d := server.NewDeleter(fsStore)
	for _, k := range keys {
		d.Enqueue(k)
	}
	d.Wait()

	for _, k := range keys {
		if _, err := os.Stat(filepath.Join(root, k)); !os.IsNotExist(err) {
			t.Errorf("expected %q to be deleted, stat err = %v", k, err)
		}
	}
}
