package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/voidxo/voidxo/internal/platform/httpserver"
)

func newAuthHandler(t *testing.T) (*Handler, *authHarness) {
	t.Helper()
	h := newAuthHarness(t)
	return NewHandler(h.service, false, h.service.logger), h
}

func csrfPair(t *testing.T) (string, *http.Cookie) {
	t.Helper()
	token, err := NewCSRFToken()
	if err != nil {
		t.Fatal(err)
	}
	return token, &http.Cookie{Name: DevelopmentCSRFCookie, Value: token}
}

func TestCSRFHandlerSetsReadableCookie(t *testing.T) {
	handler, _ := newAuthHandler(t)
	recorder := httptest.NewRecorder()
	handler.GetCSRF(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/auth/csrf", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("got %d: %s", recorder.Code, recorder.Body.String())
	}
	cookies := recorder.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != DevelopmentCSRFCookie || cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteLaxMode {
		t.Fatalf("unexpected csrf cookie: %#v", cookies)
	}
	var body struct {
		Data struct {
			Token string `json:"csrf_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil || body.Data.Token == "" {
		t.Fatalf("unexpected csrf response: %s", recorder.Body.String())
	}
}

func TestLoginRequiresCSRFAndSetsSessionCookie(t *testing.T) {
	handler, _ := newAuthHandler(t)
	token, csrfCookie := csrfPair(t)

	missing := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"login":"alice","password":"correct-password"}`))
	handler.RequireCSRF(handler.Login)(missing, request)
	if missing.Code != http.StatusForbidden {
		t.Fatalf("missing csrf returned %d", missing.Code)
	}

	recorder := httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"login":"alice","password":"correct-password","device_id":"desktop"}`))
	request = request.WithContext(httpserver.WithClientIP(request.Context(), "203.0.113.9"))
	request.AddCookie(csrfCookie)
	request.Header.Set(CSRFHeader, token)
	handler.RequireCSRF(handler.Login)(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("login returned %d: %s", recorder.Code, recorder.Body.String())
	}
	cookies := recorder.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != DevelopmentSessionCookie || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteLaxMode {
		t.Fatalf("unexpected session cookie: %#v", cookies)
	}
	sessionCookie := cookies[0]

	me := httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	request.AddCookie(sessionCookie)
	handler.RequireAuth(handler.Me)(me, request)
	if me.Code != http.StatusOK || !strings.Contains(me.Body.String(), `"username":"alice"`) {
		t.Fatalf("me returned %d: %s", me.Code, me.Body.String())
	}

	logout := httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	request.AddCookie(sessionCookie)
	request.AddCookie(csrfCookie)
	request.Header.Set(CSRFHeader, token)
	handler.RequireAuth(handler.RequireCSRF(handler.Logout))(logout, request)
	if logout.Code != http.StatusOK {
		t.Fatalf("logout returned %d: %s", logout.Code, logout.Body.String())
	}
	cleared := logout.Result().Cookies()
	if len(cleared) == 0 || cleared[0].Name != DevelopmentSessionCookie || cleared[0].MaxAge != -1 {
		t.Fatalf("session cookie was not cleared: %#v", cleared)
	}
}
