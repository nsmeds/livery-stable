// Package storage provides an object-storage abstraction so upload/delete
// code can run against a local filesystem in dev/test and against
// Cloudflare R2 in production, without changing callers.
package storage

import (
	"context"
	"io"
)

// Store persists and removes file content addressed by an opaque key.
type Store interface {
	Put(ctx context.Context, key string, r io.Reader) error
	Delete(ctx context.Context, key string) error
}
