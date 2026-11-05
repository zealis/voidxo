package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
)

const (
	CSRFHeader               = "X-CSRF-Token"
	SecureSessionCookie      = "__Host-voidxo_session"
	DevelopmentSessionCookie = "voidxo_session"
	SecureCSRFCookie         = "__Host-voidxo_csrf"
	DevelopmentCSRFCookie    = "voidxo_csrf"
)

func SessionCookieName(secure bool) string {
	if secure {
		return SecureSessionCookie
	}
	return DevelopmentSessionCookie
}

func CSRFCookieName(secure bool) string {
	if secure {
		return SecureCSRFCookie
	}
	return DevelopmentCSRFCookie
}

func NewCSRFToken() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func ValidCSRF(cookieValue, headerValue string) bool {
	if len(cookieValue) < 32 || len(headerValue) != len(cookieValue) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(cookieValue), []byte(headerValue)) == 1
}
