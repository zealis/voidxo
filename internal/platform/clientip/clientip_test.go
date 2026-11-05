package clientip

import (
	"net/http/httptest"
	"testing"
)

func TestResolverHonorsForwardedForOnlyFromTrustedProxy(t *testing.T) {
	resolver, err := New([]string{"127.0.0.1/32"})
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest("GET", "/", nil)
	request.RemoteAddr = "203.0.113.8:54321"
	request.Header.Set("X-Forwarded-For", "198.51.100.7")
	if got := resolver.FromRequest(request); got != "203.0.113.8" {
		t.Fatalf("untrusted proxy spoofed client IP: %s", got)
	}

	request.RemoteAddr = "127.0.0.1:54321"
	request.Header.Set("X-Forwarded-For", "198.51.100.7, 127.0.0.1")
	if got := resolver.FromRequest(request); got != "198.51.100.7" {
		t.Fatalf("trusted proxy chain resolved to %s", got)
	}
}
