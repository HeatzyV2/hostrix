"use client";

import { useCallback, useEffect, useState } from "react";
import { Loader2, Plus, Trash2 } from "lucide-react";

type Node = {
  uuid: string;
  name: string;
  hostname: string;
  address: string;
  port: number;
  status: string;
  last_heartbeat_at?: string | null;
  cpu_percent: number;
  memory_usage_bytes: number;
  memory_total_bytes: number;
  disk_usage_bytes: number;
  disk_total_bytes: number;
  container_count: number;
};

function formatBytes(n: number) {
  if (!n) return "—";
  const gb = n / (1024 ** 3);
  if (gb >= 1) return `${gb.toFixed(1)} GB`;
  const mb = n / (1024 ** 2);
  return `${mb.toFixed(0)} MB`;
}

export function NodesAdmin() {
  const [nodes, setNodes] = useState<Node[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [tokenOnce, setTokenOnce] = useState<string | null>(null);
  const [creating, setCreating] = useState(false);
  const [form, setForm] = useState({
    name: "",
    hostname: "",
    address: "127.0.0.1",
    port: 8081,
  });

  const load = useCallback(async () => {
    setError(null);
    const res = await fetch("/api/v1/nodes", { credentials: "include" });
    const data = await res.json().catch(() => ({}));
    if (!res.ok) {
      setError(data.error || "Failed to load nodes");
      setLoading(false);
      return;
    }
    setNodes(data.nodes || []);
    setLoading(false);
  }, []);

  useEffect(() => {
    load();
    const t = setInterval(load, 15000);
    return () => clearInterval(t);
  }, [load]);

  async function onCreate(e: React.FormEvent) {
    e.preventDefault();
    setCreating(true);
    setError(null);
    setTokenOnce(null);
    try {
      const res = await fetch("/api/v1/nodes", {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(form),
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        setError(data.error || "Create failed");
        return;
      }
      setTokenOnce(data.token);
      setForm({ name: "", hostname: "", address: "127.0.0.1", port: 8081 });
      await load();
    } finally {
      setCreating(false);
    }
  }

  async function onDelete(uuid: string) {
    if (!confirm("Delete this node?")) return;
    const res = await fetch(`/api/v1/nodes/${uuid}`, {
      method: "DELETE",
      credentials: "include",
    });
    if (!res.ok) {
      const data = await res.json().catch(() => ({}));
      setError(data.error || "Delete failed");
      return;
    }
    await load();
  }

  if (loading) {
    return (
      <div className="flex items-center gap-2 text-ink-muted">
        <Loader2 className="h-4 w-4 animate-spin" /> Loading nodes…
      </div>
    );
  }

  return (
    <div className="space-y-8">
      {error ? <p className="text-sm text-red-400">{error}</p> : null}

      {tokenOnce ? (
        <div className="rounded-2xl border border-accent/30 bg-accent/10 p-4 text-sm">
          <p className="font-medium text-ink">Node token (copy now — shown once)</p>
          <code className="mt-2 block break-all rounded-lg bg-canvas px-3 py-2 text-accent">
            {tokenOnce}
          </code>
        </div>
      ) : null}

      <form
        onSubmit={onCreate}
        className="grid gap-3 rounded-2xl border border-line bg-canvas-raised/60 p-5 sm:grid-cols-2 lg:grid-cols-5"
      >
        <input
          required
          placeholder="Name"
          value={form.name}
          onChange={(e) => setForm({ ...form, name: e.target.value })}
          className="rounded-lg border border-line bg-canvas-overlay px-3 py-2 text-sm outline-none focus:border-accent/60"
        />
        <input
          required
          placeholder="Hostname"
          value={form.hostname}
          onChange={(e) => setForm({ ...form, hostname: e.target.value })}
          className="rounded-lg border border-line bg-canvas-overlay px-3 py-2 text-sm outline-none focus:border-accent/60"
        />
        <input
          required
          placeholder="Address"
          value={form.address}
          onChange={(e) => setForm({ ...form, address: e.target.value })}
          className="rounded-lg border border-line bg-canvas-overlay px-3 py-2 text-sm outline-none focus:border-accent/60"
        />
        <input
          type="number"
          required
          placeholder="Port"
          value={form.port}
          onChange={(e) => setForm({ ...form, port: Number(e.target.value) })}
          className="rounded-lg border border-line bg-canvas-overlay px-3 py-2 text-sm outline-none focus:border-accent/60"
        />
        <button
          type="submit"
          disabled={creating}
          className="inline-flex items-center justify-center gap-2 rounded-lg bg-accent px-3 py-2 text-sm font-medium text-canvas disabled:opacity-60"
        >
          {creating ? <Loader2 className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />}
          Create node
        </button>
      </form>

      <div className="overflow-hidden rounded-2xl border border-line">
        <table className="w-full text-left text-sm">
          <thead className="bg-canvas-raised text-ink-muted">
            <tr>
              <th className="px-4 py-3 font-medium">Node</th>
              <th className="px-4 py-3 font-medium">Status</th>
              <th className="px-4 py-3 font-medium">CPU</th>
              <th className="px-4 py-3 font-medium">Memory</th>
              <th className="px-4 py-3 font-medium">Disk</th>
              <th className="px-4 py-3 font-medium">Containers</th>
              <th className="px-4 py-3 font-medium" />
            </tr>
          </thead>
          <tbody>
            {nodes.length === 0 ? (
              <tr>
                <td colSpan={7} className="px-4 py-8 text-ink-faint">
                  No nodes yet. Create one and start the Agent with the token.
                </td>
              </tr>
            ) : (
              nodes.map((n) => (
                <tr key={n.uuid} className="border-t border-line">
                  <td className="px-4 py-3">
                    <div className="font-medium">{n.name}</div>
                    <div className="text-xs text-ink-faint">
                      {n.address}:{n.port}
                    </div>
                  </td>
                  <td className="px-4 py-3">
                    <StatusBadge status={n.status} />
                  </td>
                  <td className="px-4 py-3">
                    {n.cpu_percent > 0 ? `${n.cpu_percent.toFixed(0)}%` : "—"}
                  </td>
                  <td className="px-4 py-3">
                    {n.memory_total_bytes
                      ? `${formatBytes(n.memory_usage_bytes)} / ${formatBytes(n.memory_total_bytes)}`
                      : "—"}
                  </td>
                  <td className="px-4 py-3">
                    {n.disk_total_bytes
                      ? `${formatBytes(n.disk_usage_bytes)} / ${formatBytes(n.disk_total_bytes)}`
                      : "—"}
                  </td>
                  <td className="px-4 py-3">{n.container_count || "—"}</td>
                  <td className="px-4 py-3 text-right">
                    <button
                      type="button"
                      onClick={() => onDelete(n.uuid)}
                      className="rounded-lg p-2 text-ink-muted hover:bg-canvas-overlay hover:text-red-400"
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function StatusBadge({ status }: { status: string }) {
  const color =
    status === "ONLINE"
      ? "text-emerald-400"
      : status === "ERROR"
        ? "text-red-400"
        : status === "INSTALLING"
          ? "text-amber-400"
          : "text-ink-faint";
  return <span className={`text-xs font-medium tracking-wide ${color}`}>{status}</span>;
}
