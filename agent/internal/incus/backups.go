package incusmgr

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/hostrix/hostrix/agent/internal/containers"
	incus "github.com/lxc/incus/v6/client"
	"github.com/lxc/incus/v6/shared/api"
)

func (m *Manager) CreateBackup(ctx context.Context, containerName, backupName string) (*containers.BackupInfo, error) {
	if err := containers.ValidateContainerName(containerName); err != nil {
		return nil, err
	}
	if err := containers.ValidateBackupName(backupName); err != nil {
		return nil, err
	}
	srv, err := m.connect()
	if err != nil {
		return nil, err
	}

	op, err := srv.CreateInstanceBackup(containerName, api.InstanceBackupsPost{
		Name:                 backupName,
		ExpiresAt:            time.Time{}, // never expires via Incus; Hostrix owns lifecycle
		InstanceOnly:         true,
		OptimizedStorage:     false,
		CompressionAlgorithm: "gzip",
	})
	if err != nil {
		return nil, fmt.Errorf("create backup: %w", err)
	}
	if err := waitOp(ctx, op); err != nil {
		return nil, err
	}

	info, err := m.GetBackup(ctx, containerName, backupName)
	if err != nil {
		return &containers.BackupInfo{Name: backupName}, nil
	}
	return info, nil
}

func (m *Manager) ListBackups(ctx context.Context, containerName string) ([]containers.BackupInfo, error) {
	if err := containers.ValidateContainerName(containerName); err != nil {
		return nil, err
	}
	srv, err := m.connect()
	if err != nil {
		return nil, err
	}
	raw, err := srv.GetInstanceBackups(containerName)
	if err != nil {
		return nil, fmt.Errorf("list backups: %w", err)
	}
	out := make([]containers.BackupInfo, 0, len(raw))
	for _, b := range raw {
		out = append(out, containers.BackupInfo{
			Name:      b.Name,
			CreatedAt: b.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return out, nil
}

func (m *Manager) GetBackup(ctx context.Context, containerName, backupName string) (*containers.BackupInfo, error) {
	if err := containers.ValidateContainerName(containerName); err != nil {
		return nil, err
	}
	if err := containers.ValidateBackupName(backupName); err != nil {
		return nil, err
	}
	srv, err := m.connect()
	if err != nil {
		return nil, err
	}
	b, _, err := srv.GetInstanceBackup(containerName, backupName)
	if err != nil {
		return nil, fmt.Errorf("get backup: %w", err)
	}
	info := &containers.BackupInfo{
		Name:      b.Name,
		CreatedAt: b.CreatedAt.UTC().Format(time.RFC3339),
	}
	if size, err := m.probeBackupSize(srv, containerName, backupName); err == nil {
		info.SizeBytes = size
	}
	return info, nil
}

func (m *Manager) DeleteBackup(ctx context.Context, containerName, backupName string) error {
	if err := containers.ValidateContainerName(containerName); err != nil {
		return err
	}
	if err := containers.ValidateBackupName(backupName); err != nil {
		return err
	}
	srv, err := m.connect()
	if err != nil {
		return err
	}
	op, err := srv.DeleteInstanceBackup(containerName, backupName)
	if err != nil {
		return fmt.Errorf("delete backup: %w", err)
	}
	return waitOp(ctx, op)
}

func (m *Manager) DownloadBackup(ctx context.Context, containerName, backupName string, w io.Writer) (int64, error) {
	if err := containers.ValidateContainerName(containerName); err != nil {
		return 0, err
	}
	if err := containers.ValidateBackupName(backupName); err != nil {
		return 0, err
	}
	srv, err := m.connect()
	if err != nil {
		return 0, err
	}
	return m.streamBackupExport(ctx, srv, containerName, backupName, w)
}

func (m *Manager) RestoreBackup(ctx context.Context, containerName, backupName string) error {
	if err := containers.ValidateContainerName(containerName); err != nil {
		return err
	}
	if err := containers.ValidateBackupName(backupName); err != nil {
		return err
	}
	srv, err := m.connect()
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp("", "hostrix-backup-*.tar.gz")
	if err != nil {
		return fmt.Errorf("temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}()

	if _, err := m.streamBackupExport(ctx, srv, containerName, backupName, tmp); err != nil {
		return fmt.Errorf("export backup: %w", err)
	}
	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		return err
	}

	// Best-effort stop; ignore errors if already stopped.
	_ = m.StopContainer(ctx, containerName)
	delOp, err := srv.DeleteInstance(containerName)
	if err != nil {
		return fmt.Errorf("delete instance before restore: %w", err)
	}
	if err := waitOp(ctx, delOp); err != nil {
		return fmt.Errorf("delete instance before restore: %w", err)
	}

	createOp, err := srv.CreateInstanceFromBackup(incus.InstanceBackupArgs{
		BackupFile: tmp,
		Name:       containerName,
	})
	if err != nil {
		// Retry without name override when the Incus extension is unavailable.
		if _, seekErr := tmp.Seek(0, io.SeekStart); seekErr != nil {
			return fmt.Errorf("import backup: %w", err)
		}
		createOp, err = srv.CreateInstanceFromBackup(incus.InstanceBackupArgs{
			BackupFile: tmp,
		})
		if err != nil {
			return fmt.Errorf("import backup: %w", err)
		}
	}
	if err := waitOp(ctx, createOp); err != nil {
		return fmt.Errorf("import backup: %w", err)
	}
	return m.StartContainer(ctx, containerName)
}

func (m *Manager) streamBackupExport(ctx context.Context, srv incus.InstanceServer, containerName, backupName string, w io.Writer) (int64, error) {
	info, err := srv.GetConnectionInfo()
	if err != nil {
		return 0, err
	}
	uri := fmt.Sprintf("%s/1.0/instances/%s/backups/%s/export", info.URL, url.PathEscape(containerName), url.PathEscape(backupName))
	if info.Project != "" {
		uri += "?project=" + url.QueryEscape(info.Project)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return 0, err
	}
	resp, err := srv.DoHTTP(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return 0, fmt.Errorf("export backup: %s: %s", resp.Status, string(body))
	}
	return io.Copy(w, resp.Body)
}

// probeBackupSize issues a GET against the export endpoint and uses Content-Length when present,
// then cancels after reading headers via an immediate close (no full download).
func (m *Manager) probeBackupSize(srv incus.InstanceServer, containerName, backupName string) (int64, error) {
	info, err := srv.GetConnectionInfo()
	if err != nil {
		return 0, err
	}
	uri := fmt.Sprintf("%s/1.0/instances/%s/backups/%s/export", info.URL, url.PathEscape(containerName), url.PathEscape(backupName))
	if info.Project != "" {
		uri += "?project=" + url.QueryEscape(info.Project)
	}
	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return 0, err
	}
	resp, err := srv.DoHTTP(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("status %s", resp.Status)
	}
	if resp.ContentLength > 0 {
		return resp.ContentLength, nil
	}
	return 0, fmt.Errorf("content length unknown")
}
