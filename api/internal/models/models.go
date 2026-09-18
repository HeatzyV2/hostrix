package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID           uint64         `gorm:"primaryKey" json:"id"`
	UUID         string         `gorm:"size:36;uniqueIndex;not null" json:"uuid"`
	Username     string         `gorm:"size:64;uniqueIndex;not null" json:"username"`
	Email        string         `gorm:"size:255;uniqueIndex;not null" json:"email"`
	PasswordHash string         `gorm:"size:255;not null" json:"-"`
	IsAdmin      bool           `gorm:"not null;default:false" json:"is_admin"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

type Session struct {
	ID         uint64    `gorm:"primaryKey" json:"id"`
	UserID     uint64    `gorm:"index;not null" json:"user_id"`
	TokenHash  string    `gorm:"size:64;uniqueIndex;not null" json:"-"`
	ExpiresAt  time.Time `gorm:"index;not null" json:"expires_at"`
	IP         string    `gorm:"size:45" json:"ip"`
	UserAgent  string    `gorm:"size:512" json:"user_agent"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	User       User      `gorm:"constraint:OnDelete:CASCADE" json:"-"`
}

type Setting struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	Key       string    `gorm:"size:128;uniqueIndex;not null" json:"key"`
	Value     string    `gorm:"type:text;not null" json:"value"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Node struct {
	ID               uint64     `gorm:"primaryKey" json:"id"`
	UUID             string     `gorm:"size:36;uniqueIndex;not null" json:"uuid"`
	Name             string     `gorm:"size:128;not null" json:"name"`
	Hostname         string     `gorm:"size:255;not null" json:"hostname"`
	Address          string     `gorm:"size:255;not null" json:"address"`
	Port             int        `gorm:"not null;default:8081" json:"port"`
	Token            string     `gorm:"size:128;not null" json:"-"` // node secret; never exposed via API JSON
	Status           string     `gorm:"size:32;not null;default:OFFLINE" json:"status"`
	LastHeartbeatAt  *time.Time `json:"last_heartbeat_at"`
	CPUPercent       float64    `gorm:"not null;default:0" json:"cpu_percent"`
	MemoryUsageBytes int64      `gorm:"not null;default:0" json:"memory_usage_bytes"`
	MemoryTotalBytes int64      `gorm:"not null;default:0" json:"memory_total_bytes"`
	DiskUsageBytes   int64      `gorm:"not null;default:0" json:"disk_usage_bytes"`
	DiskTotalBytes   int64      `gorm:"not null;default:0" json:"disk_total_bytes"`
	ContainerCount   int        `gorm:"not null;default:0" json:"container_count"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type ServerTemplate struct {
	ID             uint64    `gorm:"primaryKey" json:"id"`
	UUID           string    `gorm:"size:36;uniqueIndex;not null" json:"uuid"`
	Name           string    `gorm:"size:128;not null" json:"name"`
	Slug           string    `gorm:"size:128;uniqueIndex;not null" json:"slug"`
	Description    string    `gorm:"type:text" json:"description"`
	Image          string    `gorm:"size:255;not null" json:"image"`
	StartupCommand string    `gorm:"type:text" json:"startup_command"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type Server struct {
	ID            uint64    `gorm:"primaryKey" json:"id"`
	UUID          string    `gorm:"size:36;uniqueIndex;not null" json:"uuid"`
	Name          string    `gorm:"size:128;not null" json:"name"`
	NodeID        uint64    `gorm:"index;not null" json:"node_id"`
	TemplateID    uint64    `gorm:"index;not null" json:"template_id"`
	OwnerID       uint64    `gorm:"index;not null" json:"owner_id"`
	ContainerName string    `gorm:"size:128;uniqueIndex;not null" json:"container_name"`
	MemoryMB      int       `gorm:"not null" json:"memory"`
	CPULimit      int       `gorm:"not null" json:"cpu"`
	DiskMB        int       `gorm:"not null" json:"disk"`
	Status        string    `gorm:"size:32;not null;default:INSTALLING" json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Allocation struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	NodeID    uint64    `gorm:"index;not null" json:"node_id"`
	ServerID  *uint64   `gorm:"index" json:"server_id"`
	IP        string    `gorm:"size:45;not null" json:"ip"`
	Port      int       `gorm:"not null" json:"port"`
	CreatedAt time.Time `json:"created_at"`
}

type Backup struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	UUID      string    `gorm:"size:36;uniqueIndex;not null" json:"uuid"`
	ServerID  uint64    `gorm:"index;not null" json:"server_id"`
	Name      string    `gorm:"size:255;not null" json:"name"`
	SizeBytes int64     `gorm:"not null;default:0" json:"size_bytes"`
	Status    string    `gorm:"size:32;not null;default:PENDING" json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ServerPermission struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	ServerID  uint64    `gorm:"uniqueIndex:idx_server_user;not null" json:"server_id"`
	UserID    uint64    `gorm:"uniqueIndex:idx_server_user;not null" json:"user_id"`
	CanStart  bool      `gorm:"not null;default:false" json:"can_start"`
	CanStop   bool      `gorm:"not null;default:false" json:"can_stop"`
	CanFiles  bool      `gorm:"not null;default:false" json:"can_files"`
	CanConsole bool     `gorm:"not null;default:false" json:"can_console"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
