package httpapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/hostrix/hostrix/api/internal/backups"
	"github.com/hostrix/hostrix/api/internal/models"
	"github.com/hostrix/hostrix/api/internal/permissions"
	"github.com/hostrix/hostrix/api/internal/servers"
	"github.com/hostrix/hostrix/api/internal/users"
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
	views, err := servers.WithNodeInfo(s.db, list)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load nodes")
		return
	}
	out := make([]map[string]any, 0, len(views))
	for i := range views {
		out = append(out, publicServerView(&views[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{"servers": out})
}

func (s *Server) handleCreateServer(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	if !user.IsAdmin {
		writeError(w, http.StatusForbidden, "admin required to create servers")
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
		if srv != nil {
			view := servers.ServerView{Server: *srv}
			if n, nerr := servers.GetNode(s.db, srv.NodeID); nerr == nil {
				view.NodeUUID = n.UUID
				view.NodeName = n.Name
				view.NodeStatus = n.Status
			}
			writeJSON(w, http.StatusBadGateway, map[string]any{
				"error":  err.Error(),
				"server": publicServerView(&view),
			})
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	view := servers.ServerView{Server: *srv}
	if n, nerr := servers.GetNode(s.db, srv.NodeID); nerr == nil {
		view.NodeUUID = n.UUID
		view.NodeName = n.Name
		view.NodeStatus = n.Status
	}
	writeJSON(w, http.StatusCreated, map[string]any{"server": publicServerView(&view)})
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
	if !servers.CanAccess(s.db, user, srv) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	if status, err := servers.SyncStatus(ctx, s.db, srv); err == nil {
		srv.Status = status
	}
	view := servers.ServerView{Server: *srv}
	if n, nerr := servers.GetNode(s.db, srv.NodeID); nerr == nil {
		view.NodeUUID = n.UUID
		view.NodeName = n.Name
		view.NodeStatus = n.Status
	}
	writeJSON(w, http.StatusOK, map[string]any{"server": publicServerView(&view)})
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

type createBackupRequest struct {
	Name string `json:"name"`
}

func (s *Server) handleListServerBackups(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	list, err := backups.ListForServer(s.db, user, r.PathValue("id"))
	if errors.Is(err, servers.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if errors.Is(err, backups.ErrForbidden) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list backups")
		return
	}
	out := make([]map[string]any, 0, len(list))
	for i := range list {
		out = append(out, publicBackup(&list[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{"backups": out})
}

func (s *Server) handleCreateServerBackup(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	var req createBackupRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Minute)
	defer cancel()
	b, err := backups.Create(ctx, s.db, user, r.PathValue("id"), req.Name)
	if errors.Is(err, servers.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if errors.Is(err, backups.ErrForbidden) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	if err != nil {
		if b != nil {
			writeJSON(w, http.StatusBadGateway, map[string]any{
				"error":  err.Error(),
				"backup": publicBackup(b),
			})
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"backup": publicBackup(b)})
}

func (s *Server) handleGetServerBackup(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	b, _, err := backups.GetForUser(s.db, user, r.PathValue("id"), r.PathValue("backupId"))
	if errors.Is(err, servers.ErrNotFound) || errors.Is(err, backups.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if errors.Is(err, backups.ErrForbidden) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get backup")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"backup": publicBackup(b)})
}

func (s *Server) handleDeleteServerBackup(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()
	err := backups.Delete(ctx, s.db, user, r.PathValue("id"), r.PathValue("backupId"))
	if errors.Is(err, servers.ErrNotFound) || errors.Is(err, backups.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if errors.Is(err, backups.ErrForbidden) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleRestoreServerBackup(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Minute)
	defer cancel()
	err := backups.Restore(ctx, s.db, user, r.PathValue("id"), r.PathValue("backupId"))
	if errors.Is(err, servers.ErrNotFound) || errors.Is(err, backups.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if errors.Is(err, backups.ErrForbidden) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleDownloadServerBackup(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Minute)
	defer cancel()
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", `attachment; filename="`+r.PathValue("backupId")+`.tar.gz"`)
	n, b, err := backups.Download(ctx, s.db, user, r.PathValue("id"), r.PathValue("backupId"), w)
	if err != nil {
		if n == 0 {
			if errors.Is(err, servers.ErrNotFound) || errors.Is(err, backups.ErrNotFound) {
				writeError(w, http.StatusNotFound, "not found")
				return
			}
			if errors.Is(err, backups.ErrForbidden) {
				writeError(w, http.StatusForbidden, "forbidden")
				return
			}
			writeError(w, http.StatusBadGateway, err.Error())
		}
		return
	}
	_ = b
}

func (s *Server) handleListAllBackups(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	list, err := backups.ListAccessible(s.db, user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list backups")
		return
	}
	out := make([]map[string]any, 0, len(list))
	for i := range list {
		item := publicBackup(&list[i])
		var srv models.Server
		if err := s.db.First(&srv, list[i].ServerID).Error; err == nil {
			item["server_uuid"] = srv.UUID
			item["server_name"] = srv.Name
		}
		out = append(out, item)
	}
	writeJSON(w, http.StatusOK, map[string]any{"backups": out})
}

type grantPermissionRequest struct {
	UserUUID   string `json:"user_uuid"`
	Username   string `json:"username"`
	CanStart   bool   `json:"can_start"`
	CanStop    bool   `json:"can_stop"`
	CanFiles   bool   `json:"can_files"`
	CanConsole bool   `json:"can_console"`
}

func (s *Server) handleListPermissions(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	list, err := permissions.ListForServer(s.db, user, r.PathValue("id"))
	if errors.Is(err, servers.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if errors.Is(err, permissions.ErrForbidden) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list permissions")
		return
	}
	out, err := permissions.EnrichWithUser(s.db, list)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to enrich permissions")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"permissions": out})
}

func (s *Server) handleGrantPermission(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	var req grantPermissionRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	perm, err := permissions.Grant(s.db, user, r.PathValue("id"), permissions.GrantInput{
		UserUUID:   req.UserUUID,
		Username:   req.Username,
		CanStart:   req.CanStart,
		CanStop:    req.CanStop,
		CanFiles:   req.CanFiles,
		CanConsole: req.CanConsole,
	})
	if errors.Is(err, servers.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if errors.Is(err, permissions.ErrForbidden) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	out, _ := permissions.EnrichWithUser(s.db, []models.ServerPermission{*perm})
	if len(out) == 0 {
		writeJSON(w, http.StatusCreated, map[string]any{"permission": perm})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"permission": out[0]})
}

func (s *Server) handleRevokePermission(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	err := permissions.Revoke(s.db, user, r.PathValue("id"), r.PathValue("userId"))
	if errors.Is(err, servers.ErrNotFound) || errors.Is(err, permissions.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if errors.Is(err, permissions.ErrForbidden) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	list, err := users.List(s.db)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list users")
		return
	}
	out := make([]map[string]any, 0, len(list))
	for i := range list {
		out = append(out, publicUser(&list[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{"users": out})
}
