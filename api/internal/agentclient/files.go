package agentclient

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
)

type DirEntry struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Mode int    `json:"mode"`
	Size int64  `json:"size"`
}

type ListFilesResponse struct {
	Path    string     `json:"path"`
	Entries []DirEntry `json:"entries"`
}

func (c *Client) ListFiles(ctx context.Context, name, path string) (*ListFilesResponse, error) {
	q := url.Values{}
	q.Set("path", path)
	var out ListFilesResponse
	if err := c.do(ctx, http.MethodGet, "/v1/containers/"+url.PathEscape(name)+"/files?"+q.Encode(), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) WriteFile(ctx context.Context, name, path, content string) error {
	return c.do(ctx, http.MethodPost, "/v1/containers/"+url.PathEscape(name)+"/files/write", map[string]string{
		"path":    path,
		"content": content,
	}, nil)
}

func (c *Client) WriteFileBytes(ctx context.Context, name, path string, data []byte) error {
	return c.do(ctx, http.MethodPost, "/v1/containers/"+url.PathEscape(name)+"/files/write", map[string]any{
		"path":        path,
		"content_b64": base64.StdEncoding.EncodeToString(data),
	}, nil)
}

func (c *Client) Mkdir(ctx context.Context, name, path string) error {
	return c.do(ctx, http.MethodPost, "/v1/containers/"+url.PathEscape(name)+"/files/mkdir", map[string]string{
		"path": path,
	}, nil)
}

func (c *Client) RenameFile(ctx context.Context, name, from, to string) error {
	return c.do(ctx, http.MethodPost, "/v1/containers/"+url.PathEscape(name)+"/files/rename", map[string]string{
		"from": from,
		"to":   to,
	}, nil)
}

func (c *Client) MoveFile(ctx context.Context, name, from, to string) error {
	return c.do(ctx, http.MethodPost, "/v1/containers/"+url.PathEscape(name)+"/files/move", map[string]string{
		"from": from,
		"to":   to,
	}, nil)
}

func (c *Client) DeleteFile(ctx context.Context, name, path string) error {
	q := url.Values{}
	q.Set("path", path)
	return c.do(ctx, http.MethodDelete, "/v1/containers/"+url.PathEscape(name)+"/files?"+q.Encode(), nil, nil)
}

func (c *Client) DownloadFile(ctx context.Context, name, path string) ([]byte, string, error) {
	q := url.Values{}
	q.Set("path", path)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/containers/"+url.PathEscape(name)+"/files/download?"+q.Encode(), nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 33<<20))
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode >= 300 {
		return nil, "", fmt.Errorf("agent: %s", parseAgentError(data, resp.Status))
	}
	return data, resp.Header.Get("Content-Disposition"), nil
}

func (c *Client) UploadFile(ctx context.Context, name, path, filename string, content io.Reader) error {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("path", path)
	part, err := mw.CreateFormFile("file", filename)
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, io.LimitReader(content, 33<<20)); err != nil {
		return err
	}
	if err := mw.Close(); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/containers/"+url.PathEscape(name)+"/files/upload", &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 300 {
		return fmt.Errorf("agent: %s", parseAgentError(data, resp.Status))
	}
	return nil
}

func (c *Client) ExtractArchive(ctx context.Context, name, archive, dest string) error {
	return c.do(ctx, http.MethodPost, "/v1/containers/"+url.PathEscape(name)+"/files/extract", map[string]string{
		"archive": archive,
		"dest":    dest,
	}, nil)
}

func parseAgentError(data []byte, fallback string) string {
	var er struct {
		Error string `json:"error"`
	}
	if len(data) > 0 {
		_ = json.Unmarshal(data, &er)
		if er.Error != "" {
			return er.Error
		}
	}
	return fallback
}
