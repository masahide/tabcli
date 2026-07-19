package mcpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStressRotatedTokensAreAlwaysRejected(t *testing.T) {
	const host = "127.0.0.1:43123"
	handler := SecurityMiddleware("new-token", host, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	for index := 0; index < 1000; index++ {
		request := httptest.NewRequest(http.MethodPost, "http://"+host+"/mcp", nil)
		request.Host = host
		request.Header.Set("Authorization", "Bearer old-token")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("request %d status=%d", index, response.Code)
		}
	}
}
