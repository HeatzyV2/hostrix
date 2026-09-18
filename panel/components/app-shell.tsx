"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import {
  LayoutDashboard,
  Server,
  Network,
  Boxes,
  Users,
  Archive,
  Settings,
  LogOut,
  UserRound,
} from "lucide-react";
import { cn } from "@/lib/utils";
import type { User } from "@/types";

const adminNav = [
  { href: "/dashboard", label: "Dashboard", icon: LayoutDashboard },
  { href: "/servers", label: "Servers", icon: Server },
  { href: "/nodes", label: "Nodes", icon: Network },
  { href: "/templates", label: "Templates", icon: Boxes },
  { href: "/users", label: "Users", icon: Users },
  { href: "/backups", label: "Backups", icon: Archive },
  { href: "/settings", label: "Settings", icon: Settings },
];

const userNav = [
  { href: "/dashboard", label: "Dashboard", icon: LayoutDashboard },
  { href: "/servers", label: "Servers", icon: Server },
  { href: "/account", label: "Account", icon: UserRound },
];

export function AppShell({
  user,
  children,
}: {
  user: User;
  children: React.ReactNode;
}) {
  const pathname = usePathname();
  const router = useRouter();
  const items = user.is_admin ? adminNav : userNav;

  async function logout() {
    await fetch("/api/v1/auth/logout", {
      method: "POST",
      credentials: "include",
    });
    router.replace("/login");
    router.refresh();
  }

  return (
    <div className="flex min-h-screen">
      <aside className="sticky top-0 flex h-screen w-60 flex-col border-r border-line bg-canvas-raised/70 px-4 py-6 backdrop-blur">
        <div className="mb-8 px-2">
          <p className="font-display text-xl font-semibold tracking-tight">HOSTRIX</p>
          <p className="mt-1 text-xs text-ink-faint">Infrastructure panel</p>
        </div>
        <nav className="flex flex-1 flex-col gap-1">
          {items.map((item) => {
            const active = pathname === item.href || pathname.startsWith(item.href + "/");
            const Icon = item.icon;
            return (
              <Link
                key={item.href}
                href={item.href}
                className={cn(
                  "flex items-center gap-3 rounded-lg px-3 py-2 text-sm transition",
                  active
                    ? "bg-accent/15 text-ink"
                    : "text-ink-muted hover:bg-canvas-overlay hover:text-ink"
                )}
              >
                <Icon className="h-4 w-4" />
                {item.label}
              </Link>
            );
          })}
        </nav>
        <div className="mt-auto space-y-3 border-t border-line pt-4">
          <div className="px-2">
            <p className="text-sm font-medium">{user.username}</p>
            <p className="text-xs text-ink-faint">
              {user.is_admin ? "Administrator" : "User"}
            </p>
          </div>
          <button
            type="button"
            onClick={logout}
            className="flex w-full items-center gap-3 rounded-lg px-3 py-2 text-sm text-ink-muted transition hover:bg-canvas-overlay hover:text-ink"
          >
            <LogOut className="h-4 w-4" />
            Sign out
          </button>
        </div>
      </aside>
      <main className="flex-1 px-8 py-8">{children}</main>
    </div>
  );
}
