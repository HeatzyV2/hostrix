import { getMe } from "@/lib/auth";
import { DashboardStats } from "@/components/dashboard-stats";

export default async function DashboardPage() {
  const me = await getMe();

  return (
    <div className="mx-auto max-w-5xl space-y-8 animate-fade-up">
      <header className="space-y-2">
        <h1 className="font-display text-3xl font-semibold tracking-tight">
          Dashboard
        </h1>
        <p className="text-ink-muted">
          Welcome back, {me?.username}. Values below come from live node heartbeats when available.
        </p>
      </header>

      <DashboardStats isAdmin={!!me?.is_admin} />
    </div>
  );
}
