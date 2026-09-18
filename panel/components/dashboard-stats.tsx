"use client";

import { useEffect, useState } from "react";
import { Loader2 } from "lucide-react";

type Node = {
  status: string;
  cpu_percent: number;
  memory_usage_bytes: number;
  memory_total_bytes: number;
  disk_usage_bytes: number;
  disk_total_bytes: number;
  container_count: number;
};

type Server = { uuid: string };

function formatBytes(n: number) {
  if (!n) return "—";
  const gb = n / 1024 ** 3;
  if (gb >= 1) return `${gb.toFixed(1)} GB`;
  return `${(n / 1024 ** 2).toFixed(0)} MB`;
}

export function DashboardStats({ isAdmin }: { isAdmin: boolean }) {
  const [nodes, setNodes] = useState<Node[] | null>(null);
  const [servers, setServers] = useState<Server[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    async function load() {
      try {
        const sr = await fetch("/api/v1/servers", { credentials: "include" });
        const sd = await sr.json().catch(() => ({}));
        if (sr.ok) setServers(sd.servers || []);
        if (isAdmin) {
          const nr = await fetch("/api/v1/nodes", { credentials: "include" });
          const nd = await nr.json().catch(() => ({}));
          if (nr.ok) setNodes(nd.nodes || []);
          else setError(nd.error || null);
        } else {
          setNodes([]);
        }
      } catch {
        setError("Failed to load dashboard data");
      }
    }
    load();
    const t = setInterval(load, 15000);
    return () => clearInterval(t);
  }, [isAdmin]);

  if (servers === null || nodes === null) {
    return (
      <div className="flex items-center gap-2 text-ink-muted">
        <Loader2 className="h-4 w-4 animate-spin" /> Loading…
      </div>
    );
  }

  const online = (nodes || []).filter((n) => n.status === "ONLINE");
  const memUsed = online.reduce((a, n) => a + (n.memory_usage_bytes || 0), 0);
  const memTotal = online.reduce((a, n) => a + (n.memory_total_bytes || 0), 0);
  const diskUsed = online.reduce((a, n) => a + (n.disk_usage_bytes || 0), 0);
  const diskTotal = online.reduce((a, n) => a + (n.disk_total_bytes || 0), 0);
  const cpuVals = online.map((n) => n.cpu_percent).filter((v) => v > 0);
  const cpuAvg = cpuVals.length
    ? cpuVals.reduce((a, b) => a + b, 0) / cpuVals.length
    : null;
  const containers = online.reduce((a, n) => a + (n.container_count || 0), 0);

  return (
    <div className="space-y-6">
      {error ? <p className="text-sm text-amber-400">{error}</p> : null}
      <section className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <Card
          title="CPU"
          value={cpuAvg === null ? "—" : `${cpuAvg.toFixed(0)}%`}
          hint={online.length ? "Average across online nodes" : "No online nodes reporting CPU"}
        />
        <Card
          title="Memory"
          value={memTotal ? `${formatBytes(memUsed)} / ${formatBytes(memTotal)}` : "—"}
          hint={online.length ? "From online node heartbeats" : "Awaiting node heartbeat"}
        />
        <Card
          title="Storage"
          value={diskTotal ? `${formatBytes(diskUsed)} / ${formatBytes(diskTotal)}` : "—"}
          hint={online.length ? "From online node heartbeats" : "Awaiting node heartbeat"}
        />
        <Card title="Servers" value={String(servers.length)} hint="Registered in Hostrix" />
        {isAdmin ? (
          <Card
            title="Nodes"
            value={`${online.length} / ${nodes.length}`}
            hint={`${containers} containers reported online`}
          />
        ) : null}
      </section>
    </div>
  );
}

function Card({
  title,
  value,
  hint,
}: {
  title: string;
  value: string;
  hint: string;
}) {
  return (
    <div className="rounded-2xl border border-line bg-canvas-raised/60 p-5">
      <p className="text-sm text-ink-muted">{title}</p>
      <p className="mt-3 font-display text-3xl font-semibold tracking-tight">{value}</p>
      <p className="mt-2 text-xs text-ink-faint">{hint}</p>
    </div>
  );
}
