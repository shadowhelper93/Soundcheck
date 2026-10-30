package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/soundcheck/soundcheck/internal/app"
	"github.com/soundcheck/soundcheck/internal/core/domain"
	"github.com/soundcheck/soundcheck/internal/edge/config"
	transphttp "github.com/soundcheck/soundcheck/internal/transport/http"
)

func newTestServer(t *testing.T, opts ...transphttp.ServerOption) (*transphttp.Server, *app.App) {
	t.Helper()
	appInstance, err := app.New()
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}
	srv := transphttp.NewServer(appInstance, opts...)
	return srv, appInstance
}

func TestServerHealth(t *testing.T) {
	srv, _ := newTestServer(t)
	req := httptest.NewRequest("GET", "/api/health", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
}

func TestServerDoctorJSON(t *testing.T) {
	srv, _ := newTestServer(t)
	req := httptest.NewRequest("GET", "/api/v1/doctor?deep=true", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	var res map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to parse doctor JSON: %v", err)
	}
	if _, ok := res["summary"]; !ok {
		t.Errorf("expected summary field in doctor response")
	}
}

func TestServerDoctorSSE(t *testing.T) {
	srv, _ := newTestServer(t)
	req := httptest.NewRequest("GET", "/api/v1/doctor", nil)
	req.Header.Set("Accept", "text/event-stream")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "text/event-stream") {
		t.Fatalf("expected text/event-stream content type, got %s", rec.Header().Get("Content-Type"))
	}
	body := rec.Body.String()
	if !strings.Contains(body, "event: start") || !strings.Contains(body, "event: done") {
		t.Errorf("expected start and done SSE events, got: %s", body)
	}
}

func TestServerProbeEndpoints(t *testing.T) {
	srv, _ := newTestServer(t)

	// GET /api/v1/probe
	req := httptest.NewRequest("GET", "/api/v1/probe", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	// POST /api/v1/probe/{id} - unknown adapter
	reqUnknown := httptest.NewRequest("POST", "/api/v1/probe/nonexistent-tool", nil)
	recUnknown := httptest.NewRecorder()
	srv.ServeHTTP(recUnknown, reqUnknown)
	if recUnknown.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 for unknown adapter, got %d", recUnknown.Code)
	}
}

func TestServerResolveAndManifest(t *testing.T) {
	srv, _ := newTestServer(t)

	// POST /api/v1/resolve
	body := bytes.NewBufferString(`{"role":"primary"}`)
	reqResolve := httptest.NewRequest("POST", "/api/v1/resolve", body)
	recResolve := httptest.NewRecorder()
	srv.ServeHTTP(recResolve, reqResolve)
	if recResolve.Code != http.StatusOK {
		t.Fatalf("expected status 200 from resolve POST, got %d", recResolve.Code)
	}

	// GET /api/v1/resolve?role=primary
	reqResolveGet := httptest.NewRequest("GET", "/api/v1/resolve?role=primary", nil)
	recResolveGet := httptest.NewRecorder()
	srv.ServeHTTP(recResolveGet, reqResolveGet)
	if recResolveGet.Code != http.StatusOK {
		t.Fatalf("expected status 200 from resolve GET, got %d", recResolveGet.Code)
	}

	// GET /api/v1/manifest
	reqManifest := httptest.NewRequest("GET", "/api/v1/manifest", nil)
	recManifest := httptest.NewRecorder()
	srv.ServeHTTP(recManifest, reqManifest)
	if recManifest.Code != http.StatusOK {
		t.Fatalf("expected status 200 from manifest, got %d", recManifest.Code)
	}

	// GET /api/v1/adapters
	reqAdapters := httptest.NewRequest("GET", "/api/v1/adapters", nil)
	recAdapters := httptest.NewRecorder()
	srv.ServeHTTP(recAdapters, reqAdapters)
	if recAdapters.Code != http.StatusOK {
		t.Fatalf("expected status 200 from adapters, got %d", recAdapters.Code)
	}
}

func TestServerConfigReadWrite(t *testing.T) {
	srv, appInstance := newTestServer(t)
	path := filepath.Join(t.TempDir(), "agent-doctor.yaml")
	app.WithConfigPath(path)(appInstance)

	recGet := httptest.NewRecorder()
	srv.ServeHTTP(recGet, httptest.NewRequest("GET", "/api/v1/config", nil))
	if recGet.Code != http.StatusOK {
		t.Fatalf("expected status 200 from config get, got %d", recGet.Code)
	}

	newCfg := appInstance.Config()
	newCfg.Roles[domain.RolePrimary] = []string{"claude:claude-sonnet-5-5"}
	b, _ := json.Marshal(newCfg)
	recPut := httptest.NewRecorder()
	srv.ServeHTTP(recPut, httptest.NewRequest("PUT", "/api/v1/config", bytes.NewReader(b)))
	if recPut.Code != http.StatusOK {
		t.Fatalf("expected status 200 from config put, got %d: %s", recPut.Code, recPut.Body)
	}
	onDisk, err := config.Load(path)
	if err != nil || onDisk.Roles[domain.RolePrimary][0] != "claude:claude-sonnet-5-5" {
		t.Fatalf("config not persisted: %v %+v", err, onDisk)
	}

	bad := httptest.NewRecorder()
	srv.ServeHTTP(bad, httptest.NewRequest("PUT", "/api/v1/config", bytes.NewReader([]byte(`{"version":99}`))))
	if bad.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid config, got %d", bad.Code)
	}
}

