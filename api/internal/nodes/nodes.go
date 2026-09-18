package nodes

import (
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hostrix/hostrix/api/internal/models"
	"github.com/hostrix/hostrix/api/internal/security"
	"gorm.io/gorm"
)

var (
	ErrNotFound    = errors.New("node not found")
	ErrUnauthorized = errors.New("unauthorized")
)

type CreateInput struct {
	Name     string
	Hostname string
	Address  string
	Port     int
}

type CreateResult struct {
	Node  *models.Node
	Token string // plaintext, returned once to admin
}

func Create(db *gorm.DB, in CreateInput) (*CreateResult, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Hostname = strings.TrimSpace(in.Hostname)
	in.Address = strings.TrimSpace(in.Address)
	if in.Name == "" || in.Hostname == "" || in.Address == "" {
		return nil, fmt.Errorf("name, hostname and address are required")
	}
	if in.Port <= 0 || in.Port > 65535 {
		in.Port = 8081
	}
	token, err := security.RandomToken(32)
	if err != nil {
		return nil, err
	}
	node := &models.Node{
		UUID:     uuid.NewString(),
		Name:     in.Name,
		Hostname: in.Hostname,
		Address:  in.Address,
		Port:     in.Port,
		Token:    token,
		Status:   "INSTALLING",
	}
	if err := db.Create(node).Error; err != nil {
		return nil, err
	}
	return &CreateResult{Node: node, Token: token}, nil
}

func List(db *gorm.DB) ([]models.Node, error) {
	var list []models.Node
	err := db.Order("id asc").Find(&list).Error
	return list, err
}

func GetByUUID(db *gorm.DB, id string) (*models.Node, error) {
	var node models.Node
	err := db.Where("uuid = ?", id).First(&node).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &node, err
}

func Delete(db *gorm.DB, id string) error {
	res := db.Where("uuid = ?", id).Delete(&models.Node{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

type HeartbeatInput struct {
	Status           string
	CPUPercent       float64
	MemoryUsageBytes int64
	MemoryTotalBytes int64
	DiskUsageBytes   int64
	DiskTotalBytes   int64
	Containers       int
}

func ApplyHeartbeat(db *gorm.DB, nodeUUID, token string, in HeartbeatInput) error {
	var node models.Node
	if err := db.Where("uuid = ?", nodeUUID).First(&node).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}
	if !tokensEqual(token, node.Token) {
		return ErrUnauthorized
	}
	now := time.Now().UTC()
	status := strings.ToUpper(strings.TrimSpace(in.Status))
	if status == "" {
		status = "ONLINE"
	}
	return db.Model(&node).Updates(map[string]any{
		"status":             status,
		"last_heartbeat_at":  now,
		"cpu_percent":        in.CPUPercent,
		"memory_usage_bytes": in.MemoryUsageBytes,
		"memory_total_bytes": in.MemoryTotalBytes,
		"disk_usage_bytes":   in.DiskUsageBytes,
		"disk_total_bytes":   in.DiskTotalBytes,
		"container_count":    in.Containers,
	}).Error
}

func RotateToken(db *gorm.DB, nodeUUID string) (string, error) {
	node, err := GetByUUID(db, nodeUUID)
	if err != nil {
		return "", err
	}
	token, err := security.RandomToken(32)
	if err != nil {
		return "", err
	}
	if err := db.Model(node).Updates(map[string]any{
		"token":  token,
		"status": "INSTALLING",
	}).Error; err != nil {
		return "", err
	}
	return token, nil
}

func tokensEqual(a, b string) bool {
	ha := sha256.Sum256([]byte(a))
	hb := sha256.Sum256([]byte(b))
	return subtle.ConstantTimeCompare(ha[:], hb[:]) == 1
}

func MarkStaleOffline(db *gorm.DB, olderThan time.Duration) error {
	cutoff := time.Now().UTC().Add(-olderThan)
	return db.Model(&models.Node{}).
		Where("status = ? AND (last_heartbeat_at IS NULL OR last_heartbeat_at < ?)", "ONLINE", cutoff).
		Update("status", "OFFLINE").Error
}
