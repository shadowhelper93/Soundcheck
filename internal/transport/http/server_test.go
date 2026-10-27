package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/soundcheck/soundcheck/internal/app"
)

func TestServerHealth(t *testing.T) {
	appInstance, err := app.New()
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}
	srv := NewServer(appInstance)

	req := httptest.NewRequest("GET", "/api/health", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
}
