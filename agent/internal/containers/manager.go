package containers

import "context"

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

// DirEntry is a single entry returned by ListDir.
type DirEntry struct {
	Name string `json:"name"`
	Type string `json:"type"` // file, directory, symlink
	Mode int    `json:"mode"`
	Size int64  `json:"size"`
}

// MaxFileBytes caps read/write/upload payloads (32 MiB).
const MaxFileBytes = 32 << 20

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
	ListDir(ctx context.Context, name string, path string) ([]DirEntry, error)
	Mkdir(ctx context.Context, name string, path string) error
	RenameFile(ctx context.Context, name string, from string, to string) error
	ExtractArchive(ctx context.Context, name string, archivePath string, destDir string) error
	HostMetrics(ctx context.Context) (*HostMetrics, error)
}
