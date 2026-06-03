package store

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	CreatedAt time.Time
	ExpiresAt time.Time
	RevokedAt *time.Time
}

func (s *Store) CreateSession(ctx context.Context, id, userID uuid.UUID, expiresAt time.Time) (*Session, error) {
	var sess Session
	err := s.pool.QueryRow(ctx,
		`INSERT INTO sessions (id, user_id, expires_at)
		 VALUES ($1, $2, $3)
		 RETURNING id, user_id, created_at, expires_at, revoked_at`,
		id, userID, expiresAt,
	).Scan(&sess.ID, &sess.UserID, &sess.CreatedAt, &sess.ExpiresAt, &sess.RevokedAt)
	if err != nil {
		return nil, err
	}
	return &sess, nil
}

func (s *Store) GetSession(ctx context.Context, id uuid.UUID) (*Session, error) {
	var sess Session
	err := s.pool.QueryRow(ctx,
		`SELECT id, user_id, created_at, expires_at, revoked_at FROM sessions WHERE id = $1`,
		id,
	).Scan(&sess.ID, &sess.UserID, &sess.CreatedAt, &sess.ExpiresAt, &sess.RevokedAt)
	if err != nil {
		return nil, err
	}
	return &sess, nil
}

func (s *Store) RevokeSession(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE sessions SET revoked_at = NOW() WHERE id = $1`,
		id,
	)
	return err
}
