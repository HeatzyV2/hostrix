package httpapi

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/hostrix/hostrix/api/internal/agentclient"
	"github.com/hostrix/hostrix/api/internal/servers"
)

const maxUploadBytes = 32 << 20

func (s *Server) handleListServerFiles(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	srv, node, ok := s.loadAccessibleServer(w, r, user, servers.ActionFiles)
	if !ok {
		return
	}
	dirPath := r.URL.Query().Get("path")
	if dirPath == "" {
		dirPath = "/"
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	client := agentclient.New(node.Address, node.Port, node.Token)
	out, err := client.ListFiles(ctx, srv.ContainerName, dirPath)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

type writeServerFileBody struct {
	Path       string `json:"path"`
	Content    string `json:"content"`
	ContentB64 string `json:"content_b64"`
}

func (s *Server) handleWriteServerFile(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	srv, node, ok := s.loadAccessibleServer(w, r, user, servers.ActionFiles)
	if !ok {
		return
	}
	var body writeServerFileBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if strings.TrimSpace(body.Path) == "" {
		writeError(w, http.StatusBadRequest, "path is required")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()
	client := agentclient.New(node.Address, node.Port, node.Token)

	if body.ContentB64 != "" {
		data, err := base64.StdEncoding.DecodeString(body.ContentB64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid content_b64")
			return
		}
		if len(data) > maxUploadBytes {
			writeError(w, http.StatusRequestEntityTooLarge, "file too large (max 32MB)")
			return
		}
		if err := client.WriteFileBytes(ctx, srv.ContainerName, body.Path, data); err != nil {
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}
	} else {
		if len(body.Content) > maxUploadBytes {
			writeError(w, http.StatusRequestEntityTooLarge, "file too large (max 32MB)")
			return
		}
		if !utf8.ValidString(body.Content) {
			writeError(w, http.StatusBadRequest, "content must be valid UTF-8 or use content_b64")
			return
		}
		if err := client.WriteFile(ctx, srv.ContainerName, body.Path, body.Content); err != nil {
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type mkdirServerBody struct {
	Path string `json:"path"`
}

func (s *Server) handleMkdirServerFile(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	srv, node, ok := s.loadAccessibleServer(w, r, user, servers.ActionFiles)
	if !ok {
		return
	}
	var body mkdirServerBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	client := agentclient.New(node.Address, node.Port, node.Token)
	if err := client.Mkdir(ctx, srv.ContainerName, body.Path); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type renameServerBody struct {
	From string `json:"from"`
	To   string `json:"to"`
}

func (s *Server) handleRenameServerFile(w http.ResponseWriter, r *http.Request) {
	s.handleRenameOrMoveServerFile(w, r, false)
}

func (s *Server) handleMoveServerFile(w http.ResponseWriter, r *http.Request) {
	s.handleRenameOrMoveServerFile(w, r, true)
}

func (s *Server) handleRenameOrMoveServerFile(w http.ResponseWriter, r *http.Request, move bool) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	srv, node, ok := s.loadAccessibleServer(w, r, user, servers.ActionFiles)
	if !ok {
		return
	}
	var body renameServerBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	client := agentclient.New(node.Address, node.Port, node.Token)
	var err error
	if move {
		err = client.MoveFile(ctx, srv.ContainerName, body.From, body.To)
	} else {
		err = client.RenameFile(ctx, srv.ContainerName, body.From, body.To)
	}
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleDeleteServerFile(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	srv, node, ok := s.loadAccessibleServer(w, r, user, servers.ActionFiles)
	if !ok {
		return
	}
	filePath := r.URL.Query().Get("path")
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	client := agentclient.New(node.Address, node.Port, node.Token)
	if err := client.DeleteFile(ctx, srv.ContainerName, filePath); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleDownloadServerFile(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	srv, node, ok := s.loadAccessibleServer(w, r, user, servers.ActionFiles)
	if !ok {
		return
	}
	filePath := r.URL.Query().Get("path")
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()
	client := agentclient.New(node.Address, node.Port, node.Token)
	data, disposition, err := client.DownloadFile(ctx, srv.ContainerName, filePath)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	filename := path.Base(filePath)
	if filename == "" || filename == "." || filename == "/" {
		filename = "download"
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	if disposition != "" {
		w.Header().Set("Content-Disposition", disposition)
	} else {
		w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (s *Server) handleUploadServerFile(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	srv, node, ok := s.loadAccessibleServer(w, r, user, servers.ActionFiles)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes+1<<20)
	if err := r.ParseMultipartForm(maxUploadBytes + 1<<20); err != nil {
		writeError(w, http.StatusBadRequest, "upload too large or invalid multipart")
		return
	}
	filePath := strings.TrimSpace(r.FormValue("path"))
	if filePath == "" {
		writeError(w, http.StatusBadRequest, "path is required")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()
	filename := "upload"
	if header != nil && header.Filename != "" {
		filename = path.Base(header.Filename)
	}
	limited := io.LimitReader(file, maxUploadBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read upload")
		return
	}
	if len(data) > maxUploadBytes {
		writeError(w, http.StatusRequestEntityTooLarge, "file too large (max 32MB)")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()
	client := agentclient.New(node.Address, node.Port, node.Token)
	if err := client.UploadFile(ctx, srv.ContainerName, filePath, filename, bytes.NewReader(data)); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type extractServerBody struct {
	Archive string `json:"archive"`
	Dest    string `json:"dest"`
}

func (s *Server) handleExtractServerFile(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	srv, node, ok := s.loadAccessibleServer(w, r, user, servers.ActionFiles)
	if !ok {
		return
	}
	var body extractServerBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
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
	client := agentclient.New(node.Address, node.Port, node.Token)
	if err := client.ExtractArchive(ctx, srv.ContainerName, body.Archive, body.Dest); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
