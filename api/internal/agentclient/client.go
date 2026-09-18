package agentclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func New(address string, port int, token string) *Client {
	return &Client{
		baseURL: fmt.Sprintf("http://%s:%d", address, port),
		token:   token,
		httpClient: &http.Client{
			Timeout: 10 * time.Minute,
		},
	}
}

type CreateContainerRequest struct {
	Name     string `json:"name"`
	Image    string `json:"image"`
	MemoryMB int    `json:"memory_mb"`
	CPULimit int    `json:"cpu_limit"`
	DiskMB   int    `json:"disk_mb"`
}

func (c *Client) CreateContainer(ctx context.Context, req CreateContainerRequest) error {
	return c.do(ctx, http.MethodPost, "/v1/containers", req, nil)
}

func (c *Client) DeleteContainer(ctx context.Context, name string) error {
	return c.do(ctx, http.MethodDelete, "/v1/containers/"+name, nil, nil)
}

func (c *Client) StartContainer(ctx context.Context, name string) error {
	return c.do(ctx, http.MethodPost, "/v1/containers/"+name+"/start", nil, nil)
}

func (c *Client) StopContainer(ctx context.Context, name string) error {
	return c.do(ctx, http.MethodPost, "/v1/containers/"+name+"/stop", nil, nil)
}

func (c *Client) RestartContainer(ctx context.Context, name string) error {
	return c.do(ctx, http.MethodPost, "/v1/containers/"+name+"/restart", nil, nil)
}

func (c *Client) KillContainer(ctx context.Context, name string) error {
	return c.do(ctx, http.MethodPost, "/v1/containers/"+name+"/kill", nil, nil)
}

func (c *Client) GetStatus(ctx context.Context, name string) (string, error) {
	var out struct {
		Status string `json:"status"`
	}
	if err := c.do(ctx, http.MethodGet, "/v1/containers/"+name+"/status", nil, &out); err != nil {
		return "", err
	}
	return out.Status, nil
}

func (c *Client) do(ctx context.Context, method, path string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 300 {
		var er struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(data, &er)
		if er.Error != "" {
			return fmt.Errorf("agent: %s", er.Error)
		}
		return fmt.Errorf("agent: %s", resp.Status)
	}
	if out != nil && len(data) > 0 {
		return json.Unmarshal(data, out)
	}
	return nil
}
