package config

import (
	"testing"
	"time"
)

func TestValidate(t *testing.T) {
	valid := Config{
		Environment:        "development",
		DatabaseDSN:        "user:pass@tcp(localhost:3306)/voidxo",
		RedisAddr:          "127.0.0.1:6379",
		SessionIdleTTL:     time.Hour,
		SessionAbsoluteTTL: 24 * time.Hour,
		ShutdownTimeout:    10 * time.Second,
		AllowedOrigins:     []string{"http://localhost:5173"},
		TrustedProxies:     []string{"127.0.0.1/32"},
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}

	tests := []Config{
		{},
		{DatabaseDSN: "dsn", ShutdownTimeout: time.Second, AllowedOrigins: []string{"*"}},
		{DatabaseDSN: "dsn", ShutdownTimeout: time.Second, AllowedOrigins: []string{"javascript:alert(1)"}},
		{DatabaseDSN: "dsn", ShutdownTimeout: time.Second, AllowedOrigins: []string{"http://localhost:5173/path"}},
		{DatabaseDSN: "dsn"},
		{DatabaseDSN: "dsn", RedisAddr: "localhost:6379", SessionIdleTTL: time.Hour, SessionAbsoluteTTL: time.Minute, ShutdownTimeout: time.Second},
		{DatabaseDSN: "dsn", RedisAddr: "localhost:6379", SessionIdleTTL: time.Hour, SessionAbsoluteTTL: time.Hour, ShutdownTimeout: time.Second, TrustedProxies: []string{"not-a-cidr"}},
		{DatabaseDSN: "dsn", RedisAddr: "localhost:6379", SessionIdleTTL: time.Hour, SessionAbsoluteTTL: time.Hour, ShutdownTimeout: time.Second, Environment: "production", CookieSecure: false},
	}
	for i, cfg := range tests {
		if err := cfg.Validate(); err == nil {
			t.Errorf("case %d: expected validation error", i)
		}
	}
}

func TestLoadRejectsInvalidDuration(t *testing.T) {
	t.Setenv("VOIDXO_SHUTDOWN_TIMEOUT", "soon")
	if _, err := Load(); err == nil {
		t.Fatal("expected invalid duration error")
	}
}

func TestLoadDefaultsToSecureCookieInProduction(t *testing.T) {
	t.Setenv("VOIDXO_ENV", "production")
	t.Setenv("VOIDXO_DATABASE_DSN", "dsn")
	t.Setenv("VOIDXO_COOKIE_SECURE", "")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.CookieSecure {
		t.Fatal("production must default to secure cookies")
	}
}
