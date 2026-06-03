package store_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/nsmeds/livery-stable/store"
)

func TestCreateUser(t *testing.T) {
	ctx := context.Background()
	s := store.New(testPool)

	email := fmt.Sprintf("create-%s@example.com", uniqueSuffix())
	user, err := s.CreateUser(ctx, email, "hashed", store.RoleMember)
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	t.Cleanup(func() { deleteUser(t, user.ID) })

	if user.Email != email {
		t.Errorf("email = %q, want %q", user.Email, email)
	}
	if user.Role != string(store.RoleMember) {
		t.Errorf("role = %q, want %q", user.Role, store.RoleMember)
	}
	if user.ID == (uuid.UUID{}) {
		t.Error("expected non-nil ID")
	}
}

func TestCreateUser_AdminRole(t *testing.T) {
	ctx := context.Background()
	s := store.New(testPool)

	email := fmt.Sprintf("admin-%s@example.com", uniqueSuffix())
	user, err := s.CreateUser(ctx, email, "hashed", store.RoleAdmin)
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	t.Cleanup(func() { deleteUser(t, user.ID) })

	if user.Role != string(store.RoleAdmin) {
		t.Errorf("role = %q, want %q", user.Role, store.RoleAdmin)
	}
}

func TestCreateUser_DuplicateEmail(t *testing.T) {
	ctx := context.Background()
	s := store.New(testPool)

	email := fmt.Sprintf("dup-%s@example.com", uniqueSuffix())
	user, err := s.CreateUser(ctx, email, "hashed", store.RoleMember)
	if err != nil {
		t.Fatalf("first CreateUser: %v", err)
	}
	t.Cleanup(func() { deleteUser(t, user.ID) })

	_, err = s.CreateUser(ctx, email, "hashed", store.RoleMember)
	if err == nil {
		t.Fatal("expected error for duplicate email, got nil")
	}
}

func TestGetUserByEmail(t *testing.T) {
	ctx := context.Background()
	s := store.New(testPool)

	email := fmt.Sprintf("getemail-%s@example.com", uniqueSuffix())
	created, err := s.CreateUser(ctx, email, "hashed", store.RoleMember)
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	t.Cleanup(func() { deleteUser(t, created.ID) })

	found, err := s.GetUserByEmail(ctx, email)
	if err != nil {
		t.Fatalf("GetUserByEmail: %v", err)
	}
	if found.ID != created.ID {
		t.Errorf("id = %v, want %v", found.ID, created.ID)
	}
}

func TestGetUserByEmail_NotFound(t *testing.T) {
	ctx := context.Background()
	s := store.New(testPool)

	_, err := s.GetUserByEmail(ctx, "notexist@example.com")
	if err == nil {
		t.Fatal("expected error for missing user, got nil")
	}
}

func TestGetUserByID(t *testing.T) {
	ctx := context.Background()
	s := store.New(testPool)

	email := fmt.Sprintf("getid-%s@example.com", uniqueSuffix())
	created, err := s.CreateUser(ctx, email, "hashed", store.RoleMember)
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	t.Cleanup(func() { deleteUser(t, created.ID) })

	found, err := s.GetUserByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
	if found.Email != email {
		t.Errorf("email = %q, want %q", found.Email, email)
	}
}
