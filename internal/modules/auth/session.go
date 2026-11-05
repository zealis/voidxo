package auth

import (
	"context"
	"errors"
	"time"
)

var (
	ErrSessionNotFound         = errors.New("session not found")
	ErrSessionStoreUnavailable = errors.New("session store unavailable")
)

type Session struct {
	UserID       int64     `json:"user_id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	Roles        []string  `json:"roles"`
	Permissions  []string  `json:"permissions"`
	DeviceID     string    `json:"device_id"`
	CreatedAt    time.Time `json:"created_at"`
	LastActiveAt time.Time `json:"last_active_at"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type SessionStore interface {
	Create(ctx context.Context, tokenHash string, session Session, ttl time.Duration) error
	Get(ctx context.Context, tokenHash string) (Session, error)
	Touch(ctx context.Context, tokenHash string, ttl time.Duration) error
	Delete(ctx context.Context, tokenHash string) error
}

type SessionRecord struct {
	UserID    int64
	TokenHash string
	DeviceID  string
	IPAddress string
	UserAgent string
	CreatedAt time.Time
	ExpiresAt time.Time
}

type SessionRepository interface {
	Create(ctx context.Context, record SessionRecord) error
	ActiveHashes(ctx context.Context, userID int64) ([]string, error)
	Revoke(ctx context.Context, tokenHash, reason string) error
	RevokeAll(ctx context.Context, userID int64, reason string) ([]string, error)
}
