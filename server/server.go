package server

import (
	"fmt"
	"net/http"

	"github.com/nsmeds/livery-stable/store"
)

type Server struct {
	*http.Server
	store  *store.Store
	config Config
}

func New(host string, port int, st *store.Store, cfg Config) *Server {
	s := Server{store: st, config: cfg}
	httpServer := http.Server{
		Addr:    fmt.Sprintf("%s:%d", host, port),
		Handler: s.Routes(),
	}
	s.Server = &httpServer
	return &s
}

func (s *Server) handleDefaultRequest() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"not found"}`))
	}
}

func (s *Server) handleMain() http.HandlerFunc {
	// TODO implement template
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	}
}

type audioFile struct {
	title string
	data  []byte
}

type fileMap []audioFile

func getAllFiles() (*fileMap, error) {
	var f fileMap
	// TODO abstraction over storage layer
	return &f, nil
}
