package httpapi

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hostrix/hostrix/api/internal/agentclient"
	"github.com/hostrix/hostrix/api/internal/models"
	"github.com/hostrix/hostrix/api/internal/servers"
)

var consoleUpgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (s *Server) handleConsoleTicket(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	srv, node, ok := s.loadAccessibleServer(w, r, user, servers.ActionConsole)
	if !ok {
		return
	}
	_ = node
	ticket, exp, err := s.tickets.Issue(srv.UUID, user.ID, 2*time.Minute)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to issue ticket")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ticket":     ticket,
		"expires_at": exp.UTC().Format(time.RFC3339),
		"ws_path":    "/api/v1/servers/" + srv.UUID + "/console/ws",
	})
}

func (s *Server) handleConsoleWS(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ticketID := r.URL.Query().Get("ticket")
	if ticketID == "" {
		writeError(w, http.StatusUnauthorized, "ticket required")
		return
	}
	ticket, ok := s.tickets.Consume(ticketID, id)
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid or expired ticket")
		return
	}

	var srv models.Server
	if err := s.db.Where("uuid = ?", id).First(&srv).Error; err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if srv.OwnerID != ticket.UserID {
		// allow if user is still admin — ticket already bound to user id at issue time
	}

	var node models.Node
	if err := s.db.First(&node, srv.NodeID).Error; err != nil {
		writeError(w, http.StatusBadGateway, "node unavailable")
		return
	}

	cols, _ := strconv.Atoi(r.URL.Query().Get("cols"))
	rows, _ := strconv.Atoi(r.URL.Query().Get("rows"))

	clientConn, err := consoleUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("console upgrade: %v", err)
		return
	}
	defer clientConn.Close()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	agent := agentclient.New(node.Address, node.Port, node.Token)
	agentConn, _, err := agent.DialConsole(ctx, srv.ContainerName, cols, rows)
	if err != nil {
		_ = clientConn.WriteMessage(websocket.TextMessage, []byte("console error: "+err.Error()+"\r\n"))
		return
	}
	defer agentConn.Close()

	errCh := make(chan error, 2)
	go proxyWS(clientConn, agentConn, errCh)
	go proxyWS(agentConn, clientConn, errCh)
	<-errCh
}

func proxyWS(dst, src *websocket.Conn, errCh chan<- error) {
	for {
		msgType, data, err := src.ReadMessage()
		if err != nil {
			errCh <- err
			return
		}
		if err := dst.WriteMessage(msgType, data); err != nil {
			errCh <- err
			return
		}
	}
}

func (s *Server) handleServerMetrics(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	srv, node, ok := s.loadAccessibleServer(w, r, user, servers.ActionMetrics)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	client := agentclient.New(node.Address, node.Port, node.Token)
	stats, err := client.GetStats(ctx, srv.ContainerName)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"server_uuid": srv.UUID,
		"status":      srv.Status,
		"metrics":     stats,
	})
}

func (s *Server) loadAccessibleServer(w http.ResponseWriter, r *http.Request, user *models.User, action servers.Action) (*models.Server, *models.Node, bool) {
	srv, err := servers.GetByUUID(s.db, r.PathValue("id"))
	if errors.Is(err, servers.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return nil, nil, false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load server")
		return nil, nil, false
	}
	if !servers.CanPerform(s.db, user, srv, action) {
		writeError(w, http.StatusForbidden, "forbidden")
		return nil, nil, false
	}
	var node models.Node
	if err := s.db.First(&node, srv.NodeID).Error; err != nil {
		writeError(w, http.StatusBadGateway, "node unavailable")
		return nil, nil, false
	}
	return srv, &node, true
}
