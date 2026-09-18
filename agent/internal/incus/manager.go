package incusmgr

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/hostrix/hostrix/agent/internal/containers"
	incus "github.com/lxc/incus/v6/client"
	"github.com/lxc/incus/v6/shared/api"
)

// Manager implements containers.ContainerManager using the official Incus client.
type Manager struct {
	socket string
	mu     sync.Mutex
	server incus.InstanceServer
}

func New(socket string) *Manager {
	return &Manager{socket: socket}
}

func (m *Manager) connect() (incus.InstanceServer, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.server != nil {
		return m.server, nil
	}
	args := &incus.ConnectionArgs{}
	var (
		srv incus.InstanceServer
		err error
	)
	if m.socket == "" {
		srv, err = incus.ConnectIncusUnix("", args)
	} else {
		srv, err = incus.ConnectIncusUnix(m.socket, args)
	}
	if err != nil {
		return nil, fmt.Errorf("connect incus: %w", err)
	}
	m.server = srv
	return srv, nil
}

func (m *Manager) CreateContainer(ctx context.Context, req containers.CreateRequest) error {
	if err := containers.ValidateCreateRequest(req); err != nil {
		return err
	}
	srv, err := m.connect()
	if err != nil {
		return err
	}

	alias, server, protocol := parseImage(req.Image)
	cpuCores := float64(req.CPULimit) / 100.0
	if cpuCores < 0.1 {
		cpuCores = 0.1
	}

	post := api.InstancesPost{
		Name: req.Name,
		Type: api.InstanceTypeContainer,
		Source: api.InstanceSource{
			Type:     "image",
			Alias:    alias,
			Server:   server,
			Protocol: protocol,
		},
	}
	post.Config = map[string]string{
		"limits.memory": fmt.Sprintf("%dMB", req.MemoryMB),
		"limits.cpu":    fmt.Sprintf("%.2f", cpuCores),
	}
	post.Devices = map[string]map[string]string{
		"root": {
			"type": "disk",
			"path": "/",
			"pool": "default",
			"size": fmt.Sprintf("%dMB", req.DiskMB),
		},
	}

	op, err := srv.CreateInstance(post)
	if err != nil {
		return fmt.Errorf("create instance: %w", err)
	}
	return waitOp(ctx, op)
}

func (m *Manager) DeleteContainer(ctx context.Context, name string) error {
	if err := containers.ValidateContainerName(name); err != nil {
		return err
	}
	srv, err := m.connect()
	if err != nil {
		return err
	}
	// Ensure stopped first (best effort).
	_ = m.StopContainer(ctx, name)
	op, err := srv.DeleteInstance(name)
	if err != nil {
		return fmt.Errorf("delete instance: %w", err)
	}
	return waitOp(ctx, op)
}

func (m *Manager) StartContainer(ctx context.Context, name string) error {
	return m.updateState(ctx, name, "start", false)
}

func (m *Manager) StopContainer(ctx context.Context, name string) error {
	return m.updateState(ctx, name, "stop", false)
}

func (m *Manager) RestartContainer(ctx context.Context, name string) error {
	return m.updateState(ctx, name, "restart", false)
}

func (m *Manager) KillContainer(ctx context.Context, name string) error {
	return m.updateState(ctx, name, "stop", true)
}

func (m *Manager) updateState(ctx context.Context, name, action string, force bool) error {
	if err := containers.ValidateContainerName(name); err != nil {
		return err
	}
	srv, err := m.connect()
	if err != nil {
		return err
	}
	op, err := srv.UpdateInstanceState(name, api.InstanceStatePut{
		Action:  action,
		Timeout: 120,
		Force:   force,
	}, "")
	if err != nil {
		return fmt.Errorf("%s instance: %w", action, err)
	}
	return waitOp(ctx, op)
}

func (m *Manager) GetContainerStatus(ctx context.Context, name string) (containers.Status, error) {
	if err := containers.ValidateContainerName(name); err != nil {
		return containers.StatusError, err
	}
	srv, err := m.connect()
	if err != nil {
		return containers.StatusError, err
	}
	inst, _, err := srv.GetInstance(name)
	if err != nil {
		return containers.StatusError, err
	}
	switch strings.ToLower(inst.Status) {
	case "running":
		return containers.StatusRunning, nil
	case "stopped":
		return containers.StatusStopped, nil
	case "starting":
		return containers.StatusStarting, nil
	case "stopping":
		return containers.StatusStopping, nil
	default:
		return containers.StatusUnknown, nil
	}
}

func (m *Manager) Connect() (incus.InstanceServer, error) {
	return m.connect()
}

