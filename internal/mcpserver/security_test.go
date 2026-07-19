package mcpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecurityMiddleware(t *testing.T) {
	const (
		token = "secret-token"
		host  = "127.0.0.1:43123"
	)
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	handler := SecurityMiddleware(token, host, next)

	tests := []struct {
		name       string
		authorize  string
		host       string
		origin     string
		wantStatus int
	}{
		{name: "valid", authorize: "Bearer " + token, host: host, wantStatus: http.StatusNoContent},
		{name: "missing bearer", host: host, wantStatus: http.StatusUnauthorized},
		{name: "wrong bearer", authorize: "Bearer wrong", host: host, wantStatus: http.StatusUnauthorized},
		{name: "host mismatch", authorize: "Bearer " + token, host: "localhost:43123", wantStatus: http.StatusForbidden},
		{name: "origin present", authorize: "Bearer " + token, host: host, origin: "https://example.com", wantStatus: http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "http://"+host+"/mcp", nil)
			req.Host = tt.host
			if tt.authorize != "" {
				req.Header.Set("Authorization", tt.authorize)
			}
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, req)
			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, tt.wantStatus)
			}
			if got := response.Header().Get("Access-Control-Allow-Origin"); got != "" {
				t.Fatalf("CORS header must be absent, got %q", got)
			}
		})
	}
}

func TestValidateListenAddressRejectsNonLoopback(t *testing.T) {
	for _, address := range []string{"0.0.0.0:0", "192.0.2.1:1234", ":0"} {
		t.Run(address, func(t *testing.T) {
			if err := ValidateListenAddress(address); err == nil {
				t.Fatalf("ValidateListenAddress(%q) succeeded, want rejection", address)
			}
		})
	}
	if err := ValidateListenAddress("127.0.0.1:0"); err != nil {
		t.Fatalf("ValidateListenAddress(loopback) error = %v", err)
	}
}
