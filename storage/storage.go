// Package storage provides an object-storage abstraction so upload/delete
// code can run against a local filesystem in dev/test and against
// Cloudflare R2 in production, without changing callers.
package storage

import (
	"context"
	"io"
)

// Store persists and removes file content addressed by an opaque key. size
// is the number of bytes r will yield; passing the true size (rather than
// -1 for "unknown") lets the R2 implementation upload more efficiently.
type Store interface {
	Put(ctx context.Context, key string, r io.Reader, size int64) error
	Delete(ctx context.Context, key string) error
}
