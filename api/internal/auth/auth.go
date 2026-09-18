package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hostrix/hostrix/api/internal/config"
	"github.com/hostrix/hostrix/api/internal/models"
	"github.com/hostrix/hostrix/api/internal/users"
	"gorm.io/gorm"
)

var ErrUnauthorized = errors.New("unauthorized")

type Service struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewService(db *gorm.DB, cfg *config.Config) *Service {
	return &Service{db: db, cfg: cfg}
}

type LoginResult struct {
	User      *models.User
	Token     string
	ExpiresAt time.Time
}

func (s *Service) Login(login, password, ip, userAgent string) (*LoginResult, error) {
	user, err := users.FindByLogin(s.db, login)
	if err != nil {
		return nil, users.ErrInvalidCredentials
	}
	if !users.CheckPassword(user.PasswordHash, password) {
		return nil, users.ErrInvalidCredentials
	}

	token, err := randomToken(32)
	if err != nil {
		return nil, err
	}

	expires := time.Now().UTC().Add(s.cfg.SessionTTL)
	session := models.Session{
		UserID:    user.ID,
		TokenHash: hashToken(token),
		ExpiresAt: expires,
		IP:        truncate(ip, 45),
		UserAgent: truncate(userAgent, 512),
	}
	if err := s.db.Create(&session).Error; err != nil {
		return nil, err
	}

	return &LoginResult{User: user, Token: token, ExpiresAt: expires}, nil
}

func (s *Service) Logout(token string) error {
	if token == "" {
		return nil
	}
	return s.db.Where("token_hash = ?", hashToken(token)).Delete(&models.Session{}).Error
}

func (s *Service) Authenticate(r *http.Request) (*models.User, error) {
	token := s.tokenFromRequest(r)
	if token == "" {
		return nil, ErrUnauthorized
	}

	var session models.Session
	err := s.db.Preload("User").Where("token_hash = ?", hashToken(token)).First(&session).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUnauthorized
	}
	if err != nil {
		return nil, err
	}
	if time.Now().UTC().After(session.ExpiresAt) {
		_ = s.db.Delete(&session).Error
		return nil, ErrUnauthorized
	}
	if session.User.ID == 0 {
		return nil, ErrUnauthorized
	}
	return &session.User, nil
}

func (s *Service) SetSessionCookie(w http.ResponseWriter, token string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     s.cfg.CookieName,
		Value:    token,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		Secure:   s.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (s *Service) ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     s.cfg.CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   s.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (s *Service) tokenFromRequest(r *http.Request) string {
	if c, err := r.Cookie(s.cfg.CookieName); err == nil && c.Value != "" {
		return c.Value
	}
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		return strings.TrimSpace(auth[7:])
	}
	return ""
}

func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("rand: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
