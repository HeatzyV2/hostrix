import { UsersAdmin } from "@/components/users-admin";

export default function UsersPage() {
  return (
    <div className="mx-auto max-w-5xl space-y-6 animate-fade-up">
      <div>
        <h1 className="font-display text-3xl font-semibold tracking-tight">Users</h1>
        <p className="mt-1 text-ink-muted">Accounts on this Hostrix panel.</p>
      </div>
      <UsersAdmin />
    </div>
  );
}
