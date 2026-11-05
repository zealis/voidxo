package clientip

import (
	"fmt"
	"net"
	"net/http"
	"strings"
)

type Resolver struct {
	trusted []*net.IPNet
}

func New(trustedCIDRs []string) (*Resolver, error) {
	resolver := &Resolver{}
	for _, cidr := range trustedCIDRs {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			return nil, fmt.Errorf("parse trusted proxy %q: %w", cidr, err)
		}
		resolver.trusted = append(resolver.trusted, network)
	}
	return resolver, nil
}

func (r *Resolver) FromRequest(request *http.Request) string {
	remote := host(request.RemoteAddr)
	if remote == "" {
		remote = request.RemoteAddr
	}
	if !r.isTrusted(remote) {
		return remote
	}
	forwarded := strings.Split(request.Header.Get("X-Forwarded-For"), ",")
	for i := len(forwarded) - 1; i >= 0; i-- {
		candidate := strings.TrimSpace(forwarded[i])
		if parsed := net.ParseIP(candidate); parsed != nil && !r.isTrusted(candidate) {
			return parsed.String()
		}
	}
	return remote
}

func (r *Resolver) isTrusted(value string) bool {
	parsed := net.ParseIP(value)
	if parsed == nil {
		return false
	}
	for _, network := range r.trusted {
		if network.Contains(parsed) {
			return true
		}
	}
	return false
}

func host(address string) string {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return strings.Trim(address, "[]")
	}
	return host
}
