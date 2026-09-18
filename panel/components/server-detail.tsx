"use client";

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import {
  ArrowLeft,
  Loader2,
  Play,
  Square,
  RotateCcw,
  Skull,
  Trash2,
} from "lucide-react";
import { ServerConsole } from "@/components/server-console";
import { ServerMetrics } from "@/components/server-metrics";
import { ServerBackups } from "@/components/server-backups";
import { ServerPermissions } from "@/components/server-permissions";

type Server = {
  uuid: string;
  name: string;
  status: string;
  memory: number;
  cpu: number;
  disk: number;
  container_name: string;
  node_uuid?: string;
  node_name?: string;
  node_status?: string;
};

export function ServerDetail({ isAdmin }: { isAdmin: boolean }) {
  const params = useParams<{ id: string }>();
  const id = params.id;
  const router = useRouter();
  const [server, setServer] = useState<Server | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    const res = await fetch(`/api/v1/servers/${id}`, { credentials: "include" });
    const data = await res.json().catch(() => ({}));
    if (!res.ok) {
      setError(data.error || "Failed to load server");
      return;
    }
    setServer(data.server);
    setError(null);
  }, [id]);

  useEffect(() => {
    load();
  }, [load]);

  async function power(action: string) {
    setBusy(true);
    try {
      const res = await fetch(`/api/v1/servers/${id}/${action}`, {
        method: "POST",
        credentials: "include",
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) setError(data.error || `${action} failed`);
      await load();
    } finally {
      setBusy(false);
    }
  }

  async function onDelete() {
    if (!confirm("Delete this server and container?")) return;
    setBusy(true);
    try {
      const res = await fetch(`/api/v1/servers/${id}`, {
        method: "DELETE",
        credentials: "include",
      });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        setError(data.error || "Delete failed");
        return;
      }
      router.push("/servers");
    } finally {
      setBusy(false);
    }
  }

  if (!server && !error) {
    return (
      <div className="flex items-center gap-2 text-ink-muted">
        <Loader2 className="h-4 w-4 animate-spin" /> Loading…
      </div>
    );
  }

  if (!server) {
    return <p className="text-red-400">{error || "Not found"}</p>;
  }

  return (
    <div className="space-y-8 animate-fade-up">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div className="space-y-2">
          <Link
            href="/servers"
            className="inline-flex items-center gap-2 text-sm text-ink-muted hover:text-ink"
          >
            <ArrowLeft className="h-4 w-4" /> Servers
          </Link>
          <h1 className="font-display text-3xl font-semibold tracking-tight">{server.name}</h1>
          <p className="text-sm text-ink-muted">
            {server.container_name} · <span className="tracking-wide">{server.status}</span>
            {server.node_name ? (
              <>
                {" "}
                · {server.node_name}{" "}
                <span className="text-ink-faint">({server.node_status})</span>
              </>
            ) : null}
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          <Action disabled={busy} onClick={() => power("start")} icon={<Play className="h-4 w-4" />} label="Start" />
          <Action disabled={busy} onClick={() => power("stop")} icon={<Square className="h-4 w-4" />} label="Stop" />
          <Action disabled={busy} onClick={() => power("restart")} icon={<RotateCcw className="h-4 w-4" />} label="Restart" />
          <Action disabled={busy} onClick={() => power("kill")} icon={<Skull className="h-4 w-4" />} label="Kill" />
          <Action disabled={busy} onClick={onDelete} icon={<Trash2 className="h-4 w-4" />} label="Delete" danger />
        </div>
      </div>

      {error ? <p className="text-sm text-red-400">{error}</p> : null}

      <div className="grid gap-3 sm:grid-cols-3">
        <Info label="Memory" value={`${server.memory} MB`} />
        <Info label="CPU" value={`${server.cpu}%`} />
        <Info label="Disk" value={`${server.disk} MB`} />
      </div>

      <ServerMetrics serverId={server.uuid} />
      <ServerConsole serverId={server.uuid} />
      <ServerBackups serverId={server.uuid} />
      <ServerPermissions serverId={server.uuid} isAdmin={isAdmin} />
    </div>
  );
}

function Info({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-2xl border border-line bg-canvas-raised/60 p-4">
      <p className="text-xs text-ink-muted">{label}</p>
      <p className="mt-1 font-medium">{value}</p>
    </div>
  );
}

function Action({
  label,
  icon,
  onClick,
  disabled,
  danger,
}: {
  label: string;
  icon: React.ReactNode;
  onClick: () => void;
  disabled?: boolean;
  danger?: boolean;
}) {
  return (
    <button
      type="button"
      disabled={disabled}
      onClick={onClick}
      className={`inline-flex items-center gap-2 rounded-lg border px-3 py-2 text-sm transition disabled:opacity-40 ${
        danger
          ? "border-red-500/30 text-red-300 hover:bg-red-500/10"
          : "border-line text-ink-muted hover:bg-canvas-overlay hover:text-ink"
      }`}
    >
      {icon}
      {label}
    </button>
  );
}
