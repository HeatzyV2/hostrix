"use client";

import { useCallback, useEffect, useState } from "react";
import { Loader2 } from "lucide-react";

type Template = {
  uuid: string;
  name: string;
  slug: string;
  description: string;
  image: string;
  startup_command: string;
};

export function TemplatesPanel() {
  const [templates, setTemplates] = useState<Template[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setError(null);
    const res = await fetch("/api/v1/templates", { credentials: "include" });
    const data = await res.json().catch(() => ({}));
    if (!res.ok) {
      setError(data.error || "Failed to load templates");
      setLoading(false);
      return;
    }
    setTemplates(data.templates || []);
    setLoading(false);
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  if (loading) {
    return (
      <div className="flex items-center gap-2 text-ink-muted">
        <Loader2 className="h-4 w-4 animate-spin" /> Loading templates…
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {error ? <p className="text-sm text-red-400">{error}</p> : null}

      <div className="overflow-hidden rounded-2xl border border-line">
        <table className="w-full text-left text-sm">
          <thead className="bg-canvas-raised text-ink-muted">
            <tr>
              <th className="px-4 py-3 font-medium">Name</th>
              <th className="px-4 py-3 font-medium">Slug</th>
              <th className="px-4 py-3 font-medium">Image</th>
              <th className="px-4 py-3 font-medium">Startup</th>
            </tr>
          </thead>
          <tbody>
            {templates.length === 0 ? (
              <tr>
                <td colSpan={4} className="px-4 py-8 text-ink-faint">
                  No templates synced yet. Ensure the API can read{" "}
                  <code className="text-ink-muted">HOSTRIX_TEMPLATES_DIR</code>.
                </td>
              </tr>
            ) : (
              templates.map((t) => (
                <tr key={t.uuid} className="border-t border-line align-top">
                  <td className="px-4 py-3">
                    <div className="font-medium">{t.name}</div>
                    {t.description ? (
                      <div className="mt-0.5 text-xs text-ink-faint">{t.description}</div>
                    ) : null}
                  </td>
                  <td className="px-4 py-3 font-mono text-xs text-ink-muted">{t.slug}</td>
                  <td className="px-4 py-3 font-mono text-xs">{t.image}</td>
                  <td className="max-w-md px-4 py-3 font-mono text-xs text-ink-muted">
                    {t.startup_command || "—"}
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
