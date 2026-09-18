"use client";

import { useCallback, useEffect, useState } from "react";
import { Loader2 } from "lucide-react";

type UserRow = {
  uuid: string;
  username: string;
  email: string;
  is_admin: boolean;
};

export function UsersAdmin() {
  const [users, setUsers] = useState<UserRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    const res = await fetch("/api/v1/users", { credentials: "include" });
    const data = await res.json().catch(() => ({}));
    if (!res.ok) {
      setError(data.error || "Failed to load users");
      setLoading(false);
      return;
    }
    setUsers(data.users || []);
    setLoading(false);
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  if (loading) {
    return (
      <div className="flex items-center gap-2 text-ink-muted">
        <Loader2 className="h-4 w-4 animate-spin" /> Loading users…
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {error ? <p className="text-sm text-red-400">{error}</p> : null}
      <p className="text-sm text-ink-muted">
        Manage server shares from each server&apos;s Access section. User creation remains bootstrap / installer for now.
      </p>
      <div className="overflow-hidden rounded-2xl border border-line">
        <table className="w-full text-left text-sm">
          <thead className="bg-canvas-raised text-ink-muted">
            <tr>
              <th className="px-4 py-3 font-medium">Username</th>
              <th className="px-4 py-3 font-medium">Email</th>
              <th className="px-4 py-3 font-medium">Role</th>
            </tr>
          </thead>
          <tbody>
            {users.map((u) => (
              <tr key={u.uuid} className="border-t border-line">
                <td className="px-4 py-3 font-medium">{u.username}</td>
                <td className="px-4 py-3 text-ink-muted">{u.email}</td>
                <td className="px-4 py-3 text-xs tracking-wide">
                  {u.is_admin ? "ADMIN" : "USER"}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
