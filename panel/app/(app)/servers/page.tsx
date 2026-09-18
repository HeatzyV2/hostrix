import { ServersPanel } from "@/components/servers-panel";
import { getMe } from "@/lib/auth";

export default async function ServersPage() {
  const me = await getMe();
  return (
    <div className="mx-auto max-w-6xl space-y-6 animate-fade-up">
      <header className="space-y-2">
        <h1 className="font-display text-3xl font-semibold tracking-tight">Servers</h1>
        <p className="text-ink-muted">
          Create and control LXC containers through Hostrix Agents.
        </p>
      </header>
      <ServersPanel isAdmin={!!me?.is_admin} />
    </div>
  );
}
