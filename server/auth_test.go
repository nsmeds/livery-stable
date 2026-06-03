package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRegister(t *testing.T) {
	srv := newTestServer(t)
	email := uniqueEmail()
	t.Cleanup(func() { deleteUserByEmail(t, email) })

	body, _ := json.Marshal(map[string]string{"email": email, "password": "password123"})
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusCreated, w.Body.String())
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	srv := newTestServer(t)
	email := uniqueEmail()
	t.Cleanup(func() { deleteUserByEmail(t, email) })

	body, _ := json.Marshal(map[string]string{"email": email, "password": "password123"})

	for i, wantStatus := range []int{http.StatusCreated, http.StatusConflict} {
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		srv.Handler.ServeHTTP(w, req)
		if w.Code != wantStatus {
			t.Errorf("request %d: status = %d, want %d", i+1, w.Code, wantStatus)
		}
	}
}

func TestRegister_MissingFields(t *testing.T) {
	srv := newTestServer(t)

	cases := []struct {
		body string
	}{
		{`{"email":"","password":"pass"}`},
		{`{"email":"a@b.com","password":""}`},
		{`{}`},
	}

	for _, tc := range cases {
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString(tc.body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		srv.Handler.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("body %q: status = %d, want %d", tc.body, w.Code, http.StatusBadRequest)
		}
	}
}

func TestLogin(t *testing.T) {
	srv := newTestServer(t)
	email := uniqueEmail()
	t.Cleanup(func() { deleteUserByEmail(t, email) })

	cookie := registerAndLogin(t, srv, email, "password123")

	if cookie.HttpOnly != true {
		t.Error("session cookie should be HttpOnly")
	}
	if cookie.Value == "" {
		t.Error("session cookie value should not be empty")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	srv := newTestServer(t)
	email := uniqueEmail()
	t.Cleanup(func() { deleteUserByEmail(t, email) })

	// register
	body, _ := json.Marshal(map[string]string{"email": email, "password": "correctpassword"})
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	srv.Handler.ServeHTTP(httptest.NewRecorder(), req)

	// login with wrong password
	body, _ = json.Marshal(map[string]string{"email": email, "password": "wrongpassword"})
	req = httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestLogin_UnknownEmail(t *testing.T) {
	srv := newTestServer(t)

	body, _ := json.Marshal(map[string]string{"email": "nobody@example.com", "password": "pass"})
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestLogout(t *testing.T) {
	srv := newTestServer(t)
	email := uniqueEmail()
	t.Cleanup(func() { deleteUserByEmail(t, email) })

	cookie := registerAndLogin(t, srv, email, "password123")

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	srv.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusNoContent, w.Body.String())
	}
}

func TestLogout_Unauthenticated(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	w := httptest.NewRecorder()
	srv.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestMe(t *testing.T) {
	srv := newTestServer(t)
	email := uniqueEmail()
	t.Cleanup(func() { deleteUserByEmail(t, email) })

	cookie := registerAndLogin(t, srv, email, "password123")

	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	srv.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["email"] != email {
		t.Errorf("email = %q, want %q", resp["email"], email)
	}
}

func TestMe_Unauthenticated(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	w := httptest.NewRecorder()
	srv.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}
