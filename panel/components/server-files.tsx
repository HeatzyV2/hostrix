"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  Folder,
  File,
  Upload,
  Download,
  FolderPlus,
  Trash2,
  Pencil,
  FilePenLine,
  RefreshCw,
  ChevronRight,
  Loader2,
  X,
  Save,
  FileArchive,
} from "lucide-react";

type DirEntry = {
  name: string;
  type: string;
  mode: number;
  size: number;
};

type Props = {
  serverId: string;
};

const TEXT_EXT = new Set([
  "txt",
  "md",
  "json",
  "yml",
  "yaml",
  "toml",
  "ini",
  "cfg",
  "conf",
  "properties",
  "xml",
  "html",
  "htm",
  "css",
  "js",
  "ts",
  "tsx",
  "jsx",
  "go",
  "py",
  "sh",
  "bash",
  "env",
  "log",
  "sql",
  "service",
  "php",
  "rb",
  "rs",
  "c",
  "h",
  "cpp",
  "hpp",
  "java",
  "kt",
  "swift",
  "dockerfile",
]);

const ARCHIVE_EXT = new Set(["zip", "tar", "gz", "tgz", "bz2", "xz", "tbz2", "txz"]);

function joinPath(base: string, name: string) {
  if (base === "/" || base === "") return `/${name}`;
  return `${base.replace(/\/+$/, "")}/${name}`;
}

function parentPath(p: string) {
  if (p === "/" || p === "") return "/";
  const trimmed = p.replace(/\/+$/, "");
  const idx = trimmed.lastIndexOf("/");
  if (idx <= 0) return "/";
  return trimmed.slice(0, idx);
}

function extOf(name: string) {
  const lower = name.toLowerCase();
  if (lower.endsWith(".tar.gz")) return "tar.gz";
  if (lower.endsWith(".tar.bz2")) return "tar.bz2";
  if (lower.endsWith(".tar.xz")) return "tar.xz";
  const i = lower.lastIndexOf(".");
  return i >= 0 ? lower.slice(i + 1) : "";
}

function isTextFile(name: string) {
  const e = extOf(name);
  return TEXT_EXT.has(e) || name.toLowerCase() === "dockerfile" || name.toLowerCase() === "makefile";
}

function isArchive(name: string) {
  const e = extOf(name);
  return ARCHIVE_EXT.has(e) || name.toLowerCase().endsWith(".tar.gz");
}

