import { BackupsPanel } from "@/components/backups-panel";

export default function BackupsPage() {
  return (
    <div className="mx-auto max-w-5xl space-y-6 animate-fade-up">
      <div>
        <h1 className="font-display text-3xl font-semibold tracking-tight">Backups</h1>
        <p className="mt-1 text-ink-muted">All container backups across your accessible servers.</p>
      </div>
      <BackupsPanel />
    </div>
  );
}
