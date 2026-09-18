package containers

import (
	"context"
	"io"
)

// CreateRequest defines validated container creation parameters.
// Callers must never pass raw user shell input.
type CreateRequest struct {
	Name     string
	Image    string // e.g. "ubuntu/24.04"
	MemoryMB int
	CPULimit int // percent of one core; 400 = 4 cores
	DiskMB   int
}

type Status string

const (
	StatusRunning  Status = "RUNNING"
	StatusStopped  Status = "STOPPED"
	StatusStarting Status = "STARTING"
	StatusStopping Status = "STOPPING"
	StatusError    Status = "ERROR"
	StatusUnknown  Status = "UNKNOWN"
)

type Stats struct {
	CPUPercent       float64 `json:"cpu_percent"`
	MemoryUsageBytes int64   `json:"memory_usage_bytes"`
	MemoryLimitBytes int64   `json:"memory_limit_bytes"`
	DiskUsageBytes   int64   `json:"disk_usage_bytes"`
	NetworkRxBytes   int64   `json:"network_rx_bytes"`
	NetworkTxBytes   int64   `json:"network_tx_bytes"`
}

type HostMetrics struct {
	CPUPercent       float64 `json:"cpu_percent"`
	MemoryUsageBytes int64   `json:"memory_usage_bytes"`
	MemoryTotalBytes int64   `json:"memory_total_bytes"`
	DiskUsageBytes   int64   `json:"disk_usage_bytes"`
	DiskTotalBytes   int64   `json:"disk_total_bytes"`
	Containers       int     `json:"containers"`
}

type BackupInfo struct {
	Name      string `json:"name"`
	CreatedAt string `json:"created_at,omitempty"`
	SizeBytes int64  `json:"size_bytes"`
}

// ContainerManager abstracts the runtime (Incus today).
type ContainerManager interface {
	CreateContainer(ctx context.Context, req CreateRequest) error
	DeleteContainer(ctx context.Context, name string) error
	StartContainer(ctx context.Context, name string) error
	StopContainer(ctx context.Context, name string) error
	RestartContainer(ctx context.Context, name string) error
	KillContainer(ctx context.Context, name string) error
	GetContainerStatus(ctx context.Context, name string) (Status, error)
	GetContainerStats(ctx context.Context, name string) (*Stats, error)
	ExecuteCommand(ctx context.Context, name string, command []string) (stdout string, stderr string, err error)
	ReadFile(ctx context.Context, name string, path string) ([]byte, error)
	WriteFile(ctx context.Context, name string, path string, data []byte) error
	DeleteFile(ctx context.Context, name string, path string) error
	HostMetrics(ctx context.Context) (*HostMetrics, error)
	CreateBackup(ctx context.Context, containerName, backupName string) (*BackupInfo, error)
	ListBackups(ctx context.Context, containerName string) ([]BackupInfo, error)
	GetBackup(ctx context.Context, containerName, backupName string) (*BackupInfo, error)
	DeleteBackup(ctx context.Context, containerName, backupName string) error
	DownloadBackup(ctx context.Context, containerName, backupName string, w io.Writer) (int64, error)
	RestoreBackup(ctx context.Context, containerName, backupName string) error
}