function formatSize(n: number) {
  if (!n || n < 0) return "—";
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / (1024 * 1024)).toFixed(1)} MB`;
}

export function ServerFiles({ serverId }: Props) {
  const [path, setPath] = useState("/");
  const [entries, setEntries] = useState<DirEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [editor, setEditor] = useState<{ path: string; content: string } | null>(null);
  const [editingName, setEditingName] = useState<string | null>(null);
  const [renameValue, setRenameValue] = useState("");
  const fileInputRef = useRef<HTMLInputElement>(null);

  const crumbs = useMemo(() => {
    const parts = path === "/" ? [] : path.replace(/^\/+|\/+$/g, "").split("/");
    const items = [{ label: "/", path: "/" }];
    let acc = "";
    for (const part of parts) {
      acc += `/${part}`;
      items.push({ label: part, path: acc });
    }
    return items;
  }, [path]);

  const load = useCallback(async (dir = path) => {
    setLoading(true);
    try {
      const res = await fetch(
        `/api/v1/servers/${serverId}/files?path=${encodeURIComponent(dir)}`,
        { credentials: "include" }
      );
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        setError(data.error || "Failed to list files");
        return;
      }
      const list: DirEntry[] = Array.isArray(data.entries) ? data.entries : [];
      list.sort((a, b) => {
        if (a.type === "directory" && b.type !== "directory") return -1;
        if (a.type !== "directory" && b.type === "directory") return 1;
        return a.name.localeCompare(b.name);
      });
      setEntries(list);
      setPath(data.path || dir);
      setError(null);
    } finally {
      setLoading(false);
    }
  }, [path, serverId]);

  useEffect(() => {
    load("/");
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [serverId]);

  async function apiJSON(url: string, init?: RequestInit) {
    setBusy(true);
    try {
      const res = await fetch(url, { credentials: "include", ...init });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        setError(data.error || "Request failed");
        return false;
      }
      setError(null);
      return true;
    } finally {
      setBusy(false);
    }
  }

  async function openDir(name: string) {
    await load(joinPath(path, name));
  }

  async function createFolder() {
    const name = prompt("Folder name");
    if (!name?.trim()) return;
    const ok = await apiJSON(`/api/v1/servers/${serverId}/files/mkdir`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ path: joinPath(path, name.trim()) }),
    });
    if (ok) await load(path);
  }

  async function removeEntry(entry: DirEntry) {
    if (!confirm(`Delete ${entry.name}?`)) return;
    const ok = await apiJSON(
      `/api/v1/servers/${serverId}/files?path=${encodeURIComponent(joinPath(path, entry.name))}`,
      { method: "DELETE" }
    );
    if (ok) await load(path);
  }

  async function commitRename(entry: DirEntry) {
    const next = renameValue.trim();
    setEditingName(null);
    if (!next || next === entry.name) return;
    const ok = await apiJSON(`/api/v1/servers/${serverId}/files/rename`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        from: joinPath(path, entry.name),
        to: joinPath(path, next),
      }),
    });
    if (ok) await load(path);
  }

  async function uploadFiles(files: FileList | null) {
    if (!files?.length) return;
    setBusy(true);
    try {
      for (const file of Array.from(files)) {
        const form = new FormData();
        form.set("path", joinPath(path, file.name));
        form.set("file", file);
        const res = await fetch(`/api/v1/servers/${serverId}/files/upload`, {
          method: "POST",
          credentials: "include",
          body: form,
        });
        const data = await res.json().catch(() => ({}));
        if (!res.ok) {
          setError(data.error || `Upload failed: ${file.name}`);
          return;
        }
      }
      setError(null);
      await load(path);
    } finally {
      setBusy(false);
      if (fileInputRef.current) fileInputRef.current.value = "";
    }
  }

  async function downloadEntry(entry: DirEntry) {
    const filePath = joinPath(path, entry.name);
    const res = await fetch(
      `/api/v1/servers/${serverId}/files/download?path=${encodeURIComponent(filePath)}`,
      { credentials: "include" }
    );
    if (!res.ok) {
      const data = await res.json().catch(() => ({}));
      setError(data.error || "Download failed");
      return;
    }
    const blob = await res.blob();
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = entry.name;
    a.click();
    URL.revokeObjectURL(url);
  }

  async function openEditor(entry: DirEntry) {
    if (entry.size > 1024 * 1024) {
      setError("File too large to edit in browser (max 1MB)");
      return;
    }
    setBusy(true);
    try {
      const filePath = joinPath(path, entry.name);
      const res = await fetch(
        `/api/v1/servers/${serverId}/files/download?path=${encodeURIComponent(filePath)}`,
        { credentials: "include" }
      );
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        setError(data.error || "Failed to open file");
        return;
      }
      const buf = await res.arrayBuffer();
      const text = new TextDecoder("utf-8", { fatal: false }).decode(buf);
      setEditor({ path: filePath, content: text });
      setError(null);
    } finally {
      setBusy(false);
    }
  }

  async function saveEditor() {
    if (!editor) return;
    const ok = await apiJSON(`/api/v1/servers/${serverId}/files/write`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ path: editor.path, content: editor.content }),
    });
    if (ok) {
      setEditor(null);
      await load(path);
    }
  }

  async function extractEntry(entry: DirEntry) {
    if (!confirm(`Extract ${entry.name} into ${path}?`)) return;
    const ok = await apiJSON(`/api/v1/servers/${serverId}/files/extract`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        archive: joinPath(path, entry.name),
        dest: path,
      }),
    });
    if (ok) await load(path);
  }

  return (
    <section className="space-y-4 animate-fade-up">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 className="font-display text-lg font-semibold tracking-tight">Files</h2>
          <p className="text-sm text-ink-muted">Browse and edit files inside the container</p>
        </div>
        <div className="flex flex-wrap gap-2">
          <ToolBtn
            disabled={busy}
            onClick={() => load(path)}
            icon={<RefreshCw className={`h-4 w-4 ${loading ? "animate-spin" : ""}`} />}
            label="Refresh"
          />
          <ToolBtn disabled={busy} onClick={createFolder} icon={<FolderPlus className="h-4 w-4" />} label="New folder" />
          <ToolBtn
            disabled={busy}
            onClick={() => fileInputRef.current?.click()}
            icon={<Upload className="h-4 w-4" />}
            label="Upload"
          />
          <input
            ref={fileInputRef}
            type="file"
            className="hidden"
            multiple
            onChange={(e) => uploadFiles(e.target.files)}
          />
        </div>
      </div>

      <nav className="flex flex-wrap items-center gap-1 text-sm text-ink-muted">
        {crumbs.map((c, i) => (
          <span key={c.path} className="inline-flex items-center gap-1">
            {i > 0 ? <ChevronRight className="h-3.5 w-3.5 text-ink-faint" /> : null}
            <button
              type="button"
              className="rounded px-1.5 py-0.5 hover:bg-canvas-overlay hover:text-ink"
              onClick={() => load(c.path)}
            >
              {c.label}
            </button>
          </span>
        ))}
      </nav>

      {error ? <p className="text-sm text-red-400">{error}</p> : null}

      <div className="overflow-hidden rounded-2xl border border-line bg-canvas-raised/50">
        {loading ? (
          <div className="flex items-center gap-2 px-4 py-10 text-ink-muted">
            <Loader2 className="h-4 w-4 animate-spin" /> Loading…
          </div>
        ) : (
          <table className="w-full text-left text-sm">
            <thead className="border-b border-line text-xs uppercase tracking-wide text-ink-faint">
              <tr>
                <th className="px-4 py-3 font-medium">Name</th>
                <th className="hidden px-4 py-3 font-medium sm:table-cell">Size</th>
                <th className="px-4 py-3 font-medium text-right">Actions</th>
              </tr>
            </thead>
            <tbody>
              {path !== "/" ? (
                <tr className="border-b border-line/60 hover:bg-canvas-overlay/40">
                  <td className="px-4 py-2.5" colSpan={3}>
                    <button
                      type="button"
                      className="inline-flex items-center gap-2 text-ink-muted hover:text-ink"
                      onClick={() => load(parentPath(path))}
                    >
                      <Folder className="h-4 w-4 text-accent" /> ..
                    </button>
                  </td>
                </tr>
              ) : null}
              {entries.length === 0 ? (
                <tr>
                  <td className="px-4 py-8 text-ink-muted" colSpan={3}>
                    Empty directory
                  </td>
                </tr>
              ) : (
                entries.map((entry) => {
                  const isDir = entry.type === "directory";
                  return (
                    <tr key={entry.name} className="border-b border-line/60 hover:bg-canvas-overlay/40">
                      <td className="px-4 py-2.5">
                        {editingName === entry.name ? (
                          <form
                            className="flex items-center gap-2"
                            onSubmit={(e) => {
                              e.preventDefault();
                              commitRename(entry);
                            }}
                          >
                            <input
                              autoFocus
                              value={renameValue}
                              onChange={(e) => setRenameValue(e.target.value)}
                              onBlur={() => commitRename(entry)}
                              className="w-full max-w-xs rounded-lg border border-line bg-canvas px-2 py-1 text-ink outline-none focus:border-accent"
                            />
                          </form>
                        ) : (
                          <button
                            type="button"
                            className="inline-flex max-w-full items-center gap-2 truncate text-left hover:text-accent"
                            onClick={() => {
                              if (isDir) openDir(entry.name);
                              else if (isTextFile(entry.name)) openEditor(entry);
                            }}
                          >
                            {isDir ? (
                              <Folder className="h-4 w-4 shrink-0 text-accent" />
                            ) : (
                              <File className="h-4 w-4 shrink-0 text-ink-muted" />
                            )}
                            <span className="truncate">{entry.name}</span>
                          </button>
                        )}
                      </td>
                      <td className="hidden px-4 py-2.5 text-ink-muted sm:table-cell">
                        {isDir ? "—" : formatSize(entry.size)}
                      </td>
                      <td className="px-4 py-2.5">
                        <div className="flex justify-end gap-1">
                          {!isDir ? (
                            <IconBtn title="Download" onClick={() => downloadEntry(entry)} icon={<Download className="h-3.5 w-3.5" />} />
                          ) : null}
                          {!isDir && isTextFile(entry.name) ? (
                            <IconBtn title="Edit" onClick={() => openEditor(entry)} icon={<FilePenLine className="h-3.5 w-3.5" />} />
                          ) : null}
                          {!isDir && isArchive(entry.name) ? (
                            <IconBtn title="Extract" onClick={() => extractEntry(entry)} icon={<FileArchive className="h-3.5 w-3.5" />} />
                          ) : null}
                          <IconBtn
                            title="Rename"
                            onClick={() => {
                              setEditingName(entry.name);
                              setRenameValue(entry.name);
                            }}
                            icon={<Pencil className="h-3.5 w-3.5" />}
                          />
                          <IconBtn title="Delete" danger onClick={() => removeEntry(entry)} icon={<Trash2 className="h-3.5 w-3.5" />} />
                        </div>
                      </td>
                    </tr>
                  );
                })
              )}
            </tbody>
          </table>
        )}
      </div>

      {editor ? (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4 backdrop-blur-sm">
          <div className="flex max-h-[90vh] w-full max-w-3xl flex-col overflow-hidden rounded-2xl border border-line bg-canvas-raised shadow-glow">
            <div className="flex items-center justify-between gap-3 border-b border-line px-4 py-3">
              <div className="min-w-0">
                <p className="truncate font-medium">{editor.path}</p>
                <p className="text-xs text-ink-muted">Text editor · max 1MB</p>
              </div>
              <div className="flex gap-2">
                <ToolBtn disabled={busy} onClick={saveEditor} icon={<Save className="h-4 w-4" />} label="Save" />
                <button
                  type="button"
                  className="rounded-lg border border-line p-2 text-ink-muted hover:bg-canvas-overlay hover:text-ink"
                  onClick={() => setEditor(null)}
                >
                  <X className="h-4 w-4" />
                </button>
              </div>
            </div>
            <textarea
              value={editor.content}
              onChange={(e) => setEditor({ ...editor, content: e.target.value })}
              className="min-h-[50vh] flex-1 resize-none bg-canvas px-4 py-3 font-mono text-sm text-ink outline-none"
              spellCheck={false}
            />
          </div>
        </div>
      ) : null}
    </section>
  );
}

function ToolBtn({
  label,
  icon,
  onClick,
  disabled,
}: {
  label: string;
  icon: React.ReactNode;
  onClick: () => void;
  disabled?: boolean;
}) {
  return (
    <button
      type="button"
      disabled={disabled}
      onClick={onClick}
      className="inline-flex items-center gap-2 rounded-lg border border-line px-3 py-2 text-sm text-ink-muted transition hover:bg-canvas-overlay hover:text-ink disabled:opacity-40"
    >
      {icon}
      {label}
    </button>
  );
}

function IconBtn({
  icon,
  onClick,
  title,
  danger,
}: {
  icon: React.ReactNode;
  onClick: () => void;
  title: string;
  danger?: boolean;
}) {
  return (
    <button
      type="button"
      title={title}
      onClick={onClick}
      className={`rounded-md border border-transparent p-1.5 transition ${
        danger
          ? "text-ink-muted hover:border-red-500/30 hover:bg-red-500/10 hover:text-red-300"
          : "text-ink-muted hover:border-line hover:bg-canvas-overlay hover:text-ink"
      }`}
    >
      {icon}
    </button>
  );
}
