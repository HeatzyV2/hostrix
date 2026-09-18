package httpapi

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/hostrix/hostrix/agent/internal/containers"
)

const maxUploadBytes = containers.MaxFileBytes

func (s *Server) handleListFiles(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	dirPath := r.URL.Query().Get("path")
	if dirPath == "" {
		dirPath = "/"
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	entries, err := s.mgr.ListDir(ctx, name, dirPath)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	clean, _ := containers.SanitizeContainerPathAllowRoot(dirPath)
	writeJSON(w, http.StatusOK, map[string]any{
		"path":    clean,
		"entries": entries,
	})
}

type writeFileBody struct {
	Path       string `json:"path"`
	Content    string `json:"content"`
	ContentB64 string `json:"content_b64"`
}

func (s *Server) handleWriteFile(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	var body writeFileBody
	if err := decodeJSON(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	data, err := decodeFileContent(body.Content, body.ContentB64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()
	if err := s.mgr.WriteFile(ctx, name, body.Path, data); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type mkdirBody struct {
	Path string `json:"path"`
}

func (s *Server) handleMkdir(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	var body mkdirBody
	if err := decodeJSON(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	if err := s.mgr.Mkdir(ctx, name, body.Path); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type renameBody struct {
	From string `json:"from"`
	To   string `json:"to"`
}

func (s *Server) handleRenameFile(w http.ResponseWriter, r *http.Request) {
	s.renameOrMove(w, r)
}

func (s *Server) handleMoveFile(w http.ResponseWriter, r *http.Request) {
	s.renameOrMove(w, r)
}

func (s *Server) renameOrMove(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	var body renameBody
	if err := decodeJSON(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	if err := s.mgr.RenameFile(ctx, name, body.From, body.To); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleDeleteFile(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	filePath := r.URL.Query().Get("path")
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	if err := s.mgr.DeleteFile(ctx, name, filePath); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleDownloadFile(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	filePath := r.URL.Query().Get("path")
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()
	data, err := s.mgr.ReadFile(ctx, name, filePath)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	clean, _ := containers.SanitizeContainerPath(filePath)
	filename := path.Base(clean)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", `attachment; filename="`+sanitizeFilename(filename)+`"`)
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (s *Server) handleUploadFile(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes+1<<20)
	if err := r.ParseMultipartForm(maxUploadBytes + 1<<20); err != nil {
		writeErr(w, http.StatusBadRequest, "upload too large or invalid multipart")
		return
	}
	filePath := strings.TrimSpace(r.FormValue("path"))
	if filePath == "" {
		writeErr(w, http.StatusBadRequest, "path is required")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeErr(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()
	_ = header
	data, err := io.ReadAll(io.LimitReader(file, maxUploadBytes+1))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "failed to read upload")
		return
	}
	if len(data) > maxUploadBytes {
		writeErr(w, http.StatusRequestEntityTooLarge, "file too large (max 32MB)")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()
	if err := s.mgr.WriteFile(ctx, name, filePath, data); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type extractBody struct {
	Archive string `json:"archive"`
	Dest    string `json:"dest"`
}

func (s *Server) handleExtractArchive(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	var body extractBody
	if err := decodeJSON(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if body.Dest == "" {
		body.Dest = path.Dir(body.Archive)
		if body.Dest == "" {
			body.Dest = "/"
		}
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()
	if err := s.mgr.ExtractArchive(ctx, name, body.Archive, body.Dest); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func decodeFileContent(text, b64 string) ([]byte, error) {
	if b64 != "" {
		data, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			return nil, err
		}
		if len(data) > maxUploadBytes {
			return nil, errFileTooLarge
		}
		return data, nil
	}
	data := []byte(text)
	if len(data) > maxUploadBytes {
		return nil, errFileTooLarge
	}
	if !utf8.Valid(data) {
		return nil, errInvalidUTF8
	}
	return data, nil
}

var (
	errFileTooLarge = &simpleError{"file too large (max 32MB)"}
	errInvalidUTF8  = &simpleError{"content must be valid UTF-8 or use content_b64"}
)

type simpleError struct{ msg string }

func (e *simpleError) Error() string { return e.msg }

func sanitizeFilename(name string) string {
	name = strings.ReplaceAll(name, `"`, "")
	name = strings.ReplaceAll(name, "\n", "")
	name = strings.ReplaceAll(name, "\r", "")
	if name == "" || name == "." || name == ".." {
		return "download"
	}
	return name
}
