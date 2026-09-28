package server

import (
	"context"
	"net"
	"net/http"
	"strings"
)

const clientIPKey contextKey = "client_ip"

func parseTrustedProxies(entries []string) ([]*net.IPNet, []string) {
	var nets []*net.IPNet
	var invalid []string
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if !strings.Contains(entry, "/") {
			if ip := net.ParseIP(entry); ip != nil {
				bits := 128
				if ip.To4() != nil {
					bits = 32
				}
				nets = append(nets, &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)})
				continue
			}
			invalid = append(invalid, entry)
			continue
		}
		_, ipNet, err := net.ParseCIDR(entry)
		if err != nil {
			invalid = append(invalid, entry)
			continue
		}
		nets = append(nets, ipNet)
	}
	return nets, invalid
}

func isTrusted(trusted []*net.IPNet, ip net.IP) bool {
	if ip == nil {
		return false
	}
	for _, n := range trusted {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

func remoteHost(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func resolveClientIP(trusted []*net.IPNet, r *http.Request) string {
	remote := remoteHost(r)
	if !isTrusted(trusted, net.ParseIP(remote)) {
		return remote
	}

	var hops []string
	for _, header := range r.Header.Values("X-Forwarded-For") {
		hops = append(hops, strings.Split(header, ",")...)
	}
	for i := len(hops) - 1; i >= 0; i-- {
		hop := strings.TrimSpace(hops[i])
		ip := net.ParseIP(hop)
		if ip == nil {
			return remote
		}
		if !isTrusted(trusted, ip) {
			return ip.String()
		}
	}
	return remote
}

func (s *Server) clientIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := resolveClientIP(s.trustedProxies, r)
		ctx := context.WithValue(r.Context(), clientIPKey, ip)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func getRealIP(r *http.Request) string {
	if ip, ok := r.Context().Value(clientIPKey).(string); ok && ip != "" {
		return ip
	}
	return remoteHost(r)
}
