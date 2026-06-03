package server

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type contextKey string

const (
	contextKeyUserID contextKey = "user_id"
	contextKeyJTI    contextKey = "jti"
)

func userIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(contextKeyUserID).(uuid.UUID)
	return id, ok
}

func jtiFromContext(ctx context.Context) (uuid.UUID, bool) {
	jti, ok := ctx.Value(contextKeyJTI).(uuid.UUID)
	return jti, ok
}

func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		c, err := parseToken(cookie.Value, s.config.JWTSecret)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		if c.ExpiresAt != nil && c.ExpiresAt.Before(time.Now()) {
			writeError(w, http.StatusUnauthorized, "token expired")
			return
		}

		jti, err := uuid.Parse(c.ID)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		session, err := s.store.GetSession(r.Context(), jti)
		if err != nil || session.RevokedAt != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		userID, err := uuid.Parse(c.Subject)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		ctx := context.WithValue(r.Context(), contextKeyUserID, userID)
		ctx = context.WithValue(ctx, contextKeyJTI, jti)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
