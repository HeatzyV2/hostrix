import { NodesAdmin } from "@/components/nodes-admin";

export default function NodesPage() {
  return (
    <div className="mx-auto max-w-6xl space-y-6 animate-fade-up">
      <header className="space-y-2">
        <h1 className="font-display text-3xl font-semibold tracking-tight">Nodes</h1>
        <p className="text-ink-muted">
          Register Agents, monitor heartbeat status, and manage node connectivity.
        </p>
      </header>
      <NodesAdmin />
    </div>
  );
}
