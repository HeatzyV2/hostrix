function Placeholder({
  title,
  body,
}: {
  title: string;
  body: string;
}) {
  return (
    <div className="mx-auto max-w-5xl space-y-3 animate-fade-up">
      <h1 className="font-display text-3xl font-semibold tracking-tight">{title}</h1>
      <p className="text-ink-muted">{body}</p>
    </div>
  );
}

export default function ServersPage() {
  return (
    <Placeholder
      title="Servers"
      body="Server lifecycle management lands in Phase 2–3 (Incus + UI)."
    />
  );
}
