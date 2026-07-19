package mcpserver

import (
	"crypto/subtle"
	"errors"
	"net"
	"net/http"
	"strings"
)

var ErrNonLoopbackAddress = errors.New("MCP server must listen on an explicit loopback address")

func ValidateListenAddress(address string) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return ErrNonLoopbackAddress
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return ErrNonLoopbackAddress
	}
	return nil
}

func SecurityMiddleware(token, allowedHost string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Origin") != "" {
			http.Error(response, "origin is not allowed", http.StatusForbidden)
			return
		}
		if request.Host != allowedHost {
			http.Error(response, "host is not allowed", http.StatusForbidden)
			return
		}
		provided := strings.TrimPrefix(request.Header.Get("Authorization"), "Bearer ")
		validPrefix := strings.HasPrefix(request.Header.Get("Authorization"), "Bearer ")
		if !validPrefix || len(provided) != len(token) || subtle.ConstantTimeCompare([]byte(provided), []byte(token)) != 1 {
			response.Header().Set("WWW-Authenticate", "Bearer")
			http.Error(response, "authentication required", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(response, request)
	})
}
