// Package http provides the HTTP server and REST/SSE/WS transport layer for Soundcheck.
package http

import (
	"context"
	"crypto/subtle"
	"embed"
	"errors"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/soundcheck/soundcheck/internal/app"
	"github.com/soundcheck/soundcheck/internal/edge/exec"
	"github.com/soundcheck/soundcheck/ui"
)

//go:embed static/*
var staticFiles embed.FS

// Server represents the Soundcheck HTTP and WebSocket transport server.
type Server struct {
	app            *app.App
	addr           string
	authToken      string
	allowedOrigins []string
	ptyRunner      exec.PTYRunner
	assets         fs.FS
	fileServer     http.Handler
	mux            *http.ServeMux
	server         *http.Server
}

// ServerOption configures Server parameters.
type ServerOption func(*Server)

// WithAddr configures the listening address.
func WithAddr(addr string) ServerOption {
	return func(s *Server) { s.addr = addr }
}

// WithAuthToken sets the optional Bearer authentication token.
func WithAuthToken(token string) ServerOption {
	return func(s *Server) { s.authToken = token }
}

// WithAllowedOrigins configures explicit allowed origins.
func WithAllowedOrigins(origins ...string) ServerOption {
	return func(s *Server) { s.allowedOrigins = append(s.allowedOrigins, origins...) }
}

// WithPTYRunner sets the PTY runner for WebSocket terminal sessions.
func WithPTYRunner(runner exec.PTYRunner) ServerOption {
	return func(s *Server) { s.ptyRunner = runner }
}

// WithAssets configures the static UI assets filesystem.
func WithAssets(assets fs.FS) ServerOption {
	return func(s *Server) {
		s.assets = assets
		if assets != nil {
			s.fileServer = http.FileServer(http.FS(assets))
		}
	}
}

// NewServer initializes and returns a new Server instance.
func NewServer(a *app.App, opts ...ServerOption) *Server {
	s := &Server{
		app:  a,
		addr: "127.0.0.1:3000",
		mux:  http.NewServeMux(),
	}
	for _, opt := range opts {
		opt(s)
	}
	if s.assets == nil {
		if a, err := ui.Assets(); err == nil {
			s.assets = a
		} else if subFS, err := fs.Sub(staticFiles, "static"); err == nil {
			s.assets = subFS
		}
	}
	if s.assets != nil && s.fileServer == nil {
		s.fileServer = http.FileServer(http.FS(s.assets))
	}
	s.registerRoutes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !s.checkOrigin(w, r) {
		return
	}
	if !s.checkAuth(w, r) {
		return
	}
	s.mux.ServeHTTP(w, r)
}

func (s *Server) checkOrigin(w http.ResponseWriter, r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	if !s.isOriginAllowed(origin) {
		http.Error(w, "Forbidden: invalid origin", http.StatusForbidden)
		return false
	}
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return false
	}
	return true
}

func (s *Server) isOriginAllowed(origin string) bool {
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	host := u.Hostname()
	if isLoopback(host) {
		return true
	}
	for _, allowed := range s.allowedOrigins {
		if strings.EqualFold(origin, allowed) || strings.EqualFold(host, allowed) {
			return true
		}
	}
	return false
}

func isLoopback(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (s *Server) checkAuth(w http.ResponseWriter, r *http.Request) bool {
	if s.authToken == "" || r.URL.Path == "/api/health" || r.Method == http.MethodOptions {
		return true
	}
	token := extractToken(r)
	if token != "" && subtle.ConstantTimeCompare([]byte(token), []byte(s.authToken)) == 1 {
		return true
	}
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
	return false
}

func extractToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return r.URL.Query().Get("token")
}

// Start launches the HTTP server with graceful shutdown when ctx is canceled.
func (s *Server) Start(ctx context.Context) error {
	s.server = &http.Server{
		Addr:              s.addr,
		Handler:           s,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.server.Shutdown(shutCtx)
	case err := <-errCh:
		return err
	}
}

// Addr returns the configured address.
func (s *Server) Addr() string {
	return s.addr
}
