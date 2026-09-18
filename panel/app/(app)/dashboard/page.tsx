import { getMe } from "@/lib/auth";

export default async function DashboardPage() {
  const me = await getMe();

  return (
    <div className="mx-auto max-w-5xl space-y-8 animate-fade-up">
      <header className="space-y-2">
        <h1 className="font-display text-3xl font-semibold tracking-tight">
          Dashboard
        </h1>
        <p className="text-ink-muted">
          Welcome back, {me?.username}. Metrics appear here once Nodes report real data.
        </p>
      </header>

      <section className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <EmptyMetric title="CPU" hint="Awaiting node heartbeat (Phase 2)" />
        <EmptyMetric title="Memory" hint="Awaiting node heartbeat (Phase 2)" />
        <EmptyMetric title="Storage" hint="Awaiting node heartbeat (Phase 2)" />
        <EmptyMetric title="Servers" hint="No servers yet" value="—" />
        <EmptyMetric title="Nodes" hint="No nodes yet" value="—" />
      </section>

      <section className="rounded-2xl border border-line bg-canvas-raised/60 p-6">
        <h2 className="font-display text-lg font-medium">Getting started</h2>
        <p className="mt-2 max-w-2xl text-sm leading-relaxed text-ink-muted">
          Phase 1 is online: authentication, MariaDB schema, and this panel shell.
          Incus container management, Agents, and live metrics arrive in Phase 2.
        </p>
      </section>
    </div>
  );
}

function EmptyMetric({
  title,
  hint,
  value = "—",
}: {
  title: string;
  hint: string;
  value?: string;
}) {
  return (
    <div className="rounded-2xl border border-line bg-canvas-raised/60 p-5">
      <p className="text-sm text-ink-muted">{title}</p>
      <p className="mt-3 font-display text-3xl font-semibold tracking-tight">{value}</p>
      <p className="mt-2 text-xs text-ink-faint">{hint}</p>
    </div>
  );
}
