package store_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/nsmeds/livery-stable/store"
)

func TestCreateFile(t *testing.T) {
	ctx := context.Background()
	s := store.New(testPool)
	user := createTestUser(t)

	duration := float32(123.45)
	f, err := s.CreateFile(ctx, uuid.New(), user.ID, "song.wav", "wav", 1024, &duration, "owner/key.wav")
	if err != nil {
		t.Fatalf("CreateFile: %v", err)
	}

	if f.OwnerID != user.ID {
		t.Errorf("owner_id = %v, want %v", f.OwnerID, user.ID)
	}
	if f.Filename != "song.wav" {
		t.Errorf("filename = %q, want %q", f.Filename, "song.wav")
	}
	if f.Format != "wav" {
		t.Errorf("format = %q, want %q", f.Format, "wav")
	}
	if f.SizeBytes != 1024 {
		t.Errorf("size_bytes = %d, want %d", f.SizeBytes, 1024)
	}
	if f.DurationSeconds == nil || *f.DurationSeconds != duration {
		t.Errorf("duration_seconds = %v, want %v", f.DurationSeconds, duration)
	}
	if f.DeletedAt != nil {
		t.Error("expected deleted_at to be nil")
	}
}

func TestCreateFile_NilDuration(t *testing.T) {
	ctx := context.Background()
	s := store.New(testPool)
	user := createTestUser(t)

	f, err := s.CreateFile(ctx, uuid.New(), user.ID, "song.mp3", "mp3", 2048, nil, "owner/key.mp3")
	if err != nil {
		t.Fatalf("CreateFile: %v", err)
	}
	if f.DurationSeconds != nil {
		t.Errorf("duration_seconds = %v, want nil", f.DurationSeconds)
	}
}

func TestGetFile(t *testing.T) {
	ctx := context.Background()
	s := store.New(testPool)
	user := createTestUser(t)

	created, err := s.CreateFile(ctx, uuid.New(), user.ID, "song.aiff", "aiff", 4096, nil, "owner/key.aiff")
	if err != nil {
		t.Fatalf("CreateFile: %v", err)
	}

	found, err := s.GetFile(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetFile: %v", err)
	}
	if found.ID != created.ID {
		t.Errorf("id = %v, want %v", found.ID, created.ID)
	}
}

func TestGetFile_NotFound(t *testing.T) {
	ctx := context.Background()
	s := store.New(testPool)

	_, err := s.GetFile(ctx, uuid.New())
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestListFilesByOwner(t *testing.T) {
	ctx := context.Background()
	s := store.New(testPool)
	owner := createTestUser(t)
	other := createTestUser(t)

	if _, err := s.CreateFile(ctx, uuid.New(), owner.ID, "a.mp3", "mp3", 1, nil, "k1"); err != nil {
		t.Fatalf("CreateFile: %v", err)
	}
	if _, err := s.CreateFile(ctx, uuid.New(), owner.ID, "b.mp3", "mp3", 1, nil, "k2"); err != nil {
		t.Fatalf("CreateFile: %v", err)
	}
	if _, err := s.CreateFile(ctx, uuid.New(), other.ID, "c.mp3", "mp3", 1, nil, "k3"); err != nil {
		t.Fatalf("CreateFile: %v", err)
	}

	files, err := s.ListFilesByOwner(ctx, owner.ID)
	if err != nil {
		t.Fatalf("ListFilesByOwner: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("len(files) = %d, want 2", len(files))
	}
	for _, f := range files {
		if f.OwnerID != owner.ID {
			t.Errorf("file %v owner_id = %v, want %v", f.ID, f.OwnerID, owner.ID)
		}
	}
}

func TestListFilesByOwner_ExcludesDeleted(t *testing.T) {
	ctx := context.Background()
	s := store.New(testPool)
	owner := createTestUser(t)

	f, err := s.CreateFile(ctx, uuid.New(), owner.ID, "a.mp3", "mp3", 1, nil, "k1")
	if err != nil {
		t.Fatalf("CreateFile: %v", err)
	}
	if _, err := s.SoftDeleteFile(ctx, f.ID, owner.ID); err != nil {
		t.Fatalf("SoftDeleteFile: %v", err)
	}

	files, err := s.ListFilesByOwner(ctx, owner.ID)
	if err != nil {
		t.Fatalf("ListFilesByOwner: %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("len(files) = %d, want 0", len(files))
	}
}

func TestSoftDeleteFile(t *testing.T) {
	ctx := context.Background()
	s := store.New(testPool)
	owner := createTestUser(t)

	f, err := s.CreateFile(ctx, uuid.New(), owner.ID, "a.mp3", "mp3", 1, nil, "k1")
	if err != nil {
		t.Fatalf("CreateFile: %v", err)
	}

	storageKey, err := s.SoftDeleteFile(ctx, f.ID, owner.ID)
	if err != nil {
		t.Fatalf("SoftDeleteFile: %v", err)
	}
	if storageKey != "k1" {
		t.Errorf("storageKey = %q, want %q", storageKey, "k1")
	}

	found, err := s.GetFile(ctx, f.ID)
	if err != nil {
		t.Fatalf("GetFile: %v", err)
	}
	if found.DeletedAt == nil {
		t.Error("expected deleted_at to be set")
	}
}

func TestSoftDeleteFile_WrongOwner(t *testing.T) {
	ctx := context.Background()
	s := store.New(testPool)
	owner := createTestUser(t)
	other := createTestUser(t)

	f, err := s.CreateFile(ctx, uuid.New(), owner.ID, "a.mp3", "mp3", 1, nil, "k1")
	if err != nil {
		t.Fatalf("CreateFile: %v", err)
	}

	_, err = s.SoftDeleteFile(ctx, f.ID, other.ID)
	if err != store.ErrNotFound {
		t.Fatalf("SoftDeleteFile error = %v, want %v", err, store.ErrNotFound)
	}

	found, err := s.GetFile(ctx, f.ID)
	if err != nil {
		t.Fatalf("GetFile: %v", err)
	}
	if found.DeletedAt != nil {
		t.Error("expected deleted_at to remain nil after unauthorized delete attempt")
	}
}

func TestSoftDeleteFile_AlreadyDeleted(t *testing.T) {
	ctx := context.Background()
	s := store.New(testPool)
	owner := createTestUser(t)

	f, err := s.CreateFile(ctx, uuid.New(), owner.ID, "a.mp3", "mp3", 1, nil, "k1")
	if err != nil {
		t.Fatalf("CreateFile: %v", err)
	}
	if _, err := s.SoftDeleteFile(ctx, f.ID, owner.ID); err != nil {
		t.Fatalf("first SoftDeleteFile: %v", err)
	}

	_, err = s.SoftDeleteFile(ctx, f.ID, owner.ID)
	if err != store.ErrNotFound {
		t.Fatalf("second SoftDeleteFile error = %v, want %v", err, store.ErrNotFound)
	}
}
