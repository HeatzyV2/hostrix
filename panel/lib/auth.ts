export type User = {
  uuid: string;
  username: string;
  email: string;
  is_admin: boolean;
};

const API_BASE =
  process.env.HOSTRIX_API_INTERNAL_URL ||
  process.env.HOSTRIX_API_URL ||
  "http://127.0.0.1:8080";

export async function getMe(): Promise<User | null> {
  try {
    const { cookies } = await import("next/headers");
    const cookieStore = await cookies();
    const session = cookieStore.get("hostrix_session");
    if (!session?.value) return null;

    const res = await fetch(`${API_BASE}/api/v1/auth/me`, {
      headers: {
        Cookie: `hostrix_session=${session.value}`,
      },
      cache: "no-store",
    });
    if (!res.ok) return null;
    const data = (await res.json()) as { user: User };
    return data.user;
  } catch {
    return null;
  }
}
