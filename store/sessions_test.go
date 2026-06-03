package store_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/nsmeds/livery-stable/store"
)

func createTestUser(t *testing.T) *store.User {
	t.Helper()
	s := store.New(testPool)
	email := fmt.Sprintf("sess-%s@example.com", uniqueSuffix())
	user, err := s.CreateUser(context.Background(), email, "hashed", store.RoleMember)
	if err != nil {
		t.Fatalf("createTestUser: %v", err)
	}
	t.Cleanup(func() { deleteUser(t, user.ID) })
	return user
}

func TestCreateSession(t *testing.T) {
	ctx := context.Background()
	s := store.New(testPool)
	user := createTestUser(t)

	id := uuid.New()
	expiresAt := time.Now().Add(24 * time.Hour)

	sess, err := s.CreateSession(ctx, id, user.ID, expiresAt)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	if sess.ID != id {
		t.Errorf("id = %v, want %v", sess.ID, id)
	}
	if sess.UserID != user.ID {
		t.Errorf("user_id = %v, want %v", sess.UserID, user.ID)
	}
	if sess.RevokedAt != nil {
		t.Error("expected revoked_at to be nil")
	}
}

func TestGetSession(t *testing.T) {
	ctx := context.Background()
	s := store.New(testPool)
	user := createTestUser(t)

	id := uuid.New()
	_, err := s.CreateSession(ctx, id, user.ID, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	sess, err := s.GetSession(ctx, id)
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if sess.ID != id {
		t.Errorf("id = %v, want %v", sess.ID, id)
	}
}

func TestGetSession_NotFound(t *testing.T) {
	ctx := context.Background()
	s := store.New(testPool)

	_, err := s.GetSession(ctx, uuid.New())
	if err == nil {
		t.Fatal("expected error for missing session, got nil")
	}
}

func TestRevokeSession(t *testing.T) {
	ctx := context.Background()
	s := store.New(testPool)
	user := createTestUser(t)

	id := uuid.New()
	if _, err := s.CreateSession(ctx, id, user.ID, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	if err := s.RevokeSession(ctx, id); err != nil {
		t.Fatalf("RevokeSession: %v", err)
	}

	sess, err := s.GetSession(ctx, id)
	if err != nil {
		t.Fatalf("GetSession after revoke: %v", err)
	}
	if sess.RevokedAt == nil {
		t.Error("expected revoked_at to be set")
	}
}