func TestServerOriginSecurity(t *testing.T) {
	srv, _ := newTestServer(t)

	// Loopback origin - allowed
	reqLocal := httptest.NewRequest("GET", "/api/health", nil)
	reqLocal.Header.Set("Origin", "http://localhost:3000")
	recLocal := httptest.NewRecorder()
	srv.ServeHTTP(recLocal, reqLocal)
	if recLocal.Code != http.StatusOK {
		t.Errorf("expected status 200 for localhost origin, got %d", recLocal.Code)
	}

	// 127.0.0.1 origin - allowed
	reqIP := httptest.NewRequest("GET", "/api/health", nil)
	reqIP.Header.Set("Origin", "http://127.0.0.1:8080")
	recIP := httptest.NewRecorder()
	srv.ServeHTTP(recIP, reqIP)
	if recIP.Code != http.StatusOK {
		t.Errorf("expected status 200 for 127.0.0.1 origin, got %d", recIP.Code)
	}

	// Disallowed origin - rejected
	reqEvil := httptest.NewRequest("GET", "/api/health", nil)
	reqEvil.Header.Set("Origin", "http://evil.com")
	recEvil := httptest.NewRecorder()
	srv.ServeHTTP(recEvil, reqEvil)
	if recEvil.Code != http.StatusForbidden {
		t.Errorf("expected status 403 for evil.com origin, got %d", recEvil.Code)
	}

	// Attacker disguised 127 origin (CSWSH) - rejected
	reqEvil127 := httptest.NewRequest("GET", "/api/health", nil)
	reqEvil127.Header.Set("Origin", "http://127.evil.com")
	recEvil127 := httptest.NewRecorder()
	srv.ServeHTTP(recEvil127, reqEvil127)
	if recEvil127.Code != http.StatusForbidden {
		t.Errorf("expected status 403 for 127.evil.com origin, got %d", recEvil127.Code)
	}

	// IPv6 loopback origin - allowed
	reqIPv6 := httptest.NewRequest("GET", "/api/health", nil)
	reqIPv6.Header.Set("Origin", "http://[::1]:8080")
	recIPv6 := httptest.NewRecorder()
	srv.ServeHTTP(recIPv6, reqIPv6)
	if recIPv6.Code != http.StatusOK {
		t.Errorf("expected status 200 for [::1] origin, got %d", recIPv6.Code)
	}
}

func TestServerBearerAuth(t *testing.T) {
	srv, _ := newTestServer(t, transphttp.WithAuthToken("secret-token-123"))
	testHeaderAuth(t, srv)
	testCookieAuth(t, srv)
}

func testHeaderAuth(t *testing.T, srv *transphttp.Server) {
	t.Helper()
	reqNoAuth := httptest.NewRequest("GET", "/api/v1/manifest", nil)
	recNoAuth := httptest.NewRecorder()
	srv.ServeHTTP(recNoAuth, reqNoAuth)
	if recNoAuth.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", recNoAuth.Code)
	}

	reqWrong := httptest.NewRequest("GET", "/api/v1/manifest", nil)
	reqWrong.Header.Set("Authorization", "Bearer wrong-token")
	recWrong := httptest.NewRecorder()
	srv.ServeHTTP(recWrong, reqWrong)
	if recWrong.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", recWrong.Code)
	}

	req := httptest.NewRequest("GET", "/api/v1/manifest", nil)
	req.Header.Set("Authorization", "Bearer secret-token-123")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func testCookieAuth(t *testing.T, srv *transphttp.Server) {
	t.Helper()
	req := httptest.NewRequest("GET", "/api/v1/manifest?token=secret-token-123", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var authCookie *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == "soundcheck_token" {
			authCookie = c
			break
		}
	}
	if authCookie == nil || authCookie.Value != "secret-token-123" {
		t.Fatalf("expected soundcheck_token cookie, got %v", authCookie)
	}

	reqCookie := httptest.NewRequest("GET", "/api/v1/adapters", nil)
	reqCookie.AddCookie(authCookie)
	recCookie := httptest.NewRecorder()
	srv.ServeHTTP(recCookie, reqCookie)
	if recCookie.Code != http.StatusOK {
		t.Errorf("expected 200 with cookie auth, got %d", recCookie.Code)
	}
}

func TestServerGracefulShutdown(t *testing.T) {
	srv, _ := newTestServer(t, transphttp.WithAddr("127.0.0.1:0"))
	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start(ctx)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("unexpected server shutdown error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server failed to shutdown within timeout")
	}
}
