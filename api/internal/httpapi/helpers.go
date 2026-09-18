package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/hostrix/hostrix/api/internal/auth"
	"github.com/hostrix/hostrix/api/internal/models"
	"github.com/hostrix/hostrix/api/internal/servers"
)

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

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

// logAuthFail appends a Fail2Ban-friendly line for login abuse.
func logAuthFail(ip, reason string) {
	line := fmt.Sprintf("%s auth_fail ip=%s reason=%s\n", time.Now().UTC().Format(time.RFC3339), ip, reason)
	path := "/var/log/hostrix/auth-fail.log"
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o640)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.WriteString(line)
}


func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func publicUser(u *models.User) map[string]any {
	return map[string]any{
		"uuid":     u.UUID,
		"username": u.Username,
		"email":    u.Email,
		"is_admin": u.IsAdmin,
	}
}

func publicNode(n *models.Node) map[string]any {
	return map[string]any{
		"uuid":               n.UUID,
		"name":               n.Name,
		"hostname":           n.Hostname,
		"address":            n.Address,
		"port":               n.Port,
		"status":             n.Status,
		"last_heartbeat_at":  n.LastHeartbeatAt,
		"cpu_percent":        n.CPUPercent,
		"memory_usage_bytes": n.MemoryUsageBytes,
		"memory_total_bytes": n.MemoryTotalBytes,
		"disk_usage_bytes":   n.DiskUsageBytes,
		"disk_total_bytes":   n.DiskTotalBytes,
		"container_count":    n.ContainerCount,
		"created_at":         n.CreatedAt,
	}
}

func publicServer(s *models.Server) map[string]any {
	return map[string]any{
		"uuid":           s.UUID,
		"name":           s.Name,
		"node_id":        s.NodeID,
		"template_id":    s.TemplateID,
		"container_name": s.ContainerName,
		"memory":         s.MemoryMB,
		"cpu":            s.CPULimit,
		"disk":           s.DiskMB,
		"status":         s.Status,
		"created_at":     s.CreatedAt,
	}
}

func publicServerView(v *servers.ServerView) map[string]any {
	out := publicServer(&v.Server)
	out["node_uuid"] = v.NodeUUID
	out["node_name"] = v.NodeName
	out["node_status"] = v.NodeStatus
	return out
}

func publicBackup(b *models.Backup) map[string]any {
	return map[string]any{
		"uuid":       b.UUID,
		"server_id":  b.ServerID,
		"name":       b.Name,
		"size_bytes": b.SizeBytes,
		"status":     b.Status,
		"created_at": b.CreatedAt,
		"updated_at": b.UpdatedAt,
	}
}

func (s *Server) requireUser(w http.ResponseWriter, r *http.Request) (*models.User, bool) {
	user, err := s.auth.Authenticate(r)
	if errors.Is(err, auth.ErrUnauthorized) {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return nil, false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "auth failed")
		return nil, false
	}
	return user, true
}

func (s *Server) requireAdmin(w http.ResponseWriter, r *http.Request) (*models.User, bool) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return nil, false
	}
	if !user.IsAdmin {
		writeError(w, http.StatusForbidden, "admin required")
		return nil, false
	}
	return user, true
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(strings.ToLower(h), "bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return ""
}
