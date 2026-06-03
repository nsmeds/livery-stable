package store_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nsmeds/livery-stable/db"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	connStr := os.Getenv("TEST_DATABASE_URL")
	if connStr == "" {
		connStr = "postgres://postgres:postgres@localhost:5433/livery_stable_test?sslmode=disable"
	}

	if err := db.Migrate(connStr); err != nil {
		fmt.Fprintf(os.Stderr, "store tests: migrate: %v\n", err)
		os.Exit(1)
	}

	pool, err := db.Connect(context.Background(), connStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "store tests: connect: %v\n", err)
		os.Exit(1)
	}
	testPool = pool

	code := m.Run()
	pool.Close()
	os.Exit(code)
}

func uniqueSuffix() string {
	return uuid.New().String()[:8]
}

func deleteUser(t *testing.T, id uuid.UUID) {
	t.Helper()
	_, err := testPool.Exec(context.Background(), "DELETE FROM users WHERE id = $1", id)
	if err != nil {
		t.Logf("deleteUser cleanup: %v", err)
	}
}
