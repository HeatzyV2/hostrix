package agentclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type BackupInfo struct {
	Name      string `json:"name"`
	CreatedAt string `json:"created_at,omitempty"`
	SizeBytes int64  `json:"size_bytes"`
}

func (c *Client) ListBackups(ctx context.Context, containerName string) ([]BackupInfo, error) {
	var out struct {
		Backups []BackupInfo `json:"backups"`
	}
	if err := c.do(ctx, http.MethodGet, "/v1/containers/"+url.PathEscape(containerName)+"/backups", nil, &out); err != nil {
		return nil, err
	}
	return out.Backups, nil
}

func (c *Client) CreateBackup(ctx context.Context, containerName, backupName string) (*BackupInfo, error) {
	var out struct {
		Backup BackupInfo `json:"backup"`
	}
	if err := c.do(ctx, http.MethodPost, "/v1/containers/"+url.PathEscape(containerName)+"/backups", map[string]string{
		"name": backupName,
	}, &out); err != nil {
		return nil, err
	}
	return &out.Backup, nil
}

func (c *Client) GetBackup(ctx context.Context, containerName, backupName string) (*BackupInfo, error) {
	var out struct {
		Backup BackupInfo `json:"backup"`
	}
	path := "/v1/containers/" + url.PathEscape(containerName) + "/backups/" + url.PathEscape(backupName)
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return &out.Backup, nil
}

func (c *Client) DeleteBackup(ctx context.Context, containerName, backupName string) error {
	path := "/v1/containers/" + url.PathEscape(containerName) + "/backups/" + url.PathEscape(backupName)
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}

func (c *Client) RestoreBackup(ctx context.Context, containerName, backupName string) error {
	path := "/v1/containers/" + url.PathEscape(containerName) + "/backups/" + url.PathEscape(backupName) + "/restore"
	return c.do(ctx, http.MethodPost, path, nil, nil)
}

func (c *Client) DownloadBackup(ctx context.Context, containerName, backupName string, w io.Writer) (int64, error) {
	path := "/v1/containers/" + url.PathEscape(containerName) + "/backups/" + url.PathEscape(backupName) + "/download"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		var er struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(data, &er)
		if er.Error != "" {
			return 0, fmt.Errorf("agent: %s", er.Error)
		}
		return 0, fmt.Errorf("agent: %s", resp.Status)
	}
	return io.Copy(w, resp.Body)
}
