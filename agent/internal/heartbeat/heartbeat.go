package heartbeat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/hostrix/hostrix/agent/internal/config"
	"github.com/hostrix/hostrix/agent/internal/containers"
)

type Reporter struct {
	cfg    *config.Config
	mgr    containers.ContainerManager
	client *http.Client
}

func New(cfg *config.Config, mgr containers.ContainerManager) *Reporter {
	return &Reporter{
		cfg: cfg,
		mgr: mgr,
		client: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

func (r *Reporter) Run(ctx context.Context) {
	t := time.NewTicker(r.cfg.HeartbeatEvery)
	defer t.Stop()
	r.beat(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			r.beat(ctx)
		}
	}
}

func (r *Reporter) beat(ctx context.Context) {
	metrics, err := r.mgr.HostMetrics(ctx)
	status := "ONLINE"
	payload := map[string]any{
		"status": status,
	}
	if err != nil {
		log.Printf("heartbeat metrics: %v", err)
		payload["status"] = "ERROR"
		payload["error"] = err.Error()
	} else {
		payload["cpu_percent"] = metrics.CPUPercent
		payload["memory_usage_bytes"] = metrics.MemoryUsageBytes
		payload["memory_total_bytes"] = metrics.MemoryTotalBytes
		payload["disk_usage_bytes"] = metrics.DiskUsageBytes
		payload["disk_total_bytes"] = metrics.DiskTotalBytes
		payload["containers"] = metrics.Containers
	}

	body, _ := json.Marshal(payload)
	url := fmt.Sprintf("%s/api/v1/nodes/%s/heartbeat", r.cfg.APIURL, r.cfg.NodeUUID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		log.Printf("heartbeat request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+r.cfg.NodeToken)
	resp, err := r.client.Do(req)
	if err != nil {
		log.Printf("heartbeat send: %v", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		log.Printf("heartbeat rejected: %s", resp.Status)
	}
}
