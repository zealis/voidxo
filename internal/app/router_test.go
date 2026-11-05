package app

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/voidxo/voidxo/internal/platform/config"
)

type optionRepositoryStub struct{}

func (optionRepositoryStub) FindByNames(context.Context, []string) (map[string]string, error) {
	return map[string]string{"site_name": "VOIDXO", "question_can_edit": "false"}, nil
}

type pingerStub struct{ err error }

func (p pingerStub) PingContext(context.Context) error { return p.err }

func testRouter() http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewRouter(config.Config{AllowedOrigins: []string{"http://localhost:5173"}}, logger, Dependencies{
		Database: pingerStub{},
		Options:  optionRepositoryStub{},
	})
}

func TestHealthAndOptions(t *testing.T) {
	for _, path := range []string{"/health/live", "/health/ready", "/api/v1/options"} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, path, nil)
		testRouter().ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("%s returned %d: %s", path, recorder.Code, recorder.Body.String())
		}
		if recorder.Header().Get("X-Request-Id") == "" || recorder.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Fatalf("%s missing security headers", path)
		}
	}
}

func TestNotFoundUsesErrorEnvelope(t *testing.T) {
	recorder := httptest.NewRecorder()
	testRouter().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/missing", nil))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("got %d", recorder.Code)
	}
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil || body.Error.Code != "not_found" {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
}

func TestMethodNotAllowedUsesErrorEnvelope(t *testing.T) {
	recorder := httptest.NewRecorder()
	testRouter().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/options", strings.NewReader("{}")))
	if recorder.Code != http.StatusMethodNotAllowed || recorder.Header().Get("Allow") != http.MethodGet {
		t.Fatalf("got %d, allow=%q: %s", recorder.Code, recorder.Header().Get("Allow"), recorder.Body.String())
	}
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil || body.Error.Code != "method_not_allowed" {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
}

func TestReadyFailsWhenRedisIsUnavailable(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := NewRouter(config.Config{}, logger, Dependencies{
		Database:   pingerStub{},
		RedisCheck: func(context.Context) error { return errors.New("redis unavailable") },
		Options:    optionRepositoryStub{},
	})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/health/ready", nil))
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestReadyFailsWhenDatabaseIsUnavailable(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := NewRouter(config.Config{}, logger, Dependencies{
		Database: pingerStub{err: errors.New("connection refused")},
		Options:  optionRepositoryStub{},
	})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/health/ready", nil))
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestCORSAllowsOnlyConfiguredOrigin(t *testing.T) {
	for _, test := range []struct {
		origin string
		status int
	}{
		{"http://localhost:5173", http.StatusNoContent},
		{"https://malicious.example", http.StatusForbidden},
	} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodOptions, "/api/v1/options", strings.NewReader(""))
		request.Header.Set("Origin", test.origin)
		testRouter().ServeHTTP(recorder, request)
		if recorder.Code != test.status {
			t.Errorf("origin %s: got %d, want %d", test.origin, recorder.Code, test.status)
		}
	}
}
