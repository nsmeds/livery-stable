package server

import "net/http"

// Routes returns a configured router
func (s *Server) Routes() *http.ServeMux {
	router := http.NewServeMux()
	router.HandleFunc("/", s.handleDefaultRequest())
	router.HandleFunc("/main", s.handleMain())

	return router
}
