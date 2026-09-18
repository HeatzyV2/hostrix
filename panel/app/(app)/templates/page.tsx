import { TemplatesPanel } from "@/components/templates-panel";

export default function TemplatesPage() {
  return (
    <div className="mx-auto max-w-5xl space-y-6 animate-fade-up">
      <div>
        <h1 className="font-display text-3xl font-semibold tracking-tight">Templates</h1>
        <p className="mt-1 text-ink-muted">
          Service stacks synced from YAML under <code className="text-ink">templates/</code>. Used when
          creating servers.
        </p>
      </div>
      <TemplatesPanel />
    </div>
  );
}
