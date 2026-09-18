package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/hostrix/hostrix/api/internal/models"
	"github.com/hostrix/hostrix/api/internal/templates"
)

type createTemplateRequest struct {
	Name           string `json:"name"`
	Slug           string `json:"slug"`
	Description    string `json:"description"`
	Image          string `json:"image"`
	StartupCommand string `json:"startup_command"`
}

type updateTemplateRequest struct {
	Name           *string `json:"name"`
	Slug           *string `json:"slug"`
	Description    *string `json:"description"`
	Image          *string `json:"image"`
	StartupCommand *string `json:"startup_command"`
}

func publicTemplate(t *models.ServerTemplate) map[string]any {
	return map[string]any{
		"uuid":            t.UUID,
		"name":            t.Name,
		"slug":            t.Slug,
		"description":     t.Description,
		"image":           t.Image,
		"startup_command": t.StartupCommand,
		"created_at":      t.CreatedAt,
	}
}

func (s *Server) handleListTemplates(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireUser(w, r); !ok {
		return
	}
	list, err := templates.List(s.db)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list templates")
		return
	}
	out := make([]map[string]any, 0, len(list))
	for i := range list {
		out = append(out, publicTemplate(&list[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{"templates": out})
}

func (s *Server) handleGetTemplate(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireUser(w, r); !ok {
		return
	}
	id := r.PathValue("id")
	tmpl, err := templates.GetByUUID(s.db, id)
	if errors.Is(err, templates.ErrNotFound) {
		tmpl, err = templates.GetBySlug(s.db, id)
	}
	if errors.Is(err, templates.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get template")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"template": publicTemplate(tmpl)})
}

func (s *Server) handleCreateTemplate(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var req createTemplateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	tmpl, err := templates.Create(s.db, templates.CreateInput{
		Name:           req.Name,
		Slug:           req.Slug,
		Description:    req.Description,
		Image:          req.Image,
		StartupCommand: req.StartupCommand,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"template": publicTemplate(tmpl)})
}

func (s *Server) handleUpdateTemplate(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var req updateTemplateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	tmpl, err := templates.Update(s.db, r.PathValue("id"), templates.UpdateInput{
		Name:           req.Name,
		Slug:           req.Slug,
		Description:    req.Description,
		Image:          req.Image,
		StartupCommand: req.StartupCommand,
	})
	if errors.Is(err, templates.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"template": publicTemplate(tmpl)})
}

func (s *Server) handleDeleteTemplate(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	err := templates.Delete(s.db, r.PathValue("id"))
	if errors.Is(err, templates.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "in use") {
			writeError(w, http.StatusConflict, msg)
			return
		}
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
