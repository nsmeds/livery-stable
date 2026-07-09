package server_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nsmeds/livery-stable/server"
)

func uploadHelper(t *testing.T, srv *server.Server, cookie *http.Cookie, filename string, content []byte) *httptest.ResponseRecorder {
	t.Helper()

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/files", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, req)
	return rec
}

func wavFixture(dataSize uint32) []byte {
	var buf bytes.Buffer
	buf.WriteString("RIFF")
	writeUint32LE(&buf, 36+dataSize)
	buf.WriteString("WAVE")
	buf.WriteString("fmt ")
	writeUint32LE(&buf, 16)
	writeUint16LE(&buf, 1)     // PCM
	writeUint16LE(&buf, 1)     // mono
	writeUint32LE(&buf, 44100) // sample rate
	writeUint32LE(&buf, 88200) // byte rate
	writeUint16LE(&buf, 2)     // block align
	writeUint16LE(&buf, 16)    // bits per sample
	buf.WriteString("data")
	writeUint32LE(&buf, dataSize)
	buf.Write(make([]byte, dataSize))
	return buf.Bytes()
}

func writeUint32LE(buf *bytes.Buffer, v uint32) {
	buf.WriteByte(byte(v))
	buf.WriteByte(byte(v >> 8))
	buf.WriteByte(byte(v >> 16))
	buf.WriteByte(byte(v >> 24))
}

func writeUint16LE(buf *bytes.Buffer, v uint16) {
	buf.WriteByte(byte(v))
	buf.WriteByte(byte(v >> 8))
}

func TestUploadFile(t *testing.T) {
	srv := newTestServer(t)
	email := uniqueEmail()
	t.Cleanup(func() { deleteUserByEmail(t, email) })
	cookie := registerAndLogin(t, srv, email, "password123")

	rec := uploadHelper(t, srv, cookie, "song.wav", wavFixture(88200))
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["filename"] != "song.wav" {
		t.Errorf("filename = %v, want %q", resp["filename"], "song.wav")
	}
	if resp["format"] != "wav" {
		t.Errorf("format = %v, want %q", resp["format"], "wav")
	}
	if resp["duration_seconds"] == nil {
		t.Error("expected duration_seconds to be set")
	}
}

func TestUploadFile_UnrecognizedFormat(t *testing.T) {
	srv := newTestServer(t)
	email := uniqueEmail()
	t.Cleanup(func() { deleteUserByEmail(t, email) })
	cookie := registerAndLogin(t, srv, email, "password123")

	rec := uploadHelper(t, srv, cookie, "notaudio.txt", []byte("this is definitely not audio"))
	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d, body = %s", rec.Code, http.StatusUnprocessableEntity, rec.Body.String())
	}
}

func TestUploadFile_Unauthenticated(t *testing.T) {
	srv := newTestServer(t)

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	w.Close()

	req := httptest.NewRequest(http.MethodPost, "/files", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestListFiles_ScopedToOwner(t *testing.T) {
	srv := newTestServer(t)
	email1 := uniqueEmail()
	email2 := uniqueEmail()
	t.Cleanup(func() { deleteUserByEmail(t, email1) })
	t.Cleanup(func() { deleteUserByEmail(t, email2) })

	cookie1 := registerAndLogin(t, srv, email1, "password123")
	cookie2 := registerAndLogin(t, srv, email2, "password123")

	uploadHelper(t, srv, cookie1, "mine.wav", wavFixture(4410))

	req := httptest.NewRequest(http.MethodGet, "/files", nil)
	req.AddCookie(cookie2)
	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var files []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &files); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("len(files) = %d, want 0 (other user's files should not be listed)", len(files))
	}
}

func TestDeleteFile(t *testing.T) {
	srv := newTestServer(t)
	email := uniqueEmail()
	t.Cleanup(func() { deleteUserByEmail(t, email) })
	cookie := registerAndLogin(t, srv, email, "password123")

	uploadRec := uploadHelper(t, srv, cookie, "song.wav", wavFixture(4410))
	var uploaded map[string]any
	if err := json.Unmarshal(uploadRec.Body.Bytes(), &uploaded); err != nil {
		t.Fatalf("unmarshal upload response: %v", err)
	}
	id := uploaded["id"].(string)

	delReq := httptest.NewRequest(http.MethodDelete, "/files/"+id, nil)
	delReq.AddCookie(cookie)
	delRec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(delRec, delReq)
	if delRec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want %d, body = %s", delRec.Code, http.StatusNoContent, delRec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/files", nil)
	listReq.AddCookie(cookie)
	listRec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(listRec, listReq)
	var files []map[string]any
	if err := json.Unmarshal(listRec.Body.Bytes(), &files); err != nil {
		t.Fatalf("unmarshal list response: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("len(files) = %d, want 0 after delete", len(files))
	}
}

func TestDeleteFile_WrongOwner(t *testing.T) {
	srv := newTestServer(t)
	email1 := uniqueEmail()
	email2 := uniqueEmail()
	t.Cleanup(func() { deleteUserByEmail(t, email1) })
	t.Cleanup(func() { deleteUserByEmail(t, email2) })

	cookie1 := registerAndLogin(t, srv, email1, "password123")
	cookie2 := registerAndLogin(t, srv, email2, "password123")

	uploadRec := uploadHelper(t, srv, cookie1, "song.wav", wavFixture(4410))
	var uploaded map[string]any
	if err := json.Unmarshal(uploadRec.Body.Bytes(), &uploaded); err != nil {
		t.Fatalf("unmarshal upload response: %v", err)
	}
	id := uploaded["id"].(string)

	delReq := httptest.NewRequest(http.MethodDelete, "/files/"+id, nil)
	delReq.AddCookie(cookie2)
	delRec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(delRec, delReq)
	if delRec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", delRec.Code, http.StatusNotFound)
	}
}
