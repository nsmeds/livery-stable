package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nsmeds/livery-stable/db"
	"github.com/nsmeds/livery-stable/server"
	"github.com/nsmeds/livery-stable/storage"
	"github.com/nsmeds/livery-stable/store"
)

var testPool *pgxpool.Pool
var testJWTSecret = []byte("test-jwt-secret-for-testing-only")

func TestMain(m *testing.M) {
	connStr := os.Getenv("TEST_DATABASE_URL")
	if connStr == "" {
		connStr = "postgres://postgres:postgres@localhost:5433/livery_stable_test?sslmode=disable"
	}

	if err := db.Migrate(connStr); err != nil {
		fmt.Fprintf(os.Stderr, "server tests: migrate: %v\n", err)
		os.Exit(1)
	}

	pool, err := db.Connect(context.Background(), connStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "server tests: connect: %v\n", err)
		os.Exit(1)
	}
	testPool = pool

	code := m.Run()
	pool.Close()
	os.Exit(code)
}

func newTestServer(t *testing.T) *server.Server {
	t.Helper()
	st := store.New(testPool)
	fsStore := storage.NewFilesystemStore(t.TempDir())
	deleter := server.NewDeleter(fsStore)
	cfg := server.Config{JWTSecret: testJWTSecret, Storage: fsStore}
	return server.New("localhost", 0, st, deleter, cfg)
}

func registerAndLogin(t *testing.T, srv *server.Server, email, password string) *http.Cookie {
	t.Helper()

	// register
	body, _ := json.Marshal(map[string]string{"email": email, "password": password})
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("register: status = %d, body = %s", w.Code, w.Body.String())
	}

	// login
	req = httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	srv.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("login: status = %d, body = %s", w.Code, w.Body.String())
	}

	var cookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == "session" {
			cookie = c
			break
		}
	}
	if cookie == nil {
		t.Fatal("no session cookie in login response")
	}
	return cookie
}

func deleteUserByEmail(t *testing.T, email string) {
	t.Helper()
	_, err := testPool.Exec(context.Background(), "DELETE FROM users WHERE email = $1", email)
	if err != nil {
		t.Logf("deleteUserByEmail cleanup: %v", err)
	}
}

func uniqueEmail() string {
	return fmt.Sprintf("test-%s@example.com", uuid.New().String()[:8])
}
