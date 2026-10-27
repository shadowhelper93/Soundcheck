package http_test

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	transphttp "github.com/soundcheck/soundcheck/internal/transport/http"
	"github.com/soundcheck/soundcheck/ui"
)

func TestStaticRootAndFallback(t *testing.T) {
	srv, _ := newTestServer(t)

	reqRoot := httptest.NewRequest("GET", "/", nil)
	recRoot := httptest.NewRecorder()
	srv.ServeHTTP(recRoot, reqRoot)
	if recRoot.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recRoot.Code)
	}
	if ct := recRoot.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Errorf("expected text/html, got %s", ct)
	}

	reqDoctor := httptest.NewRequest("GET", "/doctor", nil)
	recDoctor := httptest.NewRecorder()
	srv.ServeHTTP(recDoctor, reqDoctor)
	if recDoctor.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recDoctor.Code)
	}
	if !strings.Contains(recDoctor.Body.String(), "<!doctype html>") && !strings.Contains(recDoctor.Body.String(), "<!DOCTYPE html>") {
		t.Errorf("expected html body on fallback")
	}
}

func TestStaticAssetsAndMissing(t *testing.T) {
	srv, _ := newTestServer(t)

	if assets, err := ui.Assets(); err == nil {
		if entries, err := fs.ReadDir(assets, "assets"); err == nil {
			for _, entry := range entries {
				if !entry.IsDir() {
					reqAsset := httptest.NewRequest("GET", "/assets/"+entry.Name(), nil)
					recAsset := httptest.NewRecorder()
					srv.ServeHTTP(recAsset, reqAsset)
					if recAsset.Code != http.StatusOK {
						t.Fatalf("expected 200 for asset %s, got %d", entry.Name(), recAsset.Code)
					}
					break
				}
			}
		}
	}

	reqMissingAsset := httptest.NewRequest("GET", "/assets/missing.js", nil)
	recMissingAsset := httptest.NewRecorder()
	srv.ServeHTTP(recMissingAsset, reqMissingAsset)
	if recMissingAsset.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing asset, got %d", recMissingAsset.Code)
	}

	reqMissingFile := httptest.NewRequest("GET", "/missing.png", nil)
	recMissingFile := httptest.NewRecorder()
	srv.ServeHTTP(recMissingFile, reqMissingFile)
	if recMissingFile.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing file, got %d", recMissingFile.Code)
	}
}

func TestStaticMethodNotAllowed(t *testing.T) {
	srv, _ := newTestServer(t)
	req := httptest.NewRequest("POST", "/", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestCustomWithAssetsOption(t *testing.T) {
	mockFS := fstest.MapFS{
		"index.html":       {Data: []byte("<html>custom mock ui</html>")},
		"assets/bundle.js": {Data: []byte("console.log('custom');")},
	}
	srv, _ := newTestServer(t, transphttp.WithAssets(mockFS))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "custom mock ui") {
		t.Fatalf("expected custom mock UI, got %s", rec.Body.String())
	}

	reqRoute := httptest.NewRequest("GET", "/terminal", nil)
	recRoute := httptest.NewRecorder()
	srv.ServeHTTP(recRoute, reqRoute)
	if recRoute.Code != http.StatusOK || !strings.Contains(recRoute.Body.String(), "custom mock ui") {
		t.Fatalf("expected fallback to custom mock UI, got %s", recRoute.Body.String())
	}
}
