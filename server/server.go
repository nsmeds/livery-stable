package server

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

type Server struct {
	*http.Server
	// TODO logger
	// TODO metrics
}

func New(host string, port int) *Server {
	s := Server{}
	httpServer := http.Server{
		Addr:    fmt.Sprintf("%s:%d", host, port),
		Handler: s.Routes(),
	}
	s.Server = &httpServer
	return &s
}

func (s *Server) handleDefaultRequest() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		processStartedAt := time.Now().Format(time.RFC3339Nano)
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.Header().Set("content-type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"message": "internal system error"}`))
			return
		}
		message := fmt.Sprintf("received %v at %s", string(body), processStartedAt)
		w.Header().Set("content-type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(message))
	}
}

func (s *Server) handleMain() http.HandlerFunc {
	// TODO check method - only GET is allowed
	// TODO implement template
	return func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("content-type", "application/json")
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
	// TODO abstraction ov 
	return &f, nil
}
