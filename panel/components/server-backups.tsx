"use client";

import { useCallback, useEffect, useState } from "react";
import { Download, Loader2, Plus, RotateCcw, Trash2 } from "lucide-react";

type Backup = {
  uuid: string;
  name: string;
  size_bytes: number;
  status: string;
  created_at: string;
};

function formatBytes(n: number) {
  if (!n) return "—";
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  if (n < 1024 * 1024 * 1024) return `${(n / (1024 * 1024)).toFixed(1)} MB`;
  return `${(n / (1024 * 1024 * 1024)).toFixed(2)} GB`;
}

export function ServerBackups({ serverId }: { serverId: string }) {
  const [backups, setBackups] = useState<Backup[]>([]);
  const [name, setName] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState<string | null>(null);

  const load = useCallback(async () => {
    const res = await fetch(`/api/v1/servers/${serverId}/backups`, { credentials: "include" });
    const data = await res.json().catch(() => ({}));
    if (!res.ok) {
      setError(data.error || "Failed to load backups");
      return;
    }
    setBackups(data.backups || []);
    setError(null);
  }, [serverId]);

  useEffect(() => {
    load();
  }, [load]);

  async function create(e: React.FormEvent) {
    e.preventDefault();
    setBusy("create");
    setError(null);
    try {
      const res = await fetch(`/api/v1/servers/${serverId}/backups`, {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name: name.trim() || undefined }),
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        setError(data.error || "Backup failed");
        return;
      }
      setName("");
      await load();
    } finally {
      setBusy(null);
    }
  }

  async function restore(uuid: string) {
    if (!confirm("Restore this backup? The current container will be replaced.")) return;
    setBusy(uuid + "restore");
    try {
      const res = await fetch(`/api/v1/servers/${serverId}/backups/${uuid}/restore`, {
        method: "POST",
        credentials: "include",
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) setError(data.error || "Restore failed");
      await load();
    } finally {
      setBusy(null);
    }
  }

  async function remove(uuid: string) {
    if (!confirm("Delete this backup?")) return;
    setBusy(uuid + "delete");
    try {
      const res = await fetch(`/api/v1/servers/${serverId}/backups/${uuid}`, {
        method: "DELETE",
        credentials: "include",
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) setError(data.error || "Delete failed");
      await load();
    } finally {
      setBusy(null);
    }
  }

  return (
    <section className="space-y-4">
      <div>
        <h2 className="font-display text-xl font-semibold tracking-tight">Backups</h2>
        <p className="mt-1 text-sm text-ink-muted">Incus instance backups for this container.</p>
      </div>
      {error ? <p className="text-sm text-red-400">{error}</p> : null}
      <form onSubmit={create} className="flex flex-wrap gap-2">
        <input
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="Optional name"
          className="min-w-[12rem] flex-1 rounded-lg border border-line bg-canvas-overlay px-3 py-2 text-sm outline-none focus:border-accent/60"
        />
        <button
          type="submit"
          disabled={busy === "create"}
          className="inline-flex items-center gap-2 rounded-lg bg-accent px-3 py-2 text-sm font-medium text-canvas disabled:opacity-60"
        >
          {busy === "create" ? <Loader2 className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />}
          Create backup
        </button>
      </form>
      <div className="overflow-hidden rounded-2xl border border-line">
        <table className="w-full text-left text-sm">
          <thead className="bg-canvas-raised text-ink-muted">
            <tr>
              <th className="px-4 py-3 font-medium">Name</th>
              <th className="px-4 py-3 font-medium">Status</th>
              <th className="px-4 py-3 font-medium">Size</th>
              <th className="px-4 py-3 font-medium">Created</th>
              <th className="px-4 py-3 font-medium">Actions</th>
            </tr>
          </thead>
          <tbody>
            {backups.length === 0 ? (
              <tr>
                <td colSpan={5} className="px-4 py-6 text-ink-faint">
                  No backups for this server.
                </td>
              </tr>
            ) : (
              backups.map((b) => (
                <tr key={b.uuid} className="border-t border-line">
                  <td className="px-4 py-3 font-medium">{b.name}</td>
                  <td className="px-4 py-3 text-xs tracking-wide">{b.status}</td>
                  <td className="px-4 py-3">{formatBytes(b.size_bytes)}</td>
                  <td className="px-4 py-3 text-ink-muted">
                    {b.created_at ? new Date(b.created_at).toLocaleString() : "—"}
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex flex-wrap gap-1">
                      <a
                        href={`/api/v1/servers/${serverId}/backups/${b.uuid}/download`}
                        title="Download"
                        className="rounded-lg border border-line p-2 text-ink-muted transition hover:bg-canvas-overlay hover:text-ink"
                      >
                        <Download className="h-3.5 w-3.5" />
                      </a>
                      <button
                        type="button"
                        title="Restore"
                        disabled={!!busy || b.status !== "COMPLETED"}
                        onClick={() => restore(b.uuid)}
                        className="rounded-lg border border-line p-2 text-ink-muted transition hover:bg-canvas-overlay hover:text-ink disabled:opacity-40"
                      >
                        <RotateCcw className="h-3.5 w-3.5" />
                      </button>
                      <button
                        type="button"
                        title="Delete"
                        disabled={!!busy}
                        onClick={() => remove(b.uuid)}
                        className="rounded-lg border border-line p-2 text-ink-muted transition hover:bg-canvas-overlay hover:text-ink disabled:opacity-40"
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                      </button>
                    </div>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </section>
  );
}
