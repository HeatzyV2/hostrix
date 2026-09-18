"use client";

import { useEffect, useRef, useState } from "react";
import { Loader2, RefreshCw } from "lucide-react";

type Metrics = {
  cpu_percent: number;
  memory_usage_bytes: number;
  memory_limit_bytes: number;
  disk_usage_bytes: number;
  network_rx_bytes: number;
  network_tx_bytes: number;
};

function formatBytes(n: number) {
  if (!n && n !== 0) return "—";
  const gb = n / 1024 ** 3;
  if (gb >= 1) return `${gb.toFixed(2)} GB`;
  const mb = n / 1024 ** 2;
  if (mb >= 1) return `${mb.toFixed(1)} MB`;
  return `${Math.round(n / 1024)} KB`;
}

export function ServerMetrics({ serverId }: { serverId: string }) {
  const [metrics, setMetrics] = useState<Metrics | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  async function load() {
    setError(null);
    try {
      const res = await fetch(`/api/v1/servers/${serverId}/metrics`, {
        credentials: "include",
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        setMetrics(null);
        setError(data.error || "Metrics unavailable");
        return;
      }
      setMetrics(data.metrics);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    load();
    const t = setInterval(load, 5000);
    return () => clearInterval(t);
  }, [serverId]);

  if (loading && !metrics) {
    return (
      <div className="flex items-center gap-2 text-sm text-ink-muted">
        <Loader2 className="h-4 w-4 animate-spin" /> Loading metrics…
      </div>
    );
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <h2 className="font-display text-lg font-medium">Live metrics</h2>
        <button
          type="button"
          onClick={load}
          className="rounded-lg border border-line p-2 text-ink-muted hover:bg-canvas-overlay"
          title="Refresh"
        >
          <RefreshCw className="h-4 w-4" />
        </button>
      </div>
      {error ? <p className="text-sm text-amber-400">{error}</p> : null}
      {metrics ? (
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          <MetricCard label="CPU" value={`${metrics.cpu_percent.toFixed(1)}%`} />
          <MetricCard
            label="Memory"
            value={`${formatBytes(metrics.memory_usage_bytes)} / ${formatBytes(metrics.memory_limit_bytes)}`}
          />
          <MetricCard label="Disk used" value={formatBytes(metrics.disk_usage_bytes)} />
          <MetricCard label="Network RX" value={formatBytes(metrics.network_rx_bytes)} />
          <MetricCard label="Network TX" value={formatBytes(metrics.network_tx_bytes)} />
        </div>
      ) : null}
    </div>
  );
}

function MetricCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-2xl border border-line bg-canvas-raised/60 p-4">
      <p className="text-xs text-ink-muted">{label}</p>
      <p className="mt-2 font-display text-xl font-semibold tracking-tight">{value}</p>
    </div>
  );
}
