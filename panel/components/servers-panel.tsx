"use client";

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { Loader2, Plus, Play, Square, RotateCcw, Skull, Trash2 } from "lucide-react";

type Server = {
  uuid: string;
  name: string;
  status: string;
  memory: number;
  cpu: number;
  disk: number;
  container_name: string;
  node_id: number;
};

type Node = {
  uuid: string;
  name: string;
  status: string;
};

type Template = {
  uuid: string;
  name: string;
  slug: string;
  image: string;
};

export function ServersPanel({ isAdmin }: { isAdmin: boolean }) {
  const [servers, setServers] = useState<Server[]>([]);
  const [nodes, setNodes] = useState<Node[]>([]);
  const [templates, setTemplates] = useState<Template[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState<string | null>(null);
  const [form, setForm] = useState({
    name: "",
    node_uuid: "",
    template_uuid: "",
    memory: 1024,
    cpu: 100,
    disk: 10240,
  });

  const load = useCallback(async () => {
    setError(null);
    const res = await fetch("/api/v1/servers", { credentials: "include" });
    const data = await res.json().catch(() => ({}));
    if (!res.ok) {
      setError(data.error || "Failed to load servers");
      setLoading(false);
      return;
    }
    setServers(data.servers || []);
    if (isAdmin) {
      const [nr, tr] = await Promise.all([
        fetch("/api/v1/nodes", { credentials: "include" }),
        fetch("/api/v1/templates", { credentials: "include" }),
      ]);
      const nd = await nr.json().catch(() => ({}));
      const td = await tr.json().catch(() => ({}));
      if (nr.ok) {
        const list = nd.nodes || [];
        setNodes(list);
        setForm((f) => (f.node_uuid || !list[0]?.uuid ? f : { ...f, node_uuid: list[0].uuid }));
      }
      if (tr.ok) {
        const list: Template[] = td.templates || [];
        setTemplates(list);
        setForm((f) => {
          if (f.template_uuid || !list[0]?.uuid) return f;
          return { ...f, template_uuid: list[0].uuid };
        });
      }
    }
    setLoading(false);
  }, [isAdmin]);

  useEffect(() => {
    load();
  }, [load]);

  async function power(uuid: string, action: string) {
    setBusy(uuid + action);
    setError(null);
    try {
      const res = await fetch(`/api/v1/servers/${uuid}/${action}`, {
        method: "POST",
        credentials: "include",
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) setError(data.error || `${action} failed`);
      await load();
    } finally {
      setBusy(null);
    }
  }

  async function onDelete(uuid: string) {
    if (!confirm("Delete this server and its container?")) return;
    setBusy(uuid + "delete");
    try {
      const res = await fetch(`/api/v1/servers/${uuid}`, {
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

  async function onCreate(e: React.FormEvent) {
    e.preventDefault();
    setBusy("create");
    setError(null);
    try {
      const res = await fetch("/api/v1/servers", {
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
      setForm((f) => ({ ...f, name: "" }));
      await load();
    } finally {
      setBusy(null);
    }
  }

  if (loading) {
    return (
      <div className="flex items-center gap-2 text-ink-muted">
        <Loader2 className="h-4 w-4 animate-spin" /> Loading servers…
      </div>
    );
  }

  return (
    <div className="space-y-8">
      {error ? <p className="text-sm text-red-400">{error}</p> : null}

      {isAdmin ? (
        <form
          onSubmit={onCreate}
          className="grid gap-3 rounded-2xl border border-line bg-canvas-raised/60 p-5 lg:grid-cols-7"
        >
          <input
            required
            placeholder="Name"
            value={form.name}
            onChange={(e) => setForm({ ...form, name: e.target.value })}
            className="rounded-lg border border-line bg-canvas-overlay px-3 py-2 text-sm outline-none focus:border-accent/60"
          />
          <select
            required
            value={form.node_uuid}
            onChange={(e) => setForm({ ...form, node_uuid: e.target.value })}
            className="rounded-lg border border-line bg-canvas-overlay px-3 py-2 text-sm outline-none focus:border-accent/60"
          >
            <option value="">Select node</option>
            {nodes.map((n) => (
              <option key={n.uuid} value={n.uuid}>
                {n.name} ({n.status})
              </option>
            ))}
          </select>
          <select
            required
            value={form.template_uuid}
            onChange={(e) => setForm({ ...form, template_uuid: e.target.value })}
            className="rounded-lg border border-line bg-canvas-overlay px-3 py-2 text-sm outline-none focus:border-accent/60"
          >
            <option value="">Select template</option>
            {templates.map((t) => (
              <option key={t.uuid} value={t.uuid}>
                {t.name}
              </option>
            ))}
          </select>
          <input
            type="number"
            value={form.memory}
            onChange={(e) => setForm({ ...form, memory: Number(e.target.value) })}
            className="rounded-lg border border-line bg-canvas-overlay px-3 py-2 text-sm outline-none focus:border-accent/60"
            title="Memory MB"
          />
          <input
            type="number"
            value={form.cpu}
            onChange={(e) => setForm({ ...form, cpu: Number(e.target.value) })}
            className="rounded-lg border border-line bg-canvas-overlay px-3 py-2 text-sm outline-none focus:border-accent/60"
            title="CPU %"
          />
          <input
            type="number"
            value={form.disk}
            onChange={(e) => setForm({ ...form, disk: Number(e.target.value) })}
            className="rounded-lg border border-line bg-canvas-overlay px-3 py-2 text-sm outline-none focus:border-accent/60"
            title="Disk MB"
          />
          <button
            type="submit"
            disabled={busy === "create"}
            className="inline-flex items-center justify-center gap-2 rounded-lg bg-accent px-3 py-2 text-sm font-medium text-canvas disabled:opacity-60"
          >
            {busy === "create" ? <Loader2 className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />}
            Create
          </button>
        </form>
      ) : null}

      <div className="overflow-hidden rounded-2xl border border-line">
        <table className="w-full text-left text-sm">
          <thead className="bg-canvas-raised text-ink-muted">
            <tr>
              <th className="px-4 py-3 font-medium">Name</th>
              <th className="px-4 py-3 font-medium">Status</th>
              <th className="px-4 py-3 font-medium">RAM</th>
              <th className="px-4 py-3 font-medium">CPU</th>
              <th className="px-4 py-3 font-medium">Disk</th>
              <th className="px-4 py-3 font-medium">Actions</th>
            </tr>
          </thead>
          <tbody>
            {servers.length === 0 ? (
              <tr>
                <td colSpan={6} className="px-4 py-8 text-ink-faint">
                  No servers yet.
                </td>
              </tr>
            ) : (
              servers.map((s) => (
                <tr key={s.uuid} className="border-t border-line">
                  <td className="px-4 py-3">
                    <Link href={`/servers/${s.uuid}`} className="font-medium hover:text-accent">
                      {s.name}
                    </Link>
                    <div className="text-xs text-ink-faint">{s.container_name}</div>
                  </td>
                  <td className="px-4 py-3 text-xs font-medium tracking-wide">{s.status}</td>
                  <td className="px-4 py-3">{s.memory} MB</td>
                  <td className="px-4 py-3">{s.cpu}%</td>
                  <td className="px-4 py-3">{s.disk} MB</td>
                  <td className="px-4 py-3">
                    <div className="flex flex-wrap gap-1">
                      <IconBtn disabled={!!busy} onClick={() => power(s.uuid, "start")} title="Start">
                        <Play className="h-3.5 w-3.5" />
                      </IconBtn>
                      <IconBtn disabled={!!busy} onClick={() => power(s.uuid, "stop")} title="Stop">
                        <Square className="h-3.5 w-3.5" />
                      </IconBtn>
                      <IconBtn disabled={!!busy} onClick={() => power(s.uuid, "restart")} title="Restart">
                        <RotateCcw className="h-3.5 w-3.5" />
                      </IconBtn>
                      <IconBtn disabled={!!busy} onClick={() => power(s.uuid, "kill")} title="Kill">
                        <Skull className="h-3.5 w-3.5" />
                      </IconBtn>
                      <IconBtn disabled={!!busy} onClick={() => onDelete(s.uuid)} title="Delete">
                        <Trash2 className="h-3.5 w-3.5" />
                      </IconBtn>
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

function IconBtn({
  children,
  onClick,
  disabled,
  title,
}: {
  children: React.ReactNode;
  onClick: () => void;
  disabled?: boolean;
  title: string;
}) {
  return (
    <button
      type="button"
      title={title}
      disabled={disabled}
      onClick={onClick}
      className="rounded-lg border border-line p-2 text-ink-muted transition hover:bg-canvas-overlay hover:text-ink disabled:opacity-40"
    >
      {children}
    </button>
  );
}
