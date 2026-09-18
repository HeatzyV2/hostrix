package agentclient

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type Stats struct {
	CPUPercent       float64 `json:"cpu_percent"`
	MemoryUsageBytes int64   `json:"memory_usage_bytes"`
	MemoryLimitBytes int64   `json:"memory_limit_bytes"`
	DiskUsageBytes   int64   `json:"disk_usage_bytes"`
	NetworkRxBytes   int64   `json:"network_rx_bytes"`
	NetworkTxBytes   int64   `json:"network_tx_bytes"`
}

func (c *Client) GetStats(ctx context.Context, name string) (*Stats, error) {
	var out Stats
	if err := c.do(ctx, http.MethodGet, "/v1/containers/"+url.PathEscape(name)+"/stats", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DialConsole opens a WebSocket to the agent console endpoint.
func (c *Client) DialConsole(ctx context.Context, name string, cols, rows int) (*websocket.Conn, *http.Response, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, nil, err
	}
	if strings.EqualFold(u.Scheme, "https") {
		u.Scheme = "wss"
	} else {
		u.Scheme = "ws"
	}
	u.Path = "/v1/containers/" + name + "/console/ws"
	q := u.Query()
	q.Set("cols", strconv.Itoa(cols))
	q.Set("rows", strconv.Itoa(rows))
	u.RawQuery = q.Encode()

	hdr := http.Header{}
	hdr.Set("Authorization", "Bearer "+c.token)

	dialer := websocket.Dialer{HandshakeTimeout: 15 * time.Second}
	return dialer.DialContext(ctx, u.String(), hdr)
}
