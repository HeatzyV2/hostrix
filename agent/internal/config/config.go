package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr      string
	APIURL        string
	NodeUUID      string
	NodeToken     string
	IncusSocket   string
	HeartbeatEvery time.Duration
}

func Load() (*Config, error) {
	cfg := &Config{
		HTTPAddr:       getEnv("HOSTRIX_AGENT_ADDR", ":8081"),
		APIURL:         strings.TrimRight(getEnv("HOSTRIX_API_URL", "http://127.0.0.1:8080"), "/"),
		NodeUUID:       strings.TrimSpace(getEnv("HOSTRIX_NODE_UUID", "")),
		NodeToken:      strings.TrimSpace(getEnv("HOSTRIX_NODE_TOKEN", "")),
		IncusSocket:    getEnv("HOSTRIX_INCUS_SOCKET", ""),
		HeartbeatEvery: time.Duration(getEnvInt("HOSTRIX_HEARTBEAT_SECONDS", 15)) * time.Second,
	}
	if cfg.NodeUUID == "" {
		return nil, fmt.Errorf("HOSTRIX_NODE_UUID is required")
	}
	if cfg.NodeToken == "" {
		return nil, fmt.Errorf("HOSTRIX_NODE_TOKEN is required")
	}
	if cfg.APIURL == "" {
		return nil, fmt.Errorf("HOSTRIX_API_URL is required")
	}
	if cfg.HeartbeatEvery < 5*time.Second {
		cfg.HeartbeatEvery = 5 * time.Second
	}
	return cfg, nil
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
