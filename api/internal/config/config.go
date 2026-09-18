package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr string
	PanelOrigin string

	DBHost     string
	DBPort     int
	DBUser     string
	DBPassword string
	DBName     string

	SessionTTL   time.Duration
	BcryptCost   int
	CookieSecure bool
	CookieName   string

	BootstrapAdminUsername string
	BootstrapAdminEmail    string
	BootstrapAdminPassword string
}

func Load() (*Config, error) {
	cfg := &Config{
		HTTPAddr:               getEnv("HOSTRIX_HTTP_ADDR", ":8080"),
		PanelOrigin:            strings.TrimRight(getEnv("HOSTRIX_PANEL_ORIGIN", "http://localhost:3000"), "/"),
		DBHost:                 getEnv("HOSTRIX_DB_HOST", "127.0.0.1"),
		DBPort:                 getEnvInt("HOSTRIX_DB_PORT", 3306),
		DBUser:                 getEnv("HOSTRIX_DB_USER", "hostrix"),
		DBPassword:             getEnv("HOSTRIX_DB_PASSWORD", "change-me"),
		DBName:                 getEnv("HOSTRIX_DB_NAME", "hostrix"),
		SessionTTL:             time.Duration(getEnvInt("HOSTRIX_SESSION_TTL_HOURS", 72)) * time.Hour,
		BcryptCost:             getEnvInt("HOSTRIX_BCRYPT_COST", 12),
		CookieSecure:           getEnvBool("HOSTRIX_COOKIE_SECURE", false),
		CookieName:             getEnv("HOSTRIX_COOKIE_NAME", "hostrix_session"),
		BootstrapAdminUsername: getEnv("HOSTRIX_BOOTSTRAP_ADMIN_USERNAME", "admin"),
		BootstrapAdminEmail:    getEnv("HOSTRIX_BOOTSTRAP_ADMIN_EMAIL", "admin@localhost"),
		BootstrapAdminPassword: getEnv("HOSTRIX_BOOTSTRAP_ADMIN_PASSWORD", "changeme"),
	}

	if cfg.BcryptCost < 10 || cfg.BcryptCost > 14 {
		return nil, fmt.Errorf("HOSTRIX_BCRYPT_COST must be between 10 and 14")
	}
	if cfg.DBPassword == "" {
		return nil, fmt.Errorf("HOSTRIX_DB_PASSWORD is required")
	}
	return cfg, nil
}

func (c *Config) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=UTC",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func getEnvBool(key string, fallback bool) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if v == "" {
		return fallback
	}
	return v == "1" || v == "true" || v == "yes"
}
