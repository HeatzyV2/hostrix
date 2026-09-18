package httpapi

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hostrix/hostrix/agent/internal/console"
	incusmgr "github.com/hostrix/hostrix/agent/internal/incus"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true // agent is not browser-facing; API proxies
	},
}

func (s *Server) handleConsoleWS(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	cols, _ := strconv.Atoi(r.URL.Query().Get("cols"))
	rows, _ := strconv.Atoi(r.URL.Query().Get("rows"))

	mgr, ok := s.mgr.(*incusmgr.Manager)
	if !ok {
		writeErr(w, http.StatusInternalServerError, "console unavailable")
		return
	}
	srv, err := mgr.Connect()
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("console upgrade: %v", err)
		return
	}
	defer conn.Close()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	if err := console.AttachInteractiveShell(ctx, srv, name, cols, rows, conn); err != nil {
		log.Printf("console session %s: %v", name, err)
	}
}

func (s *Server) applyLongLivedTimeouts() {
	// WebSockets require disabling per-request read/write deadlines on the server.
	s.http.ReadTimeout = 0
	s.http.WriteTimeout = 0
	s.http.IdleTimeout = 120 * time.Second
}
