// Package http provides the HTTP server and web UI transport layer for Soundcheck.
package http

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"

	"github.com/soundcheck/soundcheck/internal/app"
)

//go:embed static/*
var staticFiles embed.FS

// Server represents the Soundcheck HTTP server and web UI.
type Server struct {
	app *app.App
	mux *http.ServeMux
}

// NewServer initializes and returns a new Server instance.
func NewServer(a *app.App) *Server {
	s := &Server{
		app: a,
		mux: http.NewServeMux(),
	}
	s.routes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	s.mux.HandleFunc("/api/health", s.handleHealth)
	s.mux.HandleFunc("/api/doctor", s.handleDoctor)
	s.mux.HandleFunc("/api/probe", s.handleProbe)
	s.mux.HandleFunc("/api/resolve", s.handleResolve)
	s.mux.HandleFunc("/api/manifest", s.handleManifest)

	subFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		panic(err)
	}
	fileServer := http.FileServer(http.FS(subFS))
	s.mux.Handle("/", fileServer)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]string{"status": "ok", "version": "1.4.0"})
}

func (s *Server) handleDoctor(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	deep := r.URL.Query().Get("deep") == "true"
	force := r.URL.Query().Get("force") == "true"

	var opts []app.DoctorOption
	if deep {
		opts = append(opts, app.WithDoctorDeep(true))
	}
	if force {
		opts = append(opts, app.WithDoctorForce(true))
	}

	report, err := s.app.Doctor(ctx, opts...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, report)
}

func (s *Server) handleProbe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	adapterID := r.URL.Query().Get("adapter")
	if adapterID != "" {
		adapter, err := s.app.ProbeOne(ctx, adapterID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		writeJSON(w, adapter)
		return
	}

	report, err := s.app.Doctor(ctx, app.WithDoctorDeep(true))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, report.Adapters)
}

func (s *Server) handleResolve(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	roleFilter := r.URL.Query().Get("role")
	resolutions, err := s.app.Resolve(ctx, roleFilter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, resolutions)
}

func (s *Server) handleManifest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	manifest, err := s.app.Manifest(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, manifest)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(v)
}
