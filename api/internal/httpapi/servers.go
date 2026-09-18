package httpapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/hostrix/hostrix/api/internal/servers"
)

type createServerRequest struct {
	Name         string `json:"name"`
	NodeUUID     string `json:"node_uuid"`
	TemplateID   uint64 `json:"template_id"`
	TemplateUUID string `json:"template_uuid"`
	TemplateSlug string `json:"template_slug"`
	Image        string `json:"image"`
	MemoryMB     int    `json:"memory"`
	CPULimit     int    `json:"cpu"`
	DiskMB       int    `json:"disk"`
}

func (s *Server) handleListServers(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	list, err := servers.ListForUser(s.db, user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list servers")
		return
	}
	out := make([]map[string]any, 0, len(list))
	for i := range list {
		out = append(out, publicServer(&list[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{"servers": out})
}

func (s *Server) handleCreateServer(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	if !user.IsAdmin {
		writeError(w, http.StatusForbidden, "admin required to create servers in phase 2")
		return
	}
	var req createServerRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
	defer cancel()
	srv, err := servers.Create(ctx, s.db, servers.CreateInput{
		Name:         req.Name,
		NodeUUID:     req.NodeUUID,
		TemplateID:   req.TemplateID,
		TemplateUUID: req.TemplateUUID,
		TemplateSlug: req.TemplateSlug,
		Image:        req.Image,
		MemoryMB:     req.MemoryMB,
		CPULimit:     req.CPULimit,
		DiskMB:       req.DiskMB,
		OwnerID:      user.ID,
	})
	if err != nil {
		// may still return partial server with ERROR status
		if srv != nil {
			writeJSON(w, http.StatusBadGateway, map[string]any{
				"error":  err.Error(),
				"server": publicServer(srv),
			})
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"server": publicServer(srv)})
}

func (s *Server) handleGetServer(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	srv, err := servers.GetByUUID(s.db, r.PathValue("id"))
	if errors.Is(err, servers.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get server")
		return
	}
	if !servers.CanAccess(user, srv) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	if status, err := servers.SyncStatus(ctx, s.db, srv); err == nil {
		srv.Status = status
	}
	writeJSON(w, http.StatusOK, map[string]any{"server": publicServer(srv)})
}

func (s *Server) handleDeleteServer(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()
	err := servers.Delete(ctx, s.db, user, r.PathValue("id"))
	if errors.Is(err, servers.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if errors.Is(err, servers.ErrForbidden) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleServerPower(action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := s.requireUser(w, r)
		if !ok {
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Minute)
		defer cancel()
		err := servers.Power(ctx, s.db, user, r.PathValue("id"), action)
		if errors.Is(err, servers.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		if errors.Is(err, servers.ErrForbidden) {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}
		if err != nil {
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}
