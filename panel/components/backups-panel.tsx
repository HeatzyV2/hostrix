"use client";

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { Loader2, RotateCcw, Trash2 } from "lucide-react";

type Backup = {
  uuid: string;
  name: string;
  size_bytes: number;
  status: string;
  created_at: string;
  server_uuid?: string;
  server_name?: string;
};

function formatBytes(n: number) {
  if (!n) return "—";
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  if (n < 1024 * 1024 * 1024) return `${(n / (1024 * 1024)).toFixed(1)} MB`;
  return `${(n / (1024 * 1024 * 1024)).toFixed(2)} GB`;
}

export function BackupsPanel() {
  const [backups, setBackups] = useState<Backup[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState<string | null>(null);

  const load = useCallback(async () => {
    setError(null);
    const res = await fetch("/api/v1/backups", { credentials: "include" });
    const data = await res.json().catch(() => ({}));
    if (!res.ok) {
      setError(data.error || "Failed to load backups");
      setLoading(false);
      return;
    }
    setBackups(data.backups || []);
    setLoading(false);
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  async function restore(b: Backup) {
    if (!b.server_uuid) return;
    if (!confirm(`Restore "${b.name}" onto ${b.server_name || "server"}? The running container will be replaced.`)) {
      return;
    }
    setBusy(b.uuid + "restore");
    try {
      const res = await fetch(`/api/v1/servers/${b.server_uuid}/backups/${b.uuid}/restore`, {
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

  async function remove(b: Backup) {
    if (!b.server_uuid) return;
    if (!confirm(`Delete backup "${b.name}"?`)) return;
    setBusy(b.uuid + "delete");
    try {
      const res = await fetch(`/api/v1/servers/${b.server_uuid}/backups/${b.uuid}`, {
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

  if (loading) {
    return (
      <div className="flex items-center gap-2 text-ink-muted">
        <Loader2 className="h-4 w-4 animate-spin" /> Loading backups…
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {error ? <p className="text-sm text-red-400">{error}</p> : null}
      <div className="overflow-hidden rounded-2xl border border-line">
        <table className="w-full text-left text-sm">
          <thead className="bg-canvas-raised text-ink-muted">
            <tr>
              <th className="px-4 py-3 font-medium">Name</th>
              <th className="px-4 py-3 font-medium">Server</th>
              <th className="px-4 py-3 font-medium">Status</th>
              <th className="px-4 py-3 font-medium">Size</th>
              <th className="px-4 py-3 font-medium">Created</th>
              <th className="px-4 py-3 font-medium">Actions</th>
            </tr>
          </thead>
          <tbody>
            {backups.length === 0 ? (
              <tr>
                <td colSpan={6} className="px-4 py-8 text-ink-faint">
                  No backups yet. Create one from a server detail page.
                </td>
              </tr>
            ) : (
              backups.map((b) => (
                <tr key={b.uuid} className="border-t border-line">
                  <td className="px-4 py-3 font-medium">{b.name}</td>
                  <td className="px-4 py-3">
                    {b.server_uuid ? (
                      <Link href={`/servers/${b.server_uuid}`} className="hover:text-accent">
                        {b.server_name || b.server_uuid}
                      </Link>
                    ) : (
                      "—"
                    )}
                  </td>
                  <td className="px-4 py-3 text-xs tracking-wide">{b.status}</td>
                  <td className="px-4 py-3">{formatBytes(b.size_bytes)}</td>
                  <td className="px-4 py-3 text-ink-muted">
                    {b.created_at ? new Date(b.created_at).toLocaleString() : "—"}
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex gap-1">
                      <button
                        type="button"
                        title="Restore"
                        disabled={!!busy || b.status !== "COMPLETED"}
                        onClick={() => restore(b)}
                        className="rounded-lg border border-line p-2 text-ink-muted transition hover:bg-canvas-overlay hover:text-ink disabled:opacity-40"
                      >
                        {busy === b.uuid + "restore" ? (
                          <Loader2 className="h-3.5 w-3.5 animate-spin" />
                        ) : (
                          <RotateCcw className="h-3.5 w-3.5" />
                        )}
                      </button>
                      <button
                        type="button"
                        title="Delete"
                        disabled={!!busy}
                        onClick={() => remove(b)}
                        className="rounded-lg border border-line p-2 text-ink-muted transition hover:bg-canvas-overlay hover:text-ink disabled:opacity-40"
                      >
                        {busy === b.uuid + "delete" ? (
                          <Loader2 className="h-3.5 w-3.5 animate-spin" />
                        ) : (
                          <Trash2 className="h-3.5 w-3.5" />
                        )}
                      </button>
                    </div>
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
