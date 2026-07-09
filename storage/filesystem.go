package storage

import (
	"context"
	"io"
	"os"
	"path/filepath"
)

// FilesystemStore stores objects as files under a root directory. It's the
// default Store used for local development and tests, before Cloudflare R2
// credentials are provisioned.
type FilesystemStore struct {
	root string
}

func NewFilesystemStore(root string) *FilesystemStore {
	return &FilesystemStore{root: root}
}

func (s *FilesystemStore) Put(ctx context.Context, key string, r io.Reader, size int64) error {
	path := filepath.Join(s.root, key)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, r)
	return err
}

func (s *FilesystemStore) Delete(ctx context.Context, key string) error {
	err := os.Remove(filepath.Join(s.root, key))
	if err != nil && os.IsNotExist(err) {
		return nil
	}
	return err
}
