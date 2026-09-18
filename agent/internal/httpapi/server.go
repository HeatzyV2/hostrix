package httpapi

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/hostrix/hostrix/agent/internal/config"
	"github.com/hostrix/hostrix/agent/internal/containers"
)

type Server struct {
	cfg     *config.Config
	mgr     containers.ContainerManager
	tokenHash []byte
	http    *http.Server
}

func New(cfg *config.Config, mgr containers.ContainerManager) *Server {
	sum := sha256.Sum256([]byte(cfg.NodeToken))
	s := &Server{
		cfg:       cfg,
		mgr:       mgr,
		tokenHash: sum[:],
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("POST /v1/containers", s.auth(s.handleCreate))
	mux.HandleFunc("DELETE /v1/containers/{name}", s.auth(s.handleDelete))
	mux.HandleFunc("POST /v1/containers/{name}/start", s.auth(s.handleStart))
	mux.HandleFunc("POST /v1/containers/{name}/stop", s.auth(s.handleStop))
	mux.HandleFunc("POST /v1/containers/{name}/restart", s.auth(s.handleRestart))
	mux.HandleFunc("POST /v1/containers/{name}/kill", s.auth(s.handleKill))
	mux.HandleFunc("GET /v1/containers/{name}/status", s.auth(s.handleStatus))
	mux.HandleFunc("GET /v1/containers/{name}/stats", s.auth(s.handleStats))
	mux.HandleFunc("GET /v1/containers/{name}/console/ws", s.auth(s.handleConsoleWS))
	mux.HandleFunc("GET /v1/containers/{name}/backups", s.auth(s.handleListBackups))
	mux.HandleFunc("POST /v1/containers/{name}/backups", s.auth(s.handleCreateBackup))
	mux.HandleFunc("GET /v1/containers/{name}/backups/{backup}", s.auth(s.handleGetBackup))
	mux.HandleFunc("DELETE /v1/containers/{name}/backups/{backup}", s.auth(s.handleDeleteBackup))
	mux.HandleFunc("GET /v1/containers/{name}/backups/{backup}/download", s.auth(s.handleDownloadBackup))
	mux.HandleFunc("POST /v1/containers/{name}/backups/{backup}/restore", s.auth(s.handleRestoreBackup))
	mux.HandleFunc("GET /v1/metrics/host", s.auth(s.handleHostMetrics))

	s.http = &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	s.applyLongLivedTimeouts()
	return s
}

func (s *Server) ListenAndServe() error {
	log.Printf("hostrix-agent listening on %s", s.cfg.HTTPAddr)
	return s.http.ListenAndServe()
}

func (s *Server) Close() error {
	return s.http.Close()
}

func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		if token == "" {
			writeErr(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		sum := sha256.Sum256([]byte(token))
		if subtle.ConstantTimeCompare(sum[:], s.tokenHash) != 1 {
			writeErr(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next(w, r)
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "hostrix-agent",
		"version": "0.6.0",
	})
}

type createBody struct {
	Name     string `json:"name"`
	Image    string `json:"image"`
	MemoryMB int    `json:"memory_mb"`
	CPULimit int    `json:"cpu_limit"`
	DiskMB   int    `json:"disk_mb"`
}

func (s *Server) handleCreate(w http.ResponseWriter, r *http.Request) {
	var body createBody
	if err := decodeJSON(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
	defer cancel()
	err := s.mgr.CreateContainer(ctx, containers.CreateRequest{
		Name: body.Name, Image: body.Image, MemoryMB: body.MemoryMB, CPULimit: body.CPULimit, DiskMB: body.DiskMB,
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "created", "name": body.Name})
}

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()
	if err := s.mgr.DeleteContainer(ctx, name); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleStart(w http.ResponseWriter, r *http.Request) {
	s.action(w, r, s.mgr.StartContainer)
}
func (s *Server) handleStop(w http.ResponseWriter, r *http.Request) {
	s.action(w, r, s.mgr.StopContainer)
}
func (s *Server) handleRestart(w http.ResponseWriter, r *http.Request) {
	s.action(w, r, s.mgr.RestartContainer)
}
func (s *Server) handleKill(w http.ResponseWriter, r *http.Request) {
	s.action(w, r, s.mgr.KillContainer)
}

func (s *Server) action(w http.ResponseWriter, r *http.Request, fn func(context.Context, string) error) {
	name := r.PathValue("name")
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Minute)
	defer cancel()
	if err := fn(ctx, name); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	st, err := s.mgr.GetContainerStatus(ctx, name)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": string(st)})
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	stats, err := s.mgr.GetContainerStats(ctx, name)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func (s *Server) handleHostMetrics(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	m, err := s.mgr.HostMetrics(ctx)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, m)
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(strings.ToLower(h), "bearer ") {
		return strings.TrimSpace(h[7:])
	}
	if t := strings.TrimSpace(r.URL.Query().Get("token")); t != "" {
		return t
	}
	return ""
}

func decodeJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
