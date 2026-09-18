"use client";

import { useCallback, useEffect, useState } from "react";
import { Loader2, Plus, Trash2 } from "lucide-react";

type Permission = {
  user_uuid: string;
  username: string;
  email: string;
  can_start: boolean;
  can_stop: boolean;
  can_files: boolean;
  can_console: boolean;
};

type User = {
  uuid: string;
  username: string;
  email: string;
  is_admin: boolean;
};

export function ServerPermissions({ serverId, isAdmin }: { serverId: string; isAdmin: boolean }) {
  const [perms, setPerms] = useState<Permission[]>([]);
  const [users, setUsers] = useState<User[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [hidden, setHidden] = useState(false);
  const [form, setForm] = useState({
    user_uuid: "",
    username: "",
    can_start: true,
    can_stop: true,
    can_files: false,
    can_console: false,
  });

  const load = useCallback(async () => {
    const res = await fetch(`/api/v1/servers/${serverId}/permissions`, { credentials: "include" });
    const data = await res.json().catch(() => ({}));
    if (res.status === 403) {
      setHidden(true);
      return;
    }
    if (!res.ok) {
      setError(data.error || "Failed to load permissions");
      return;
    }
    setHidden(false);
    setPerms(data.permissions || []);
    setError(null);
    if (isAdmin) {
      const ur = await fetch("/api/v1/users", { credentials: "include" });
      const ud = await ur.json().catch(() => ({}));
      if (ur.ok) {
        setUsers(((ud.users || []) as User[]).filter((u) => !u.is_admin));
      }
    }
  }, [serverId, isAdmin]);

  useEffect(() => {
    load();
  }, [load]);

  if (hidden) return null;

  async function grant(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      const body: Record<string, unknown> = {
        can_start: form.can_start,
        can_stop: form.can_stop,
        can_files: form.can_files,
        can_console: form.can_console,
      };
      if (form.user_uuid) body.user_uuid = form.user_uuid;
      else if (form.username.trim()) body.username = form.username.trim();

      const res = await fetch(`/api/v1/servers/${serverId}/permissions`, {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        setError(data.error || "Grant failed");
        return;
      }
      setForm((f) => ({ ...f, username: "", user_uuid: "" }));
      await load();
    } finally {
      setBusy(false);
    }
  }

  async function revoke(userUUID: string) {
    if (!confirm("Revoke access for this user?")) return;
    setBusy(true);
    try {
      const res = await fetch(`/api/v1/servers/${serverId}/permissions/${userUUID}`, {
        method: "DELETE",
        credentials: "include",
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) setError(data.error || "Revoke failed");
      await load();
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className="space-y-4">
      <div>
        <h2 className="font-display text-xl font-semibold tracking-tight">Access</h2>
        <p className="mt-1 text-sm text-ink-muted">
          Share start, stop, files, and console access with other users.
        </p>
      </div>
      {error ? <p className="text-sm text-red-400">{error}</p> : null}

      <form onSubmit={grant} className="grid gap-3 rounded-2xl border border-line bg-canvas-raised/40 p-4 lg:grid-cols-6">
        {isAdmin && users.length > 0 ? (
          <select
            value={form.user_uuid}
            onChange={(e) => setForm({ ...form, user_uuid: e.target.value, username: "" })}
            className="rounded-lg border border-line bg-canvas-overlay px-3 py-2 text-sm outline-none focus:border-accent/60 lg:col-span-2"
          >
            <option value="">Select user</option>
            {users.map((u) => (
              <option key={u.uuid} value={u.uuid}>
                {u.username} ({u.email})
              </option>
            ))}
          </select>
        ) : (
          <input
            required
            placeholder="Username or email"
            value={form.username}
            onChange={(e) => setForm({ ...form, username: e.target.value, user_uuid: "" })}
            className="rounded-lg border border-line bg-canvas-overlay px-3 py-2 text-sm outline-none focus:border-accent/60 lg:col-span-2"
          />
        )}
        {(
          [
            ["can_start", "Start"],
            ["can_stop", "Stop"],
            ["can_files", "Files"],
            ["can_console", "Console"],
          ] as const
        ).map(([key, label]) => (
          <label key={key} className="flex items-center gap-2 text-sm text-ink-muted">
            <input
              type="checkbox"
              checked={form[key]}
              onChange={(e) => setForm({ ...form, [key]: e.target.checked })}
            />
            {label}
          </label>
        ))}
        <button
          type="submit"
          disabled={busy || (!form.user_uuid && !form.username.trim())}
          className="inline-flex items-center justify-center gap-2 rounded-lg bg-accent px-3 py-2 text-sm font-medium text-canvas disabled:opacity-60 lg:col-span-6"
        >
          {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />}
          Grant access
        </button>
      </form>

      <div className="overflow-hidden rounded-2xl border border-line">
        <table className="w-full text-left text-sm">
          <thead className="bg-canvas-raised text-ink-muted">
            <tr>
              <th className="px-4 py-3 font-medium">User</th>
              <th className="px-4 py-3 font-medium">Start</th>
              <th className="px-4 py-3 font-medium">Stop</th>
              <th className="px-4 py-3 font-medium">Files</th>
              <th className="px-4 py-3 font-medium">Console</th>
              <th className="px-4 py-3 font-medium" />
            </tr>
          </thead>
          <tbody>
            {perms.length === 0 ? (
              <tr>
                <td colSpan={6} className="px-4 py-6 text-ink-faint">
                  No shared access yet.
                </td>
              </tr>
            ) : (
              perms.map((p) => (
                <tr key={p.user_uuid} className="border-t border-line">
                  <td className="px-4 py-3">
                    <div className="font-medium">{p.username}</div>
                    <div className="text-xs text-ink-faint">{p.email}</div>
                  </td>
                  <td className="px-4 py-3">{p.can_start ? "Yes" : "—"}</td>
                  <td className="px-4 py-3">{p.can_stop ? "Yes" : "—"}</td>
                  <td className="px-4 py-3">{p.can_files ? "Yes" : "—"}</td>
                  <td className="px-4 py-3">{p.can_console ? "Yes" : "—"}</td>
                  <td className="px-4 py-3">
                    <button
                      type="button"
                      disabled={busy}
                      onClick={() => revoke(p.user_uuid)}
                      className="rounded-lg border border-line p-2 text-ink-muted transition hover:bg-canvas-overlay hover:text-ink disabled:opacity-40"
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </button>
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