func (m *Manager) GetContainerStats(ctx context.Context, name string) (*containers.Stats, error) {
	if err := containers.ValidateContainerName(name); err != nil {
		return nil, err
	}
	srv, err := m.connect()
	if err != nil {
		return nil, err
	}

	s1, _, err := srv.GetInstanceState(name)
	if err != nil {
		return nil, err
	}
	t1 := time.Now()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(250 * time.Millisecond):
	}

	s2, _, err := srv.GetInstanceState(name)
	if err != nil {
		return nil, err
	}
	elapsed := time.Since(t1).Seconds()
	if elapsed <= 0 {
		elapsed = 0.25
	}

	cpuDelta := float64(s2.CPU.Usage - s1.CPU.Usage) // nanoseconds
	cpuPercent := (cpuDelta / 1e9) / elapsed * 100
	if cpuPercent < 0 {
		cpuPercent = 0
	}

	stats := &containers.Stats{
		CPUPercent:       cpuPercent,
		MemoryUsageBytes: s2.Memory.Usage,
		MemoryLimitBytes: s2.Memory.Total,
	}
	for _, disk := range s2.Disk {
		stats.DiskUsageBytes += disk.Usage
	}
	for _, nic := range s2.Network {
		stats.NetworkRxBytes += nic.Counters.BytesReceived
		stats.NetworkTxBytes += nic.Counters.BytesSent
	}
	return stats, nil
}

func (m *Manager) ExecuteCommand(ctx context.Context, name string, command []string) (string, string, error) {
	if err := containers.ValidateContainerName(name); err != nil {
		return "", "", err
	}
	if len(command) == 0 {
		return "", "", fmt.Errorf("command required")
	}
	for _, part := range command {
		if strings.ContainsAny(part, "\n\r\x00") {
			return "", "", fmt.Errorf("invalid command argument")
		}
	}
	srv, err := m.connect()
	if err != nil {
		return "", "", err
	}

	var stdout, stderr bytes.Buffer
	op, err := srv.ExecInstance(name, api.InstanceExecPost{
		Command:     command,
		WaitForWS:   true,
		Interactive: false,
	}, &incus.InstanceExecArgs{
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil {
		return "", "", err
	}
	if err := waitOp(ctx, op); err != nil {
		return stdout.String(), stderr.String(), err
	}
	return stdout.String(), stderr.String(), nil
}

func (m *Manager) ReadFile(ctx context.Context, name string, filePath string) ([]byte, error) {
	if err := containers.ValidateContainerName(name); err != nil {
		return nil, err
	}
	clean, err := containers.SanitizeContainerPath(filePath)
	if err != nil {
		return nil, err
	}
	srv, err := m.connect()
	if err != nil {
		return nil, err
	}
	reader, _, err := srv.GetInstanceFile(name, clean)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(io.LimitReader(reader, 32<<20)) // 32 MiB cap
}

func (m *Manager) WriteFile(ctx context.Context, name string, filePath string, data []byte) error {
	if err := containers.ValidateContainerName(name); err != nil {
		return err
	}
	clean, err := containers.SanitizeContainerPath(filePath)
	if err != nil {
		return err
	}
	if len(data) > 32<<20 {
		return fmt.Errorf("file too large")
	}
	srv, err := m.connect()
	if err != nil {
		return err
	}
	return srv.CreateInstanceFile(name, clean, incus.InstanceFileArgs{
		Content: bytes.NewReader(data),
		Mode:    0o644,
		Type:    "file",
	})
}

func (m *Manager) DeleteFile(ctx context.Context, name string, filePath string) error {
	if err := containers.ValidateContainerName(name); err != nil {
		return err
	}
	clean, err := containers.SanitizeContainerPath(filePath)
	if err != nil {
		return err
	}
	srv, err := m.connect()
	if err != nil {
		return err
	}
	return srv.DeleteInstanceFile(name, clean)
}

func (m *Manager) HostMetrics(ctx context.Context) (*containers.HostMetrics, error) {
	srv, err := m.connect()
	if err != nil {
		return nil, err
	}
	instances, err := srv.GetInstances(api.InstanceTypeAny)
	if err != nil {
		return nil, err
	}
	res, err := srv.GetServerResources()
	if err != nil {
		return &containers.HostMetrics{Containers: len(instances)}, nil
	}

	var diskTotal int64
	for _, disk := range res.Storage.Disks {
		diskTotal += int64(disk.Size)
	}

	// Prefer storage pool usage when available.
	var diskUsed int64
	pools, err := srv.GetStoragePools()
	if err == nil {
		for _, pool := range pools {
			poolRes, err := srv.GetStoragePoolResources(pool.Name)
			if err != nil {
				continue
			}
			diskUsed += int64(poolRes.Space.Used)
			if poolRes.Space.Total > 0 {
				diskTotal = int64(poolRes.Space.Total) // use pool capacity as authoritative total when present
			}
		}
	}

	return &containers.HostMetrics{
		CPUPercent:       0, // instantaneous host CPU % not provided by resources endpoint
		MemoryUsageBytes: int64(res.Memory.Used),
		MemoryTotalBytes: int64(res.Memory.Total),
		DiskUsageBytes:   diskUsed,
		DiskTotalBytes:   diskTotal,
		Containers:       len(instances),
	}, nil
}

func parseImage(image string) (alias, server, protocol string) {
	server = "https://images.linuxcontainers.org"
	protocol = "simplestreams"
	alias = strings.TrimSpace(image)
	if strings.Contains(alias, ":") {
		parts := strings.SplitN(alias, ":", 2)
		if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
			// allow remote:alias form later; for now treat as alias only if unknown
			alias = parts[1]
		}
	}
	return alias, server, protocol
}

func waitOp(ctx context.Context, op incus.Operation) error {
	done := make(chan error, 1)
	go func() { done <- op.Wait() }()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		return err
	case <-time.After(10 * time.Minute):
		return fmt.Errorf("operation timed out")
	}
}
