package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/nsmeds/livery-stable/audio"
	"github.com/nsmeds/livery-stable/store"
)

// maxUploadSizeBytes is the file size limit from the roadmap's file
// constraints (accommodates a 45-minute 24-bit/96kHz stereo WAV).
const maxUploadSizeBytes = 2 * 1024 * 1024 * 1024

// maxMultipartMemory is how much of the multipart form Go buffers in memory
// before spilling remaining parts (i.e. the uploaded file) to a temp file on
// disk - which is also what makes the file seekable for duration parsing.
const maxMultipartMemory = 32 << 20 // 32MB

type fileResponse struct {
	ID              string   `json:"id"`
	Filename        string   `json:"filename"`
	Format          string   `json:"format"`
	SizeBytes       int64    `json:"size_bytes"`
	DurationSeconds *float32 `json:"duration_seconds,omitempty"`
	UploadedAt      string   `json:"uploaded_at"`
}

func newFileResponse(f *store.File) fileResponse {
	return fileResponse{
		ID:              f.ID.String(),
		Filename:        f.Filename,
		Format:          f.Format,
		SizeBytes:       f.SizeBytes,
		DurationSeconds: f.DurationSeconds,
		UploadedAt:      f.UploadedAt.Format(time.RFC3339),
	}
}

func (s *Server) handleUploadFile() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := userIDFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, maxUploadSizeBytes)
		if err := r.ParseMultipartForm(maxMultipartMemory); err != nil {
			writeError(w, http.StatusBadRequest, "file too large or invalid multipart form")
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			writeError(w, http.StatusBadRequest, "missing file field")
			return
		}
		defer file.Close()

		magic := make([]byte, 12)
		if _, err := io.ReadFull(file, magic); err != nil {
			writeError(w, http.StatusBadRequest, "file too short to be a valid audio file")
			return
		}
		format, err := audio.DetectFormat(magic)
		if err != nil {
			writeError(w, http.StatusUnprocessableEntity, "unsupported or unrecognized audio format")
			return
		}

		var duration *float32
		if d, err := audio.Duration(format, file, header.Size); err != nil {
			log.Printf("upload: could not compute duration for %q: %v", header.Filename, err)
		} else {
			seconds := float32(d.Seconds())
			duration = &seconds
		}

		if _, err := file.Seek(0, io.SeekStart); err != nil {
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}

		id := uuid.New()
		storageKey := fmt.Sprintf("%s/%s.%s", userID, id, format)
		if err := s.config.Storage.Put(r.Context(), storageKey, file); err != nil {
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}

		f, err := s.store.CreateFile(r.Context(), id, userID, header.Filename, string(format), header.Size, duration, storageKey)
		if err != nil {
			if delErr := s.config.Storage.Delete(r.Context(), storageKey); delErr != nil {
				log.Printf("upload: cleanup after failed CreateFile: %v", delErr)
			}
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(newFileResponse(f))
	}
}

func (s *Server) handleListFiles() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := userIDFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		files, err := s.store.ListFilesByOwner(r.Context(), userID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}

		resp := make([]fileResponse, len(files))
		for i := range files {
			resp[i] = newFileResponse(&files[i])
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

func (s *Server) handleDeleteFile() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := userIDFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		id, err := uuid.Parse(r.PathValue("id"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid file id")
			return
		}

		storageKey, err := s.store.SoftDeleteFile(r.Context(), id, userID)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				writeError(w, http.StatusNotFound, "file not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}

		s.deleter.Enqueue(storageKey)
		w.WriteHeader(http.StatusNoContent)
	}
}
