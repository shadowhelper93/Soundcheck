package http

import (
	"bytes"
	"io"
	"net/http"
	"path"
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
	clean := path.Clean(strings.TrimPrefix(r.URL.Path, "/"))
	if strings.HasPrefix(clean, "api/") || clean == "api" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
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

func (s *Server) serveFile(w http.ResponseWriter, r *http.Request, filePath string) bool {
	f, err := s.assets.Open(filePath)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()
	stat, err := f.Stat()
	if err != nil || stat.IsDir() {
		return false
	}
	if s.fileServer != nil {
		s.fileServer.ServeHTTP(w, r)
	} else {
		http.FileServer(http.FS(s.assets)).ServeHTTP(w, r)
	}
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

func shouldFallbackToIndex(p string) bool {
	if strings.HasPrefix(p, "assets/") || strings.HasPrefix(p, "api/") || p == "api" {
		return false
	}
	return path.Ext(p) == ""
}
