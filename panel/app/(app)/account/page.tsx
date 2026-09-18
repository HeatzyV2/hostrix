import { getMe } from "@/lib/auth";

export default async function AccountPage() {
  const me = await getMe();
  return (
    <div className="mx-auto max-w-5xl space-y-6 animate-fade-up">
      <h1 className="font-display text-3xl font-semibold tracking-tight">Account</h1>
      <div className="rounded-2xl border border-line bg-canvas-raised/60 p-6 space-y-3">
        <Row label="Username" value={me?.username ?? "—"} />
        <Row label="Email" value={me?.email ?? "—"} />
        <Row label="Role" value={me?.is_admin ? "Administrator" : "User"} />
      </div>
    </div>
  );
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between gap-4 border-b border-line py-3 last:border-0">
      <span className="text-sm text-ink-muted">{label}</span>
      <span className="text-sm font-medium">{value}</span>
    </div>
  );
}
