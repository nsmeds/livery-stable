package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var ErrNotFound = errors.New("store: not found")

type File struct {
	ID              uuid.UUID
	OwnerID         uuid.UUID
	Filename        string
	Format          string
	SizeBytes       int64
	DurationSeconds *float32
	StorageKey      string
	UploadedAt      time.Time
	DeletedAt       *time.Time
}

func (s *Store) CreateFile(ctx context.Context, id, ownerID uuid.UUID, filename, format string, sizeBytes int64, durationSeconds *float32, storageKey string) (*File, error) {
	var f File
	err := s.pool.QueryRow(ctx,
		`INSERT INTO files (id, owner_id, filename, format, size_bytes, duration_seconds, storage_key)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, owner_id, filename, format, size_bytes, duration_seconds, storage_key, uploaded_at, deleted_at`,
		id, ownerID, filename, format, sizeBytes, durationSeconds, storageKey,
	).Scan(&f.ID, &f.OwnerID, &f.Filename, &f.Format, &f.SizeBytes, &f.DurationSeconds, &f.StorageKey, &f.UploadedAt, &f.DeletedAt)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (s *Store) GetFile(ctx context.Context, id uuid.UUID) (*File, error) {
	var f File
	err := s.pool.QueryRow(ctx,
		`SELECT id, owner_id, filename, format, size_bytes, duration_seconds, storage_key, uploaded_at, deleted_at
		 FROM files WHERE id = $1`,
		id,
	).Scan(&f.ID, &f.OwnerID, &f.Filename, &f.Format, &f.SizeBytes, &f.DurationSeconds, &f.StorageKey, &f.UploadedAt, &f.DeletedAt)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

// ListFilesByOwner returns the owner's files, excluding soft-deleted ones,
// most recently uploaded first.
func (s *Store) ListFilesByOwner(ctx context.Context, ownerID uuid.UUID) ([]File, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, owner_id, filename, format, size_bytes, duration_seconds, storage_key, uploaded_at, deleted_at
		 FROM files WHERE owner_id = $1 AND deleted_at IS NULL ORDER BY uploaded_at DESC`,
		ownerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	files := []File{}
	for rows.Next() {
		var f File
		if err := rows.Scan(&f.ID, &f.OwnerID, &f.Filename, &f.Format, &f.SizeBytes, &f.DurationSeconds, &f.StorageKey, &f.UploadedAt, &f.DeletedAt); err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return files, nil
}

// SoftDeleteFile marks a file deleted, scoped to its owner so a user can't
// delete another user's file, and returns its storage key so the caller can
// schedule the object for removal from storage. Returns ErrNotFound if no
// matching, undeleted row exists.
func (s *Store) SoftDeleteFile(ctx context.Context, id, ownerID uuid.UUID) (string, error) {
	var storageKey string
	err := s.pool.QueryRow(ctx,
		`UPDATE files SET deleted_at = NOW() WHERE id = $1 AND owner_id = $2 AND deleted_at IS NULL
		 RETURNING storage_key`,
		id, ownerID,
	).Scan(&storageKey)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}
	return storageKey, nil
}
