package http

import (
	"bytes"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

func (s *Server) registerStatic() {
	if s.assets == nil {
		return
	}
	s.mux.HandleFunc("/", s.handleStatic)
}

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	clean := filepath.Clean(strings.TrimPrefix(r.URL.Path, "/"))
	if clean == "." || clean == "" {
		s.serveIndex(w, r)
		return
	}
	if s.serveFile(w, r, clean) {
		return
	}
	if shouldFallbackToIndex(clean) {
		s.serveIndex(w, r)
		return
	}
	http.NotFound(w, r)
}

func (s *Server) serveFile(w http.ResponseWriter, r *http.Request, path string) bool {
	f, err := s.assets.Open(path)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()
	stat, err := f.Stat()
	if err != nil || stat.IsDir() {
		return false
	}
	http.FileServer(http.FS(s.assets)).ServeHTTP(w, r)
	return true
}

func (s *Server) serveIndex(w http.ResponseWriter, r *http.Request) {
	f, err := s.assets.Open("index.html")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer func() { _ = f.Close() }()
	content, err := io.ReadAll(f)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	http.ServeContent(w, r, "index.html", time.Time{}, bytes.NewReader(content))
}

func shouldFallbackToIndex(path string) bool {
	if strings.HasPrefix(path, "assets/") {
		return false
	}
	return filepath.Ext(path) == ""
}
