package httpapi

import (
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/hostrix/hostrix/api/internal/auth"
	"github.com/hostrix/hostrix/api/internal/config"
	"github.com/hostrix/hostrix/api/internal/models"
	"github.com/hostrix/hostrix/api/internal/users"
	"golang.org/x/time/rate"
	"gorm.io/gorm"
)

type Server struct {
	cfg         *config.Config
	db          *gorm.DB
	auth        *auth.Service
	httpServer  *http.Server
	loginLimit  *ipRateLimiter
}

func NewServer(cfg *config.Config, db *gorm.DB) *Server {
	s := &Server{
		cfg:        cfg,
		db:         db,
		auth:       auth.NewService(db, cfg),
		loginLimit: newIPRateLimiter(rate.Every(time.Minute/10), 10),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", s.handleHealth)
	mux.HandleFunc("POST /api/v1/auth/login", s.handleLogin)
	mux.HandleFunc("POST /api/v1/auth/logout", s.handleLogout)
	mux.HandleFunc("GET /api/v1/auth/me", s.handleMe)

	s.httpServer = &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           s.withMiddleware(mux),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	return s
}

func (s *Server) ListenAndServe() error {
	return s.httpServer.ListenAndServe()
}

func (s *Server) Close() error {
	return s.httpServer.Close()
}

func (s *Server) withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")

		origin := r.Header.Get("Origin")
		if origin == s.cfg.PanelOrigin || origin == "" {
			if origin != "" {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			}
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	sqlDB, err := s.db.DB()
	status := "ok"
	code := http.StatusOK
	if err != nil || sqlDB.Ping() != nil {
		status = "degraded"
		code = http.StatusServiceUnavailable
	}
	writeJSON(w, code, map[string]any{
		"status":  status,
		"service": "hostrix-api",
		"version": "0.1.0",
	})
}

type loginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)
	limiter := s.loginLimit.get(ip)
	if !limiter.Allow() {
		writeError(w, http.StatusTooManyRequests, "too many login attempts")
		return
	}

	var req loginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	req.Login = strings.TrimSpace(req.Login)
	if req.Login == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "login and password are required")
		return
	}

	result, err := s.auth.Login(req.Login, req.Password, ip, r.UserAgent())
	if errors.Is(err, users.ErrInvalidCredentials) {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "login failed")
		return
	}

	s.auth.SetSessionCookie(w, result.Token, result.ExpiresAt)
	writeJSON(w, http.StatusOK, map[string]any{
		"user":       publicUser(result.User),
		"expires_at": result.ExpiresAt.UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	token := ""
	if c, err := r.Cookie(s.cfg.CookieName); err == nil {
		token = c.Value
	}
	_ = s.auth.Logout(token)
	s.auth.ClearSessionCookie(w)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	user, err := s.auth.Authenticate(r)
	if errors.Is(err, auth.ErrUnauthorized) {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "auth failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": publicUser(user)})
}

func publicUser(u *models.User) map[string]any {
	return map[string]any{
		"uuid":     u.UUID,
		"username": u.Username,
		"email":    u.Email,
		"is_admin": u.IsAdmin,
	}
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

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

type ipRateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	r        rate.Limit
	b        int
}

func newIPRateLimiter(r rate.Limit, b int) *ipRateLimiter {
	return &ipRateLimiter{
		limiters: make(map[string]*rate.Limiter),
		r:        r,
		b:        b,
	}
}

func (i *ipRateLimiter) get(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()
	if lim, ok := i.limiters[ip]; ok {
		return lim
	}
	lim := rate.NewLimiter(i.r, i.b)
	i.limiters[ip] = lim
	return lim
}
