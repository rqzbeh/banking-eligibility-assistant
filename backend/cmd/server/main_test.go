package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCORSMiddlewareAllowsCustomerWrites(t *testing.T) {
	req := httptest.NewRequest(http.MethodOptions, "/api/rbci/customers/1234567890", nil)
	w := httptest.NewRecorder()

	corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected preflight 204, got %d", w.Code)
	}
	methods := w.Header().Get("Access-Control-Allow-Methods")
	for _, method := range []string{http.MethodPut, http.MethodDelete} {
		if !strings.Contains(methods, method) {
			t.Fatalf("expected %s in CORS methods %q", method, methods)
		}
	}
}
