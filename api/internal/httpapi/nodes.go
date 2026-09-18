package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/hostrix/hostrix/api/internal/nodes"
)

type createNodeRequest struct {
	Name     string `json:"name"`
	Hostname string `json:"hostname"`
	Address  string `json:"address"`
	Port     int    `json:"port"`
}

func (s *Server) handleListNodes(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	list, err := nodes.List(s.db)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list nodes")
		return
	}
	out := make([]map[string]any, 0, len(list))
	for i := range list {
		out = append(out, publicNode(&list[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{"nodes": out})
}

func (s *Server) handleCreateNode(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var req createNodeRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	result, err := nodes.Create(s.db, nodes.CreateInput{
		Name: req.Name, Hostname: req.Hostname, Address: req.Address, Port: req.Port,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"node":  publicNode(result.Node),
		"token": result.Token, // shown once
	})
}

func (s *Server) handleGetNode(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	node, err := nodes.GetByUUID(s.db, r.PathValue("id"))
	if errors.Is(err, nodes.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get node")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"node": publicNode(node)})
}

func (s *Server) handleDeleteNode(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	if err := nodes.Delete(s.db, r.PathValue("id")); errors.Is(err, nodes.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete node")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleRotateNodeToken(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	token, err := nodes.RotateToken(s.db, r.PathValue("id"))
	if errors.Is(err, nodes.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to rotate token")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}

type heartbeatRequest struct {
	Status           string  `json:"status"`
	CPUPercent       float64 `json:"cpu_percent"`
	MemoryUsageBytes int64   `json:"memory_usage_bytes"`
	MemoryTotalBytes int64   `json:"memory_total_bytes"`
	DiskUsageBytes   int64   `json:"disk_usage_bytes"`
	DiskTotalBytes   int64   `json:"disk_total_bytes"`
	Containers       int     `json:"containers"`
	Error            string  `json:"error"`
}

func (s *Server) handleNodeHeartbeat(w http.ResponseWriter, r *http.Request) {
	token := bearerToken(r)
	if token == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req heartbeatRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	status := req.Status
	if strings.TrimSpace(req.Error) != "" && status == "" {
		status = "ERROR"
	}
	err := nodes.ApplyHeartbeat(s.db, r.PathValue("id"), token, nodes.HeartbeatInput{
		Status:           status,
		CPUPercent:       req.CPUPercent,
		MemoryUsageBytes: req.MemoryUsageBytes,
		MemoryTotalBytes: req.MemoryTotalBytes,
		DiskUsageBytes:   req.DiskUsageBytes,
		DiskTotalBytes:   req.DiskTotalBytes,
		Containers:       req.Containers,
	})
	if errors.Is(err, nodes.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if errors.Is(err, nodes.ErrUnauthorized) {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "heartbeat failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
