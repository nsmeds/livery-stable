package server

import "net/http"

// Routes returns a configured router
func (s *Server) Routes() *http.ServeMux {
	router := http.NewServeMux()

	router.HandleFunc("/", s.handleDefaultRequest())

	// Unauthenticated auth routes
	router.HandleFunc("POST /auth/register", s.handleRegister())
	router.HandleFunc("POST /auth/login", s.handleLogin())

	// Protected auth routes
	router.Handle("POST /auth/logout", s.requireAuth(s.handleLogout()))
	router.Handle("GET /auth/me", s.requireAuth(s.handleMe()))

	// Protected app routes
	router.Handle("GET /main", s.requireAuth(s.handleMain()))

	// Protected file routes
	router.Handle("POST /files", s.requireAuth(s.handleUploadFile()))
	router.Handle("GET /files", s.requireAuth(s.handleListFiles()))
	router.Handle("DELETE /files/{id}", s.requireAuth(s.handleDeleteFile()))

	return router
}
