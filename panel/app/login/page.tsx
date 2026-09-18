import { LoginForm } from "@/components/login-form";
import { getMe } from "@/lib/auth";
import { redirect } from "next/navigation";

export default async function LoginPage() {
  const me = await getMe();
  if (me) redirect("/dashboard");

  return (
    <main className="relative flex min-h-screen items-center justify-center px-6">
      <div className="absolute inset-0 overflow-hidden pointer-events-none">
        <div className="absolute left-1/2 top-1/3 h-64 w-64 -translate-x-1/2 rounded-full bg-accent/10 blur-3xl animate-pulse-soft" />
      </div>
      <div className="relative w-full max-w-md space-y-8 rounded-2xl border border-line bg-canvas-raised/80 p-8 backdrop-blur">
        <div className="space-y-2 text-center">
          <p className="font-display text-3xl font-semibold tracking-tight text-ink">
            HOSTRIX
          </p>
          <p className="text-sm text-ink-muted">Your Infrastructure. Simplified.</p>
        </div>
        <LoginForm />
      </div>
    </main>
  );
}
