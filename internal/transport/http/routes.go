package http

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"strings"

	"github.com/soundcheck/soundcheck/internal/app"
	"github.com/soundcheck/soundcheck/internal/core/domain"
	"github.com/soundcheck/soundcheck/internal/transport/ws"
)

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("GET /api/v1/doctor", s.handleDoctor)
	s.mux.HandleFunc("GET /api/v1/probe", s.handleProbeList)
	s.mux.HandleFunc("POST /api/v1/probe/{id}", s.handleProbeID)
	s.mux.HandleFunc("GET /api/v1/resolve", s.handleResolve)
	s.mux.HandleFunc("POST /api/v1/resolve", s.handleResolve)
	s.mux.HandleFunc("GET /api/v1/manifest", s.handleManifest)
	s.mux.HandleFunc("GET /api/v1/adapters", s.handleAdapters)
	s.mux.HandleFunc("GET /api/v1/config", s.handleConfigGet)
	s.mux.HandleFunc("PUT /api/v1/config", s.handleConfigPut)
	s.mux.Handle("/api/v1/terminal/ws", ws.NewTerminalHandler(s.ptyRunner))

	s.mux.HandleFunc("/api/health", s.handleHealth)
	s.mux.HandleFunc("/api/doctor", s.handleDoctor)
	s.mux.HandleFunc("/api/probe", s.handleProbeCompat)
	s.mux.HandleFunc("/api/resolve", s.handleResolve)
	s.mux.HandleFunc("/api/manifest", s.handleManifest)

	s.registerStatic()
}

func (s *Server) handleDoctor(w http.ResponseWriter, r *http.Request) {
	if strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
		s.streamDoctorSSE(w, r)
		return
	}
	opts := parseDoctorOpts(r)
	report, err := s.app.Doctor(r.Context(), opts...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (s *Server) streamDoctorSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}
	setSSEHeaders(w)
	flusher.Flush()
	_, _ = fmt.Fprintf(w, "event: start\ndata: {\"status\":\"running\"}\n\n")
	flusher.Flush()

	opts := parseDoctorOpts(r)
	report, err := s.app.Doctor(r.Context(), opts...)
	if err != nil {
		_, _ = fmt.Fprintf(w, "event: error\ndata: {\"error\":%q}\n\n", err.Error())
		flusher.Flush()
		return
	}
	streamReportAdapters(w, flusher, report.Adapters)
	b, _ := json.Marshal(report)
	_, _ = fmt.Fprintf(w, "event: report\ndata: %s\n\n", string(b))
	_, _ = fmt.Fprintf(w, "event: done\ndata: {\"status\":\"complete\"}\n\n")
	flusher.Flush()
}

func streamReportAdapters(w http.ResponseWriter, flusher http.Flusher, adapters []domain.Adapter) {
	for _, a := range adapters {
		b, err := json.Marshal(a)
		if err == nil {
			_, _ = fmt.Fprintf(w, "event: adapter\ndata: %s\n\n", string(b))
			flusher.Flush()
		}
	}
}

func setSSEHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleProbeList(w http.ResponseWriter, r *http.Request) {
	opts := parseDoctorOpts(r)
	report, err := s.app.Doctor(r.Context(), opts...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, report.Adapters)
}

func (s *Server) handleProbeID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		id = strings.TrimPrefix(r.URL.Path, "/api/v1/probe/")
	}
	if id == "" {
		id = r.URL.Query().Get("id")
	}
	adapter, err := s.app.ProbeOne(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("adapter not found or probe failed: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, adapter)
}

type resolveRequest struct {
	Role  string   `json:"role"`
	Roles []string `json:"roles"`
	Task  string   `json:"task"`
}

func (s *Server) handleResolve(w http.ResponseWriter, r *http.Request) {
	var req resolveRequest
	if r.Body != nil && r.ContentLength > 0 {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	role := req.Role
	if role == "" && len(req.Roles) > 0 {
		role = req.Roles[0]
	}
	if role == "" {
		role = r.URL.Query().Get("role")
	}
	resolutions, err := s.app.Resolve(r.Context(), role)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resolutions)
}

func (s *Server) handleManifest(w http.ResponseWriter, r *http.Request) {
	m, err := s.app.Manifest(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, m)
}

func (s *Server) handleAdapters(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.app.Definitions())
}

func (s *Server) handleConfigGet(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.app.Config())
}

func (s *Server) handleConfigPut(w http.ResponseWriter, r *http.Request) {
	var cfg domain.Config
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid config json: %v", err))
		return
	}
	s.app.SetConfig(cfg)
	writeJSON(w, http.StatusOK, s.app.Config())
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "version": "1.4.0"})
}

func (s *Server) handleProbeCompat(w http.ResponseWriter, r *http.Request) {
	if id := r.URL.Query().Get("id"); id != "" {
		s.handleProbeID(w, r)
		return
	}
	s.handleProbeList(w, r)
}

func (s *Server) registerStatic() {
	subFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return
	}
	s.mux.Handle("/", http.FileServer(http.FS(subFS)))
}

func parseDoctorOpts(r *http.Request) []app.DoctorOption {
	var opts []app.DoctorOption
	if r.URL.Query().Get("deep") == "true" {
		opts = append(opts, app.WithDoctorDeep(true))
	}
	if r.URL.Query().Get("force") == "true" {
		opts = append(opts, app.WithDoctorForce(true))
	}
	return opts
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
